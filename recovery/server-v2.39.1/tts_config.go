package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const ttsTestText = "아리아니 공구14님이 테스트 완료"

type TTSConfig struct {
	Enabled bool    `json:"enabled"`
	Voice   string  `json:"voice"`
	Speed   float64 `json:"speed"`
}

func defaultTTSConfig() TTSConfig { return TTSConfig{true, "gtts", 1.10} }
func (c TTSConfig) validate() error {
	if c.Voice != "gtts" && c.Voice != "sunhi" && c.Voice != "injoon" && c.Voice != "sapi" {
		return errors.New("선택한 목소리가 올바르지 않습니다")
	}
	if math.IsNaN(c.Speed) || math.IsInf(c.Speed, 0) || c.Speed < .8 || c.Speed > 1.5 || math.Abs(c.Speed*20-math.Round(c.Speed*20)) > 1e-7 {
		return errors.New("배속은 0.80~1.50, 0.05 간격이어야 합니다")
	}
	return nil
}
func (c TTSConfig) label() string {
	return map[string]string{"gtts": "구글 전자녀 (gTTS)", "sunhi": "MS 선희", "injoon": "MS 인준", "sapi": "윈도우 기본 내장 (SAPI)"}[c.Voice]
}
func (a *App) loadTTSConfig() {
	c := defaultTTSConfig()
	c.Enabled = !a.state.Integration.DesktopSpeechMuted
	if a.state.Integration.TTS != nil {
		c = *a.state.Integration.TTS
	}
	b, err := os.ReadFile(filepath.Join(a.dir, "tts_config.json"))
	if err == nil {
		if err = json.Unmarshal(b, &c); err != nil {
			a.ttsConfigError = "tts_config.json: " + err.Error()
			c.Enabled = false
		}
	}
	if err != nil && !os.IsNotExist(err) {
		a.ttsConfigError = "tts_config.json: " + err.Error()
		c.Enabled = false
	}
	if err = c.validate(); err != nil {
		a.ttsConfigError = err.Error()
		c.Enabled = false
	}
	a.ttsConfig = c
	if a.ttsConfigError == "" {
		data, _ := json.MarshalIndent(c, "", "  ")
		if err = atomicFile(filepath.Join(a.dir, "tts_config.json"), append(data, '\n')); err != nil {
			a.ttsConfigError = err.Error()
		}
	}
}

// Caller holds App.mu. The sidecar and legacy backup mirror are rolled back on save failure.
func (a *App) saveTTSConfig(c TTSConfig) error {
	if err := c.validate(); err != nil {
		return err
	}
	name := filepath.Join(a.dir, "tts_config.json")
	old, readErr := os.ReadFile(name)
	if readErr != nil && !os.IsNotExist(readErr) {
		return readErr
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	if err := atomicFile(name, append(b, '\n')); err != nil {
		return err
	}
	if err := a.transaction(func(s *State) error {
		s.Integration.DesktopSpeechMuted = !c.Enabled
		copy := c
		s.Integration.TTS = &copy
		return nil
	}); err != nil {
		var rollback error
		if readErr == nil {
			rollback = atomicFile(name, old)
		} else {
			rollback = os.Remove(name)
		}
		if rollback != nil {
			return fmt.Errorf("설정 저장 실패: %v / JSON 복원 실패: %v", err, rollback)
		}
		return err
	}
	a.ttsConfig = c
	a.ttsConfigError = ""
	return nil
}
func (a *App) playConfiguredSpeech(ctx context.Context, text string) error {
	a.mu.Lock()
	c, configErr := a.ttsConfig, a.ttsConfigError
	a.mu.Unlock()
	if configErr != "" {
		return errors.New(configErr)
	}
	_, err := runTTSPython(ctx, text, "speak", c)
	return err
}
func (a *App) ttsConfigHTTP(w http.ResponseWriter, r *http.Request, s Session) bool {
	if r.URL.Path != "/tts-settings" && r.URL.Path != "/tts-settings/test" && r.URL.Path != "/tts-settings/status" {
		return false
	}
	if !localRequest(r) {
		http.Error(w, "PC 음성 설정은 회사 서버 PC에서 변경하세요", 403)
		return true
	}
	if r.Method == "GET" && r.URL.Path == "/tts-settings" {
		a.render(w, "tts-settings.html", map[string]any{"CSRF": s.CSRF, "TTS": a.ttsConfig, "Error": a.ttsConfigError})
		return true
	}
	if r.Method == "GET" && r.URL.Path == "/tts-settings/status" {
		status := a.ttsConfigError
		if a.nativeQueue != nil && status == "" {
			status = a.nativeQueue.state()
		}
		jsonReply(w, map[string]any{"config": a.ttsConfig, "status": status})
		return true
	}
	if r.Method != "POST" {
		http.Error(w, "지원하지 않는 요청", 405)
		return true
	}
	if r.URL.Path == "/tts-settings/test" {
		if a.integration.SpeechLeases["local"].Expires.After(time.Now()) {
			http.Error(w, "이 PC의 다른 화면에서 음성을 읽고 있습니다. 그 화면의 음성 알림을 끈 뒤 테스트해 주세요.", 409)
			return true
		}
		if a.nativeQueue == nil {
			http.Error(w, "PC 프로그램 창이 실행 중이지 않습니다", 409)
			return true
		}
		if a.ttsConfigError != "" {
			http.Error(w, a.ttsConfigError, 409)
			return true
		}
		snapshot := a.ttsConfig
		a.nativeQueue.enqueue(desktopEvent{SpeechPreview: true, SpeechConfig: &snapshot})
		jsonReply(w, map[string]any{"ok": true, "queued": true})
		return true
	}
	speed, err := strconv.ParseFloat(r.FormValue("speed"), 64)
	if err != nil {
		http.Error(w, "배속 오류", 400)
		return true
	}
	c := TTSConfig{r.FormValue("enabled") == "1", "" + r.FormValue("voice"), speed}
	if err = a.saveTTSConfig(c); err != nil {
		http.Error(w, err.Error(), 400)
		return true
	}
	jsonReply(w, map[string]any{"ok": true, "config": c})
	return true
}

// A portable restore projects the restored mirror into the new local sidecar.
func (a *App) restoreTTSConfig() {
	c := defaultTTSConfig()
	c.Enabled = !a.state.Integration.DesktopSpeechMuted
	if a.state.Integration.TTS != nil {
		c = *a.state.Integration.TTS
	}
	a.ttsConfigError = ""
	if err := c.validate(); err != nil {
		a.ttsConfigError = err.Error()
		c.Enabled = false
	}
	a.ttsConfig = c
	if a.ttsConfigError != "" {
		return
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	if err := atomicFile(filepath.Join(a.dir, "tts_config.json"), append(b, '\n')); err != nil {
		a.ttsConfigError = "복원된 음성 설정 저장 실패: " + err.Error()
	}
}
