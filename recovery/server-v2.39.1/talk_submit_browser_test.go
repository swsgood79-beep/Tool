package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Run only when explicitly requested; uses a temporary app and real Chromium multipart POSTs.
func TestTalkSubmitBrowserIntegration(t *testing.T) {
	if os.Getenv("TALK_SUBMIT_BROWSER") != "1" {
		t.Skip("optional real-browser temporary server")
	}
	a := fixture(t)
	a.state.Integration.Connection.Provider = "pages"
	a.state.Integration.Connection.PublicURL = "https://old-worker.trycloudflare.com"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()
		a.ServeHTTP(rec, r)
		for k, vs := range rec.Header() {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		body := rec.Body.Bytes()
		if strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
			body = bytes.ReplaceAll(body, []byte(`data-origin="https://old-worker.trycloudflare.com"`), []byte(`data-origin="http://`+r.Host+`"`))
		}
		w.WriteHeader(rec.Code)
		w.Write(body)
	}))
	defer server.Close()
	origin := server.URL
	cmd := exec.Command("node", "audit/browser-talk-submit-v1.4.46.cjs", origin, a.state.Board)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.state.Community) != 4 {
		t.Fatal("unexpected posts after browser submissions", len(a.state.Community))
	}
	for _, p := range a.state.Community {
		if len(p.Photos) != 1 || !strings.HasPrefix(p.Title, "수정 완료") {
			t.Fatal("post/photo/edit not stored", p.Title, len(p.Photos))
		}
	}
	t.Log("Real browser native multipart uploads and edits stored in temporary Go app; four posts with one photo each; no company data used")
}
