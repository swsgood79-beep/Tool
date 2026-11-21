package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

const shippingAttachmentChunk = 4 << 20
const shippingAttachmentTTL = 2 * time.Hour

type ShippingAttachment struct {
	ID, Key, Name, Kind, MIME string
	Size, OriginalSize        int64
	CreatedAt                 time.Time
	Parts                     []Photo
	Hashes                    []string
}

func (f ShippingAttachment) Path() string { return f.ID + "/" + url.PathEscape(f.Name) }
func (f ShippingAttachment) SizeLabel() string {
	return fmt.Sprintf("%.1f MB", float64(f.Size)/(1<<20))
}
func (f ShippingAttachment) ReducedLabel() string {
	if f.OriginalSize > f.Size {
		return fmt.Sprintf("원본 %.1f MB → %s", float64(f.OriginalSize)/(1<<20), f.SizeLabel())
	}
	return f.SizeLabel()
}
func attachmentParts(r ShippingRecord) []Photo {
	var out []Photo
	for _, f := range r.Attachments {
		out = append(out, f.Parts...)
	}
	return out
}

func shippingStoredAssets(r ShippingRecord) []Photo {
	out := append([]Photo{}, r.Project.Photos...)
	out = append(out, r.Photos...)
	return append(out, attachmentParts(r)...)
}

type shippingAttachmentStage struct {
	ID, Record, Owner, Dir string
	File                   ShippingAttachment
	Received               int64
	Updated                time.Time
}

// Upload I/O has its own lock; App.mu is held only for authorization and metadata commits.
type shippingAttachmentTransfers struct {
	sync.Mutex
	Stages map[string]*shippingAttachmentStage
}

func cleanAttachmentName(s string) string {
	s = filepath.Base(strings.ReplaceAll(s, "\\", "/"))
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(s)
}
func attachmentFileMeta(name, kind string, size int64) (string, error) {
	name = cleanAttachmentName(name)
	if name == "" || name == "." || utf8.RuneCountInString(name) > 180 || size < 1 {
		return "", errors.New("파일 이름과 용량을 확인해 주세요.")
	}
	limit := int64(100 << 20)
	if kind == "video" {
		limit = 300 << 20
		ext := strings.ToLower(filepath.Ext(name))
		if !strings.Contains("|.mp4|.mov|.webm|.m4v|.3gp|", "|"+ext+"|") {
			return "", errors.New("MP4·MOV·WebM·3GP 동영상을 선택해 주세요.")
		}
	} else if kind != "file" {
		return "", errors.New("첨부 종류가 올바르지 않습니다.")
	}
	if size > limit {
		return "", errors.New("파일은 100MB, 동영상은 300MB 이하로 등록해 주세요.")
	}
	return name, nil
}
func (a *App) cleanupAttachmentStagesLocked() {
	now := time.Now()
	for id, s := range a.attachmentTransfers.Stages {
		if now.Sub(s.Updated) > shippingAttachmentTTL {
			_ = os.RemoveAll(s.Dir)
			delete(a.attachmentTransfers.Stages, id)
		}
	}
	a.mu.Lock()
	storageDir := a.dir
	a.mu.Unlock()
	root := filepath.Join(storageDir, ".shipping-uploads")
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if !e.IsDir() || !validPhotoID.MatchString(e.Name()) || a.attachmentTransfers.Stages[e.Name()] != nil {
			continue
		}
		st, err := e.Info()
		if err == nil && now.Sub(st.ModTime()) > shippingAttachmentTTL {
			_ = os.RemoveAll(filepath.Join(root, e.Name()))
		}
	}
}

