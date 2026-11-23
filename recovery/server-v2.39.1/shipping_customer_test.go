package main

import (
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

func TestShippingCustomerConventionEverySurface(t *testing.T) {
	a := exportFixture(t)
	cookie := adminCookie(a)
	for _, tc := range []struct{ name, stored, want string }{
		{"은일 12축", "", "은일"},
		{"쿼드메디슨 3축, 긴급건", "쿼드메디슨", "쿼드메디슨"},
		{"비엔에스 4축 (선진행, 예가)", "다른 템플릿 이름", "비엔에스"},
		{"고객 사명 2 축", "", "고객 사명"},
		{"축 정보 없는 수주", "기존 고객사", "기존 고객사"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a.state.ClientNotes = nil
			r := a.state.ShippingRecords[0]
			r.Project.Name = tc.name
			r.ClientName = tc.stored
			r.ClientSourceID = ""
			r.ClientSourcePhotoIDs = nil
			a.state.ShippingRecords[0] = r
			before := string(jsonRaw(a.state))
			got := a.shippingCurrentNotes(&a.state, r)
			if got.ClientName != tc.want {
				t.Fatal("detail customer", got.ClientName, tc.want)
			}
			list, e := shippingListView(&a.state, "/shipping-archive-admin", url.Values{"q": {tc.want}}, time.Now())
			if e != nil || !strings.Contains(string(jsonRaw(list)), `"ClientName":"`+tc.want+`"`) {
				t.Fatal("list customer/search", e, list)
			}
			for _, base := range []string{"/shipping-archive-admin/", "/shipping-records/" + a.state.Board + "/"} {
				w := call(a, "GET", base+r.ID, nil, cookie)
				if w.Code != 200 || !strings.Contains(w.Body.String(), "고객사 <b>"+tc.want+"</b>") {
					t.Fatal("customer omitted in HTTP", w.Code)
				}
			}
			if before != string(jsonRaw(a.state)) {
				t.Fatal("view rewrote archived data")
			}
			job, records, _, e := a.newShippingExportLocked(hashToken(cookie.Value), token(), r.ID)
			if e != nil || records[0].ClientName != tc.want {
				t.Fatal("export customer mismatch", e)
			}
			a.runShippingExport(job, records)
			if job.State != "ready" {
				t.Fatal(job.Message)
			}
			f, e := excelize.OpenFile(filepath.Join(job.Dir, "export.xlsx"))
			if e != nil {
				t.Fatal(e)
			}
			value, _ := f.GetCellValue(f.GetSheetName(1), "C8")
			f.Close()
			if value != tc.want {
				t.Fatal("XLSX customer", value)
			}
		})
	}
}
func TestShippingCustomerSavedWithoutTemplateAndRestart(t *testing.T) {
	a := fixture(t)
	a.state.Projects[0].Name = "은일 12축"
	a.state.Projects[0].ClientID = ""
	r := approveShipment(t, a)
	if r.ClientName != "은일" {
		t.Fatal("approval capture requires a template", r.ClientName)
	}
	if w := shippingUpload(t, a, r.ID, 1, false); w.Code != 303 {
		t.Fatal(w.Code)
	}
	fresh, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	r = *shippingRecord(&fresh.state, r.ID)
	if r.ClientName != "은일" {
		t.Fatal("archive/restart lost customer", r.ClientName)
	}
	// Older archives may have no saved customer and no connected project left.
	r.ClientName = ""
	fresh.state.Projects = nil
	if got := fresh.shippingCurrentNotes(&fresh.state, r).ClientName; got != "은일" {
		t.Fatal("legacy archive needs live order", got)
	}
}
func TestShippingCustomerExplicitSourceAndUnknown(t *testing.T) {
	a := exportFixture(t)
	cookie := adminCookie(a)
	r := a.state.ShippingRecords[0]
	r.ClientName = ""
	r.Project.Name = "축 표기 없는 수주"
	r.ClientSourceID = "bound-client"
	a.state.ClientNotes = []ClientNote{{ID: r.ClientSourceID, Name: "연결 고객사"}}
	a.state.Projects = nil
	if got := shippingCustomerName(&a.state, r); got != "연결 고객사" {
		t.Fatal("explicit source not used", got)
	}
	r.ClientSourceID = ""
	a.state.ShippingRecords[0] = r
	w := call(a, "GET", "/shipping-archive-admin/"+r.ID, nil, cookie)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "고객사 <b>미등록</b>") {
		t.Fatal("unknown customer label silently hidden", w.Code)
	}
	if got := shippingCustomerName(&a.state, r); got != "" {
		t.Fatal("guessed unrelated customer", got)
	}
}
