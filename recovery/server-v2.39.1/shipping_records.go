package main

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// An independent snapshot, captured in the approval transaction. The live schedule
// remains a projection of the connected sheet and never owns archive records.
type ShippingRecord struct {
	PhotoUploadKeys []string `json:",omitempty"`
	ClientName      string
	ID              string
	Project         Project
	Receipts        []MaterialReceipt
	InspectedAt     time.Time
	ArchivedAt      *time.Time
	Photos          []Photo
	Memo            string
}

func shippingRecord(s *State, id string) *ShippingRecord {
	for i := range s.ShippingRecords {
		if s.ShippingRecords[i].ID == id {
			return &s.ShippingRecords[i]
		}
	}
	return nil
}
func (a *App) captureShippingRecord(s *State, p Project) ([]Photo, error) {
	for _, record := range s.ShippingRecords {
		if record.Project.ID == p.ID && record.Project.Shipping.EventID == p.Shipping.EventID {
			return nil, nil
		}
	}
	for _, d := range s.Designers {
		if d.ID == p.DesignerID {
			p.DesignerName, p.DesignerPhone = d.Name, d.Phone
		}
	}
	clientName := ""
	source := append([]Photo{}, p.Photos...)
	for _, c := range s.ClientNotes {
		if c.ID == p.ClientID {
			clientName = c.Name
			p.SpecialInstructions = c.Text
			for _, ph := range c.Photos {
				selected := !p.ClientPhotosSelected
				for _, id := range p.ClientPhotoIDs {
					if id == ph.ID {
						selected = true
					}
				}
				if selected {
					source = append(source, ph)
				}
			}
		}
	}
	copies := []Photo{}
	fail := func(err error) ([]Photo, error) {
		for _, ph := range copies {
			_ = os.Remove(filepath.Join(a.dir, "photos", ph.ID))
		}
		return nil, err
	}
	for _, ph := range source {
		var data []byte
		var err error
		if ph.Legacy {
			data, err = a.legacyPhoto(ph)
		} else {
			data, err = os.ReadFile(filepath.Join(a.dir, "photos", ph.ID))
			if err == nil {
				data, err = unprotect(data)
			}
		}
		if err != nil {
			return fail(fmt.Errorf("수주 특이사항 사진 보관 실패: %w", err))
		}
		encrypted, err := protect(data)
		if err != nil {
			return fail(err)
		}
		ph.ID, ph.Legacy = token(), false
		if err = os.MkdirAll(filepath.Join(a.dir, "photos"), 0700); err != nil {
			return fail(err)
		}
		if err = atomicFile(filepath.Join(a.dir, "photos", ph.ID), encrypted); err != nil {
			return fail(err)
		}
		copies = append(copies, ph)
	}
	p.Photos = copies
	// Snapshot form tokens and source bindings are not actions in an archive.
	p.SettingForm, p.ShippingForm, p.CombinedForm, p.NoteForm = "", "", "", ""
	p.DesignerID, p.ClientID = "", ""
	p.ClientPhotoIDs = nil
	s.ShippingRecords = append(s.ShippingRecords, ShippingRecord{ID: token(), ClientName: clientName, Project: p, Receipts: append([]MaterialReceipt{}, s.MaterialReceipts[p.Number]...), InspectedAt: *p.Shipping.Completed})
	return copies, nil
}
func (a *App) saveShippingRecord(id string, r *http.Request) error {
	record := shippingRecord(&a.state, id)
	if record == nil {
		return errors.New("보관할 수주를 찾을 수 없습니다.")
	}
	if record.ArchivedAt != nil {
		return nil
	} // repeat POST after successful redirect is harmless
	memo := strings.TrimSpace(r.FormValue("memo"))
	if utf8.RuneCountInString(memo) > 2000 {
		return errors.New("메모는 2,000자 이내로 입력해 주세요.")
	}
	count := 0
	if r.MultipartForm != nil {
		count = len(r.MultipartForm.File["photos"])
	}
	if count < 1 || count > 20 {
		return errors.New("출하사진을 1장 이상, 최대 20장 등록해 주세요.")
	}
	photos, err := a.uploadReducedPhotos(r, 20)
	if err != nil {
		return err
	}
	err = a.commitAnnouncement(func(s *State) error {
		record := shippingRecord(s, id)
		if record == nil || record.ArchivedAt != nil {
			return errors.New("이미 변경된 보관 자료입니다.")
		}
		now := time.Now()
		record.Photos, record.Memo, record.ArchivedAt = photos, memo, &now
		return nil
	})
	if err != nil {
		a.pruneCommunityPhotos(photos)
	}
	return err
}
func (a *App) shippingRecordsHTTP(w http.ResponseWriter, r *http.Request) bool {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 0 || parts[0] != "shipping-records" && parts[0] != "shipping-archive-admin" {
		return false
	}
	admin := parts[0] == "shipping-archive-admin"
	csrf, base := a.state.Board, "/shipping-records/"+a.state.Board
	var rest []string
	if admin {
		session, ok := a.admin(w, r)
		if !ok {
			return true
		}
		csrf, base = session.CSRF, "/shipping-archive-admin"
		rest = parts[1:]
	} else {
		if len(parts) < 2 || !equal(parts[1], a.state.Board) {
			http.NotFound(w, r)
			return true
		}
		rest = parts[2:]
	}
	if r.Method == "POST" && !equal(r.FormValue("csrf"), csrf) {
		http.Error(w, "인증 정보가 올바르지 않습니다.", 403)
		return true
	}
	if admin && len(rest) == 1 && rest[0] == "recover" && r.Method == "POST" {
		if err := a.recoverShippingRecord(number(r.FormValue("project"))); err != nil {
			http.Error(w, err.Error(), 400)
			return true
		}
		http.Redirect(w, r, base+"?pending=1&recovered=1", http.StatusSeeOther)
		return true
	}
	if len(rest) == 1 && rest[0] == "status" && r.Method == "GET" {
		pending := []any{}
		for _, record := range a.state.ShippingRecords {
			if record.ArchivedAt == nil {
				pending = append(pending, map[string]any{"id": record.ID, "project_id": record.Project.ID, "number": record.Project.Number, "name": record.Project.Name})
			}
		}
		jsonReply(w, map[string]any{"pending": pending, "pending_signature": shippingPendingSignature(&a.state)})
		return true
	}
	calendarURL := "/work/" + a.state.Board + "?entry=skip&design-tab=calendar"
	if admin {
		calendarURL = "/?design-tab=calendar"
	}
	view := map[string]any{"PhotoReset": r.URL.Query().Get("photo_reset") == "1", "Recovered": r.URL.Query().Get("recovered") == "1", "CalendarURL": calendarURL, "Board": a.state.Board, "CSRF": csrf, "Base": base, "ShippingAdmin": admin, "WorkerURL": "/work/" + a.state.Board}
	if len(rest) == 0 && r.Method == "GET" {
		if admin {
			missing := []Project{}
			for _, p := range a.state.Projects {
				if shippingRecordRecoverable(&a.state, p) {
					missing = append(missing, p)
				}
			}
			view["MissingProjects"] = missing
		}
		list, err := shippingListView(&a.state, base, r.URL.Query(), time.Now())
		if err != nil {
			http.Error(w, err.Error(), 400)
			return true
		}
		view["PendingSignature"] = shippingPendingSignature(&a.state)
		for key, value := range list {
			view[key] = value
		}
		a.render(w, "shipping-records.html", view)
		return true
	}
	if len(rest) > 0 {
		record := shippingRecord(&a.state, rest[0])
		if record == nil {
			http.NotFound(w, r)
			return true
		}
		if len(rest) == 3 && rest[1] == "photo" && r.Method == "GET" {
			photos := append(append([]Photo{}, record.Project.Photos...), record.Photos...)
			if r.URL.Query().Get("download") == "1" {
				w.Header().Set("Content-Disposition", `attachment; filename="shipment-photo.jpg"`)
			}
			a.photo(w, r, photos, rest[2])
			return true
		}
		if len(rest) == 2 && r.Method == "POST" {
			if rest[1] == "add-photos" {
				if err := a.addShippingPhotos(record.ID, r); err != nil {
					http.Error(w, err.Error(), 400)
					return true
				}
				http.Redirect(w, r, base+"/"+record.ID+"?photos_added=1", http.StatusSeeOther)
				return true
			}
			if rest[1] == "save" && !admin {
				if err := a.saveShippingRecord(record.ID, r); err != nil {
					http.Error(w, err.Error(), 400)
					return true
				}
				http.Redirect(w, r, base+"/"+record.ID+"?saved=1", http.StatusSeeOther)
				return true
			}
			if rest[1] == "delete" && admin {
				if r.FormValue("confirm") != "1" {
					http.Error(w, "삭제 확인이 필요합니다.", 400)
					return true
				}
				reset, err := a.deleteShippingArchive(record.ID)
				if err != nil {
					http.Error(w, err.Error(), 400)
					return true
				}
				target := base
				if reset {
					target += "?pending=1&photo_reset=1"
				}
				http.Redirect(w, r, target, http.StatusSeeOther)
				return true
			}
		}
		if len(rest) == 1 && r.Method == "GET" {
			view["ResetToPending"] = shippingSourceComplete(&a.state, *record)
			view["CanAddPhotos"] = record.ArchivedAt != nil && len(record.Photos) < 20
			view["PhotoRevision"], view["UploadKey"] = shippingPhotoRevision(record), token()
			view["PhotosAdded"] = r.URL.Query().Get("photos_added") == "1"
			view["ShippingPending"] = record.ArchivedAt == nil
			view["Record"], view["Saved"] = record, r.URL.Query().Get("saved") == "1"
			phone := cleanPhone(record.Project.DesignerPhone)
			if len(phone) >= 7 && len(phone) <= 30 {
				view["CallURL"] = template.URL("tel:" + phone)
			}
			a.render(w, "shipping-records.html", view)
			return true
		}
	}
	http.NotFound(w, r)
	return true
}
func (a *App) shippingRecordMarkup(s, name string, value any) string {
	if !strings.Contains(s, "</body>") {
		return s
	}
	worker := name == "work.html"
	if name == "work.html" || name == "admin.html" {
		board := template.HTMLEscapeString(a.state.Board)
		base := "/shipping-records/" + board
		if !worker {
			base = "/shipping-archive-admin"
		}
		tabs := `<nav class="shipment-tabs" aria-label="출고 자료"><span class="selected">출고 일정</span><a href="` + base + `">출하 기록</a></nav>`
		s = strings.Replace(s, `<div class="calendar-body">`, tabs+`<div class="calendar-body">`, 1)
		if worker {
			n := 0
			for _, record := range a.state.ShippingRecords {
				if record.ArchivedAt == nil {
					n++
				}
			}
			link := `<a class="shipment-pending-link" href="` + base + `?pending=1">📷 출하사진 필요 <strong data-shipping-pending-count>` + fmt.Sprint(n) + `</strong><span>사진 등록 후 완료·보관 ›</span></a>`
			s = strings.Replace(s, `<div class="search-meta">`, link+`<div class="search-meta">`, 1)
		}
		s = strings.Replace(s, "</head>", `<meta name="shipping-records-board" content="`+board+`"><script defer src="/shipping-records-v1.4.53.js"></script></head>`, 1)
	}
	if name == "shipping-records.html" || name == "work.html" || name == "admin.html" {
		s = strings.Replace(s, "</head>", `<link rel="stylesheet" href="/shipping-records-v1.4.52.css"></head>`, 1)
	}
	return s
}
