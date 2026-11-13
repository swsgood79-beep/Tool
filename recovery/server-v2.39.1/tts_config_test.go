package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"filippo.io/age"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTTSConfigDefaultsValidationAndRestart(t *testing.T) {
	a := fixture(t)
	if a.ttsConfig != defaultTTSConfig() {
		t.Fatal(a.ttsConfig)
	}
	if out := os.Getenv("TTS_DEFAULT_HTML"); out != "" {
		page := call(a, "GET", "/tts-settings", nil, adminCookie(a))
		if err := os.WriteFile(out, page.Body.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := TTSConfig{false, "injoon", .95, 1}
	if err := a.saveTTSConfig(cfg); err != nil {
		t.Fatal(err)
	}
	b, err := newApp(a.dir)
	if err != nil || b.ttsConfig != cfg {
		t.Fatal("lost TTS settings", err, b.ttsConfig)
	}
	raw, _ := os.ReadFile(filepath.Join(a.dir, "tts_config.json"))
	var saved TTSConfig
	if json.Unmarshal(raw, &saved) != nil || saved != cfg || !b.state.Integration.DesktopSpeechMuted || b.state.Integration.TTS == nil {
		t.Fatal("sidecar/backup mirror mismatch")
	}
	for _, c := range []TTSConfig{{true, "other", 1.1, 1}, {true, "gtts", .75, 1}, {true, "sunhi", 1.55, 1}, {true, "sapi", 1.11, 1}} {
		if a.saveTTSConfig(c) == nil {
			t.Fatal("invalid settings accepted", c)
		}
	}
}
func TestTTSConfigCorruptFileDoesNotChooseFallback(t *testing.T) {
	a := fixture(t)
	name := filepath.Join(a.dir, "tts_config.json")
	bad := []byte("{broken")
	os.WriteFile(name, bad, 0600)
	b, err := newApp(a.dir)
	if err != nil {
		t.Fatal("TTS error stopped company app", err)
	}
	if b.ttsConfigError == "" || b.ttsConfig.Enabled {
		t.Fatal("corruption silently replaced settings")
	}
	if err = b.playConfiguredSpeech(context.Background(), "test"); err == nil || !strings.Contains(err.Error(), "tts_config.json") {
		t.Fatal("corrupt config played", err)
	}
	got, _ := os.ReadFile(name)
	if !bytes.Equal(got, bad) {
		t.Fatal("corrupt file overwritten")
	}
	if call(b, "GET", "/", nil, adminCookie(b)).Code != 200 {
		t.Fatal("orders unavailable")
	}
	if err = b.saveTTSConfig(defaultTTSConfig()); err != nil || b.ttsConfigError != "" {
		t.Fatal("explicit repair failed", err)
	}
}
func TestTTSConfigSaveFailureRollsBackFile(t *testing.T) {
	a := fixture(t)
	name := filepath.Join(a.dir, "tts_config.json")
	before, _ := os.ReadFile(name)
	cfg := a.ttsConfig
	a.directHook = func(string, string) error { return errors.New("synthetic storage failure") }
	if err := a.saveTTSConfig(TTSConfig{true, "sunhi", 1.2, 1}); err == nil {
		t.Fatal("failed mirror accepted")
	}
	after, _ := os.ReadFile(name)
	if !bytes.Equal(before, after) || a.ttsConfig != cfg {
		t.Fatal("sidecar rollback failed")
	}
	a.directHook = nil
}
func TestTTSSettingsPanelSaveAndManualPreviewWhenOff(t *testing.T) {
	a := fixture(t)
	c := adminCookie(a)
	f := url.Values{"csrf": {"admin-csrf"}, "enabled": {"0"}, "voice": {"sunhi"}, "speed": {"1.10"}}
	if r := call(a, "POST", "/tts-settings", f, c); r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	started, release := make(chan string, 1), make(chan struct{})
	a.nativeQueue = newInspectionAnnouncer(false, func(ctx context.Context, text string) error {
		started <- text
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	defer a.nativeQueue.close()
	r := call(a, "GET", "/tts-settings", nil, c)
	for _, s := range []string{"구글 전자녀 (gTTS - 크롬 기본)", "MS 선희", "MS 인준", "윈도우 기본 내장", `step="0.05"`, `min="0.8"`, `max="1.5"`, ttsTestText} {
		if !strings.Contains(r.Body.String(), s) {
			t.Fatal("setting control missing", s)
		}
	}
	if out := os.Getenv("TTS_SETTINGS_HTML"); out != "" {
		if err := os.WriteFile(out, r.Body.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if r = call(a, "POST", "/tts-settings/test", url.Values{"csrf": {"admin-csrf"}}, c); r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	select {
	case text := <-started:
		if text != ttsTestText {
			t.Fatal(text)
		}
	case <-time.After(time.Second):
		t.Fatal("manual preview blocked by OFF")
	}
	if call(a, "GET", "/orders/panel", nil, c).Code != 200 {
		t.Fatal("preview blocked UI")
	}
	close(release)
}
func TestTTSAudioSelectedEngineAndRateNoDoubleSpeedOrFallback(t *testing.T) {
	a := fixture(t)
	c := adminCookie(a)
	calls := 0
	a.speechSynthesizeConfigured = func(_ context.Context, text string, cfg TTSConfig) ([]byte, error) {
		calls++
		if text != ttsTestText {
			t.Error(text)
		}
		if cfg.Voice == "injoon" {
			return nil, errors.New("WSServerHandshakeError: HTTP403")
		}
		if cfg.Voice == "sapi" {
			return []byte("RIFFsyntheticWAV"), nil
		}
		return []byte("ID3syntheticMP3"), nil
	}
	for _, voice := range []string{"gtts", "sunhi", "injoon", "sapi"} {
		f := url.Values{"csrf": {"admin-csrf"}, "preview": {"1"}, "voice": {voice}, "speed": {"1.25"}}
		r := call(a, "POST", "/admin-speech/audio", f, c)
		if voice == "injoon" {
			if r.Code != 502 || !strings.Contains(r.Body.String(), "MS 인준") || !strings.Contains(r.Body.String(), "HTTP403") {
				t.Fatal("error cause hidden", r.Code, r.Body.String())
			}
			continue
		}
		want := "1.25"
		if voice == "sunhi" {
			want = "1.00"
		}
		if r.Code != 200 || r.Header().Get("X-TTS-Playback-Rate") != want {
			t.Fatal(voice, r.Code, r.Header())
		}
		if voice == "sapi" && r.Header().Get("Content-Type") != "audio/wav" {
			t.Fatal("SAPI WAV mislabeled")
		}
	}
	if calls != 4 {
		t.Fatal("another voice used as fallback", calls)
	}
	if r := call(a, "POST", "/tts-settings", url.Values{"csrf": {"wrong"}}, c); r.Code != 403 {
		t.Fatal(r.Code)
	}
	if r := call(a, "GET", "/tts-settings", nil); r.Code != 303 {
		t.Fatal("settings exposed", r.Code)
	}
}

func TestTTSQueuedPreviewKeepsClickedVoiceAndSpeed(t *testing.T) {
	started := make(chan TTSConfig, 4)
	gate := make(chan struct{})
	s := newInspectionAnnouncer(false, func(context.Context, string) error { return errors.New("preview config was not passed") })
	defer s.close()
	s.mu.Lock()
	s.runConfigured = func(ctx context.Context, text string, c TTSConfig) error {
		started <- c
		select {
		case <-gate:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	s.mu.Unlock()
	first := TTSConfig{false, "sunhi", 1.1, 1}
	s.enqueue(desktopEvent{SpeechPreview: true, SpeechConfig: &first})
	if c := awaitSpeech(t, started); c != first {
		t.Fatal(c)
	}
	second := TTSConfig{false, "injoon", .9, 1}
	s.enqueue(desktopEvent{SpeechPreview: true, SpeechConfig: &second})
	second.Voice = "gtts"
	close(gate)
	if c := awaitSpeech(t, started); c.Voice != "injoon" || c.Speed != .9 {
		t.Fatal("queued preview changed its chosen settings", c)
	}
}

func TestTTSPreferencesSurvivePortableBackup(t *testing.T) {
	a := fixture(t)
	cfg := TTSConfig{false, "sapi", 1.35, 2}
	if err := a.saveTTSConfig(cfg); err != nil {
		t.Fatal(err)
	}
	identity, _ := age.GenerateX25519Identity()
	key := RecoveryKey{1, repository, token(), identity.String()}
	a.state.Integration.Backup.Key = key
	data, err := a.portable(a.state)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := newApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = fresh.restorePortable(key, data); err != nil {
		t.Fatal(err)
	}
	if fresh.ttsConfig != cfg {
		t.Fatal("backup lost TTS preferences", fresh.ttsConfig)
	}
	restarted, err := newApp(fresh.dir)
	if err != nil || restarted.ttsConfig != cfg {
		t.Fatal("restored sidecar changed on restart", err)
	}
}

func TestTTSRepeatMigrationPersistenceAndSingleGeneration(t *testing.T) {
	a := fixture(t)
	name := filepath.Join(a.dir, "tts_config.json")
	if err := os.WriteFile(name, []byte(`{"enabled":true,"voice":"gtts","speed":1.1}`), 0600); err != nil {
		t.Fatal(err)
	}
	b, err := newApp(a.dir)
	if err != nil || b.ttsConfig.Repeat != 1 {
		t.Fatal("old settings did not migrate to one read", err)
	}
	cfg := b.ttsConfig
	cfg.Repeat = 2
	if err = b.saveTTSConfig(cfg); err != nil {
		t.Fatal(err)
	}
	restarted, err := newApp(b.dir)
	if err != nil || restarted.ttsConfig.Repeat != 2 {
		t.Fatal("repeat not persisted", err)
	}
	cookie := adminCookie(restarted)
	calls := 0
	restarted.speechSynthesizeConfigured = func(context.Context, string, TTSConfig) ([]byte, error) {
		calls++
		return []byte("ID3single-audio"), nil
	}
	r := call(restarted, "POST", "/admin-speech/audio", url.Values{"csrf": {"admin-csrf"}, "preview": {"1"}}, cookie)
	if r.Code != 200 || r.Header().Get("X-TTS-Repeat") != "2" || calls != 1 {
		t.Fatal("repeat regenerated or missing", r.Code, r.Header(), calls)
	}
	legacyForm := url.Values{"csrf": {"admin-csrf"}, "enabled": {"1"}, "voice": {"gtts"}, "speed": {"1.10"}}
	if r = call(restarted, "POST", "/tts-settings", legacyForm, cookie); r.Code != 200 || restarted.ttsConfig.Repeat != 2 {
		t.Fatal("old GUI reset repetition")
	}
	for _, repeat := range []int{0, 3, -1} {
		bad := cfg
		bad.Repeat = repeat
		if restarted.saveTTSConfig(bad) == nil {
			t.Fatal("invalid repeat accepted")
		}
	}
}
