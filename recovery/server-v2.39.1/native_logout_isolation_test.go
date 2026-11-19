package main

import (
	"net/url"
	"strings"
	"testing"
)

// Logging out on one physical phone must not revoke another phone or the PC,
// even when all three are signed into the same administrator account.
func TestNativeLogoutOnlyCurrentPhone(t *testing.T) {
	a := accountFixture(t)
	a.state.Integration.MobileEnabled = true
	cfg, _, _, _ := nativeTestConfig(t)
	a.state.NativeAdminPush = &cfg
	phoneA, _ := accountLogin(t, a, "wssong", accountOwnerPassword)
	phoneB, _ := accountLogin(t, a, "wssong", accountOwnerPassword)
	desktop, _ := accountLogin(t, a, "wssong", accountOwnerPassword)
	deviceA, deviceB := token(), token()
	grantA := enrollNativeLogin(t, a, phoneA, deviceA)
	grantB := enrollNativeLogin(t, a, phoneB, deviceB)
	nativeEnroll(t, a, phoneA, deviceA, strings.Repeat("A", 80))
	nativeEnroll(t, a, phoneB, deviceB, strings.Repeat("B", 80))
	csrfA := a.sessions[hashToken(phoneA.Value)].CSRF
	if w := nativeLoginCall(a, phoneA, "/logout", url.Values{"csrf": {csrfA}}); w.Code != 303 {
		t.Fatal("logout", w.Code)
	}
	if w := nativeLoginCall(a, nil, "/native-admin/session/restore", grantA); w.Code != 403 {
		t.Fatal("logged-out phone restored", w.Code)
	}
	if d := nativeDevice(&a.state, deviceA); d.Enabled || d.Token != "" || d.SecretHash != "" {
		t.Fatal("logged-out phone still receives push")
	}
	restoredCookie(t, nativeLoginCall(a, nil, "/native-admin/session/restore", grantB))
	if d := nativeDevice(&a.state, deviceB); !d.Enabled || d.Token == "" || d.SecretHash == "" {
		t.Fatal("other phone revoked")
	}
	if w := call(a, "GET", "/", nil, desktop); w.Code != 200 {
		t.Fatal("desktop session revoked", w.Code)
	}
}
