package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func notesTestPhoto(t *testing.T, a *App, name string, red uint8) Photo {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, 4, 3))
	im.Set(0, 0, color.RGBA{R: red, A: 255})
	var raw bytes.Buffer
	png.Encode(&raw, im)
	data, e := protect(raw.Bytes())
	if e != nil {
		t.Fatal(e)
	}
	id := token()
	os.MkdirAll(filepath.Join(a.dir, "photos"), 0700)
	if e = os.WriteFile(filepath.Join(a.dir, "photos", id), data, 0600); e != nil {
		t.Fatal(e)
	}
	return Photo{ID: id, Name: name, MIME: "image/png", Width: 4, Height: 3, Size: raw.Len()}
}
func notesEditClient(t *testing.T, a *App, cookie *http.Cookie, text string, add bool, remove string) {
	t.Helper()
	c := a.state.ClientNotes[0]
	var body bytes.Buffer
	m := multipart.NewWriter(&body)
	for k, v := range map[string]string{"csrf": "admin-csrf", "action": "save", "id": c.ID, "revision": fmtID(c.Revision), "client_name": c.Name, "client_text": text} {
		m.WriteField(k, v)
	}
	if remove != "" {
		m.WriteField("remove_photo", remove)
	}
	if add {
		part, _ := m.CreateFormFile("photos", "추가 원본.png")
		im := image.NewRGBA(image.Rect(0, 0, 9, 7))
		png.Encode(part, im)
	}
	m.Close()
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791/client-notes", &body)
	r.Header.Set("Content-Type", m.FormDataContentType())
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != 303 {
		t.Fatal("source edit", w.Code, w.Body.String())
	}
}
func TestShippingNotesFollowSourcePreserveEvidence(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "bound", true: "legacy"}[legacy], func(t *testing.T) {
			a := fixture(t)
			cookie := adminCookie(a)
			own := notesTestPhoto(t, a, "수주 전용.png", 10)
			source := notesTestPhoto(t, a, "업체 원본.png", 20)
			a.state.ClientNotes = []ClientNote{{ID: "client", Name: "회사", Text: "이전 공통 내용", Revision: 1, Photos: []Photo{source}}}
			a.state.Projects[0].ClientID = "client"
			a.state.Projects[0].Photos = []Photo{own}
			a.state.Projects[0].AdditionalInstructions = "수주만의 메모"
			r := approveShipment(t, a)
			if w := shippingUpload(t, a, r.ID, 1, false); w.Code != 303 {
				t.Fatal(w.Code)
			}
			r = *shippingRecord(&a.state, r.ID)
			before := string(jsonRaw([]any{r.Project.Setting, r.Project.Shipping, r.Receipts, r.Photos, r.Attachments, r.InspectedAt, r.ArchivedAt, r.Memo}))
			if legacy {
				p := shippingRecord(&a.state, r.ID)
				p.ClientSourceID = ""
				p.ClientSourcePhotoIDs = nil
				p.ClientSourceRevision = 0
			}
			notesEditClient(t, a, cookie, "수정된 공통 내용", true, source.ID)
			current := shippingCurrentNotes(&a.state, *shippingRecord(&a.state, r.ID))
			if current.Project.SpecialInstructions != "수정된 공통 내용" || current.Project.AdditionalInstructions != "수주만의 메모" || len(current.Project.Photos) != 2 || current.Project.Photos[0].ID != own.ID || current.Project.Photos[1].Name != "추가 원본.png" {
				t.Fatalf("source/own photos mismatch %+v", current)
			}
			after := string(jsonRaw([]any{current.Project.Setting, current.Project.Shipping, current.Receipts, current.Photos, current.Attachments, current.InspectedAt, current.ArchivedAt, current.Memo}))
			if before != after {
				t.Fatal("inspection or shipping evidence changed")
			}
			for _, base := range []string{"/shipping-archive-admin/", "/shipping-records/" + a.state.Board + "/"} {
				w := call(a, "GET", base+r.ID, nil, cookie)
				if w.Code != 200 || !strings.Contains(w.Body.String(), "수정된 공통 내용") {
					t.Fatal("detail did not refresh", w.Code)
				}
				w = call(a, "GET", base+r.ID+"/photo/"+current.Project.Photos[1].ID, nil, cookie)
				if w.Code != 200 {
					t.Fatal("new photo not accessible", w.Code)
				}
			}
			job, list, _, e := a.newShippingExportLocked(hashToken(cookie.Value), token(), r.ID)
			if e != nil || len(list) != 1 || list[0].Project.SpecialInstructions != "수정된 공통 내용" || len(list[0].Project.Photos) != 2 {
				t.Fatal("export snapshot stale", e)
			}
			a.runShippingExport(job, list)
			if job.State != "ready" {
				t.Fatal("export failed", job.State, job.Message)
			}
			if e = a.transaction(func(s *State) error { s.Projects = nil; return nil }); e != nil {
				t.Fatal(e)
			}
			notesEditClient(t, a, cookie, "수주 제거 후 공통 수정", true, "")
			fresh, e := newApp(a.dir)
			if e != nil {
				t.Fatal(e)
			}
			r = *shippingRecord(&fresh.state, r.ID)
			if r.Project.SpecialInstructions != "수주 제거 후 공통 수정" || len(r.Project.Photos) != 3 || r.Project.AdditionalInstructions != "수주만의 메모" {
				t.Fatal("detached archive/restart lost notes", r)
			}
			c := fresh.state.ClientNotes[0]
			cookie = adminCookie(fresh)
			w := call(fresh, "POST", "/client-notes", url.Values{"csrf": {"admin-csrf"}, "action": {"delete"}, "id": {c.ID}, "revision": {fmtID(c.Revision)}}, cookie)
			if w.Code != 303 {
				t.Fatal(w.Code, w.Body.String())
			}
			r = shippingCurrentNotes(&fresh.state, *shippingRecord(&fresh.state, r.ID))
			if r.Project.SpecialInstructions != "수주 제거 후 공통 수정" || len(r.Project.Photos) != 3 {
				t.Fatal("source deletion erased archived notes")
			}
		})
	}
}
func TestShippingNotesLegacyDoesNotGuessAnotherCycle(t *testing.T) {
	a := fixture(t)
	a.state.ClientNotes = []ClientNote{{ID: "client", Text: "다른 검사 내용", Revision: 2}}
	p := a.state.Projects[0]
	p.ClientID = "client"
	a.state.Projects[0] = p
	r := ShippingRecord{Project: p}
	r.Project.SpecialInstructions = "과거 내용"
	r.Project.Shipping.EventID = 999
	got := shippingCurrentNotes(&a.state, r)
	if got.Project.SpecialInstructions != "과거 내용" || got.ClientSourceID != "" {
		t.Fatal("unrelated cycle linked")
	}
}
