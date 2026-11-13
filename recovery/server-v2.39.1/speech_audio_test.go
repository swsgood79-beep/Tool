package main

import (
	"archive/zip"
	"bytes"
	"context"
	"debug/pe"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSpeechAudioAuthorizationPrivacyAndServerText(t *testing.T) {
	a, id := pendingReview(t, "setting")
	cookie := adminCookie(a)
	calls := 0
	a.speechSynthesize = func(ctx context.Context, text string) ([]byte, error) {
		calls++
		if !strings.Contains(text, "신청 작업자님이") {
			t.Error("client text replaced server message", text)
		}
		return []byte("ID3synthetic"), nil
	}
	form := url.Values{"csrf": {"admin-csrf"}, "event": {fmtID(id)}, "text": {"UNTRUSTED"}}
	if r := call(a, "POST", "/admin-speech/audio", form); r.Code != 401 {
		t.Fatal(r.Code)
	}
	form.Set("csrf", "bad")
	if r := call(a, "POST", "/admin-speech/audio", form, cookie); r.Code != 403 {
		t.Fatal(r.Code)
	}
	form.Set("csrf", "admin-csrf")
	r := call(a, "POST", "/admin-speech/audio", form, cookie)
	if r.Code != 200 || r.Header().Get("Content-Type") != "audio/mpeg" || r.Body.String() != "ID3synthetic" {
		t.Fatal(r.Code, r.Body.String())
	}
	before, _ := json.Marshal(a.state)
	path := "/worker-speech/" + a.state.Board + "/audio"
	form.Set("csrf", a.state.Board)
	if r = call(a, "POST", path, form); r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	after, _ := json.Marshal(a.state)
	if !bytes.Equal(before, after) {
		t.Fatal("audio changed business data")
	}
	a.state.Events = append(a.state.Events, Event{ID: 999, ProjectID: 1, IsNote: true, Unread: true, Worker: "PRIVATE", Note: "PRIVATE"})
	form.Set("event", "999")
	if r = call(a, "POST", path, form); r.Code != 409 {
		t.Fatal("private note audio exposed", r.Code)
	}
	if calls != 2 {
		t.Fatal("unauthorized synthesis invoked", calls)
	}
	if r = call(a, "POST", "/worker-speech/invalid/audio", form); r.Code != 404 {
		t.Fatal(r.Code)
	}
	if r = call(a, "GET", path, nil); r.Code != 405 {
		t.Fatal(r.Code)
	}
}
func TestSlowSpeechAudioDoesNotBlockOrdersOrCrashOnFailure(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	started, release := make(chan struct{}), make(chan struct{})
	a.speechSynthesize = func(ctx context.Context, text string) ([]byte, error) {
		close(started)
		select {
		case <-release:
			return nil, errors.New("network down")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	f := url.Values{"csrf": {"admin-csrf"}, "preview": {"1"}}
	done := make(chan int, 1)
	go func() { done <- call(a, "POST", "/admin-speech/audio", f, cookie).Code }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("synth not started")
	}
	page := make(chan int, 1)
	go func() { page <- call(a, "GET", "/orders/panel", nil, cookie).Code }()
	select {
	case code := <-page:
		if code != 200 {
			t.Fatal(code)
		}
	case <-time.After(time.Second):
		t.Fatal("TTS blocked App.mu / orders")
	}
	close(release)
	if <-done != 502 {
		t.Fatal("failure not handled")
	}
	if r := call(a, "GET", "/", nil, cookie); r.Code != 200 {
		t.Fatal("app stopped after TTS failure")
	}
	a.speechSynthesize = func(context.Context, string) ([]byte, error) { return []byte("ID3recovered"), nil }
	if r := call(a, "POST", "/admin-speech/audio", f, cookie); r.Code != 200 {
		t.Fatal("next audio failed")
	}
}
func TestSpeechAudioCancellation(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	started := make(chan struct{})
	a.speechSynthesize = func(ctx context.Context, text string) ([]byte, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	f := url.Values{"csrf": {"admin-csrf"}, "preview": {"1"}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791/admin-speech/audio", strings.NewReader(f.Encode())).WithContext(ctx)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(cookie)
	done := make(chan struct{})
	go func() { a.ServeHTTP(httptest.NewRecorder(), r); close(done) }()
	<-started
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancel ignored")
	}
}
func TestBundledGTTSRuntimeChecksumsAndWindowsArchitecture(t *testing.T) {
	data, _ := ttsAssets.ReadFile("tts_assets/python-gtts-win-x64.zip")
	manifest, _ := ttsAssets.ReadFile("tts_assets/runtime-manifest.json")
	var m struct{ Files map[string]string }
	if err := json.Unmarshal(manifest, &m); err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, f := range z.File {
		in, _ := f.Open()
		b, e := io.ReadAll(in)
		in.Close()
		if e != nil || digest(b) != m.Files[f.Name] {
			t.Fatal("runtime checksum", f.Name)
		}
		seen[f.Name] = true
		if f.Name == "python.exe" {
			p, e := pe.NewFile(bytes.NewReader(b))
			if e != nil || p.Machine != pe.IMAGE_FILE_MACHINE_AMD64 {
				t.Fatal("wrong Python architecture")
			}
		}
	}
	for _, n := range []string{"python.exe", "python312.dll", "LICENSE.txt", "Lib/site-packages/edge_tts/communicate.py", "Lib/site-packages/comtypes/client/__init__.py", "Lib/site-packages/gtts/tts.py", "Lib/site-packages/pygame/__init__.py"} {
		if !seen[n] {
			t.Fatal("runtime missing", n)
		}
	}
	if len(seen) != len(m.Files) {
		t.Fatal("runtime manifest differs")
	}
}

func TestGTTSRuntimeExtractionAndCorruptionDoesNotOverwrite(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("LOCALAPPDATA", cache)
	ttsRuntimeMu.Lock()
	old := ttsRuntimePath
	ttsRuntimePath = ""
	ttsRuntimeMu.Unlock()
	defer func() { ttsRuntimeMu.Lock(); ttsRuntimePath = old; ttsRuntimeMu.Unlock() }()
	dir, err := prepareTTSRuntime()
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "python.exe")
	before, err := os.ReadFile(exe)
	if err != nil || len(before) < 100 {
		t.Fatal("runtime extraction failed", err)
	}
	same, err := prepareTTSRuntime()
	if err != nil || same != dir {
		t.Fatal("runtime was not reused", err)
	}
	if err = os.WriteFile(exe, []byte("corrupt-synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	ttsRuntimeMu.Lock()
	ttsRuntimePath = ""
	ttsRuntimeMu.Unlock()
	if _, err = prepareTTSRuntime(); err == nil {
		t.Fatal("corrupt runtime accepted")
	}
	after, _ := os.ReadFile(exe)
	if string(after) != "corrupt-synthetic" {
		t.Fatal("existing damaged cache was overwritten")
	}
}
