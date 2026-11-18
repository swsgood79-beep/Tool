package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type workerSubmitGroup struct{ Setting, Shipping, Combined, Note int64 }

func TestWorkerAllSubmissionsBrowser(t *testing.T) {
	if os.Getenv("WORKER_SUBMIT_BROWSER") != "1" {
		t.Skip("optional real browser worker submission matrix")
	}
	a := fixture(t)
	a.state.Integration.Connection.Provider = "pages"
	a.state.Integration.Connection.PublicURL = "https://old-worker.trycloudflare.com"
	groups := []workerSubmitGroup{}
	native := workerSubmitGroup{}
	add := func(s *State, name string, shipping bool) int64 {
		p := Project{ID: s.next(), Number: "SUBMIT-" + fmtID(s.NextID), Name: name, Revision: 1, InfoRevision: 1, ShippingDueDate: day(time.Now()), ShippingDueManaged: true}
		p.rotate()
		if shipping {
			now := time.Now()
			p.Setting = Stage{Worker: "기존 세팅 작업자", Completed: &now}
		}
		s.Projects = append(s.Projects, p)
		return p.ID
	}
	if e := a.transaction(func(s *State) error {
		for i := 0; i < 4; i++ {
			groups = append(groups, workerSubmitGroup{add(s, "세팅 신청 "+fmtID(int64(i)), false), add(s, "출하 신청 "+fmtID(int64(i)), true), add(s, "함께 신청 "+fmtID(int64(i)), false), add(s, "비고 전달 "+fmtID(int64(i)), false)})
		}
		native = workerSubmitGroup{Setting: add(s, "기본 전송 세팅", false), Shipping: add(s, "기본 전송 출하", true), Combined: add(s, "기본 전송 함께", false)}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__test__/state" {
			a.mu.Lock()
			defer a.mu.Unlock()
			jsonReply(w, map[string]any{"projects": a.state.Projects, "events": a.state.Events, "community": a.state.Community})
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
		if strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
			body = bytes.ReplaceAll(body, []byte(`data-origin="https://old-worker.trycloudflare.com"`), []byte(`data-origin="http://`+r.Host+`"`))
		}
		w.WriteHeader(rec.Code)
		w.Write(body)
	}))
	defer server.Close()
	metadata, _ := json.Marshal(map[string]any{"groups": groups, "native": native})
	cmd := exec.Command("node", "audit/browser-worker-submissions-v1.4.54.cjs", server.URL, a.state.Board, string(metadata))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, g := range groups {
		if !a.state.project(g.Setting).Setting.Pending || !a.state.project(g.Shipping).Shipping.Pending || !a.state.project(g.Combined).Setting.Pending || !a.state.project(g.Combined).Shipping.Pending {
			t.Fatal("inspection request missing")
		}
		count := 0
		for _, e := range a.state.Events {
			if e.ProjectID == g.Note && e.IsNote {
				count++
			}
		}
		if count != 1 {
			t.Fatal("note missing or duplicate", g.Note, count)
		}
	}
	if len(a.state.Events) != 24 || len(a.state.Community) != 0 {
		t.Fatal("unexpected writes", len(a.state.Events), len(a.state.Community))
	}
	files, _ := os.ReadDir(filepath.Join(a.dir, "photos"))
	if len(files) != 0 {
		t.Fatal("deleted test post photos not cleaned")
	}
	t.Log("Actual temporary Go app: 24 expected inspection/note events; all test posts/comments deleted and photo files cleaned; no company data")
}
