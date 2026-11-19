package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestNativeThemeBrowser(t *testing.T) {
	if os.Getenv("NATIVE_THEME_BROWSER") != "1" {
		t.Skip("optional native app web theme audit")
	}
	a := accountFixture(t)
	a.state.Integration.MobileEnabled = true
	c := adminCookie(a)
	id := token()
	designer := token()
	if err := a.transaction(func(s *State) error {
		s.ClientNotes = []ClientNote{{ID: id, Name: "테마 시험 업체", Text: "검사 주의사항", Revision: 1}}
		s.Designers = []Designer{{ID: designer, Name: "설계 담당", Phone: "01012345678", Revision: 1}}
		s.Projects[0].ClientID = id
		s.Projects[0].DesignerID = designer
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a)
	defer srv.Close()
	paths := []string{"/mobile-admin", "/projects/1/edit", "/projects/1/inspection", "/client-notes", "/client-notes/" + id + "/edit", "/designers", "/announcements", "/community-admin", "/inspection-requests", "/shipping-archive-admin", "/android-admin-settings", "/admin-accounts", "/mobile-access", "/tts-settings", "/market-settings", "/schedule", "/backup", "/settings", "/orders/import", "/data-storage", "/report-settings", "/community/" + a.state.Board, "/notices/" + a.state.Board, "/shipping-records/" + a.state.Board}
	cfg := map[string]any{"cookie": map[string]string{"name": c.Name, "value": c.Value}, "paths": paths, "board": a.state.Board}
	cmd := exec.Command("node", "audit/browser-native-theme-v1.4.75.cjs", srv.URL, string(jsonRaw(cfg)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}
