package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTalkSubmitUsesUpdatedDraftGuard(t *testing.T) {
	a := fixture(t)
	a.state.Integration.Connection.Provider = "pages"
	a.state.Integration.Connection.PublicURL = "https://old-worker.trycloudflare.com"
	w := call(a, "GET", "/community/"+a.state.Board+"/new", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "worker-reconnect-v1.4.48.js") || !strings.Contains(w.Body.String(), `id="community-form"`) {
		t.Fatal("Talk editor guard missing", w.Code)
	}
	if dir := os.Getenv("TALK_SUBMIT_HTML"); dir != "" {
		os.MkdirAll(dir, 0700)
		os.WriteFile(filepath.Join(dir, "new.html"), w.Body.Bytes(), 0600)
		os.WriteFile(filepath.Join(dir, "fixture.json"), jsonRaw(map[string]string{"board": a.state.Board, "origin": a.state.Integration.Connection.PublicURL, "release": releaseVersion}), 0600)
	}
}
