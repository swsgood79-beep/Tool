package main

import (
	"context"
	"net/http"
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
		text = "홍길동님이 알파 작업 확인의 세팅 및 출하 통합검사를 신청했습니다."
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
	synth := a.speechSynthesize
	a.mu.Unlock()
	if text == "" {
		http.Error(w, "재생할 새 알림이 없습니다", 409)
		return true
	}
	if synth == nil {
		synth = func(ctx context.Context, s string) ([]byte, error) { return runGTTSPython(ctx, s, "synth") }
	}
	audio, err := synth(r.Context(), text)
	if err != nil || len(audio) < 4 || len(audio) > 8<<20 {
		http.Error(w, "Google 음성 생성 실패 · 인터넷 연결을 확인하세요", 502)
		return true
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(audio)
	return true
}
