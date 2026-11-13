package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func fmtID(id int64) string { return strconv.FormatInt(id, 10) }
func (a *App) adminGET(w http.ResponseWriter, r *http.Request, v map[string]any) bool {
	if a.reportSettingsGET(w, r, v) {
		return true
	}
	if a.importGET(w, r, Session{CSRF: v["CSRF"].(string)}) {
		return true
	}
	path := r.URL.Path
	switch path {
	case "/", "/mobile-admin":
		a.render(w, "admin.html", v)
		return true
	case "/projects/new":
		a.render(w, "project-create.html", v)
		return true
	case "/orders/panel":
		a.render(w, "admin-order-items", v)
		return true
	case "/inspection-requests":
		a.render(w, "inspection-queue.html", v)
		return true
	case "/inspection-requests/panel":
		a.render(w, "inspection-request-items", v)
		return true
	case "/projects/find":
		p := findOrderByNumber(&a.state, r.URL.Query().Get("number"))
		if p == nil {
			http.Error(w, "등록된 수주번호가 없습니다.", http.StatusNotFound)
			return true
		}
		a.render(w, "duplicate-order.html", map[string]any{"Existing": *p, "ExistingStatus": duplicateOrderStatus(*p)})
		return true
	case "/designers":
		a.render(w, "designers.html", v)
		return true
	case "/designers.json":
		jsonReply(w, a.state.Designers)
		return true
	case "/client-notes":
		a.render(w, "client-notes.html", v)
		return true
	case "/client-notes/list.json":
		jsonReply(w, a.state.ClientNotes)
		return true
	case "/unread":
		events := []Event{}
		for _, e := range a.view(v["CSRF"].(string))["Events"].([]Event) {
			if e.Unread {
				events = append(events, e)
			}
		}
		v["Events"] = events
		a.render(w, "unread.html", v)
		return true
	case "/export.csv":
		a.csv(w)
		return true
	case "/report.xlsx":
		b, e := workbook(a.state)
		if e != nil {
			http.Error(w, "엑셀 생성 실패", 500)
		} else {
			w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			w.Header().Set("Content-Disposition", `attachment; filename="inspection-records.xlsx"`)
			w.Write(b)
		}
		return true
	}
	p := strings.Split(strings.Trim(path, "/"), "/")
	if len(p) >= 3 && p[0] == "projects" {
		project := a.state.project(number(p[1]))
		if project == nil {
			http.NotFound(w, r)
			return true
		}
		pv := a.projectView(*project)
		v["Projects"] = []ProjectView{pv}
		v["SelectedDesigner"] = project.DesignerID
		v["SelectedClient"] = project.ClientID
		switch p[2] {
		case "edit":
			a.render(w, "project-edit.html", v)
		case "inspection":
			v["Review"] = pv
			a.render(w, "inspection-review.html", v)
		case "info":
			v["AdminInfo"] = true
			a.render(w, "project-info.html", v)
		case "delete":
			a.render(w, "project-delete.html", v)
		case "photos":
			if len(p) == 4 {
				a.photo(w, r, pv.Photos, p[3])
			} else {
				http.NotFound(w, r)
			}
		default:
			return false
		}
		return true
	}
	if len(p) >= 3 && p[0] == "client-notes" {
		for _, c := range a.state.ClientNotes {
			if c.ID == p[1] {
				switch p[2] {
				case "data":
					jsonReply(w, c)
				case "edit":
					v["ClientNote"] = c
					a.render(w, "client-note-edit.html", v)
				case "photos":
					if len(p) == 4 {
						a.photo(w, r, c.Photos, p[3])
					} else {
						http.NotFound(w, r)
					}
				default:
					return false
				}
				return true
			}
		}
		http.NotFound(w, r)
		return true
	}
	if len(p) == 3 && p[0] == "records" && p[2] == "edit" {
		for _, e := range v["Events"].([]Event) {
			if e.ID == number(p[1]) {
				v["Record"] = e
				a.render(w, "edit.html", v)
				return true
			}
		}
		for _, e := range a.state.Events {
			if e.ID == number(p[1]) {
				v["Record"] = e
				a.render(w, "edit.html", v)
				return true
			}
		}
		http.NotFound(w, r)
		return true
	}
	return false
}
func (a *App) adminPOST(w http.ResponseWriter, r *http.Request, session Session) bool {
	if a.reportSettingsPOST(w, r) {
		return true
	}
	if a.importPOST(w, r, session) {
		return true
	}
	path := r.URL.Path
	redirect := "/"
	var err error
	handled := true
	p := strings.Split(strings.Trim(path, "/"), "/")
	switch {
	case path == "/logout" || path == "/mobile-admin/logout":
		if c, e := r.Cookie(a.sessionCookie()); e == nil {
			delete(a.sessions, hashToken(c.Value))
		}
		http.SetCookie(w, &http.Cookie{Name: a.sessionCookie(), Path: "/", Value: "", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
		http.Redirect(w, r, "/login", 303)
		return true
	case path == "/projects":
		err = a.saveProject(r, 0)
	case path == "/designers":
		err = a.saveDesigner(r)
		redirect = "/designers"
	case path == "/client-notes":
		err = a.saveClient(r)
		redirect = "/client-notes"
	case path == "/archive":
		err = a.transaction(func(s *State) error {
			p := s.project(number(r.FormValue("id")))
			if p == nil {
				return errors.New("수주를 찾을 수 없습니다.")
			}
			if p.Shipping.Completed != nil || p.LegacyCompleted {
				p.LegacyCompleted = false
				p.Setting = Stage{}
				p.Shipping = Stage{}
				p.CombinedPending = false
				p.Hidden = false
				p.rotate()
			} else {
				p.Hidden = !p.Hidden
			}
			p.touch()
			return nil
		})
	case path == "/unread/read-all":
		through := number(r.FormValue("through"))
		err = a.transaction(func(s *State) error {
			for i := range s.Events {
				if s.Events[i].ID <= through {
					s.Events[i].Unread = false
				}
			}
			return nil
		})
		redirect = "/unread"
	case len(p) == 3 && p[0] == "projects":
		id := number(p[1])
		switch p[2] {
		case "edit":
			err = a.saveProject(r, id)
		case "inspection":
			err = a.approve(id, r.FormValue("inspection"), number(r.FormValue("request")))
			redirect = "/projects/" + p[1] + "/inspection"
			if r.FormValue("return_queue") == "1" {
				redirect = "/inspection-requests"
			}
		case "delete":
			err = a.deleteProject(r, id)
		default:
			handled = false
		}
	case len(p) == 3 && p[0] == "records":
		id := number(p[1])
		switch p[2] {
		case "read":
			err = a.transaction(func(s *State) error {
				for i := range s.Events {
					if s.Events[i].ID == id {
						s.Events[i].Unread = false
						return nil
					}
				}
				return errors.New("알림을 찾을 수 없습니다.")
			})
			if err == nil {
				jsonReply(w, map[string]any{"ok": true})
				return true
			}
		case "edit":
			err = a.editRecord(r, id)
		default:
			handled = false
		}
	default:
		handled = false
	}
	if !handled {
		return false
	}
	if err != nil {
		var duplicate *duplicateOrderError
		if errors.As(err, &duplicate) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusConflict)
			a.render(w, "duplicate-order.html", map[string]any{"Existing": duplicate.Project, "ExistingStatus": duplicateOrderStatus(duplicate.Project)})
			return true
		}
		http.Error(w, err.Error(), 409)
		return true
	}
	http.Redirect(w, r, redirect, 303)
	return true
}
func (a *App) saveProject(r *http.Request, id int64) error {
	n, name := strings.TrimSpace(r.FormValue("number")), strings.TrimSpace(r.FormValue("name"))
	if !validText(n, 80) || !validText(name, 120) {
		return errors.New("수주번호와 프로젝트명을 확인하세요.")
	}
	due := r.FormValue("shipping_due_date")
	if due != "" {
		t, e := time.Parse("2006-01-02", due)
		if e != nil || t.Format("2006-01-02") != due {
			return errors.New("납기일은 YYYY-MM-DD 날짜여야 합니다.")
		}
	}
	instructions, additional := r.FormValue("instructions"), r.FormValue("additional_instructions")
	if utf8.RuneCountInString(instructions) > 5000 || utf8.RuneCountInString(additional) > 5000 {
		return errors.New("특이사항은 5000자 이내로 입력하세요.")
	}
	photos, e := a.uploadPhotos(r)
	if e != nil {
		return e
	}
	return a.transaction(func(s *State) error {
		for _, p := range s.Projects {
			if p.ID != id && strings.EqualFold(p.Number, n) {
				return &duplicateOrderError{Project: p}
			}
		}
		p := s.project(id)
		if id == 0 {
			p = &Project{ID: s.next(), Revision: 1, InfoRevision: 1, Number: n}
			p.rotate()
		} else if p == nil {
			return errors.New("수주를 찾을 수 없습니다.")
		} else if number(r.FormValue("revision")) != p.InfoRevision {
			return errors.New("다른 화면에서 수정되었습니다. 새로고침 후 다시 저장하세요.")
		}
		p.Number = n
		p.Name = name
		p.ShippingDueDate = due
		p.SpecialInstructions = instructions
		p.AdditionalInstructions = additional
		p.DesignerID = r.FormValue("designer_id")
		p.DesignerName = strings.TrimSpace(r.FormValue("designer_name"))
		p.DesignerPhone = cleanPhone(r.FormValue("designer_phone"))
		if p.DesignerID != "" {
			found := false
			for _, d := range s.Designers {
				if d.ID == p.DesignerID {
					p.DesignerName = d.Name
					p.DesignerPhone = d.Phone
					found = true
				}
			}
			if !found {
				return errors.New("선택한 설계담당자가 없습니다.")
			}
		}
		if utf8.RuneCountInString(p.DesignerName) > 40 || len(p.DesignerPhone) > 30 || (p.DesignerName == "") != (p.DesignerPhone == "") {
			return errors.New("담당자 이름과 전화번호를 함께 입력하세요.")
		}
		p.Photos = removePhotos(p.Photos, r.Form["remove_photo"])
		p.Photos = append(p.Photos, photos...)
		clientID := r.FormValue("client_template_id")
		if clientID == "" && r.FormValue("keep_client_link") == "1" {
			clientID = p.ClientID
		}
		if clientID != "" {
			found := false
			for _, c := range s.ClientNotes {
				if c.ID != clientID {
					continue
				}
				found = true
				if rev := r.FormValue("client_template_revision"); rev != "" && number(rev) != c.Revision {
					return errors.New("업체 원본이 변경되었습니다. 다시 불러오세요.")
				}
				if r.FormValue("sync_client") == "1" {
					p.ClientID = c.ID
					if r.FormValue("client_template_id") != "" {
						p.ClientPhotoIDs = append([]string{}, r.Form["template_photo"]...)
						p.ClientPhotosSelected = true
					}
					if len(c.Photos)+len(p.Photos) > 5 {
						return errors.New("업체 사진을 포함해 최대 5장입니다.")
					}
				} else {
					p.ClientID = ""
					if r.FormValue("client_template_id") != "" {
						for _, ph := range c.Photos {
							for _, selected := range r.Form["template_photo"] {
								if selected == ph.ID {
									p.Photos = append(p.Photos, ph)
								}
							}
						}
					}
				}
			}
			if !found {
				return errors.New("선택한 업체 원본을 찾을 수 없습니다.")
			}
		} else {
			p.ClientID = ""
		}
		if len(p.Photos) > 5 {
			return errors.New("수주 사진은 최대 5장입니다.")
		}
		p.touch()
		if id == 0 {
			s.Projects = append(s.Projects, *p)
		}
		return nil
	})
}
func cleanPhone(s string) string {
	var b strings.Builder
	for _, c := range s {
		if c >= '0' && c <= '9' || c == '+' && b.Len() == 0 {
			b.WriteRune(c)
		}
	}
	return b.String()
}
func (a *App) saveDesigner(r *http.Request) error {
	return a.transaction(func(s *State) error {
		action, id := r.FormValue("action"), r.FormValue("id")
		name, phone := strings.TrimSpace(r.FormValue("name")), cleanPhone(r.FormValue("phone"))
		if action != "delete" && (!validText(name, 40) || len(phone) < 7 || len(phone) > 30) {
			return errors.New("담당자 이름과 전화번호를 확인하세요.")
		}
		if action == "create" {
			for _, d := range s.Designers {
				if d.Name == name && d.Phone == phone {
					return errors.New("이미 등록된 담당자입니다.")
				}
			}
			s.Designers = append(s.Designers, Designer{token(), name, phone, 1})
			return nil
		}
		for i, d := range s.Designers {
			if d.ID != id {
				continue
			}
			if number(r.FormValue("revision")) != d.Revision {
				return errors.New("담당자 정보가 변경되었습니다. 새로고침하세요.")
			}
			if action != "delete" && action != "save" {
				return errors.New("담당자 요청 구분 오류")
			}
			for j := range s.Projects {
				p := &s.Projects[j]
				if p.DesignerID == id {
					if action == "delete" {
						p.DesignerName = d.Name
						p.DesignerPhone = d.Phone
						p.DesignerID = ""
					}
					p.touch()
				}
			}
			if action == "delete" {
				s.Designers = append(s.Designers[:i], s.Designers[i+1:]...)
			} else {
				s.Designers[i].Name = name
				s.Designers[i].Phone = phone
				s.Designers[i].Revision++
			}
			return nil
		}
		return errors.New("담당자를 찾을 수 없습니다.")
	})
}
func (a *App) saveClient(r *http.Request) error {
	photos, e := a.uploadPhotos(r)
	if e != nil {
		return e
	}
	return a.transaction(func(s *State) error {
		action, id := r.FormValue("action"), r.FormValue("id")
		name, text := strings.TrimSpace(r.FormValue("client_name")), r.FormValue("client_text")
		if action != "delete" && (!validText(name, 80) || utf8.RuneCountInString(text) > 5000) {
			return errors.New("업체명과 특이사항 내용을 확인하세요.")
		}
		if action == "create" {
			if text == "" && len(photos) == 0 {
				return errors.New("글 또는 사진을 등록하세요.")
			}
			s.ClientNotes = append(s.ClientNotes, ClientNote{token(), name, text, 1, photos})
			return nil
		}
		for i, c := range s.ClientNotes {
			if c.ID != id {
				continue
			}
			if c.Revision != number(r.FormValue("revision")) {
				return errors.New("업체 원본이 변경되었습니다. 다시 열어 주세요.")
			}
			updated := c
			updated.Photos = append(removePhotos(c.Photos, r.Form["remove_photo"]), photos...)
			updated.Name = name
			updated.Text = text
			updated.Revision++
			if action != "delete" && action != "save" {
				return errors.New("업체 요청 구분 오류")
			}
			if action == "save" && (len(updated.Photos) > 5 || (text == "" && len(updated.Photos) == 0)) {
				return errors.New("업체 원본에는 글 또는 사진이 필요하며 최대 5장입니다.")
			}
			for j := range s.Projects {
				p := &s.Projects[j]
				if p.ClientID == id {
					if action == "delete" {
						p.SpecialInstructions = c.Text
						p.Photos = append(append([]Photo{}, c.Photos...), p.Photos...)
						p.ClientID = ""
					} else if len(updated.Photos)+len(p.Photos) > 5 {
						return errors.New("연결된 수주와 합쳐 사진이 5장을 넘습니다.")
					}
					if action == "save" {
						p.ClientPhotosSelected = false
						p.ClientPhotoIDs = nil
					}
					p.touch()
				}
			}
			if action == "delete" {
				s.ClientNotes = append(s.ClientNotes[:i], s.ClientNotes[i+1:]...)
			} else {
				s.ClientNotes[i] = updated
			}
			return nil
		}
		return errors.New("업체 원본을 찾을 수 없습니다.")
	})
}
func removePhotos(photos []Photo, removed []string) []Photo {
	out := []Photo{}
	for _, p := range photos {
		drop := false
		for _, id := range removed {
			if p.ID == id {
				drop = true
			}
		}
		if !drop {
			out = append(out, p)
		}
	}
	return out
}
func (a *App) uploadPhotos(r *http.Request) ([]Photo, error) {
	result := []Photo{}
	if r.MultipartForm == nil {
		return result, nil
	}
	files := r.MultipartForm.File["photos"]
	if len(files) > 5 {
		return nil, errors.New("사진은 최대 5장입니다.")
	}
	for _, f := range files {
		if f.Size > 8<<20 {
			return nil, errors.New("사진은 한 장당 8MB 이하입니다.")
		}
		stream, e := f.Open()
		if e != nil {
			return nil, e
		}
		b, e := io.ReadAll(io.LimitReader(stream, (8<<20)+1))
		stream.Close()
		if e != nil || len(b) > 8<<20 {
			return nil, errors.New("사진을 읽을 수 없습니다.")
		}
		mime := http.DetectContentType(b)
		if mime != "image/jpeg" && mime != "image/png" && mime != "image/gif" && mime != "image/webp" {
			return nil, errors.New("JPG, PNG, GIF, WebP 사진만 등록할 수 있습니다.")
		}
		cfg, _, e := image.DecodeConfig(bytes.NewReader(b))
		if mime != "image/webp" && (e != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 40_000_000) {
			return nil, errors.New("사진 크기 또는 형식을 확인하세요.")
		}
		id := token()
		encrypted, e := protect(b)
		if e != nil {
			return nil, e
		}
		if e = os.MkdirAll(filepath.Join(a.dir, "photos"), 0700); e != nil {
			return nil, e
		}
		if e = atomicFile(filepath.Join(a.dir, "photos", id), encrypted); e != nil {
			return nil, e
		}
		result = append(result, Photo{ID: id, Name: filepath.Base(f.Filename), MIME: mime, Width: cfg.Width, Height: cfg.Height, Size: len(b)})
	}
	return result, nil
}

