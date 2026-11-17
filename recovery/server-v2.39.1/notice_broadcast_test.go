package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func broadcastFixture(t *testing.T) *App {
	a := fixture(t)
	a.state.Integration.Connection.PublicURL = "https://broadcast-test.trycloudflare.com"
	a.state.Announcements = []Announcement{{ID: token(), Title: "오후 안전교육", Text: "3시에 회의실로 모여주세요.", Urgent: true, Published: true, Revision: 1, UpdatedAt: time.Now()}}
	return a
}
func broadcastWorkerCall(a *App, path string, f url.Values, ua string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", a.state.Integration.Connection.PublicURL+path, strings.NewReader(f.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("User-Agent", ua)
	a.ServeHTTP(w, r)
	return w
}
func broadcastRegister(t *testing.T, a *App, owner, device, ua string, baseline bool) url.Values {
	t.Helper()
	f := workerSpeechForm(a, owner, device, 0, baseline)
	f.Set("broadcast_enabled", "1")
	w := broadcastWorkerCall(a, "/worker-speech/"+a.state.Board, f, ua)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	return f
}
func broadcastSend(a *App, request string) *httptest.ResponseRecorder {
	n := a.state.Announcements[0]
	return call(a, "POST", "/announcements/broadcast", url.Values{"csrf": {"admin-csrf"}, "id": {n.ID}, "revision": {fmtID(n.Revision)}, "request": {request}}, adminCookie(a))
}
func TestNoticeBroadcastAuthorizationTargetsAudioAndNoAutomaticPosting(t *testing.T) {
	a := broadcastFixture(t)
	n := a.state.Announcements[0]
	f := url.Values{"id": {n.ID}, "revision": {"1"}, "request": {token()}, "csrf": {"admin-csrf"}}
	if w := call(a, "POST", "/announcements/broadcast", f); w.Code != 401 {
		t.Fatal("unauthenticated broadcast", w.Code)
	}
	f.Set("csrf", "wrong")
	if w := call(a, "POST", "/announcements/broadcast", f, adminCookie(a)); w.Code != 403 {
		t.Fatal("csrf bypass")
	}
	if w := broadcastSend(a, token()); w.Code != 409 {
		t.Fatal("no receivers accepted", w.Code)
	}
	owner, device := token(), token()
	form := broadcastRegister(t, a, owner, device, "Mozilla/5.0 (Windows NT 10.0; Win64; x64)", true)
	broadcastRegister(t, a, token(), token(), "Mozilla/5.0 (Linux; Android 14) Mobile", true)
	if len(a.integration.NoticeReceivers) != 1 || len(a.integration.NoticeBroadcasts) != 0 {
		t.Fatal("phone registered or automatic send")
	}
	request := token()
	w := broadcastSend(a, request)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"receivers":1`) {
		t.Fatal(w.Body.String())
	}
	broadcastSend(a, request)
	if len(a.integration.NoticeBroadcasts) != 1 {
		t.Fatal("duplicate enqueue")
	}
	b := a.integration.NoticeBroadcasts[0]
	form.Del("baseline")
	w = broadcastWorkerCall(a, "/worker-speech/"+a.state.Board, form, "Windows")
	if !strings.Contains(w.Body.String(), b.ID) {
		t.Fatal("receiver missed broadcast", w.Body.String())
	}
	audio := url.Values{"csrf": {a.state.Board}, "owner": {owner}, "device": {device}, "broadcast_enabled": {"1"}, "broadcast": {b.ID}, "text": {"UNTRUSTED"}}
	calls := 0
	a.speechSynthesize = func(ctx context.Context, text string) ([]byte, error) {
		calls++
		if text != "긴급 공지입니다. 오후 안전교육. 3시에 회의실로 모여주세요." {
			t.Error("wrong server text", text)
		}
		return []byte("ID3synthetic"), nil
	}
	a.ttsConfig.Repeat = 2
	w = broadcastWorkerCall(a, "/worker-speech/"+a.state.Board+"/audio", audio, "Windows")
	if w.Code != 200 || calls != 1 || w.Header().Get("X-TTS-Repeat") != "1" {
		t.Fatal("audio", w.Code, w.Body.String())
	}
	w = broadcastWorkerCall(a, "/worker-speech/"+a.state.Board+"/audio", audio, "iPhone Mobile")
	if w.Code != 409 || calls != 1 {
		t.Fatal("phone audio allowed")
	}
	audio.Set("owner", token())
	if w = broadcastWorkerCall(a, "/worker-speech/"+a.state.Board+"/audio", audio, "Windows"); w.Code != 409 {
		t.Fatal("untargeted audio allowed")
	}
	audio.Set("owner", owner)
	a.state.Announcements[0].Published = false
	if w = broadcastWorkerCall(a, "/worker-speech/"+a.state.Board+"/audio", audio, "Windows"); w.Code != 409 {
		t.Fatal("ended announcement played")
	}
}
func TestNoticeBroadcastOptOutNewTabsAndStaleRevision(t *testing.T) {
	a := broadcastFixture(t)
	owner, device := token(), token()
	f := broadcastRegister(t, a, owner, device, "Windows", true)
	f.Set("broadcast_enabled", "0")
	f.Del("baseline")
	broadcastWorkerCall(a, "/worker-speech/"+a.state.Board, f, "Windows")
	if w := broadcastSend(a, token()); w.Code != 409 {
		t.Fatal("optout target retained")
	}
	f = broadcastRegister(t, a, owner, device, "Windows", true)
	w := broadcastSend(a, token())
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	id := a.integration.NoticeBroadcasts[0].ID
	// Re-establishing a lease must skip broadcasts queued before reconnect.
	w = broadcastWorkerCall(a, "/worker-speech/"+a.state.Board, f, "Windows")
	if strings.Contains(w.Body.String(), id) {
		t.Fatal("baseline replay")
	}
	f.Del("baseline")
	w = broadcastWorkerCall(a, "/worker-speech/"+a.state.Board, f, "Windows")
	if strings.Contains(w.Body.String(), id) {
		t.Fatal("post-baseline replay")
	}
	newForm := broadcastRegister(t, a, token(), token(), "Windows", true)
	newForm.Del("baseline")
	if w = broadcastWorkerCall(a, "/worker-speech/"+a.state.Board, newForm, "Windows"); strings.Contains(w.Body.String(), id) {
		t.Fatal("new device replay")
	}
	a.state.Announcements[0].Urgent = false
	if w = broadcastSend(a, token()); w.Code != 400 {
		t.Fatal("normal notice broadcast")
	}
	a.state.Announcements[0].Urgent = true
	a.integration.NoticeReceivers = map[string]noticeReceiver{"expired": {Owner: "old", Expires: time.Now().Add(-time.Second)}}
	if w = broadcastSend(a, token()); w.Code != 409 {
		t.Fatal("offline PC target")
	}
}
func TestNoticeBroadcastAdminButtonsAndBrowserFixture(t *testing.T) {
	a := broadcastFixture(t)
	cookie := adminCookie(a)
	list := call(a, "GET", "/announcements", nil, cookie)
	detail := call(a, "GET", "/announcements/"+a.state.Announcements[0].ID, nil, cookie)
	worker := call(a, "GET", "/work/"+a.state.Board+"?entry=skip", nil)
	public := call(a, "GET", "/notices/"+a.state.Board, nil)
	for _, w := range []*httptest.ResponseRecorder{list, detail} {
		if w.Code != 200 || !strings.Contains(w.Body.String(), `class="notice-broadcast-form"`) {
			t.Fatal("missing admin action", w.Code, w.Body.String())
		}
	}
	if strings.Contains(public.Body.String(), `class="notice-broadcast-form"`) {
		t.Fatal("worker broadcast action")
	}
	if dir := os.Getenv("NOTICE_BROADCAST_HTML"); dir != "" {
		os.MkdirAll(dir, 0700)
		for name, w := range map[string]*httptest.ResponseRecorder{"admin.html": list, "detail.html": detail, "work.html": worker, "speech-admin.html": call(a, "GET", "/", nil, cookie)} {
			os.WriteFile(filepath.Join(dir, name), w.Body.Bytes(), 0600)
		}
		b, _ := json.Marshal(map[string]string{"board": a.state.Board, "notice": a.state.Announcements[0].ID})
		os.WriteFile(filepath.Join(dir, "fixture.json"), b, 0600)
	}
}
func TestNoticeBroadcastLongTextSegments(t *testing.T) {
	a := broadcastFixture(t)
	a.state.Announcements[0].Text = strings.Repeat("안전교육 공지입니다. ", 60)
	owner, device := token(), token()
	broadcastRegister(t, a, owner, device, "Windows", true)
	w := broadcastSend(a, token())
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	b := a.integration.NoticeBroadcasts[0]
	parts := noticeSpeechParts(b.Text)
	if len(parts) < 2 || strings.Join(parts, "") != b.Text {
		t.Fatal("split lost text")
	}
	var actual []string
	a.speechSynthesize = func(ctx context.Context, text string) ([]byte, error) {
		actual = append(actual, text)
		return []byte("ID3synthetic"), nil
	}
	for i := range parts {
		audio := url.Values{"csrf": {a.state.Board}, "owner": {owner}, "device": {device}, "broadcast_enabled": {"1"}, "broadcast": {b.ID}, "part": {fmtID(int64(i))}}
		w = broadcastWorkerCall(a, "/worker-speech/"+a.state.Board+"/audio", audio, "Windows")
		if w.Code != 200 || w.Header().Get("X-TTS-Part-Count") != fmtID(int64(len(parts))) {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if strings.Join(actual, "") != b.Text {
		t.Fatal("TTS segments differ from saved content")
	}
	a.state.Announcements[0].Revision++
	audio := url.Values{"csrf": {a.state.Board}, "owner": {owner}, "device": {device}, "broadcast_enabled": {"1"}, "broadcast": {b.ID}}
	if w = broadcastWorkerCall(a, "/worker-speech/"+a.state.Board+"/audio", audio, "Windows"); w.Code != 409 {
		t.Fatal("edited notice old broadcast allowed")
	}
}
