package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func nativeAppGet(a *App, c *http.Cookie, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", "http://127.0.0.1:18791"+path, nil)
	r.Header.Set("User-Agent", "Android WebView AlphaAdminAndroid/1.0.1")
	if c != nil {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func TestNativeAdminAppPresentationIsolationAndAuthentication(t *testing.T) {
	a := accountFixture(t)
	a.state.Integration.MobileEnabled = true
	owner := adminCookie(a)
	board := a.state.Board
	original := string(jsonRaw(a.state.Projects))
	for _, path := range []string{"/mobile-admin", "/community/" + board, "/shipping-records/" + board, "/client-directory/" + board, "/notices/" + board} {
		w := nativeAppGet(a, nil, path)
		if w.Code != 303 || !strings.HasPrefix(w.Header().Get("Location"), "/login") {
			t.Fatal("unauthenticated app content", path, w.Code)
		}
	}
	for _, path := range []string{"/mobile-admin", "/inspection-requests", "/projects/" + fmtID(a.state.Projects[0].ID) + "/inspection", "/community/" + board, "/shipping-archive-admin", "/client-directory/" + board, "/notices/" + board} {
		w := nativeAppGet(a, owner, path)
		if w.Code != 200 {
			t.Fatal(path, w.Code, w.Body.String())
		}
		s := w.Body.String()
		for _, want := range []string{`id="alpha-admin-app"`, `data-admin-app="true"`, "mobile-design-app-v1.4.105.js", "native-admin-app-v1.4.65.css", `data-csrf="` + a.sessions[hashToken(owner.Value)].CSRF + `"`} {
			if !strings.Contains(s, want) {
				t.Fatal("missing native app shell", path, want)
			}
		}
		if strings.Contains(s, `src="/worker-app-v1.4.63.js"`) || strings.Contains(s, `src="/mobile-design-worker-v1.4.105.js"`) {
			t.Fatal("app used browser shell", path)
		}
	}
	w := nativeAppGet(a, owner, "/mobile-admin?design-tab=calendar")
	if !strings.Contains(w.Body.String(), `<body data-design-tab="calendar"`) {
		t.Fatal("calendar first paint destination missing")
	}
	for _, path := range []string{"/", "/mobile-admin", "/inspection-requests", "/community/" + board, "/shipping-archive-admin"} {
		w := call(a, "GET", path, nil, owner)
		if w.Code != 200 || strings.Contains(w.Body.String(), `id="alpha-admin-app"`) || !strings.Contains(w.Body.String(), "mobile-design-worker-v1.4.105.js") {
			t.Fatal("PC or normal worker changed", path, w.Code)
		}
	}
	w = nativeAppGet(a, owner, "/work/"+board+"?entry=skip&design-tab=calendar")
	if w.Code != 303 || w.Header().Get("Location") != "/mobile-admin?design-tab=calendar" {
		t.Fatal("worker back link left authenticated shell")
	}
	if string(jsonRaw(a.state.Projects)) != original {
		t.Fatal("presentation modified orders")
	}
	// The APK marker does not grant settings permissions to another inspector.
	createAccount(t, a, owner, "qa_app")
	member, _ := accountLogin(t, a, "qa_app", accountMemberPassword)
	if w := nativeAppGet(a, member, "/android-admin-settings"); w.Code != 403 {
		t.Fatal("native marker escalated member")
	}
	if w := nativeAppGet(a, member, "/mobile-admin"); !strings.Contains(w.Body.String(), `data-owner="false"`) {
		t.Fatal("member account presentation")
	}
	// Existing APK 1.0.0 sessions also get the shell after the server update.
	f := enrollNativeLogin(t, a, owner, token())
	c := restoredCookie(t, nativeLoginCall(a, nil, "/native-admin/session/restore", f))
	w = call(a, "GET", "/mobile-admin", nil, c)
	if !strings.Contains(w.Body.String(), `id="alpha-admin-app"`) || !strings.Contains(w.Body.String(), `data-native-controls="false"`) {
		t.Fatal("existing APK session missed shell")
	}
}
func TestNativeAdminAppBrowser(t *testing.T) {
	if os.Getenv("NATIVE_ADMIN_APP_BROWSER") != "1" {
		t.Skip("optional actual browser APK-shell flow")
	}
	a := accountFixture(t)
	a.state.Integration.MobileEnabled = true
	owner := adminCookie(a)
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
	info := map[string]any{"cookie": map[string]string{"name": owner.Name, "value": owner.Value}, "board": a.state.Board, "project": p.ID}
	cmd := exec.Command("node", "audit/browser-native-admin-app-v1.4.65.cjs", srv.URL, string(jsonRaw(info)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if a.state.project(p.ID).Setting.Completed == nil {
		t.Fatal("app inspector approval did not persist")
	}
	if w := call(a, "POST", "/archive", url.Values{"csrf": {"wrong"}, "id": {fmtID(p.ID)}}, owner); w.Code != 403 {
		t.Fatal("CSRF weakened")
	}
}
