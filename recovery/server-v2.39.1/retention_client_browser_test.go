package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestRetentionClientBrowser(t *testing.T) {
	if os.Getenv("RETENTION_CLIENT_BROWSER") != "1" {
		t.Skip("optional browser")
	}
	a := fixture(t)
	cookie := adminCookie(a)
	a.state.Projects[0].ScheduleMissing = true
	a.state.Projects[0].ShippingDueDate = "2026-11-24"
	c := ClientNote{ID: token(), Name: "원본 사진 수정 시험", Text: "안내", Revision: 1}
	copies := []Photo{}
	for _, red := range []uint8{30, 130, 230} {
		c.Photos = append(c.Photos, duplicateTestPhoto(t, a, red, false))
		copies = append(copies, duplicateTestPhoto(t, a, red, true))
	}
	own := []Photo{duplicateTestPhoto(t, a, 60, false), duplicateTestPhoto(t, a, 190, false)}
	a.state.ClientNotes = []ClientNote{c}
	a.state.Projects[0].ClientID = c.ID
	a.state.Projects[0].Photos = append(copies, own...)
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg, _ := json.Marshal(map[string]any{"cookie": map[string]string{"name": cookie.Name, "value": cookie.Value}, "client": c.ID, "board": a.state.Board})
	cmd := exec.Command("node", "audit/browser-retention-client-v1.4.95.cjs", srv.URL, string(cfg))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
