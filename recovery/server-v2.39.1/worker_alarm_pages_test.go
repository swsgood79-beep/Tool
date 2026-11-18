package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func alarmPageFixture(t *testing.T) (*App, []string) {
	a := fixture(t)
	b := a.state.Board
	n, p := token(), token()
	a.state.Announcements = []Announcement{{ID: n, Title: "알람 시험", Text: "시험", Published: true, CreatedAt: time.Now()}}
	a.state.Community = []CommunityPost{{ID: p, Title: "알람 시험", Text: "시험", Category: "work", Author: "시험", Owner: token(), CreatedAt: time.Now()}}
	record := approveShipment(t, a)
	return a, []string{"/work/" + b + "?entry=skip", "/work/" + b + "?entry=skip&design-tab=calendar", "/shipping-records/" + b, "/shipping-records/" + b + "/" + record.ID, "/community/" + b, "/community/" + b + "/new", "/community/" + b + "/" + p, "/client-directory/" + b, "/project-info/" + b + "/1", "/notices/" + b, "/notices/" + b + "/" + n}
}
func TestWorkerAlarmOnAllFullPages(t *testing.T) {
	a, paths := alarmPageFixture(t)
	for _, path := range paths {
		w := call(a, "GET", path, nil)
		s := w.Body.String()
		if w.Code != 200 || strings.Count(s, `id="electron-speech"`) != 1 || !strings.Contains(s, `data-mode="shared" data-endpoint="/worker-speech/`+a.state.Board+`"`) {
			t.Fatal("missing alarm", path, w.Code)
		}
	}
	owner := adminCookie(a)
	for _, path := range []string{"/", "/shipping-archive-admin", "/community-admin", "/announcements", "/projects/1/info"} {
		w := call(a, "GET", path, nil, owner)
		if strings.Contains(w.Body.String(), `data-mode="shared"`) {
			t.Fatal("public receiver in admin", path)
		}
	}
	for _, path := range []string{"/notice-panel/" + a.state.Board, "/status/" + a.state.Board} {
		if strings.Contains(call(a, "GET", path, nil).Body.String(), `id="electron-speech"`) {
			t.Fatal("receiver duplicated in fragment", path)
		}
	}
}
func TestWorkerAlarmAllPagesBrowser(t *testing.T) {
	if os.Getenv("WORKER_ALARM_BROWSER") != "1" {
		t.Skip("optional actual page alarm browser flow")
	}
	a, paths := alarmPageFixture(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/__qa_request" {
			a.ServeHTTP(w, r)
			return
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		p := Project{ID: a.state.next(), Number: "ALARM", Name: "새 검사 신청", Revision: 1, InfoRevision: 1}
		p.rotate()
		a.state.Projects = append(a.state.Projects, p)
		if err := a.requestInspection(p.ID, "combined", "알람 시험", p.SettingForm, p.CombinedForm); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		jsonReply(w, map[string]any{"event": a.state.project(p.ID).Setting.EventID})
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()
	cmd := exec.Command("node", "audit/browser-worker-alarm-v1.4.58.cjs", srv.URL, a.state.Board, string(jsonRaw(paths)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}
