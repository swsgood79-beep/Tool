package main

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPCSettingsOnReclaimsBrowserLeaseAndPreview(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	owner, device := token(), token()
	f := speechForm(owner, device, 0, true)
	if !speechPoll(t, a, f, cookie).Active {
		t.Fatal("browser did not acquire lease")
	}
	a.nativeQueue = newInspectionAnnouncer(false, func(_ context.Context, text string) error { return nil })
	defer a.nativeQueue.close()
	// This is the user's screenshot state: PC ON, live browser lease, native OFF.
	blocked := call(a, "POST", "/tts-settings/test", url.Values{"csrf": {"admin-csrf"}}, cookie)
	if blocked.Code != 409 {
		t.Fatal("fixture did not reproduce blocked preview", blocked.Code)
	}
	status := call(a, "GET", "/tts-settings/status", nil, cookie)
	if !strings.Contains(status.Body.String(), "브라우저 화면에서") {
		t.Fatal("suppression cause not visible")
	}
	selected := make(chan TTSConfig, 1)
	a.nativeQueue.mu.Lock()
	a.nativeQueue.runConfigured = func(_ context.Context, text string, cfg TTSConfig) error {
		if text != ttsTestText {
			t.Error("wrong preview")
		}
		selected <- cfg
		return nil
	}
	a.nativeQueue.mu.Unlock()
	saved := call(a, "POST", "/tts-settings", url.Values{"csrf": {"admin-csrf"}, "enabled": {"1"}, "voice": {"sunhi"}, "speed": {"1.05"}, "repeat": {"1"}}, cookie)
	if saved.Code != 200 {
		t.Fatal(saved.Body.String())
	}
	if !a.integration.NativeSpeechSelected || a.integration.SpeechLeases["local"].Owner != "" {
		t.Fatal("PC ON failed to reclaim local speaker")
	}
	// A stale renewal or another waiting tab must not steal it back.
	for _, old := range []url.Values{f, speechForm(token(), device, 0, true)} {
		r := call(a, "POST", "/admin-speech/poll", old, cookie)
		var v map[string]any
		if json.Unmarshal(r.Body.Bytes(), &v) != nil || v["active"] != false || v["native_selected"] != true {
			t.Fatal("stale tab stole speaker", r.Body.String())
		}
	}
	r := call(a, "POST", "/tts-settings/test", url.Values{"csrf": {"admin-csrf"}}, cookie)
	if r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	select {
	case cfg := <-selected:
		if cfg.Voice != "sunhi" || cfg.Speed != 1.05 || cfg.Repeat != 1 {
			t.Fatal(cfg)
		}
	case <-time.After(time.Second):
		t.Fatal("test did not reach configured native engine")
	}
	// A deliberate browser ON can still select the browser again.
	f.Set("select_browser", "1")
	if !speechPoll(t, a, f, cookie).Active || a.integration.NativeSpeechSelected {
		t.Fatal("explicit browser selection blocked")
	}
}

func TestPCSettingsFailedSaveDoesNotTakeSpeaker(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	f := speechForm(token(), token(), 0, true)
	speechPoll(t, a, f, cookie)
	lease := a.integration.SpeechLeases["local"]
	r := call(a, "POST", "/tts-settings", url.Values{"csrf": {"admin-csrf"}, "enabled": {"1"}, "voice": {"bad"}, "speed": {"1.05"}}, cookie)
	if r.Code != 400 || a.integration.NativeSpeechSelected || a.integration.SpeechLeases["local"] != lease {
		t.Fatal("invalid settings changed speaker ownership")
	}
}
