package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkerReconnectMetadataScopeAndReadiness(t *testing.T) {
	a := fixture(t)
	a.state.Integration.Connection.Provider = "pages"
	a.state.Integration.Connection.PublicURL = "https://old-worker.trycloudflare.com"
	a.state.Integration.Connection.GitHubToken = "TEST-PRIVATE-NOT-A-REAL-TOKEN"
	w := call(a, "GET", "/work/"+a.state.Board+"?entry=skip", nil)
	body := w.Body.String()
	if w.Code != 200 || !strings.Contains(body, `id="worker-reconnect-config"`) || !strings.Contains(body, "worker-reconnect-v1.4.42.js") || strings.Contains(body, "TEST-PRIVATE-NOT-A-REAL-TOKEN") {
		t.Fatal("missing recovery metadata or credential leak")
	}
	for _, path := range []string{"/community/" + a.state.Board, "/notices/" + a.state.Board, "/client-directory/" + a.state.Board, "/project-info/" + a.state.Board + "/1"} {
		if w := call(a, "GET", path, nil); w.Code != 200 || !strings.Contains(w.Body.String(), `id="worker-reconnect-config"`) {
			t.Fatal("worker page recovery missing", path, w.Code)
		}
	}
	if w := call(a, "GET", "/community-admin", nil, adminCookie(a)); strings.Contains(w.Body.String(), `id="worker-reconnect-config"`) {
		t.Fatal("admin received worker recovery")
	}
	if !strings.Contains(w.Header().Get("Content-Security-Policy"), "worker-src 'self'") {
		t.Fatal("service worker blocked")
	}
	sw := call(a, "GET", "/worker-reconnect-sw-v1.4.42.js?board="+a.state.Board, nil)
	if sw.Code != 200 || sw.Header().Get("Service-Worker-Allowed") != "/" || sw.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("worker script scope/cache", sw.Code, sw.Header())
	}
	var ready struct{ Application, Release string }
	r := call(a, "GET", "/ready/"+a.state.Board, nil)
	if json.Unmarshal(r.Body.Bytes(), &ready) != nil || ready.Application != "wanryotalk" || ready.Release != releaseVersion {
		t.Fatal("readiness version")
	}
	a.state.Integration.Connection.Provider = "quick"
	if w := call(a, "GET", "/work/"+a.state.Board, nil); strings.Contains(w.Body.String(), `id="worker-reconnect-config"`) {
		t.Fatal("changed non-Pages behavior")
	}
	a.state.Integration.Connection.Provider = "pages"
	a.testMode = true
	if w := call(a, "GET", "/work/"+a.state.Board, nil); strings.Contains(w.Body.String(), `id="worker-reconnect-config"`) {
		t.Fatal("test mode installs external recovery")
	}
}
func TestWorkerReconnectBrowserFixture(t *testing.T) {
	a := fixture(t)
	a.state.Integration.Connection.Provider = "pages"
	a.state.Integration.Connection.PublicURL = "https://old-worker.trycloudflare.com"
	w := call(a, "GET", "/work/"+a.state.Board+"?entry=skip", nil)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	if out := os.Getenv("WORKER_RECONNECT_HTML_OUT"); out != "" {
		os.WriteFile(filepath.Join(out, "work.html"), w.Body.Bytes(), 0600)
		os.WriteFile(filepath.Join(out, "fixture.json"), jsonRaw(map[string]string{"board": a.state.Board, "release": releaseVersion, "csp": w.Header().Get("Content-Security-Policy"), "swCSP": workerServiceWorkerCSP()}), 0600)
	}
}
