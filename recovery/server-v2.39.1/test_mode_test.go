package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsolatedModeRejectsExistingDataWithoutChanges(t *testing.T) {
	original := fixture(t)
	before := originalBytes(t, original.dir)
	if _, e := newTestApp(original.dir); e == nil {
		t.Fatal("existing data imported")
	}
	assertOriginalBytes(t, original.dir, before)
	if _, e := os.Stat(filepath.Join(original.dir, testMarker)); !os.IsNotExist(e) {
		t.Fatal("existing folder marked")
	}
	for _, n := range []string{"WanryoTalk", "AlphaWorkRecoveredServer"} {
		p := filepath.Join(t.TempDir(), n)
		if _, e := prepareTestDir(p); e == nil {
			t.Fatal("reserved original folder accepted")
		}
		if _, e := os.Stat(p); !os.IsNotExist(e) {
			t.Fatal("reserved folder modified")
		}
	}
}
func TestIsolatedModeFreshDemoRestartAndNetworkBlocked(t *testing.T) {
	a, e := newTestApp(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if !a.testMode || len(a.state.Projects) != 2 || a.state.Projects[0].Number != "TEST-001" {
		t.Fatal("isolated demo missing")
	}
	board := a.state.Board
	if _, e = a.httpClient().Get("https://github.com/"); e == nil {
		t.Fatal("external request allowed")
	}
	a.startTunnel()
	if a.integration.CancelTunnel != nil {
		t.Fatal("test tunnel started")
	}
	next, e := newTestApp(a.dir)
	if e != nil || next.state.Board != board || len(next.state.Projects) != 2 {
		t.Fatal("test restart lost or duplicated data", e)
	}
	for _, p := range []string{"/connection", "/backup/now", "/phone/register", "/mobile-admin/passkey/start", "/passkeys/auth/begin"} {
		r := call(next, "POST", p, nil)
		if r.Code != 403 {
			t.Fatal("external route allowed", p, r.Code)
		}
	}
	if next.sessionCookie() == "aw_recovered_session" || next.nonceCookie() == "aw_login_nonce" {
		t.Fatal("cookies overlap")
	}
}
func TestIsolatedModeIndependentWorkflowAndCookies(t *testing.T) {
	original := fixture(t)
	a, e := newTestApp(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	before := originalBytes(t, original.dir)
	page := call(a, "GET", "/login", nil)
	nonce := page.Result().Cookies()[0]
	if nonce.Name != "aw_recovery_test_nonce" {
		t.Fatal(nonce.Name)
	}
	res := call(a, "POST", "/login", url.Values{"csrf": {nonce.Value}, "password": {"isolated-password-123"}, "confirm": {"isolated-password-123"}}, nonce)
	if res.Code != 303 {
		t.Fatal(res.Code, res.Body.String())
	}
	var cookie *http.Cookie
	for _, c := range res.Result().Cookies() {
		if c.Name == a.sessionCookie() {
			cookie = c
		}
	}
	if cookie == nil {
		t.Fatal("test session missing")
	}
	if r := call(a, "GET", "/", nil, cookie); r.Code != 200 {
		t.Fatal("test admin login failed")
	}
	p := a.state.Projects[0]
	if e = a.requestInspection(p.ID, "combined", "테스트 작업자", p.SettingForm, p.CombinedForm); e != nil {
		t.Fatal(e)
	}
	if e = a.approve(p.ID, "combined", a.state.Projects[0].Setting.EventID); e != nil {
		t.Fatal(e)
	}
	assertOriginalBytes(t, original.dir, before)
	if a.state.Board == original.state.Board {
		t.Fatal("worker key shared")
	}
	oldCookie := *cookie
	oldCookie.Name = "aw_recovered_session"
	if r := call(a, "GET", "/", nil, &oldCookie); r.Code != 303 {
		t.Fatal("production cookie accepted")
	}
	req := httptest.NewRequest("POST", "http://127.0.0.1:28791/mobile-admin/passkey/finish", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)
	if w.Code != 403 || !strings.Contains(w.Body.String(), "테스트판") {
		t.Fatal("external finish unblocked")
	}
}
