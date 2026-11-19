package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestNativeClientMenuBrowser(t *testing.T) {
	if os.Getenv("NATIVE_CLIENT_MENU_BROWSER") != "1" {
		t.Skip("optional administrator app client-note edit browser flow")
	}
	a := accountFixture(t)
	a.state.Integration.MobileEnabled = true
	owner := adminCookie(a)
	createAccount(t, a, owner, "qa_client_menu")
	member, _ := accountLogin(t, a, "qa_client_menu", accountMemberPassword)
	id := token()
	if err := a.transaction(func(s *State) error {
		s.ClientNotes = []ClientNote{{ID: id, Name: "메뉴 시험 업체", Text: "기존 업체 원본", Revision: 1}}
		s.Projects[0].ClientID = id
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a)
	defer srv.Close()
	info := map[string]any{"cookies": []map[string]string{{"name": owner.Name, "value": owner.Value}, {"name": member.Name, "value": member.Value}}, "board": a.state.Board, "client": id}
	cmd := exec.Command("node", "audit/browser-native-client-menu-v1.4.73.cjs", srv.URL, string(jsonRaw(info)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(a.state.ClientNotes[0].Text, "메뉴 수정 ") || a.projectView(a.state.Projects[0]).SpecialInstructions != a.state.ClientNotes[0].Text {
		t.Fatal("client source edit not reflected in linked order")
	}
}
