package main

import (
	"github.com/xuri/excelize/v2"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func dateValues(r *ShippingRecord, date string) url.Values {
	return url.Values{"csrf": {"admin-csrf"}, "date_revision": {shippingDateRevision(r)}, "shipping_date": {date}}
}
func TestShippingDatePendingArchiveAndPreservation(t *testing.T) {
	for _, pending := range []bool{true, false} {
		t.Run(map[bool]string{true: "pending", false: "archived"}[pending], func(t *testing.T) {
			a := exportFixture(t)
			cookie := adminCookie(a)
			r := a.state.ShippingRecords[0]
			r.Project.ShippingDueDate = ""
			if pending {
				r.ArchivedAt = nil
				r.Photos = nil
			}
			a.state.ShippingRecords[0] = r
			original := string(jsonRaw(r))
			events := string(jsonRaw(a.state.Events))
			projects := string(jsonRaw(a.state.Projects))
			path := "/shipping-archive-admin/" + r.ID + "/edit-date"
			v := dateValues(&r, "2026-11-24")
			w := call(a, "POST", path, v, cookie)
			if w.Code != 303 {
				t.Fatal(w.Code, w.Body.String())
			}
			out := *shippingRecord(&a.state, r.ID)
			if out.Project.ShippingDueDate != "2026-11-24" || len(out.DateEdits) != 1 || out.DateEdits[0].ActorUsername != "wssong" {
				t.Fatal("date not recorded", out)
			}
			copy := out
			copy.Project.ShippingDueDate = ""
			copy.DateRevision = 0
			copy.DateEdits = nil
			if string(jsonRaw(copy)) != original || events != string(jsonRaw(a.state.Events)) || projects != string(jsonRaw(a.state.Projects)) {
				t.Fatal("unrelated dates/evidence/source modified")
			}
			if w = call(a, "POST", path, v, cookie); w.Code != 409 {
				t.Fatal("stale date accepted", w.Code)
			}
			if pending {
				if w = shippingUpload(t, a, r.ID, 1, false); w.Code != 303 {
					t.Fatal(w.Code)
				}
			}
			job, records, _, e := a.newShippingExportLocked(hashToken(cookie.Value), token(), r.ID)
			if e != nil {
				t.Fatal(e)
			}
			a.runShippingExport(job, records)
			if job.State != "ready" {
				t.Fatal(job.Message)
			}
			f, e := excelize.OpenFile(filepath.Join(job.Dir, "export.xlsx"))
			if e != nil {
				t.Fatal(e)
			}
			got, _ := f.GetCellValue(f.GetSheetName(1), "C5")
			f.Close()
			if got != "2026.11.24" {
				t.Fatal("Excel date", got)
			}
			fresh, e := newApp(a.dir)
			if e != nil {
				t.Fatal(e)
			}
			if shippingRecord(&fresh.state, r.ID).Project.ShippingDueDate != "2026-11-24" {
				t.Fatal("restart lost date")
			}
		})
	}
}
func TestShippingDateSecurityAndInvalid(t *testing.T) {
	a := exportFixture(t)
	cookie := adminCookie(a)
	r := a.state.ShippingRecords[0]
	path := "/shipping-archive-admin/" + r.ID + "/edit-date"
	before := string(jsonRaw(a.state.ShippingRecords))
	if w := call(a, "POST", path, dateValues(&r, "2026-11-24")); w.Code != 401 {
		t.Fatal(w.Code)
	}
	bad := dateValues(&r, "2026-11-24")
	bad.Set("csrf", "bad")
	if w := call(a, "POST", path, bad, cookie); w.Code != 403 {
		t.Fatal(w.Code)
	}
	bad.Set("csrf", a.state.Board)
	if w := call(a, "POST", "/shipping-records/"+a.state.Board+"/"+r.ID+"/edit-date", bad); w.Code != 403 {
		t.Fatal(w.Code)
	}
	for _, date := range []string{"", "2026-02-30", "2026-2-3", "garbage"} {
		w := call(a, "POST", path, dateValues(&r, date), cookie)
		if w.Code != 409 || !strings.Contains(w.Body.String(), "올바른 날짜") {
			t.Fatal("validation missing", date, w.Code)
		}
	}
	if before != string(jsonRaw(a.state.ShippingRecords)) {
		t.Fatal("invalid/unauthorized date altered record")
	}
}
