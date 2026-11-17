package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestTalkReadBrowserAcrossOrigins(t *testing.T) {
	if os.Getenv("TALK_READ_BROWSER") != "1" {
		t.Skip("optional real browser cross-origin read state test")
	}
	a := readFixture(t)
	a.state.Integration.Connection.Provider = "pages"
	a.state.Integration.Connection.PublicURL = "https://old-worker.trycloudflare.com"
	ids := []string{}
	for _, p := range a.state.Community {
		ids = append(ids, p.ID)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__test__/new" {
			a.mu.Lock()
			defer a.mu.Unlock()
			id := token()
			a.commitAnnouncement(func(s *State) error {
				s.Community = append(s.Community, CommunityPost{ID: id, Title: "새 글", Text: "새 내용", Author: "다른 작업자", Owner: hashToken(token()), Category: "work", Revision: 1, CreatedAt: time.Now()})
				return nil
			})
			jsonReply(w, map[string]string{"id": id})
			return
		}
		rec := httptest.NewRecorder()
		a.ServeHTTP(rec, r)
		for k, vs := range rec.Header() {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		body := rec.Body.Bytes()
		if origin := r.Header.Get("X-Test-Origin"); origin == "https://new-worker.trycloudflare.com" {
			body = bytes.ReplaceAll(body, []byte(`data-origin="https://old-worker.trycloudflare.com"`), []byte(`data-origin="`+origin+`"`))
		}
		w.WriteHeader(rec.Code)
		w.Write(body)
	}))
	defer server.Close()
	raw, _ := json.Marshal(ids)
	cmd := exec.Command("node", "audit/browser-talk-read-v1.4.47.cjs", server.URL, a.state.Board, string(raw))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.state.TalkReaders) < 1 {
		t.Fatal("browser did not persist read states")
	}
}
