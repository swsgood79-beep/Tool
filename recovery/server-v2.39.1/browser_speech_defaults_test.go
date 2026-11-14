package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBrowserSpeechDefaultOnKeepsNativePreference(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	for _, path := range []string{"/", "/work/" + a.state.Board + "?entry=skip"} {
		w := call(a, "GET", path, nil, cookie)
		s := w.Body.String()
		if w.Code != 200 || !strings.Contains(s, `id="electron-speech-toggle" aria-pressed="true"`) || !strings.Contains(s, "tts-speech-v1.4.22.js") {
			t.Fatal("browser default ON markup missing", path)
		}
	}
	before, _ := json.Marshal(a.state)
	f := speechForm(token(), token(), 0, true) // automatic starts never select_browser=1
	if !speechPoll(t, a, f, cookie).Active || !a.ttsConfig.Enabled {
		t.Fatal("automatic browser start muted PC preference")
	}
	after, _ := json.Marshal(a.state)
	if string(before) != string(after) {
		t.Fatal("automatic start wrote company/settings data")
	}
	a.integration.NativeSpeechSelected = true
	delete(a.integration.SpeechLeases, "local")
	r := call(a, "POST", "/admin-speech/poll", f, cookie)
	var out map[string]any
	if json.Unmarshal(r.Body.Bytes(), &out) != nil || out["active"] != false || out["native_selected"] != true || !a.ttsConfig.Enabled {
		t.Fatal("automatic start stole PC speaker", r.Body.String())
	}
}
