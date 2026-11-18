package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkerAppSecurityAndIdentity(t *testing.T) {
	a := fixture(t)
	b := a.state.Board
	for _, path := range []string{"/work/" + b + "?entry=skip", "/shipping-records/" + b, "/community/" + b, "/notices/" + b} {
		w := call(a, "GET", path, nil)
		if !strings.Contains(w.Header().Get("Content-Security-Policy"), "frame-ancestors "+workerAppOrigin) || !strings.Contains(w.Body.String(), "worker-app-v1.4.61.js") {
			t.Fatal("worker app missing", path)
		}
	}
	for _, path := range []string{"/", "/login", "/work/wrong"} {
		w := call(a, "GET", path, nil)
		if !strings.Contains(w.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
			t.Fatal("admin or invalid frame permitted", path)
		}
	}
	if w := call(a, "GET", "/worker-app-entry/"+b, nil); w.Code != 200 || !strings.Contains(w.Body.String(), "worker-app-entry-v1.4.61.js") {
		t.Fatal("bootstrap missing")
	}
	owner := token()
	request := httptest.NewRequest("POST", "http://127.0.0.1:18791/worker-app-session/"+b, strings.NewReader(url.Values{"csrf": {b}, "owner": {owner}, "reader": {token()}}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, request)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 2 || !cookies[0].Secure || !cookies[0].HttpOnly || !cookies[0].Partitioned || cookies[0].SameSite != http.SameSiteNoneMode {
		t.Fatal("partitioned identity missing")
	}
	r := httptest.NewRequest("GET", "https://worker.trycloudflare.com/community/"+b, nil)
	r.AddCookie(cookies[0])
	if got := communityIdentity(httptest.NewRecorder(), r, false); got != hashToken(owner) {
		t.Fatal("app identity lost")
	}
	if bad := call(a, "POST", "/worker-app-session/"+b, url.Values{"csrf": {b}, "owner": {owner}, "reader": {token()}}); bad.Code != 403 {
		t.Fatal("insecure cookie accepted")
	}
	if bad := call(a, "POST", "/worker-app-session/"+b, url.Values{"csrf": {"wrong"}, "owner": {owner}, "reader": {token()}}); bad.Code != 403 {
		t.Fatal("csrf bypass")
	}
	if w := call(a, "GET", "/work/"+b+"?entry=skip", nil); strings.Contains(w.Body.String(), owner) {
		t.Fatal("cookie capability leaked")
	}
}
func TestWorkerAppBrowser(t *testing.T) {
	if os.Getenv("WORKER_APP_BROWSER") != "1" {
		t.Skip("optional actual installed app web flow")
	}
	a := fixture(t)
	a.state.Integration.Connection.Provider = "pages"
	a.state.Integration.Connection.PublicURL = "https://app-old.trycloudflare.com"
	ids := []string{}
	for i := 0; i < 2; i++ {
		if i > 0 {
			p := Project{ID: a.state.next(), Number: "APP", Name: "앱 사진 시험", Revision: 1, InfoRevision: 1}
			p.rotate()
			a.state.Projects = []Project{p}
		}
		r := approveShipment(t, a)
		ids = append(ids, r.ID)
	}
	srv := httptest.NewServer(a)
	defer srv.Close()
	cmd := exec.Command("node", "audit/browser-worker-app-v1.4.61.cjs", srv.URL, a.state.Board, string(jsonRaw(ids)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
	for _, r := range a.state.ShippingRecords {
		if r.ArchivedAt == nil || len(r.Photos) != 1 {
			t.Fatal("app photo not archived")
		}
	}
	if len(a.state.Community) != 2 {
		t.Fatal("app Talk save lost", len(a.state.Community))
	}
}
func TestWorkerAppServiceWorkerBrowser(t *testing.T) {
	if os.Getenv("WORKER_APP_BROWSER") != "1" {
		t.Skip("optional real service worker cache flow")
	}
	dir, err := filepath.Abs("extras/pwa-v1.4.61")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/Tool/app/", http.StripPrefix("/Tool/app/", http.FileServer(http.Dir(dir))))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cmd := exec.Command("node", "audit/browser-app-sw-v1.4.61.cjs", srv.URL)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
