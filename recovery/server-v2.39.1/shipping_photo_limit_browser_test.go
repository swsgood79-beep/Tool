package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestShippingThirtyPhotoBrowser(t *testing.T) {
	if os.Getenv("SHIPPING_LIMIT_BROWSER") != "1" {
		t.Skip("optional actual thirty photo browser flow")
	}
	a := fixture(t)
	pending, archived := []string{}, []string{}
	for i := 0; i < 8; i++ {
		if i > 0 {
			p := Project{ID: a.state.next(), Number: "LIMIT-" + fmtID(int64(i)), Name: "사진 수량 시험", Revision: 1, InfoRevision: 1}
			p.rotate()
			a.state.Projects = []Project{p}
		}
		r := approveShipment(t, a)
		if i < 4 {
			pending = append(pending, r.ID)
		} else {
			if w := shippingUpload(t, a, r.ID, 20, false); w.Code != 303 {
				t.Fatal(w.Code)
			}
			archived = append(archived, r.ID)
		}
	}
	srv := httptest.NewServer(a)
	defer srv.Close()
	cmd := exec.Command("node", "audit/browser-shipping-limit-v1.4.60.cjs", srv.URL, a.state.Board, string(jsonRaw(map[string]any{"pending": pending, "archived": archived})))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
	for _, r := range a.state.ShippingRecords {
		if len(r.Photos) != 30 || r.ArchivedAt == nil {
			t.Fatal("not archived at 30", r.ID, len(r.Photos))
		}
	}
}
