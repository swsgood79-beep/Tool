package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const manualFileLimit int64 = 300 << 20

type ManualPost struct {
	ID, Title, Text, Author, Owner string
	CreatedAt                      time.Time
	Files                          []ShippingAttachment
}

func manualParts(s State) []Photo {
	var refs []Photo
	for _, p := range s.Manuals {
		for _, f := range p.Files {
			refs = append(refs, f.Parts...)
		}
	}
	return refs
}
func (a *App) manualHTTP(w http.ResponseWriter, r *http.Request) bool {
	p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	admin := len(p) > 0 && p[0] == "manuals-admin"
	if len(p) == 0 || p[0] != "manuals" && !admin {
		return false
	}
	a.mu.Lock()
	epoch := a.epoch
	board := a.state.Board
	session, logged := a.session(r)
	if logged && purchasingSession(&a.state, session) {
		a.mu.Unlock()
		http.Error(w, "구매 계정은 호출 업무만 사용할 수 있습니다.", 403)
		return true
	}
	base, offset := "/manuals/"+board, 2
	csrf := board
	author := ""
	if admin {
		base = "/manuals-admin"
		offset = 1
		if !logged {
			if r.Method == "GET" {
				a.admin(w, r)
			} else {
				http.Error(w, "관리자 로그인이 필요합니다.", 401)
			}
			a.mu.Unlock()
			return true
		}
		if !localRequest(r) && !a.state.Integration.MobileEnabled {
			a.mu.Unlock()
			http.Error(w, "휴대폰 관리자 접속이 꺼져 있습니다.", 403)
			return true
		}
		csrf = session.CSRF
		actor, e := inspectionActor(&a.state, session)
		if e != nil {
			a.mu.Unlock()
			http.Error(w, e.Error(), 403)
			return true
		}
		author = actor.Name
	} else if len(p) < 2 || !equal(p[1], board) {
		a.mu.Unlock()
		http.NotFound(w, r)
		return true
	}
	owner := communityIdentity(w, r, r.Method == "GET")
	if admin {
		owner = "admin:" + session.AccountID
	}
	if !admin {
		w.Header().Set("Content-Security-Policy", strings.Replace(w.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'", "frame-ancestors "+workerAppOrigin, 1))
	}
	tail := p[offset:]
	if r.Method == "GET" && len(tail) == 3 && tail[0] == "file" {
		var f *ShippingAttachment
		for _, post := range a.state.Manuals {
			if post.ID == tail[1] {
				for _, file := range post.Files {
					if file.ID == tail[2] {
						v := file
						f = &v
					}
				}
			}
		}
		if f == nil {
			a.mu.Unlock()
			http.NotFound(w, r)
			return true
		}
		if f.Size < 1 || f.Size > manualFileLimit || len(f.Parts) != len(f.Hashes) || len(f.Parts) != int((f.Size+shippingAttachmentChunk-1)/shippingAttachmentChunk) {
			a.mu.Unlock()
			http.Error(w, "첨부 정보가 올바르지 않습니다.", 500)
			return true
		}
		if a.backupPins == nil {
			a.backupPins = map[string]int{}
		}
		for _, ph := range f.Parts {
			a.backupPins[ph.ID]++
		}
		dir := a.dir
		a.mu.Unlock()
		defer func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			for _, ph := range f.Parts {
				a.backupPins[ph.ID]--
				if a.backupPins[ph.ID] <= 0 {
					delete(a.backupPins, ph.ID)
				}
			}
			a.pruneCommunityPhotos(f.Parts)
		}()
		var size int64
		for i, ph := range f.Parts {
			if r.Context().Err() != nil {
				return true
			}
			raw, e := shippingExportPhoto(dir, ph)
			want := min(int64(shippingAttachmentChunk), f.Size-int64(i)*shippingAttachmentChunk)
			if e != nil || digest(raw) != f.Hashes[i] || int64(len(raw)) != want {
				http.Error(w, "첨부 파일을 확인할 수 없습니다.", 500)
				return true
			}
			size += int64(len(raw))
		}
		if size != f.Size {
			http.Error(w, "첨부 용량이 일치하지 않습니다.", 500)
			return true
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": f.Name}))
		http.ServeContent(w, r, f.Name, f.CreatedAt, &manualFileReader{dir: dir, file: *f, index: -1})
		return true
	}
	if r.Method == "GET" && len(tail) <= 1 {
		v := map[string]any{"Base": base, "Talk": "/community/" + board, "WorkerURL": "/work/" + board, "Board": board, "CSRF": csrf, "CommunityAdmin": admin, "Author": author, "UploadKey": token(), "Owner": owner, "Saved": r.URL.Query().Get("saved") == "1"}
		if admin {
			v["Talk"] = "/community-admin"
		}
		v["Mode"] = "list"
		if len(tail) == 1 {
			if tail[0] == "new" {
				v["Mode"] = "new"
			} else {
				var post *ManualPost
				for _, item := range a.state.Manuals {
					if item.ID == tail[0] {
						copy := item
						post = &copy
					}
				}
				if post == nil {
					a.mu.Unlock()
					http.NotFound(w, r)
					return true
				}
				v["Mode"], v["Post"] = "detail", post
				v["Saved"] = r.URL.Query().Get("saved") == "1"
			}
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		v["Query"] = q
		posts := []ManualPost{}
		for i := len(a.state.Manuals) - 1; i >= 0; i-- {
			post := a.state.Manuals[i]
			search := post.Title + " " + post.Text + " " + post.Author
			for _, f := range post.Files {
				search += " " + f.Name
			}
			if q == "" || strings.Contains(strings.ToLower(search), strings.ToLower(q)) {
				posts = append(posts, post)
			}
		}
		page := int(number(r.URL.Query().Get("page")))
		if page < 1 {
			page = 1
		}
		pages := max(1, (len(posts)+19)/20)
		page = min(page, pages)
		start := (page - 1) * 20
		v["Posts"], v["Count"], v["Page"], v["Pages"], v["Previous"], v["Next"] = posts[start:min(start+20, len(posts))], len(posts), page, pages, page-1, page+1
		rw, handled := a.nativeAdminPresentation(w, r)
		if !handled {
			a.render(rw, "manuals.html", v)
		}
		a.mu.Unlock()
		return true
	}
	valid := func() bool {
		if epoch != a.epoch || board != a.state.Board {
			return false
		}
		if admin {
			s, ok := a.session(r)
			return ok && !purchasingSession(&a.state, s) && equal(r.FormValue("csrf"), s.CSRF) && s.AccountID == session.AccountID && (localRequest(r) || a.state.Integration.MobileEnabled)
		}
		return owner != "" && equal(r.FormValue("csrf"), a.state.Board) && communityIdentity(w, r, false) == owner
	}
	if len(tail) == 1 && strings.HasPrefix(tail[0], "upload-") {
		a.mu.Unlock()
		a.manualTransferHTTP(w, r, tail[0], owner, board, epoch, valid)
		return true
	}
	a.mu.Unlock()
	if r.Method != "POST" || len(tail) != 1 || (tail[0] != "save" && tail[0] != "delete") {
		http.NotFound(w, r)
		return true
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1504<<20)
	var err error
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		err = r.ParseMultipartForm(2 << 20)
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
	} else {
		err = r.ParseForm()
	}
	if err != nil {
		http.Error(w, "첨부는 한 파일 300MB, 글당 5개까지 가능합니다.", 400)
		return true
	}

	a.mu.Lock()
	if !valid() {
		a.mu.Unlock()
		http.Error(w, "요청 인증을 확인해 주세요.", 403)
		return true
	}
	if tail[0] == "delete" {
		var removed []Photo
		err = a.commitAnnouncement(func(s *State) error {
			for i, post := range s.Manuals {
				if post.ID == r.FormValue("id") {
					if !admin && post.Owner != owner {
						return fmt.Errorf("작성한 브라우저에서만 삭제할 수 있습니다.")
					}
					for _, f := range post.Files {
						removed = append(removed, f.Parts...)
					}
					s.Manuals = append(s.Manuals[:i], s.Manuals[i+1:]...)
					return nil
				}
			}
			return fmt.Errorf("매뉴얼을 찾을 수 없습니다.")
		})
		if err == nil {
			a.pruneCommunityPhotos(removed)
		}
		a.mu.Unlock()
		if err != nil {
			http.Error(w, err.Error(), 409)
		} else {
			http.Redirect(w, r, base, 303)
		}
		return true
	}
	dir := a.dir
	a.mu.Unlock()
	title, text := strings.TrimSpace(r.FormValue("title")), strings.TrimSpace(r.FormValue("text"))
	if !admin {
		author = strings.TrimSpace(r.FormValue("author"))
	}
	key := r.FormValue("upload_key")
	if !validText(title, 120) || !validText(author, 40) || utf8.RuneCountInString(text) > 5000 || !validPhotoID.MatchString(key) {
		http.Error(w, "제목·작성자·파일(1~5개)을 확인해 주세요.", 400)
		return true
	}
	post := ManualPost{ID: key, Title: title, Text: text, Author: author, Owner: owner, CreatedAt: time.Now()}
	if len(r.Form["file_id"]) > 0 {
		a.finishManualTransfer(w, r, base, post, valid)
		return true
	}
	if r.MultipartForm == nil || len(r.MultipartForm.File["files"]) < 1 || len(r.MultipartForm.File["files"]) > 5 {
		http.Error(w, "파일은 1~5개 등록해 주세요.", 400)
		return true
	}
	written := []Photo{}
	err = func() error {
		for _, fh := range r.MultipartForm.File["files"] {
			name := cleanAttachmentName(fh.Filename)
			if name == "" || utf8.RuneCountInString(name) > 180 || fh.Size < 1 || fh.Size > manualFileLimit {
				return fmt.Errorf("파일은 한 개당 300MB 이하로 올려 주세요.")
			}
			src, e := fh.Open()
			if e != nil {
				return e
			}
			file := ShippingAttachment{ID: token(), Name: name, Kind: "file", MIME: "application/octet-stream", Size: fh.Size, CreatedAt: post.CreatedAt}
			var size int64
			for {
				b := make([]byte, shippingAttachmentChunk)
				n, e := io.ReadFull(src, b)
				if n > 0 {
					b = b[:n]
					size += int64(n)
					if size > manualFileLimit {
						src.Close()
						return fmt.Errorf("파일 용량 초과")
					}
					encrypted, e := protect(b)
					if e != nil {
						src.Close()
						return e
					}
					ph := Photo{ID: token(), Size: n}
					if e = os.MkdirAll(filepath.Join(dir, "photos"), 0700); e == nil {
						e = atomicFile(filepath.Join(dir, "photos", ph.ID), encrypted)
					}
					if e != nil {
						src.Close()
						return e
					}
					written = append(written, ph)
					file.Parts = append(file.Parts, ph)
					file.Hashes = append(file.Hashes, digest(b))
				}
				if e == io.EOF || e == io.ErrUnexpectedEOF {
					break
				}
				if e != nil {
					src.Close()
					return e
				}
			}
			src.Close()
			if size != fh.Size {
				return fmt.Errorf("첨부 용량 불일치")
			}
			post.Files = append(post.Files, file)
		}
		return nil
	}()
	a.mu.Lock()
	if err == nil && !valid() {
		err = fmt.Errorf("연결 정보가 변경되었습니다. 다시 등록해 주세요.")
	}
	if err == nil {
		err = a.commitAnnouncement(func(s *State) error {
			for _, old := range s.Manuals {
				if old.ID == key {
					if old.Owner != owner {
						return fmt.Errorf("등록 식별자가 다릅니다.")
					}
					return nil
				}
			}
			s.Manuals = append(s.Manuals, post)
			return nil
		})
	}
	a.pruneCommunityPhotos(written)
	a.mu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), 400)
	} else {
		http.Redirect(w, r, base+"/"+post.ID+"?saved=1", 303)
	}
	return true
}

func (f ShippingAttachment) ManualSizeLabel() string { return exportSize(f.Size) }
