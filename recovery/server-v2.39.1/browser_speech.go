package main

import (
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

type speechLease struct {
	Owner   string
	Expires time.Time
}
type speechAlert struct {
	ID      int64  `json:"id"`
	Worker  string `json:"worker"`
	Message string `json:"message"`
	Text    string `json:"text"`
}

var speechIdentity = regexp.MustCompile(`^[A-Za-z0-9_-]{16,80}$`)

// Called under App.mu, after administrator authentication and CSRF validation.
func (a *App) browserSpeechPOST(w http.ResponseWriter, r *http.Request, session Session) bool {
	if r.Method != "POST" || r.URL.Path != "/admin-speech/poll" {
		return false
	}
	a.speechPollReply(w, r, session.CSRF, false)
	return true
}

// Worker-board access permits request announcements, never administrator messages/settings.
// Called under App.mu after the board URL has been checked by worker().
func (a *App) workerSpeech(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, "음성 연결은 POST로 요청하세요.", http.StatusMethodNotAllowed)
		return
	}
	if !equal(r.FormValue("csrf"), a.state.Board) {
		http.Error(w, "작업장 요청 검증 실패", http.StatusForbidden)
		return
	}
	a.speechPollReply(w, r, a.state.Board, true)
}

func (a *App) speechPollReply(w http.ResponseWriter, r *http.Request, principal string, shared bool) {
	owner, device := r.FormValue("owner"), r.FormValue("device")
	if !speechIdentity.MatchString(owner) || !speechIdentity.MatchString(device) {
		http.Error(w, "음성 연결 식별자 오류", 400)
		return
	}
	now := time.Now()
	if a.integration.SpeechLeases == nil {
		a.integration.SpeechLeases = map[string]speechLease{}
	}
	for key, lease := range a.integration.SpeechLeases {
		if !lease.Expires.After(now) {
			delete(a.integration.SpeechLeases, key)
		}
	}
	scope := "remote:" + hashToken(device)
	if localRequest(r) {
		scope = "local"
	}
	identity := hashToken(principal + ":" + owner)
	lease := a.integration.SpeechLeases[scope]
	active := false
	if r.FormValue("enabled") == "1" {
		if lease.Owner == "" || lease.Owner == identity {
			if len(a.integration.SpeechLeases) >= 64 && lease.Owner == "" {
				http.Error(w, "음성 연결 수가 너무 많습니다.", 429)
				return
			}
			// The explicit browser ON click replaces the local default voice. OFF must
			// not unexpectedly restart the old Windows voice on the next application.
			if !shared && scope == "local" && r.FormValue("select_browser") == "1" && a.ttsConfig.Enabled {
				if err := func() error { c := a.ttsConfig; c.Enabled = false; return a.saveTTSConfig(c) }(); err != nil {
					http.Error(w, "PC 기본 음성을 끄지 못했습니다.", 500)
					return
				}
			}
			a.integration.SpeechLeases[scope] = speechLease{identity, now.Add(9 * time.Second)}
			active = true
		}
	} else if lease.Owner == identity {
		delete(a.integration.SpeechLeases, scope)
	}
	cursor := a.state.NextID
	alerts := []speechAlert{}
	after := number(r.FormValue("after"))
	if active && r.FormValue("baseline") != "1" && after >= 0 && after < a.state.NextID {
		events := append([]Event(nil), a.state.Events...)
		sort.Slice(events, func(i, j int) bool { return events[i].ID < events[j].ID })
		for _, e := range events {
			if e.ID <= after || (!shared && !e.Unread) || !notifyEvent(e) || (e.BatchID != 0 && e.BatchID != e.ID) {
				continue
			}
			if shared {
				p := a.state.project(e.ProjectID)
				if e.IsNote || !e.Pending || p == nil || p.Hidden || p.LegacyCompleted || p.Shipping.Completed != nil {
					continue
				}
			}
			if p := a.state.project(e.ProjectID); p != nil {
				e.Project, e.Order = p.Name, p.Number
			}
			n := desktopNotice(e)
			text := inspectionSpeech(n)
			if text == "" {
				continue
			}
			worker := speechField(n.Worker, "작업자", 100)
			message := strings.TrimPrefix(text, worker+"님이 ")
			alerts = append(alerts, speechAlert{e.ID, worker, message, text})
			if len(alerts) == 256 {
				cursor = e.ID
				break
			}
		}
	}
	jsonReply(w, map[string]any{"ok": true, "active": active, "cursor": cursor, "events": alerts, "native_suppressed": active && scope == "local"})
}
