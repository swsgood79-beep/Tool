package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func duplicateTestPhoto(t *testing.T, a *App, red uint8, legacy bool) Photo {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, 40, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 40; x++ {
			im.Set(x, y, color.RGBA{R: red, G: 80, B: 120, A: 255})
		}
	}
	var b bytes.Buffer
	png.Encode(&b, im)
	data := b.Bytes()
	id := token()
	p := Photo{ID: id, Name: "동일 이름.png", MIME: "image/png", Width: 40, Height: 30, Size: len(data), Legacy: legacy}
	if legacy {
		var e error
		data, _, _, e = canonicalJPEG(data)
		if e != nil {
			t.Fatal(e)
		}
		if e = a.createCanonicalPhoto(id, data); e != nil {
			t.Fatal(e)
		}
		p.MIME = "image/jpeg"
		p.Size = len(data)
	} else {
		raw, e := protect(data)
		if e != nil {
			t.Fatal(e)
		}
		os.MkdirAll(filepath.Join(a.dir, "photos"), 0700)
		if e = os.WriteFile(filepath.Join(a.dir, "photos", id), raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	return p
}
func TestShippingNotesDuplicateCopiesAndDateLabel(t *testing.T) {
	for _, known := range []bool{false, true} {
		t.Run(map[bool]string{false: "content_copy", true: "known_materialized_copy"}[known], func(t *testing.T) {
			a := fixture(t)
			cookie := adminCookie(a)
			src := duplicateTestPhoto(t, a, 200, false)
			copy := duplicateTestPhoto(t, a, 200, true)
			second := duplicateTestPhoto(t, a, 10, false)
			a.state.ClientNotes = []ClientNote{{ID: "duplicate-client", Name: "회사", Text: "공통 특이사항", Revision: 2, Photos: []Photo{src, second}}}
			a.state.Projects[0].ClientID = "duplicate-client"
			a.state.Projects[0].Photos = []Photo{copy}
			a.state.Projects[0].ShippingDueDate = "2026-11-23"
			if known {
				a.state.MaterializedPhotos = map[string]string{fmtID(a.state.Projects[0].ID) + ":" + src.ID: copy.ID}
			}
			r := approveShipment(t, a)
			if w := shippingUpload(t, a, r.ID, 1, false); w.Code != 303 {
				t.Fatal(w.Code)
			}
			r = *shippingRecord(&a.state, r.ID)
			before := string(jsonRaw(a.state))
			for n := 0; n < 3; n++ {
				out := a.shippingCurrentNotes(&a.state, r)
				if len(out.Project.Photos) != 2 {
					t.Fatalf("duplicates remain: %d", len(out.Project.Photos))
				}
				for _, ph := range out.Project.Photos {
					if ph.ID == copy.ID {
						t.Fatal("preferred old copy over source")
					}
				}
			}
			if before != string(jsonRaw(a.state)) {
				t.Fatal("view changed stored evidence")
			}
			for _, base := range []string{"/shipping-archive-admin/", "/shipping-records/" + a.state.Board + "/"} {
				w := call(a, "GET", base+r.ID, nil, cookie)
				body := w.Body.String()
				if w.Code != 200 || strings.Count(body, `alt="특이사항 사진"`) != 2 || !strings.Contains(body, "출고일자") || strings.Contains(body, "출고 예정") {
					t.Fatal("archive presentation incorrect", w.Code)
				}
			}
			key := token()
			job, records, _, e := a.newShippingExportLocked(hashToken(cookie.Value), key, r.ID)
			if e != nil || len(records) != 1 || len(records[0].Project.Photos) != 2 {
				t.Fatal("export duplicates", e)
			}
			a.runShippingExport(job, records)
			if job.State != "ready" {
				t.Fatal(job.State, job.Message)
			}
			book, e := excelize.OpenFile(filepath.Join(job.Dir, "export.xlsx"))
			if e != nil {
				t.Fatal(e)
			}
			label, _ := book.GetCellValue(book.GetSheetName(1), "A5")
			book.Close()
			if label != "출고일자" {
				t.Fatal("export date label", label)
			}
			// Source edits must not cause accumulated duplicate presentation.
			c := a.state.ClientNotes[0]
			w := call(a, "POST", "/client-notes", url.Values{"csrf": {"admin-csrf"}, "action": {"save"}, "id": {c.ID}, "revision": {fmtID(c.Revision)}, "client_name": {c.Name}, "client_text": {"새 내용"}}, cookie)
			if w.Code != 303 {
				t.Fatal(w.Code, w.Body.String())
			}
			fresh, e := newApp(a.dir)
			if e != nil {
				t.Fatal(e)
			}
			out := fresh.shippingCurrentNotes(&fresh.state, *shippingRecord(&fresh.state, r.ID))
			if len(out.Project.Photos) != 2 || out.Project.SpecialInstructions != "새 내용" {
				t.Fatal("duplicates after edit/restart")
			}
		})
	}
}
func TestShippingNotesDedupKeepsDifferentContentAndMissingFiles(t *testing.T) {
	a := fixture(t)
	src := duplicateTestPhoto(t, a, 220, false)
	own := duplicateTestPhoto(t, a, 20, false)
	missing := Photo{ID: token(), Name: src.Name, Width: src.Width, Height: src.Height, Size: src.Size}
	r := ShippingRecord{Project: Project{Photos: []Photo{own, src, missing, src}}, ClientSourcePhotoIDs: []string{src.ID}}
	before := string(jsonRaw(r))
	out := a.shippingCurrentNotes(&a.state, r)
	if len(out.Project.Photos) != 3 || out.Project.Photos[0].ID != own.ID || out.Project.Photos[2].ID != missing.ID {
		t.Fatal("different/unreadable photos hidden", out.Project.Photos)
	}
	if before != string(jsonRaw(r)) {
		t.Fatal("input slice mutated")
	}
}
