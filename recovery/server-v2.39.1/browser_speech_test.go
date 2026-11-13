package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

type speechResponse struct {
	OK, Active       bool
	Cursor           int64
	Events           []speechAlert
	NativeSuppressed bool `json:"native_suppressed"`
}

func speechForm(owner, device string, after int64, baseline bool) url.Values {
	f := url.Values{"csrf": {"admin-csrf"}, "owner": {owner}, "device": {device}, "enabled": {"1"}, "after": {fmtID(after)}}
	if baseline {
		f.Set("baseline", "1")
	}
	return f
}
func speechPoll(t *testing.T, a *App, f url.Values, c *http.Cookie) speechResponse {
	t.Helper()
	r := call(a, "POST", "/admin-speech/poll", f, c)
	if r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	var v speechResponse
	if err := json.Unmarshal(r.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func TestBrowserSpeechAlertsUseExistingRequestsAndNotesWithoutChangingData(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	owner, device := token(), token()
	base := speechPoll(t, a, speechForm(owner, device, 0, true), cookie)
	if !base.OK || !base.Active || !base.NativeSuppressed || len(base.Events) != 0 {
		t.Fatal(base)
	}
	p := a.state.Projects[0]
	if err := a.requestInspection(p.ID, "combined", "홍길동", p.SettingForm, p.CombinedForm); err != nil {
		t.Fatal(err)
	}
	if err := a.transaction(func(s *State) error {
		s.Projects[0].Name = "최신 프로젝트"
		s.Events = append(s.Events, Event{ID: s.next(), ProjectID: p.ID, IsNote: true, Unread: true, Worker: "김작업", Note: "센서를 확인해 주세요."})
		s.Events = append(s.Events, Event{ID: s.next(), ProjectID: p.ID, Inspection: "shipping", Unread: true}) // completed
		s.Events = append(s.Events, Event{ID: s.next(), ProjectID: p.ID, IsNote: true, Unread: true, Rejected: true})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(a.state)
	v := speechPoll(t, a, speechForm(owner, device, base.Cursor, false), cookie)
	if len(v.Events) != 2 || !strings.HasPrefix(v.Events[0].Text, "홍길동님이 최신 프로젝트의 세팅 및 출하 통합검사") || !strings.Contains(v.Events[1].Text, "김작업님이 최신 프로젝트 비고 메시지") {
		t.Fatal(v)
	}
	if v.Events[0].Text != v.Events[0].Worker+"님이 "+v.Events[0].Message {
		t.Fatal("message formatting mismatch")
	}
	after, _ := json.Marshal(a.state)
	if string(before) != string(after) {
		t.Fatal("speech changed order, unread, approval or persisted data")
	}
	if len(speechPoll(t, a, speechForm(owner, device, v.Cursor, false), cookie).Events) != 0 {
		t.Fatal("cursor replayed events")
	}
	if len(speechPoll(t, a, speechForm(owner, device, a.state.NextID+100, false), cookie).Events) != 0 {
		t.Fatal("restored data cursor replay")
	}
	for _, path := range []string{"/", "/inspection-requests", "/projects/1/inspection", "/unread"} {
		r := call(a, "GET", path, nil, cookie)
		if r.Code != 200 || !strings.Contains(r.Body.String(), `id="electron-speech-toggle"`) || !strings.Contains(r.Body.String(), `data-csrf="admin-csrf"`) {
			t.Fatal("missing administrator voice toolbar", path, r.Code)
		}
		if path == "/" {
			if target := os.Getenv("BROWSER_SPEECH_HTML"); target != "" {
				if err := os.WriteFile(target, r.Body.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	worker := call(a, "GET", "/work/"+a.state.Board, nil)
	if strings.Contains(worker.Body.String(), `data-csrf="admin-csrf"`) || !strings.Contains(worker.Body.String(), `data-mode="shared"`) {
		t.Fatal("worker voice toolbar is not isolated from administrator controls")
	}
}
func TestBrowserSpeechLeasePreventsSamePCDuplicatesAndExpires(t *testing.T) {
	a := fixture(t)
	c := adminCookie(a)
	one, two, device := token(), token(), token()
	first := speechPoll(t, a, speechForm(one, device, 0, true), c)
	if !first.Active {
		t.Fatal(first)
	}
	if speechPoll(t, a, speechForm(two, device, 0, true), c).Active {
		t.Fatal("two local tabs can speak")
	}
	wrongRelease := speechForm(two, device, 0, false)
	wrongRelease.Set("enabled", "0")
	speechPoll(t, a, wrongRelease, c)
	if !speechPoll(t, a, speechForm(one, device, 0, true), c).Active {
		t.Fatal("other tab released owner")
	}
	release := speechForm(one, device, 0, false)
	release.Set("enabled", "0")
	speechPoll(t, a, release, c)
	if !speechPoll(t, a, speechForm(two, device, 0, true), c).Active {
		t.Fatal("owner release failed")
	}
	lease := a.integration.SpeechLeases["local"]
	lease.Expires = time.Now().Add(-time.Second)
	a.integration.SpeechLeases["local"] = lease
	if !speechPoll(t, a, speechForm(one, device, 0, true), c).Active {
		t.Fatal("expired owner blocked new tab")
	}
}
func TestRemoteBrowserSpeechDoesNotSuppressPC(t *testing.T) {
	a := fixture(t)
	c := adminCookie(a)
	a.state.Integration.MobileEnabled = true
	a.state.Integration.Connection.PublicURL = "https://speech.example.test"
	f := speechForm(token(), token(), 0, true)
	f.Set("select_browser", "1")
	r := httptest.NewRequest("POST", "https://speech.example.test/admin-speech/poll", strings.NewReader(f.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(c)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	var v speechResponse
	_ = json.Unmarshal(w.Body.Bytes(), &v)
	if w.Code != 200 || !v.Active || v.NativeSuppressed || a.integration.SpeechLeases["local"].Owner != "" || a.state.Integration.DesktopSpeechMuted {
		t.Fatal("remote browser disabled native PC", w.Code, w.Body.String())
	}
}

func TestBrowserSpeechExplicitSelectionKeepsOldVoiceOffAfterBrowserOff(t *testing.T) {
	a := fixture(t)
	c := adminCookie(a)
	f := speechForm(token(), token(), 0, true)
	f.Set("select_browser", "1")
	before := a.state.Projects[0]
	speechPoll(t, a, f, c)
	if !a.state.Integration.DesktopSpeechMuted {
		t.Fatal("browser ON did not replace the old PC voice")
	}
	f.Set("enabled", "0")
	speechPoll(t, a, f, c)
	if !a.state.Integration.DesktopSpeechMuted || len(a.integration.SpeechLeases) != 0 {
		t.Fatal("browser OFF re-enabled default PC voice")
	}
	b, err := newApp(a.dir)
	if err != nil || !b.state.Integration.DesktopSpeechMuted || b.state.Projects[0].Name != before.Name {
		t.Fatal("voice preference persistence", err)
	}
}
func TestBrowserSpeechRequiresAuthenticationCSRFAndValidOwner(t *testing.T) {
	a := fixture(t)
	f := speechForm(token(), token(), 0, true)
	if r := call(a, "POST", "/admin-speech/poll", f); r.Code != 401 {
		t.Fatal(r.Code)
	}
	c := adminCookie(a)
	f.Set("csrf", "wrong")
	if r := call(a, "POST", "/admin-speech/poll", f, c); r.Code != 403 {
		t.Fatal(r.Code)
	}
	f.Set("csrf", "admin-csrf")
	f.Set("owner", "invalid")
	if r := call(a, "POST", "/admin-speech/poll", f, c); r.Code != 400 {
		t.Fatal(r.Code)
	}
	if len(a.integration.SpeechLeases) != 0 {
		t.Fatal("unauthorized speech lease")
	}
}
func TestBrowserSpeechPaginationDoesNotLoseLargeBurst(t *testing.T) {
	a := fixture(t)
	c := adminCookie(a)
	owner, device := token(), token()
	base := speechPoll(t, a, speechForm(owner, device, 0, true), c)
	if err := a.transaction(func(s *State) error {
		for i := 0; i < 300; i++ {
			s.Events = append(s.Events, Event{ID: s.next(), ProjectID: 1, IsNote: true, Unread: true, Worker: "시험", Note: "메시지"})
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	first := speechPoll(t, a, speechForm(owner, device, base.Cursor, false), c)
	second := speechPoll(t, a, speechForm(owner, device, first.Cursor, false), c)
	if len(first.Events) != 256 || len(second.Events) != 44 || first.Events[255].ID >= second.Events[0].ID || second.Cursor != a.state.NextID {
		t.Fatal("speech burst lost or repeated", len(first.Events), len(second.Events))
	}
}
