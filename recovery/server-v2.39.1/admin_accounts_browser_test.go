package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestIndividualAdminAccountsBrowser(t *testing.T) {
	if os.Getenv("ADMIN_ACCOUNTS_BROWSER") != "1" {
		t.Skip("optional actual two-browser account flow")
	}
	a := accountFixture(t)
	owner := adminCookie(a)
	createAccount(t, a, owner, "qa_base")
	ids := []int64{a.state.Projects[0].ID}
	for i := 1; i < 4; i++ {
		if err := a.transaction(func(s *State) error {
			p := Project{ID: s.next(), Number: "ACCOUNT-" + fmtID(int64(i)), Name: "계정 공동 검사 " + fmtID(int64(i)), Revision: 1, InfoRevision: 1}
			p.rotate()
			s.Projects = append(s.Projects, p)
			ids = append(ids, p.ID)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range ids {
		p := a.state.project(id)
		if err := a.requestInspection(id, "combined", "작업자", p.SettingForm, p.CombinedForm); err != nil {
			t.Fatal(err)
		}
	}
	srv := httptest.NewServer(a)
	defer srv.Close()
	meta := map[string]any{"ids": ids, "owner": map[string]string{"name": owner.Name, "value": owner.Value}, "password": accountMemberPassword, "ownerPassword": accountOwnerPassword}
	cmd := exec.Command("node", "audit/browser-admin-accounts-v1.4.54.cjs", srv.URL, string(jsonRaw(meta)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if !a.passwordMatches(accountOwnerPassword) {
		t.Fatal("browser changed owner password")
	}
	if len(a.state.AdminAccounts) != 5 {
		t.Fatal("wrong account count")
	}
	for _, id := range ids {
		p := a.state.project(id)
		if p.Setting.Completed == nil || p.Shipping.Completed == nil {
			t.Fatal("member did not complete shared inspection")
		}
	}
}
