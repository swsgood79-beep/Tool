package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func nativeLoginCall(a *App, c *http.Cookie, path string, f url.Values) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791"+path, strings.NewReader(f.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("X-Forwarded-Proto", "https")
	if c != nil {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func enrollNativeLogin(t *testing.T, a *App, c *http.Cookie, device string) url.Values {
	t.Helper()
	w := nativeLoginCall(a, c, "/native-admin/session/enroll", url.Values{"csrf": {a.sessions[hashToken(c.Value)].CSRF}, "device": {device}, "name": {"검사팀 휴대폰"}})
	if w.Code != 200 {
		t.Fatal("enroll", w.Code, w.Body.String())
	}
	var j map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &j); err != nil {
		t.Fatal(err)
	}
	if j["grant"] == "" || j["secret"] == "" {
		t.Fatal("missing capability")
	}
	if c := w.Result().Cookies(); len(c) != 1 || !c[0].Secure || !c[0].HttpOnly || c[0].MaxAge != 365*24*3600 {
		t.Fatal("native persistent cookie")
	}
	return url.Values{"grant": {j["grant"]}, "secret": {j["secret"]}, "device": {device}}
}
func restoredCookie(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	if w.Code != 200 {
		t.Fatal("restore", w.Code, w.Body.String())
	}
	cs := w.Result().Cookies()
	if len(cs) != 1 {
		t.Fatal("no session cookie")
	}
	return cs[0]
}
func TestNativePersistentLoginRestartLogoutAndBrowserIsolation(t *testing.T) {
	a := accountFixture(t)
	a.state.Integration.MobileEnabled = true
	owner := adminCookie(a)
	createAccount(t, a, owner, "qa02")
	browser, _ := accountLogin(t, a, "qa02", accountMemberPassword)
	original := a.sessions[hashToken(browser.Value)]
	if original.NativeGrantID != "" || time.Until(original.Expires) > 9*time.Hour {
		t.Fatal("browser lifetime changed")
	}
	appCookie, _ := accountLogin(t, a, "qa02", accountMemberPassword)
	f := enrollNativeLogin(t, a, appCookie, token())
	s := a.sessions[hashToken(appCookie.Value)]
	if s.NativeGrantID == "" || time.Until(s.Expires) < 364*24*time.Hour {
		t.Fatal("native session not extended")
	}
	if a.sessions[hashToken(browser.Value)] != original {
		t.Fatal("browser affected")
	}
	g := nativeLogin(&a.state, f.Get("grant"))
	if g.SecretHash == f.Get("secret") || g.SecretHash == "" {
		t.Fatal("raw secret stored")
	}
	fresh, err := newApp(a.dir)
	if err != nil {
		t.Fatal(err)
	}
	if !fresh.state.Integration.MobileEnabled {
		t.Fatal("remote setting not persisted")
	}
	f.Set("csrf_hint", s.CSRF)
	f.Set("account_id", "") // Client cannot elevate a named account to owner.
	c := restoredCookie(t, nativeLoginCall(fresh, nil, "/native-admin/session/restore", f))
	rs := fresh.sessions[hashToken(c.Value)]
	if rs.AccountID != s.AccountID || rs.NativeGrantID != s.NativeGrantID || rs.CSRF != s.CSRF {
		t.Fatal("principal or draft csrf lost")
	}
	if w := call(fresh, "GET", "/android-admin-settings", nil, c); w.Code != 403 {
		t.Fatal("member escalated")
	}
	if w := call(fresh, "GET", "/", nil, c); w.Code != 200 {
		t.Fatal("admin UI inaccessible")
	}
	cfg, _, _, _ := nativeTestConfig(t)
	fresh.state.NativeAdminPush = &cfg
	pushSecret := nativeEnroll(t, fresh, c, f.Get("device"), strings.Repeat("X", 80))
	if pushSecret == f.Get("secret") {
		t.Fatal("push and login capabilities shared")
	}
	if w := nativeCall(fresh, c, "/logout", nil); w.Code != 303 {
		t.Fatal("logout", w.Code)
	}
	if w := nativeLoginCall(fresh, nil, "/native-admin/session/restore", f); w.Code != 403 || !strings.Contains(w.Body.String(), "LOGIN_REVOKED") {
		t.Fatal("logout did not revoke")
	}
	if d := nativeDevice(&fresh.state, f.Get("device")); d.Enabled || d.Token != "" {
		t.Fatal("logout left push enabled")
	}
	if _, ok := fresh.sessions[hashToken(c.Value)]; ok {
		t.Fatal("logout session retained")
	}
}
func TestNativePersistentLoginRevocationAndCapabilityBoundaries(t *testing.T) {
	a := accountFixture(t)
	a.state.Integration.MobileEnabled = true
	owner := adminCookie(a)
	f := enrollNativeLogin(t, a, owner, token())
	bad := f.Clone()
	bad.Set("secret", token())
	if w := nativeLoginCall(a, nil, "/native-admin/session/restore", bad); w.Code != 403 {
		t.Fatal("invalid secret accepted")
	}
	bad = f.Clone()
	bad.Set("device", token())
	if w := nativeLoginCall(a, nil, "/native-admin/session/restore", bad); w.Code != 403 {
		t.Fatal("other device accepted")
	}
	if w := call(a, "POST", "/native-admin/session/restore", f); w.Code != 403 {
		t.Fatal("cleartext restore allowed")
	}
	before := a.state.Integration.Revision
	a.state.Integration.MobileEnabled = false
	if w := nativeLoginCall(a, nil, "/native-admin/session/restore", f); w.Code != 403 || !strings.Contains(w.Body.String(), "REMOTE_DISABLED") {
		t.Fatal("disabled remote restore")
	}
	a.state.Integration.MobileEnabled = true
	a.state.Integration.Revision++
	if w := nativeLoginCall(a, nil, "/native-admin/session/restore", f); w.Code != 403 {
		t.Fatal("owner change not revoked")
	}
	a.state.Integration.Revision = before
	c := restoredCookie(t, nativeLoginCall(a, nil, "/native-admin/session/restore", f))
	if w := nativeCall(a, c, "/android-admin-settings", url.Values{"action": {"revoke-login"}, "login": {f.Get("grant")}}); w.Code != 303 {
		t.Fatal("owner revoke", w.Code)
	}
	if w := nativeLoginCall(a, nil, "/native-admin/session/restore", f); w.Code != 403 {
		t.Fatal("owner revocation failed")
	}
	// A fresh, normal login is required to enroll again.
	owner = adminCookie(a)
	createAccount(t, a, owner, "qa03")
	member, _ := accountLogin(t, a, "qa03", accountMemberPassword)
	f = enrollNativeLogin(t, a, member, token())
	u := adminAccountByUsername(&a.state, "qa03")
	u.Revision++
	if w := nativeLoginCall(a, nil, "/native-admin/session/restore", f); w.Code != 403 {
		t.Fatal("member password revision retained login")
	}
	u.Revision--
	u.Enabled = false
	if w := nativeLoginCall(a, nil, "/native-admin/session/restore", f); w.Code != 403 {
		t.Fatal("disabled member retained login")
	}
	u.Enabled = true
	if w := nativeLoginCall(a, nil, "/native-admin/session/revoke", f); w.Code != 200 {
		t.Fatal("scoped revoke", w.Code)
	}
	if w := nativeLoginCall(a, nil, "/native-admin/session/restore", f); w.Code != 403 {
		t.Fatal("scoped revocation failed")
	}
}
