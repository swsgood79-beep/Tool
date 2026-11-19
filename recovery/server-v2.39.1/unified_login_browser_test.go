package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestUnifiedLoginAndFastWorkerBrowser(t *testing.T) {
	if os.Getenv("UNIFIED_LOGIN_BROWSER") != "1" {
		t.Skip("optional actual unified login browser")
	}
	a := accountFixture(t)
	a.state.Integration.MobileEnabled = true
	c := adminCookie(a)
	createAccount(t, a, c, "qa_login")
	srv := httptest.NewServer(a)
	defer srv.Close()
	info := map[string]any{"board": a.state.Board, "ownerPassword": accountOwnerPassword, "memberPassword": accountMemberPassword}
	cmd := exec.Command("node", "audit/browser-unified-login-v1.4.66.cjs", srv.URL, string(jsonRaw(info)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}
func TestNativeAdminAppCurrentBrowser(t *testing.T) {
	if os.Getenv("UNIFIED_LOGIN_BROWSER") != "1" {
		t.Skip("optional current native shell browser")
	}
	a := accountFixture(t)
	a.state.Integration.MobileEnabled = true
	c := adminCookie(a)
	p := a.state.Projects[0]
	if err := a.requestInspection(p.ID, "combined", "김준필", p.SettingForm, p.CombinedForm); err != nil {
		t.Fatal(err)
	}
	if err := a.transaction(func(s *State) error {
		p := Project{ID: s.next(), Number: "A261029-005", Name: "엔티이엔지 1축", Revision: 1, InfoRevision: 1, ShippingDueDate: day(time.Now())}
		p.rotate()
		s.Projects = append(s.Projects, p)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a)
	defer srv.Close()
	info := map[string]any{"cookie": map[string]string{"name": c.Name, "value": c.Value}, "board": a.state.Board, "project": p.ID}
	cmd := exec.Command("node", "audit/browser-native-admin-app-v1.4.66.cjs", srv.URL, string(jsonRaw(info)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}
