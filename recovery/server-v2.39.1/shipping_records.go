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

const shippingPhotoLimit = 30

// An independent snapshot, captured in the approval transaction. The live schedule
// remains a projection of the connected sheet and never owns archive records.
type ShippingRecord struct {
	DateRevision         int64                `json:",omitempty"`
	DateEdits            []ShippingDateEdit   `json:",omitempty"`
	WorkerRevision       int64                `json:",omitempty"`
	WorkerEdits          []ShippingWorkerEdit `json:",omitempty"`
	ClientSourceRevision int64                `json:",omitempty"`
	ClientSourceID       string               `json:",omitempty"`
	ClientSourcePhotoIDs []string             `json:",omitempty"`
	// Export-only provenance. Never persisted into an archive snapshot.
	ExportSourceNote string               `json:"-"`
	Attachments      []ShippingAttachment `json:",omitempty"`
	PhotoUploadKeys  []string             `json:",omitempty"`
	ClientName       string
	ID               string
	Project          Project
	Receipts         []MaterialReceipt
	InspectedAt      time.Time
	ArchivedAt       *time.Time
	Photos           []Photo
	Memo             string
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
	clientSourceID, ownPhotoCount := p.ClientID, len(p.Photos)
	var clientSourceRevision int64
	source := append([]Photo{}, p.Photos...)
	for _, c := range s.ClientNotes {
		if c.ID == p.ClientID {
			clientName = c.Name
			clientSourceRevision = c.Revision
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
	sourceIDs := []string{}
	for _, ph := range copies[ownPhotoCount:] {
		sourceIDs = append(sourceIDs, ph.ID)
	}
	record := ShippingRecord{ClientSourceRevision: clientSourceRevision, ClientSourceID: clientSourceID, ClientSourcePhotoIDs: sourceIDs, ID: token(), ClientName: clientName, Project: p, Receipts: append([]MaterialReceipt{}, s.MaterialReceipts[p.Number]...), InspectedAt: *p.Shipping.Completed}
	record.ClientName = shippingCustomerName(s, record)
	s.ShippingRecords = append(s.ShippingRecords, record)
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
	if count < 1 || count > shippingPhotoLimit {
		return errors.New("출하사진을 1장 이상, 최대 30장 등록해 주세요.")
	}
	photos, err := a.uploadReducedPhotos(r, shippingPhotoLimit)
	if err != nil {
		return err
	}
	err = a.commitAnnouncement(func(s *State) error {
		record := shippingRecord(s, id)
		if record == nil || record.ArchivedAt != nil {
			return errors.New("이미 변경된 보관 자료입니다.")
		}
		metadata := shippingExportMetadata(s, *record)
		record.ClientName = shippingCustomerName(s, metadata)
		record.Project.ShippingDueDate = metadata.Project.ShippingDueDate
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
	view := map[string]any{"PhotoDeleted": r.URL.Query().Get("photo_deleted") == "1", "PhotoReset": r.URL.Query().Get("photo_reset") == "1", "Recovered": r.URL.Query().Get("recovered") == "1", "CalendarURL": calendarURL, "Board": a.state.Board, "CSRF": csrf, "Base": base, "ShippingAdmin": admin, "WorkerURL": "/work/" + a.state.Board}
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
		if policy := w.Header().Get("Content-Security-Policy"); !strings.Contains(policy, "data: blob:") {
			w.Header().Set("Content-Security-Policy", strings.Replace(policy, "img-src 'self' data:", "img-src 'self' data: blob:", 1))
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
			current := shippingCurrentNotes(&a.state, *record)
			photos := append(append(append([]Photo{}, current.Project.Photos...), record.Project.Photos...), record.Photos...)
			if r.URL.Query().Get("download") == "1" {
				w.Header().Set("Content-Disposition", `attachment; filename="shipment-photo.jpg"`)
			}
			a.photo(w, r, photos, rest[2])
			return true
		}
		if len(rest) == 2 && r.Method == "POST" {
			if rest[1] == "edit-date" {
				if !admin {
					http.Error(w, "출고일자 수정은 관리자만 할 수 있습니다.", 403)
					return true
				}
				session, _ := a.session(r)
				actor, err := inspectionActor(&a.state, session)
				if err != nil {
					http.Error(w, err.Error(), 403)
					return true
				}
				err = a.saveShippingDate(record.ID, r.FormValue("date_revision"), r.FormValue("shipping_date"), actor)
				if err == nil {
					http.Redirect(w, r, base+"/"+record.ID+"?date_saved=1", 303)
					return true
				}
				view["DateError"], view["EditShippingDate"] = err.Error(), r.FormValue("shipping_date")
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(409)
			}
			if rest[1] == "edit-workers" {
				if !admin {
					http.Error(w, "작업자 이름 수정은 관리자만 할 수 있습니다.", 403)
					return true
				}
				session, _ := a.session(r)
				actor, err := inspectionActor(&a.state, session)
				if err != nil {
					http.Error(w, err.Error(), 403)
					return true
				}
				err = a.saveShippingWorkers(record.ID, r.FormValue("workers_revision"), r.FormValue("setting_worker"), r.FormValue("shipping_worker"), actor)
				if err == nil {
					http.Redirect(w, r, base+"/"+record.ID+"?workers_saved=1", http.StatusSeeOther)
					return true
				}
				view["WorkersError"], view["EditSettingWorker"], view["EditShippingWorker"] = err.Error(), r.FormValue("setting_worker"), r.FormValue("shipping_worker")
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusConflict)
			}
			if rest[1] == "add-photos" {
				if err := a.addShippingPhotos(record.ID, r); err != nil {
					http.Error(w, err.Error(), 400)
					return true
				}
				next := base + "/" + record.ID + "?photos_added=1"
				if strings.Contains(r.Header.Get("Accept"), "application/json") {
					jsonReply(w, map[string]any{"ok": true, "next": next})
				} else {
					http.Redirect(w, r, next, http.StatusSeeOther)
				}
				return true
			}
			if rest[1] == "save" && !admin {
				if err := a.saveShippingRecord(record.ID, r); err != nil {
					http.Error(w, err.Error(), 400)
					return true
				}
				next := base + "/" + record.ID + "?saved=1"
				if strings.Contains(r.Header.Get("Accept"), "application/json") {
					jsonReply(w, map[string]any{"ok": true, "next": next})
				} else {
					http.Redirect(w, r, next, http.StatusSeeOther)
				}
				return true
			}
			if rest[1] == "delete-photo" && admin {
				if r.FormValue("confirm") != "1" {
					http.Error(w, "삭제 확인이 필요합니다.", 400)
					return true
				}
				nextID, err := a.deleteShippingPhoto(record.ID, r.FormValue("photo"), r.FormValue("photo_revision"))
				if err != nil {
					http.Error(w, err.Error(), 400)
					return true
				}
				next := base + "?photo_deleted=1"
				if nextID != "" {
					next = base + "/" + nextID + "?photo_deleted=1"
				}
				if strings.Contains(r.Header.Get("Accept"), "application/json") {
					jsonReply(w, map[string]any{"ok": true, "next": next})
				} else {
					http.Redirect(w, r, next, http.StatusSeeOther)
				}
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
		if view["WorkersError"] != nil || view["DateError"] != nil {
			rest = rest[:1]
		}
		if len(rest) == 1 && (r.Method == "GET" || view["WorkersError"] != nil || view["DateError"] != nil) {
			view["DateSaved"], view["DateRevision"] = r.URL.Query().Get("date_saved") == "1", shippingDateRevision(record)
			if view["DateError"] == nil {
				view["EditShippingDate"] = record.Project.ShippingDueDate
			}
			view["WorkersSaved"] = r.URL.Query().Get("workers_saved") == "1"
			view["WorkersRevision"] = shippingWorkersRevision(record)
			if view["WorkersError"] == nil {
				view["EditSettingWorker"], view["EditShippingWorker"] = record.Project.Setting.Worker, record.Project.Shipping.Worker
			}
			view["ResetToPending"] = shippingSourceComplete(&a.state, *record)
			view["CanAddPhotos"] = record.ArchivedAt != nil && len(record.Photos) < shippingPhotoLimit
			view["PhotoRevision"], view["UploadKey"] = shippingPhotoRevision(record), token()
			view["PhotosAdded"] = r.URL.Query().Get("photos_added") == "1"
			view["ShippingPending"] = record.ArchivedAt == nil
			current := a.shippingCurrentNotes(&a.state, *record)
			view["Record"], view["Saved"] = &current, r.URL.Query().Get("saved") == "1"
			phone := cleanPhone(record.Project.DesignerPhone)
			if len(phone) >= 7 && len(phone) <= 30 {
				view["CallURL"] = template.URL("tel:" + phone)
			}
			if policy := w.Header().Get("Content-Security-Policy"); !strings.Contains(policy, "data: blob:") {
				w.Header().Set("Content-Security-Policy", strings.Replace(policy, "img-src 'self' data:", "img-src 'self' data: blob:", 1))
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
		tabs := `<nav class="shipment-tabs" aria-label="출고 자료"><span class="selected" data-shipment-kind="schedule">출고 일정</span><a data-shipment-kind="archive" href="` + base + `">출하 기록</a></nav>`
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
		s = strings.Replace(s, "</head>", `<meta name="shipping-records-board" content="`+board+`"><script defer src="/shipping-records-v1.4.76.js"></script></head>`, 1)
	}
	if name == "shipping-records.html" {
		s = strings.Replace(s, "</head>", `<link rel="stylesheet" href="/shipment-date-v1.4.96.css"><link rel="stylesheet" href="/shipment-workers-v1.4.94.css"><link rel="stylesheet" href="/shipping-export-v1.4.83.css"><script defer src="/shipping-export-v1.4.84.js"></script><link rel="stylesheet" href="/shipment-attachments-v1.4.82.css"><script defer src="/fix-webm-duration-v1.0.6.js"></script><script defer src="/shipment-video-v1.4.82.js"></script><script defer src="/shipment-attachments-v1.4.82.js"></script><link rel="stylesheet" href="/shipment-photo-delete-v1.4.77.css"><script defer src="/shipment-photo-delete-v1.4.77.js"></script><link rel="stylesheet" href="/shipment-photo-viewer-v1.4.62.css"><script defer src="/shipment-photo-viewer-v1.4.62.js"></script></head>`, 1)
	}
	if name == "shipping-records.html" || name == "work.html" || name == "admin.html" {
		s = strings.Replace(s, "</head>", `<link rel="stylesheet" href="/shipping-records-v1.4.52.css"><link rel="stylesheet" href="/shipment-colors-v1.4.87.css"></head>`, 1)
	}
	return s
}
