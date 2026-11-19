package main

import (
	"net/url"
	"strings"
	"testing"
)

func TestUnifiedOwnerUsernamePreservesPasswordPrivilegesAndAccountCount(t *testing.T) {
	a := accountFixture(t)
	owner := adminCookie(a)
	createAccount(t, a, owner, "qa_identity")
	hash := a.state.PasswordHash
	count := len(a.state.AdminAccounts)
	for _, path := range []string{"/login", "/login?mode=legacy", "/login?mode=account", "/mobile-admin/login"} {
		w := call(a, "GET", path, nil)
		s := w.Body.String()
		if w.Code != 200 || !strings.Contains(s, `name="username"`) || strings.Contains(s, "기존 관리자") || strings.Contains(s, "passkey/start") || strings.Contains(s, "mode=legacy") {
			t.Fatal("unified login form", path)
		}
	}
	c, w := accountLogin(t, a, "WSSONG", accountOwnerPassword)
	if c == nil || w.Code != 303 || a.sessions[hashToken(c.Value)].AccountID != "" {
		t.Fatal("wssong owner login", w.Code)
	}
	if a.state.PasswordHash != hash || len(a.state.AdminAccounts) != count {
		t.Fatal("username alias changed password or created extra account")
	}
	if w := call(a, "GET", "/admin-accounts", nil, c); w.Code != 200 || !strings.Contains(w.Body.String(), "<td>wssong</td>") || strings.Contains(w.Body.String(), "기존 로그인") {
		t.Fatal("owner management identity missing")
	}
	member, w := accountLogin(t, a, "qa_identity", accountMemberPassword)
	if member == nil || w.Code != 303 {
		t.Fatal("member login changed")
	}
	if w := call(a, "GET", "/admin-accounts", nil, member); w.Code != 403 {
		t.Fatal("member became owner")
	}
	if c, _ := accountLogin(t, a, "incorrect-id", accountOwnerPassword); c != nil {
		t.Fatal("owner password accepted for unknown username")
	}
	fresh, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	if c, _ := accountLogin(t, fresh, "wssong", accountOwnerPassword); c == nil {
		t.Fatal("owner alias failed after restart")
	}
	if w := accountManageCall(a, owner, url.Values{"action": {"create"}, "name": {"충돌"}, "username": {"wssong"}, "password": {accountMemberPassword}, "confirm": {accountMemberPassword}}); w.Code == 303 {
		t.Fatal("reserved owner username created as a new personal account")
	}
}
func TestPreexistingWssongPersonalAccountCannotGainOwnerPrivileges(t *testing.T) {
	a := accountFixture(t)
	owner := adminCookie(a)
	u := createAccount(t, a, owner, "old_wssong")
	id := u.ID
	u.Username = "wssong"
	c, w := accountLogin(t, a, "wssong", accountMemberPassword)
	if c == nil || w.Code != 303 || a.sessions[hashToken(c.Value)].AccountID != id {
		t.Fatal("existing personal identity lost or elevated")
	}
	c, w = accountLogin(t, a, "wssong", accountOwnerPassword)
	if c == nil || w.Code != 303 || a.sessions[hashToken(c.Value)].AccountID != "" {
		t.Fatal("owner alias lost on name collision")
	}
	u.PasswordHash = a.state.PasswordHash
	if c, w := accountLogin(t, a, "wssong", accountOwnerPassword); c != nil || w.Code != 409 {
		t.Fatal("ambiguous duplicate credentials elevated a role")
	}
}
func TestWorkerEntranceOpensOrdersWithoutDoorAssets(t *testing.T) {
	a := fixture(t)
	for _, query := range []string{"", "?entry=skip", "?entry=show", "?design-tab=calendar"} {
		w := call(a, "GET", "/work/"+a.state.Board+query, nil)
		s := w.Body.String()
		if w.Code != 200 || !strings.Contains(s, `class="work-list"`) {
			t.Fatal("work list missing")
		}
		for _, asset := range []string{"entry-motion.js", "entry-gate.js", "entry-gate.css", "data-entry-gate"} {
			if strings.Contains(s, asset) {
				t.Fatal("door animation still served", query, asset)
			}
		}
	}
}
