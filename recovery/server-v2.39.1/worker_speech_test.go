package main

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func workerSpeechForm(a *App, owner, device string, after int64, baseline bool) url.Values {
	f := speechForm(owner, device, after, baseline)
	f.Set("csrf", a.state.Board)
	return f
}
func workerSpeechPoll(t *testing.T, a *App, f url.Values, remote bool) speechResponse {
	t.Helper()
	path := "/worker-speech/" + a.state.Board
	w := httptest.NewRecorder()
	if remote {
		u := a.state.Integration.Connection.PublicURL + path
		r := httptest.NewRequest("POST", u, strings.NewReader(f.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		a.ServeHTTP(w, r)
	} else {
		w = call(a, "POST", path, f)
	}
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var v speechResponse
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func TestWorkerSpeechPublicPCReadsNewRequestsWithoutAdminOrPrivateNotes(t *testing.T) {
	a := fixture(t)
	owner, device := token(), token()
	base := workerSpeechPoll(t, a, workerSpeechForm(a, owner, device, 0, true), false)
	if !base.Active || len(base.Events) != 0 {
		t.Fatal(base)
	}
	p := a.state.Projects[0]
	if err := a.requestInspection(p.ID, "combined", "휴대폰 신청자", p.SettingForm, p.CombinedForm); err != nil {
		t.Fatal(err)
	}
	if err := a.transaction(func(s *State) error {
		for i := range s.Events {
			s.Events[i].Unread = false
		} // admin has already read it
		s.Events = append(s.Events, Event{ID: s.next(), ProjectID: 1, IsNote: true, Unread: true, Worker: "비공개 작성자", Note: "관리자 전용 비고"})
		s.Events = append(s.Events, Event{ID: s.next(), ProjectID: 1, Inspection: "shipping", Unread: true})
		s.Events = append(s.Events, Event{ID: s.next(), ProjectID: 1, Inspection: "setting", Pending: true, Unread: true, Rejected: true})
		hidden := Project{ID: s.next(), Number: "HIDDEN", Name: "숨김 수주", Hidden: true}
		s.Projects = append(s.Projects, hidden)
		s.Events = append(s.Events, Event{ID: s.next(), ProjectID: hidden.ID, Inspection: "setting", Pending: true, Unread: true})
		s.Events = append(s.Events, Event{ID: s.next(), ProjectID: 99999999, Inspection: "setting", Pending: true, Unread: true})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(a.state)
	f := workerSpeechForm(a, owner, device, base.Cursor, false)
	f.Set("select_browser", "1") // cannot modify administrator voice settings
	v := workerSpeechPoll(t, a, f, false)
	if len(v.Events) != 1 || !strings.HasPrefix(v.Events[0].Text, "휴대폰 신청자님이 ") || !strings.Contains(v.Events[0].Text, "세팅 및 출하 통합검사") {
		t.Fatal(v)
	}
	after, _ := json.Marshal(a.state)
	if string(before) != string(after) || a.state.Integration.DesktopSpeechMuted {
		t.Fatal("public speech changed business/admin state")
	}
	if len(workerSpeechPoll(t, a, workerSpeechForm(a, owner, device, v.Cursor, false), false).Events) != 0 {
		t.Fatal("old request repeated")
	}
	page := call(a, "GET", "/work/"+a.state.Board+"?entry=skip", nil)
	if page.Code != 200 || !strings.Contains(page.Body.String(), "공용 PC 신청 알림 켜기") || !strings.Contains(page.Body.String(), `data-endpoint="/worker-speech/`+a.state.Board+`"`) || strings.Contains(page.Body.String(), "admin-csrf") {
		t.Fatal("public worker voice controls", page.Code)
	}
	if out := os.Getenv("WORKER_SPEECH_HTML"); out != "" {
		if err := os.WriteFile(out, page.Body.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
func TestWorkerSpeechRequiresBoardAndMatchingBodyAndCannotAccessAdmin(t *testing.T) {
	a := fixture(t)
	f := workerSpeechForm(a, token(), token(), 0, true)
	path := "/worker-speech/" + a.state.Board
	if r := call(a, "POST", "/worker-speech/invalid", f); r.Code != 404 {
		t.Fatal(r.Code)
	}
	f.Set("csrf", "invalid")
	if r := call(a, "POST", path, f); r.Code != 403 {
		t.Fatal(r.Code)
	}
	if r := call(a, "GET", path, nil); r.Code != 405 {
		t.Fatal(r.Code)
	}
	f.Set("csrf", a.state.Board)
	if r := call(a, "POST", path+"/poll", f); r.Code != 404 {
		t.Fatal(r.Code)
	}
	if r := call(a, "POST", "/admin-speech/poll", f); r.Code != 401 {
		t.Fatal("board gained administrator rights", r.Code)
	}
	if len(a.integration.SpeechLeases) != 0 {
		t.Fatal("invalid request acquired lease")
	}
}
func TestWorkerAndAdminSpeechOnDifferentComputersAreIndependent(t *testing.T) {
	a := fixture(t)
	a.state.Integration.Connection.PublicURL = "https://shared-speech.example.test"
	c := adminCookie(a)
	adminOwner, device := token(), token()
	publicOwner := token()
	adminBase := speechPoll(t, a, speechForm(adminOwner, device, 0, true), c)
	publicBase := workerSpeechPoll(t, a, workerSpeechForm(a, publicOwner, device, 0, true), true)
	if !adminBase.Active || !publicBase.Active || publicBase.NativeSuppressed {
		t.Fatal("shared computer suppressed administrator PC")
	}
	p := a.state.Projects[0]
	if err := a.requestInspection(p.ID, "setting", "작업자", p.SettingForm, p.CombinedForm); err != nil {
		t.Fatal(err)
	}
	if len(speechPoll(t, a, speechForm(adminOwner, device, adminBase.Cursor, false), c).Events) != 1 || len(workerSpeechPoll(t, a, workerSpeechForm(a, publicOwner, device, publicBase.Cursor, false), true).Events) != 1 {
		t.Fatal("both computers did not receive the application")
	}
	rival := workerSpeechPoll(t, a, workerSpeechForm(a, token(), device, publicBase.Cursor, true), true)
	if rival.Active {
		t.Fatal("two public tabs can speak on the same browser")
	}
	// Board credentials must not release a local administrator lease, even with copied owner/device.
	release := workerSpeechForm(a, adminOwner, device, 0, false)
	release.Set("enabled", "0")
	workerSpeechPoll(t, a, release, false)
	if !speechPoll(t, a, speechForm(adminOwner, device, 0, true), c).Active {
		t.Fatal("public request released admin owner")
	}
}