// Return the record route and required authority without reading any upload body.
func (a *App) attachmentAuthority(w http.ResponseWriter, r *http.Request, admin bool, board, id string) (string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	owner, csrf := "worker:"+a.state.Board, a.state.Board
	if admin {
		if r.Method == "POST" {
			r.Form = url.Values{"csrf": {r.Header.Get("X-CSRF-Token")}}
		}
		s, ok := a.admin(w, r)
		if !ok {
			return "", false
		}
		owner = "admin:" + s.AccountID
		csrf = s.CSRF
	} else if !equal(board, a.state.Board) {
		http.NotFound(w, r)
		return "", false
	}
	if session, ok := a.session(r); ok && purchasingSession(&a.state, session) {
		http.Error(w, "구매팀 계정은 호출 업무만 이용할 수 있습니다.", 403)
		return "", false
	}
	if !admin && nativeAdminAgent(r) {
		if _, ok := a.session(r); !ok {
			http.Error(w, "관리자 로그인이 필요합니다.", 401)
			return "", false
		}
	}
	if shippingRecord(&a.state, id) == nil {
		http.NotFound(w, r)
		return "", false
	}
	if r.Method == "POST" && !equal(r.Header.Get("X-CSRF-Token"), csrf) {
		http.Error(w, "인증 정보가 올바르지 않습니다.", 403)
		return "", false
	}
	return owner, true
}
func (a *App) shippingAttachmentsHTTP(w http.ResponseWriter, r *http.Request) bool {
	p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	admin := len(p) > 0 && p[0] == "shipping-archive-admin"
	board := ""
	var rest []string
	if admin {
		rest = p[1:]
	} else if len(p) >= 2 && p[0] == "shipping-records" {
		board = p[1]
		rest = p[2:]
	} else {
		return false
	}
	if len(rest) < 2 || rest[1] != "attachment" {
		return false
	}
	if len(rest) != 3 && len(rest) != 4 && len(rest) != 5 {
		http.NotFound(w, r)
		return true
	}
	id, action := rest[0], rest[2]
	owner, ok := a.attachmentAuthority(w, r, admin, board, id)
	if !ok {
		return true
	}
	fail := func(err error) { http.Error(w, err.Error(), 400) }
	if action == "file" && (len(rest) == 4 || len(rest) == 5) && r.Method == "GET" {
		a.serveShippingAttachment(w, r, id, rest[3])
		return true
	}
	if len(rest) != 3 || r.Method != "POST" {
		http.Error(w, "POST 요청이 필요합니다.", 405)
		return true
	}
	// Metadata endpoints stay small. Chunks are bounded independently of the normal form parser.
	if action == "start" {
		var v struct {
			Key, Name, Kind    string
			Size, OriginalSize int64
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&v); err != nil {
			fail(errors.New("첨부 정보를 읽을 수 없습니다."))
			return true
		}
		name, err := attachmentFileMeta(v.Name, v.Kind, v.Size)
		if err != nil {
			fail(err)
			return true
		}
		if !validPhotoID.MatchString(v.Key) {
			fail(errors.New("업로드 정보가 올바르지 않습니다."))
			return true
		}
		a.mu.Lock()
		record := shippingRecord(&a.state, id)
		if record == nil {
			a.mu.Unlock()
			http.NotFound(w, r)
			return true
		}
		for _, f := range record.Attachments {
			if equal(f.Key, v.Key) {
				if f.Name != name || f.Kind != v.Kind || f.Size != v.Size {
					a.mu.Unlock()
					fail(errors.New("이미 저장한 첨부와 내용이 다릅니다."))
					return true
				}
				a.mu.Unlock()
				jsonReply(w, map[string]any{"ok": true, "complete": true, "id": f.ID})
				return true
			}
		}
		quota := len(record.Attachments) >= 20
		var total int64
		for _, f := range record.Attachments {
			total += f.Size
		}
		quota = quota || total+v.Size > 1<<30
		a.mu.Unlock()
		if quota {
			fail(errors.New("수주당 첨부 20개·총 1GB까지 보관할 수 있습니다."))
			return true
		}
		a.attachmentTransfers.Lock()
		defer a.attachmentTransfers.Unlock()
		a.cleanupAttachmentStagesLocked()
		if a.attachmentTransfers.Stages == nil {
			a.attachmentTransfers.Stages = map[string]*shippingAttachmentStage{}
		}
		for _, s := range a.attachmentTransfers.Stages {
			if s.Owner == owner && s.Record == id && equal(s.File.Key, v.Key) {
				if s.File.Name != name || s.File.Size != v.Size || s.File.Kind != v.Kind {
					fail(errors.New("업로드 내용이 변경되었습니다."))
					return true
				}
				jsonReply(w, map[string]any{"ok": true, "id": s.ID, "next": len(s.File.Parts), "chunk": shippingAttachmentChunk})
				return true
			}
		}
		if len(a.attachmentTransfers.Stages) >= 8 {
			fail(errors.New("진행 중인 업로드가 많습니다. 잠시 후 다시 시도해 주세요."))
			return true
		}
		upload := token()
		a.mu.Lock()
		storageDir := a.dir
		a.mu.Unlock()
		dir := filepath.Join(storageDir, ".shipping-uploads", upload)
		if err = os.MkdirAll(dir, 0700); err != nil {
			fail(errors.New("첨부 저장 공간을 준비하지 못했습니다."))
			return true
		}
		stage := &shippingAttachmentStage{ID: upload, Record: id, Owner: owner, Dir: dir, Updated: time.Now(), File: ShippingAttachment{ID: upload, Key: v.Key, Name: name, Kind: v.Kind, Size: v.Size, OriginalSize: max(v.Size, v.OriginalSize), CreatedAt: time.Now(), MIME: "application/octet-stream"}}
		a.attachmentTransfers.Stages[upload] = stage
		jsonReply(w, map[string]any{"ok": true, "id": upload, "next": 0, "chunk": shippingAttachmentChunk})
		return true
	}
	upload := r.URL.Query().Get("upload")
	if action == "delete" {
		if !admin {
			http.Error(w, "관리자만 첨부를 삭제할 수 있습니다.", 403)
			return true
		}
		var v struct{ ID string }
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&v) != nil {
			fail(errors.New("삭제 정보를 확인해 주세요."))
			return true
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		var removed []Photo
		err := a.commitAnnouncement(func(s *State) error {
			rec := shippingRecord(s, id)
			if rec == nil {
				return errors.New("기록이 변경되었습니다.")
			}
			for i, f := range rec.Attachments {
				if f.ID == v.ID {
					removed = f.Parts
					rec.Attachments = append(rec.Attachments[:i], rec.Attachments[i+1:]...)
					return nil
				}
			}
			return errors.New("첨부를 찾을 수 없습니다.")
		})
		if err != nil {
			fail(err)
		} else {
			a.pruneCommunityPhotos(removed)
			jsonReply(w, map[string]any{"ok": true})
		}
		return true
	}
	if !validPhotoID.MatchString(upload) {
		fail(errors.New("업로드 정보를 확인해 주세요."))
		return true
	}
	var data []byte
	if action == "chunk" {
		var err error
		data, err = io.ReadAll(http.MaxBytesReader(w, r.Body, shippingAttachmentChunk))
		if err != nil || len(data) == 0 {
			fail(errors.New("첨부 조각을 읽지 못했습니다. 다시 시도해 주세요."))
			return true
		}
	}
	a.attachmentTransfers.Lock()
	defer a.attachmentTransfers.Unlock()
	stage := a.attachmentTransfers.Stages[upload]
	if stage == nil || stage.Owner != owner || stage.Record != id {
		if action == "finish" {
			a.mu.Lock()
			rec := shippingRecord(&a.state, id)
			found := false
			if rec != nil {
				for _, f := range rec.Attachments {
					found = found || f.ID == upload
				}
			}
			a.mu.Unlock()
			if found {
				jsonReply(w, map[string]any{"ok": true})
				return true
			}
		}
		fail(errors.New("업로드 연결이 만료되었습니다. 파일을 다시 저장해 주세요."))
		return true
	}
	stage.Updated = time.Now()
	switch action {
	case "cancel":
		_ = os.RemoveAll(stage.Dir)
		delete(a.attachmentTransfers.Stages, upload)
		jsonReply(w, map[string]any{"ok": true})
	case "chunk":
		n, err := strconv.Atoi(r.URL.Query().Get("part"))
		if err != nil || n < 0 || n > len(stage.File.Parts) {
			fail(errors.New("첨부 조각 순서를 확인해 주세요."))
			return true
		}
		if n < len(stage.File.Parts) {
			if digest(data) != stage.File.Hashes[n] {
				fail(errors.New("이미 받은 조각과 내용이 다릅니다."))
				return true
			}
			jsonReply(w, map[string]any{"ok": true})
			return true
		}
		expected := min(int64(shippingAttachmentChunk), stage.File.Size-stage.Received)
		if int64(len(data)) != expected {
			fail(errors.New("첨부 조각의 용량이 다릅니다."))
			return true
		}
		if n == 0 && stage.File.Kind == "video" {
			m := http.DetectContentType(data)
			if len(data) >= 12 && string(data[4:8]) == "ftyp" {
				m = "video/mp4"
				if string(data[8:12]) == "qt  " {
					m = "video/quicktime"
				}
			} else if bytes.HasPrefix(data, []byte{0x1a, 0x45, 0xdf, 0xa3}) {
				m = "video/webm"
			}
			if !strings.HasPrefix(m, "video/") {
				fail(errors.New("동영상 내용을 확인할 수 없습니다. MP4나 WebM으로 변환해 주세요."))
				return true
			}
			stage.File.MIME = m
		}
		encrypted, err := protect(data)
		if err != nil {
			fail(err)
			return true
		}
		ph := Photo{ID: token(), Size: len(data)}
		if err = atomicFile(filepath.Join(stage.Dir, ph.ID), encrypted); err != nil {
			fail(errors.New("첨부 저장에 실패했습니다. 디스크 여유 공간을 확인해 주세요."))
			return true
		}
		stage.File.Parts = append(stage.File.Parts, ph)
		stage.File.Hashes = append(stage.File.Hashes, digest(data))
		stage.Received += int64(len(data))
		jsonReply(w, map[string]any{"ok": true})
	case "finish":
		if stage.Received != stage.File.Size {
			fail(errors.New("아직 전송하지 않은 첨부 조각이 있습니다."))
			return true
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		// Recheck authentication after the transfer, including logout/account revocation.
		if admin {
			s, ok := a.session(r)
			if !ok || "admin:"+s.AccountID != owner || purchasingSession(&a.state, s) {
				http.Error(w, "관리자 로그인을 다시 확인해 주세요.", 401)
				return true
			}
		} else if !equal(board, a.state.Board) {
			http.Error(w, "접속 정보가 변경되었습니다.", 403)
			return true
		}
		moved := []Photo{}
		err := os.MkdirAll(filepath.Join(a.dir, "photos"), 0700)
		if err == nil {
			for _, ph := range stage.File.Parts {
				err = os.Rename(filepath.Join(stage.Dir, ph.ID), filepath.Join(a.dir, "photos", ph.ID))
				if err != nil {
					break
				}
				moved = append(moved, ph)
			}
		}
		if err == nil {
			err = a.commitAnnouncement(func(s *State) error {
				rec := shippingRecord(s, id)
				if rec == nil {
					return errors.New("보관 자료가 변경되었습니다.")
				}
				for _, f := range rec.Attachments {
					if f.ID == upload {
						return nil
					}
				}
				var total int64
				for _, f := range rec.Attachments {
					total += f.Size
				}
				if len(rec.Attachments) >= 20 || total+stage.File.Size > 1<<30 {
					return errors.New("수주당 첨부 20개·총 1GB 한도입니다.")
				}
				rec.Attachments = append(rec.Attachments, stage.File)
				return nil
			})
		}
		if err != nil {
			for _, ph := range moved {
				_ = os.Rename(filepath.Join(a.dir, "photos", ph.ID), filepath.Join(stage.Dir, ph.ID))
			}
			fail(err)
			return true
		}
		_ = os.RemoveAll(stage.Dir)
		delete(a.attachmentTransfers.Stages, upload)
		jsonReply(w, map[string]any{"ok": true})
	default:
		http.NotFound(w, r)
	}
	return true
}

type attachmentReader struct {
	files     []*os.File
	parts     []Photo
	hashes    []string
	size, pos int64
	cached    int
	data      []byte
}

func (b *attachmentReader) Seek(off int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		off += b.pos
	case io.SeekEnd:
		off += b.size
	default:
		return b.pos, errors.New("seek")
	}
	if off < 0 {
		return b.pos, errors.New("seek")
	}
	b.pos = off
	return off, nil
}
func (b *attachmentReader) Read(p []byte) (int, error) {
	if b.pos >= b.size {
		return 0, io.EOF
	}
	index := int(b.pos / shippingAttachmentChunk)
	if index < 0 || index >= len(b.files) {
		return 0, io.ErrUnexpectedEOF
	}
	if b.cached != index {
		raw, e := io.ReadAll(io.NewSectionReader(b.files[index], 0, shippingAttachmentChunk+(1<<20)))
		if e != nil {
			return 0, e
		}
		raw, e = unprotect(raw)
		if e != nil || len(raw) != b.parts[index].Size || digest(raw) != b.hashes[index] {
			return 0, errors.New("첨부 파일이 손상되었습니다.")
		}
		b.data, b.cached = raw, index
	}
	offset := int(b.pos % shippingAttachmentChunk)
	if offset >= len(b.data) {
		return 0, io.ErrUnexpectedEOF
	}
	n := copy(p, b.data[offset:])
	b.pos += int64(n)
	return n, nil
}
func (a *App) serveShippingAttachment(w http.ResponseWriter, r *http.Request, id, fileID string) {
	a.mu.Lock()
	rec := shippingRecord(&a.state, id)
	var file ShippingAttachment
	found := false
	if rec != nil {
		for _, f := range rec.Attachments {
			if f.ID == fileID {
				file = f
				found = true
				break
			}
		}
	}
	if !found || len(file.Parts) != len(file.Hashes) {
		a.mu.Unlock()
		http.NotFound(w, r)
		return
	}
	if file.Size < 1 || len(file.Parts) != int((file.Size+shippingAttachmentChunk-1)/shippingAttachmentChunk) {
		a.mu.Unlock()
		http.Error(w, "첨부 정보 오류", 500)
		return
	}
	for i, ph := range file.Parts {
		expected := min(int64(shippingAttachmentChunk), file.Size-int64(i)*shippingAttachmentChunk)
		if !validPhotoID.MatchString(ph.ID) || int64(ph.Size) != expected {
			a.mu.Unlock()
			http.Error(w, "첨부 정보 오류", 500)
			return
		}
	}
	reader := &attachmentReader{parts: file.Parts, hashes: file.Hashes, size: file.Size, cached: -1}
	for _, ph := range file.Parts {
		if !validPhotoID.MatchString(ph.ID) {
			a.mu.Unlock()
			http.Error(w, "첨부 정보 오류", 500)
			return
		}
		f, e := os.Open(filepath.Join(a.dir, "photos", ph.ID))
		if e != nil {
			for _, f := range reader.files {
				f.Close()
			}
			a.mu.Unlock()
			http.Error(w, "첨부 파일을 읽지 못했습니다.", 500)
			return
		}
		reader.files = append(reader.files, f)
	}
	a.mu.Unlock()
	defer func() {
		for _, f := range reader.files {
			f.Close()
		}
		a.mu.Lock()
		a.pruneCommunityPhotos(file.Parts)
		a.mu.Unlock()
	}()
	disposition := "attachment"
	if file.Kind == "video" && r.URL.Query().Get("download") != "1" {
		disposition = "inline"
	}
	w.Header().Set("Content-Type", file.MIME)
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": file.Name}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, file.Name, file.CreatedAt, reader)
}