var validPhotoID = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

func (a *App) photo(w http.ResponseWriter, r *http.Request, photos []Photo, id string) {
	if !validPhotoID.MatchString(id) {
		http.NotFound(w, r)
		return
	}
	for _, p := range photos {
		if p.ID == id {
			var b []byte
			var e error
			if p.Legacy {
				b, e = a.legacyPhoto(p)
			} else {
				b, e = os.ReadFile(filepath.Join(a.dir, "photos", id))
				if e == nil {
					b, e = unprotect(b)
				}
			}
			if e != nil {
				http.Error(w, "사진을 읽을 수 없습니다.", 500)
				return
			}
			w.Header().Set("Content-Type", p.MIME)
			w.Write(b)
			return
		}
	}
	http.NotFound(w, r)
}
func (a *App) deleteProject(r *http.Request, id int64) error {
	if r.FormValue("confirm") != "1" {
		return errors.New("삭제 확인이 필요합니다.")
	}
	return a.transaction(func(s *State) error {
		found := false
		for i, p := range s.Projects {
			if p.ID == id {
				s.Projects = append(s.Projects[:i], s.Projects[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			return errors.New("수주를 찾을 수 없습니다.")
		}
		events := []Event{}
		for _, e := range s.Events {
			if e.ProjectID != id || !e.Pending {
				events = append(events, e)
			}
		}
		s.Events = events
		return nil
	})
}
func (a *App) editRecord(r *http.Request, id int64) error {
	return a.transaction(func(s *State) error {
		i := -1
		for j := range s.Events {
			if s.Events[j].ID == id {
				i = j
			}
		}
		if i < 0 {
			return errors.New("기록을 찾을 수 없습니다.")
		}
		e := &s.Events[i]
		if e.Revision != number(r.FormValue("revision")) {
			return errors.New("기록이 변경되었습니다. 다시 열어 주세요.")
		}
		action := r.FormValue("action")
		p := s.project(e.ProjectID)
		if action == "rename" {
			worker := strings.TrimSpace(r.FormValue("worker"))
			if !validText(worker, 40) {
				return errors.New("작업자 이름을 확인하세요.")
			}
			e.Worker = worker
			e.Revision++
			if p != nil {
				if p.Setting.EventID == id {
					p.Setting.Worker = worker
				}
				if p.Shipping.EventID == id {
					p.Shipping.Worker = worker
				}
				p.touch()
			}
			return nil
		}
		if action != "cancel" && action != "delete" {
			return errors.New("기록 요청 구분 오류")
		}
		if r.FormValue("confirm") != "1" {
			return errors.New("삭제 또는 취소 확인이 필요합니다.")
		}
		if e.Pending && e.BatchID != 0 {
			return errors.New("통합 신청은 개별 기록을 삭제·취소할 수 없습니다.")
		}
		remove := map[int64]bool{id: true}
		if action == "cancel" {
			if e.IsNote || p == nil {
				return errors.New("취소할 현재 검사를 찾을 수 없습니다.")
			}
			if e.Inspection == "setting" {
				if p.Setting.EventID != id {
					return errors.New("현재 검사 주기의 기록이 아닙니다.")
				}
				remove[p.Shipping.EventID] = true
				p.Setting = Stage{}
				p.Shipping = Stage{}
			} else {
				if p.Shipping.EventID != id {
					return errors.New("현재 검사 주기의 기록이 아닙니다.")
				}
				p.Shipping = Stage{}
			}
			p.CombinedPending = false
			p.Hidden = false
			p.rotate()
			p.touch()
		} else if e.Pending && p != nil {
			if p.Setting.EventID == id {
				p.Setting = Stage{}
			}
			if p.Shipping.EventID == id {
				p.Shipping = Stage{}
			}
			p.rotate()
			p.touch()
		}
		result := []Event{}
		for _, ev := range s.Events {
			if !remove[ev.ID] {
				result = append(result, ev)
			}
		}
		s.Events = result
		return nil
	})
}
func (a *App) csv(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="inspection-records.csv"`)
	w.Write([]byte{239, 187, 191})
	c := csv.NewWriter(w)
	c.Write([]string{"수주번호", "프로젝트명", "검사구분", "작업자", "완료일자", "완료시간"})
	for _, e := range completedEvents(a.state) {
		record := eventRow(a.state, e)
		for i, s := range record {
			if len(s) > 0 && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
				record[i] = "'" + s
			}
		}
		c.Write(record)
	}
	c.Flush()
}
