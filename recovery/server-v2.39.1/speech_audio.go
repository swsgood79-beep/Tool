package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// This handler authenticates/snapshots under App.mu, then releases it BEFORE network/TTS work.
func (a *App) speechAudioHTTP(w http.ResponseWriter, r *http.Request) bool {
	shared := strings.HasPrefix(r.URL.Path, "/worker-speech/") && strings.HasSuffix(r.URL.Path, "/audio")
	if r.URL.Path != "/admin-speech/audio" && !shared {
		return false
	}
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, "POST만 지원합니다", 405)
		return true
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "음성 요청 오류", 400)
		return true
	}
	a.mu.Lock()
	text := ""
	if shared {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 3 || parts[2] != "audio" || !equal(parts[1], a.state.Board) {
			a.mu.Unlock()
			http.NotFound(w, r)
			return true
		}
		if !equal(r.FormValue("csrf"), a.state.Board) {
			a.mu.Unlock()
			http.Error(w, "작업장 요청 검증 실패", 403)
			return true
		}
	} else {
		if _, ok := a.admin(w, r); !ok {
			a.mu.Unlock()
			return true
		}
	}
	if r.FormValue("preview") == "1" {
		text = ttsTestText
		if r.FormValue("welcome") == "1" {
			text = "음성 알림을 켭니다."
		}
	} else {
		id := number(r.FormValue("event"))
		for _, e := range a.state.Events {
			if e.ID != id || !notifyEvent(e) || (e.BatchID != 0 && e.BatchID != e.ID) {
				continue
			}
			p := a.state.project(e.ProjectID)
			if shared && (e.IsNote || !e.Pending || p == nil || p.Hidden || p.LegacyCompleted || p.Shipping.Completed != nil) {
				continue
			}
			if p != nil {
				e.Project = p.Name
			}
			text = inspectionSpeech(desktopNotice(e))
			break
		}
	}
	if a.ttsConfigError != "" {
		msg := a.ttsConfigError
		a.mu.Unlock()
		http.Error(w, msg, 409)
		return true
	}
	cfg := a.ttsConfig
	if v := r.FormValue("voice"); v != "" {
		cfg.Voice = v
	}
	if v := r.FormValue("speed"); v != "" {
		if x, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Speed = x
		} else {
			a.mu.Unlock()
			http.Error(w, "배속 오류", 400)
			return true
		}
	}
	if err := cfg.validate(); err != nil {
		a.mu.Unlock()
		http.Error(w, err.Error(), 400)
		return true
	}
	configSynth := a.speechSynthesizeConfigured
	synth := a.speechSynthesize
	a.mu.Unlock()
	if text == "" {
		http.Error(w, "재생할 새 알림이 없습니다", 409)
		return true
	}
	if synth == nil {
		synth = func(ctx context.Context, s string) ([]byte, error) { return runTTSPython(ctx, s, "synth", cfg) }
	}
	var audio []byte
	var err error
	if configSynth != nil {
		audio, err = configSynth(r.Context(), text, cfg)
	} else {
		audio, err = synth(r.Context(), text)
	}
	if err != nil || len(audio) < 4 || len(audio) > 8<<20 {
		detail := "잘못된 음성 데이터"
		if err != nil {
			detail = err.Error()
		}
		http.Error(w, cfg.label()+" 음성 실패: "+detail, 502)
		return true
	}
	mime := "audio/mpeg"
	if bytes.HasPrefix(audio, []byte("RIFF")) {
		mime = "audio/wav"
	}
	w.Header().Set("Content-Type", mime)
	rate := cfg.Speed
	if cfg.Voice == "sunhi" || cfg.Voice == "injoon" {
		rate = 1
	}
	w.Header().Set("X-TTS-Playback-Rate", fmt.Sprintf("%.2f", rate))
	w.Header().Set("X-TTS-Voice", cfg.Voice)
	w.Header().Set("X-TTS-Requested-Speed", fmt.Sprintf("%.2f", cfg.Speed))
	w.Header().Set("Cache-Control", "no-store")
	w.Write(audio)
	return true
}
