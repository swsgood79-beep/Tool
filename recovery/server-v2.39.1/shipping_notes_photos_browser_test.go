package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestShippingNotesPhotosBrowser(t *testing.T) {
	if os.Getenv("SHIPPING_NOTES_BROWSER") != "1" {
		t.Skip("optional shipping notes browser")
	}
	a := fixture(t)
	cookie := adminCookie(a)
	src := duplicateTestPhoto(t, a, 200, false)
	copy := duplicateTestPhoto(t, a, 200, true)
	second := duplicateTestPhoto(t, a, 10, false)
	a.state.ClientNotes = []ClientNote{{ID: "duplicate-client", Name: "시험 고객사", Text: "수정 반영된 특이사항", Revision: 2, Photos: []Photo{src, second}}}
	a.state.Projects[0].ClientID = "duplicate-client"
	a.state.Projects[0].Photos = []Photo{copy}
	a.state.Projects[0].ShippingDueDate = "2026-11-23"
	r := approveShipment(t, a)
	if w := shippingUpload(t, a, r.ID, 1, false); w.Code != 303 {
		t.Fatal(w.Code)
	}
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg, _ := json.Marshal(map[string]any{"cookie": map[string]string{"name": cookie.Name, "value": cookie.Value}, "record": r.ID, "board": a.state.Board})
	cmd := exec.Command("node", "audit/browser-shipping-notes-v1.4.92.cjs", srv.URL, string(cfg))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
