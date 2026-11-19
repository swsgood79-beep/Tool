package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func nativeViewGet(a *App, c *http.Cookie, path string, view *http.Cookie, app bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", "http://127.0.0.1:18791"+path, nil)
	if app {
		r.Header.Set("User-Agent", "AlphaAdminAndroid/1.0.7")
	}
	if c != nil {
		r.AddCookie(c)
	}
	if view != nil {
		r.AddCookie(view)
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func TestNativeWorkerViewSwitchKeepsSessionAndPermissions(t *testing.T) {
	a := accountFixture(t)
	a.state.Integration.MobileEnabled = true
	c := adminCookie(a)
	original := a.sessions[hashToken(c.Value)]
	if w := nativeViewGet(a, nil, "/native-admin/view?mode=worker", nil, true); w.Code != 303 || !strings.HasPrefix(w.Header().Get("Location"), "/login") {
		t.Fatal("unauthenticated view switch", w.Code)
	}
	w := nativeViewGet(a, c, "/native-admin/view?mode=worker", nil, true)
	if w.Code != 303 || w.Header().Get("Location") != "/work/"+a.state.Board {
		t.Fatal("worker redirect", w.Code, w.Header())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].Name != a.sessionCookie()+"-view" {
		t.Fatal("view cookie missing")
	}
	view := cookies[0]
	for _, path := range []string{"/work/" + a.state.Board, "/community/" + a.state.Board, "/shipping-records/" + a.state.Board, "/client-directory/" + a.state.Board} {
		w = nativeViewGet(a, c, path, view, true)
		s := w.Body.String()
		if w.Code != 200 || !strings.Contains(s, `data-native-worker-view="true"`) || strings.Contains(s, `id="alpha-admin-app"`) {
			t.Fatal("not actual worker layout", path, w.Code)
		}
		if strings.Contains(s, `src="/worker-app-v1.4.63.js"`) || !strings.Contains(s, "native-view-menu-v1.4.72.js") {
			t.Fatal("worker native controls", path)
		}
	}
	w = nativeViewGet(a, c, "/native-admin/bootstrap", view, true)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"worker_view":true`) || !strings.Contains(w.Body.String(), `"owner":true`) {
		t.Fatal("principal lost", w.Code, w.Body.String())
	}
	if a.sessions[hashToken(c.Value)] != original {
		t.Fatal("view switch changed session")
	}
	// A plain browser using the same presentation cookie keeps its usual layout.
	if w := nativeViewGet(a, c, "/work/"+a.state.Board, view, false); strings.Contains(w.Body.String(), `data-native-worker-view="true"`) {
		t.Fatal("PC layout affected")
	}
	w = nativeViewGet(a, c, "/native-admin/view?mode=admin", view, true)
	if w.Code != 303 || w.Header().Get("Location") != "/mobile-admin" {
		t.Fatal("admin return", w.Code)
	}
	view = w.Result().Cookies()[0]
	w = nativeViewGet(a, c, "/mobile-admin", view, true)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `id="alpha-admin-app"`) {
		t.Fatal("admin layout missing")
	}
	createAccount(t, a, c, "qa_view")
	member, _ := accountLogin(t, a, "qa_view", accountMemberPassword)
	view.Value = "worker"
	if w := nativeViewGet(a, member, "/android-admin-settings", view, true); w.Code != 403 {
		t.Fatal("view cookie elevated member", w.Code)
	}
	if w := nativeViewGet(a, nil, "/work/"+a.state.Board, view, true); w.Code != 303 {
		t.Fatal("worker view bypassed app authentication", w.Code)
	}
	if w := nativeViewGet(a, c, "/native-admin/view?mode=invalid", view, true); w.Code != 400 {
		t.Fatal("invalid view allowed", w.Code)
	}
}

func TestNativeWorkerViewBrowser(t *testing.T) {
	if os.Getenv("NATIVE_WORKER_VIEW_BROWSER") != "1" {
		t.Skip("optional worker/admin browser navigation")
	}
	a := accountFixture(t)
	a.state.Integration.MobileEnabled = true
	c := adminCookie(a)
	srv := httptest.NewServer(a)
	defer srv.Close()
	info := map[string]any{"cookie": map[string]string{"name": c.Name, "value": c.Value}, "board": a.state.Board}
	cmd := exec.Command("node", "audit/browser-native-worker-view-v1.4.72.cjs", srv.URL, string(jsonRaw(info)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}
