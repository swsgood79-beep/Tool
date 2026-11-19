package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestShippingMobilePhotoBrowser(t *testing.T) {
	if os.Getenv("SHIPPING_MOBILE_BROWSER") != "1" {
		t.Skip("optional actual mobile photo compatibility browser")
	}
	a := fixture(t)
	pending, archived := []string{}, []string{}
	for i := 0; i < 8; i++ {
		if i > 0 {
			p := Project{ID: a.state.next(), Number: "MOBILE-" + fmtID(int64(i)), Name: "휴대폰 사진 시험", Revision: 1, InfoRevision: 1}
			p.rotate()
			a.state.Projects = []Project{p}
		}
		r := approveShipment(t, a)
		if i < 4 {
			pending = append(pending, r.ID)
		} else {
			if w := shippingUpload(t, a, r.ID, 1, false); w.Code != 303 {
				t.Fatal(w.Code)
			}
			archived = append(archived, r.ID)
		}
	}
	old := map[string]ShippingRecord{}
	for _, r := range a.state.ShippingRecords {
		old[r.ID] = r
	}
	cookie := adminCookie(a)
	srv := httptest.NewServer(a)
	defer srv.Close()
	cmd := exec.Command("node", "audit/browser-shipping-mobile-v1.4.76.cjs", srv.URL, a.state.Board, string(jsonRaw(map[string]any{"pending": pending, "archived": archived, "cookie": map[string]string{"name": cookie.Name, "value": cookie.Value}})))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
	for _, r := range a.state.ShippingRecords {
		before := old[r.ID]
		if r.ArchivedAt == nil {
			t.Fatal("not archived", r.ID)
		}
		if before.ArchivedAt != nil {
			if len(r.Photos) != 3 || r.Photos[0].ID != before.Photos[0].ID || !r.ArchivedAt.Equal(*before.ArchivedAt) {
				t.Fatal("append changed old photo/date", r.ID, len(r.Photos))
			}
		} else if len(r.Photos) != 2 {
			t.Fatal("initial photo count", len(r.Photos))
		}
	}
}
