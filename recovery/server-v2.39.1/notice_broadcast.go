package main

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type noticeReceiver struct {
	Owner   string
	Expires time.Time
}
type noticeBroadcast struct {
	ID, NoticeID, Text, Request string
	Revision                    int64
	Created                     time.Time
	Targets                     map[string]string
}

func speechScope(r *http.Request, device string) string {
	if localRequest(r) {
		return "local"
	}
	return "remote:" + hashToken(device)
}
func phoneSpeech(r *http.Request) bool {
	u := strings.ToLower(r.UserAgent())
	for _, s := range []string{"android", "iphone", "ipad", "ipod", "mobile"} {
		if strings.Contains(u, s) {
			return true
		}
	}
	return r.Header.Get("Sec-CH-UA-Mobile") == "?1"
}
func (a *App) pruneNoticeBroadcasts(now time.Time) {
	for key, v := range a.integration.NoticeReceivers {
		if !v.Expires.After(now) {
			delete(a.integration.NoticeReceivers, key)
		}
	}
	keep := a.integration.NoticeBroadcasts[:0]
	for _, b := range a.integration.NoticeBroadcasts {
		if now.Sub(b.Created) < 15*time.Minute {
			keep = append(keep, b)
		}
	}
	a.integration.NoticeBroadcasts = keep
}
func (a *App) noticeBroadcastPoll(r *http.Request, shared, active bool, scope, identity string, now time.Time) []map[string]string {
	a.pruneNoticeBroadcasts(now)
	out := []map[string]string{}
	if !shared {
		return out
	}
	if a.integration.NoticeReceivers == nil {
		a.integration.NoticeReceivers = map[string]noticeReceiver{}
	}
	enabled := active && r.FormValue("broadcast_enabled") == "1" && !phoneSpeech(r)
	if !enabled {
		if a.integration.NoticeReceivers[scope].Owner == identity {
			delete(a.integration.NoticeReceivers, scope)
		}
		return out
	}
	a.integration.NoticeReceivers[scope] = noticeReceiver{identity, now.Add(9 * time.Second)}
	// New tabs and reconnects start at the current instant; old broadcasts are never replayed.
	if r.FormValue("baseline") == "1" {
		for i := range a.integration.NoticeBroadcasts {
			b := &a.integration.NoticeBroadcasts[i]
			if b.Targets[scope] == identity {
				delete(b.Targets, scope)
			}
		}
		return out
	}
	for _, b := range a.integration.NoticeBroadcasts {
		n := findAnnouncement(&a.state, b.NoticeID)
		if b.Targets[scope] == identity && n != nil && n.Published && n.Urgent && n.Revision == b.Revision {
			out = append(out, map[string]string{"id": b.ID})
		}
	}
	return out
}
func (a *App) noticeBroadcastPOST(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/announcements/broadcast" {
		return false
	}
	n := findAnnouncement(&a.state, r.FormValue("id"))
	if n == nil || !n.Published || !n.Urgent {
		http.Error(w, "게시 중인 긴급공지만 방송할 수 있습니다.", 400)
		return true
	}
	if n.Revision != number(r.FormValue("revision")) {
		http.Error(w, "공지가 변경되었습니다. 새로고침 후 다시 눌러 주세요.", 409)
		return true
	}
	request := r.FormValue("request")
	if !speechIdentity.MatchString(request) {
		http.Error(w, "방송 요청 식별자 오류", 400)
		return true
	}
	now := time.Now()
	a.pruneNoticeBroadcasts(now)
	for _, b := range a.integration.NoticeBroadcasts {
		if b.Request == request {
			jsonReply(w, map[string]any{"ok": true, "receivers": len(b.Targets), "id": b.ID})
			return true
		}
	}
	text := "긴급 공지입니다. " + strings.TrimSpace(n.Title) + ". " + strings.TrimSpace(n.Text)
	if utf8.RuneCountInString(text) > 1600 {
		http.Error(w, "음성 방송은 제목과 내용을 합쳐 1,580자 이내로 작성해 주세요.", 400)
		return true
	}
	targets := map[string]string{}
	for scope, v := range a.integration.NoticeReceivers {
		lease := a.integration.SpeechLeases[scope]
		if v.Expires.After(now) && lease.Expires.After(now) && lease.Owner == v.Owner {
			targets[scope] = v.Owner
		}
	}
	if len(targets) == 0 {
		http.Error(w, "수신 중인 공용 PC가 없습니다. 공용 PC 작업자 화면에서 음성 알림과 ‘긴급공지 음성 수신’을 켜 주세요.", 409)
		return true
	}
	if len(a.integration.NoticeBroadcasts) >= 100 {
		http.Error(w, "방송 요청이 많습니다. 잠시 후 다시 시도해 주세요.", 429)
		return true
	}
	for _, b := range a.integration.NoticeBroadcasts {
		if b.NoticeID == n.ID && now.Sub(b.Created) < 3*time.Second {
			http.Error(w, "방송을 접수했습니다. 다시 방송하려면 잠시 기다려 주세요.", 409)
			return true
		}
	}
	b := noticeBroadcast{ID: token(), NoticeID: n.ID, Revision: n.Revision, Text: text, Request: request, Created: now, Targets: targets}
	a.integration.NoticeBroadcasts = append(a.integration.NoticeBroadcasts, b)
	jsonReply(w, map[string]any{"ok": true, "receivers": len(targets), "id": b.ID})
	return true
}
func (a *App) noticeBroadcastText(r *http.Request) string {
	if phoneSpeech(r) {
		return ""
	}
	scope := speechScope(r, r.FormValue("device"))
	identity := hashToken(a.state.Board + ":" + r.FormValue("owner"))
	receiver := a.integration.NoticeReceivers[scope]
	lease := a.integration.SpeechLeases[scope]
	now := time.Now()
	if r.FormValue("broadcast_enabled") != "1" || receiver.Owner != identity || !receiver.Expires.After(now) || lease.Owner != identity || !lease.Expires.After(now) {
		return ""
	}
	for _, b := range a.integration.NoticeBroadcasts {
		if b.ID != r.FormValue("broadcast") || now.Sub(b.Created) >= 15*time.Minute || b.Targets[scope] != identity {
			continue
		}
		n := findAnnouncement(&a.state, b.NoticeID)
		if n != nil && n.Published && n.Urgent && n.Revision == b.Revision {
			return b.Text
		}
	}
	return ""
}

// Bound each generated audio segment so long notices do not exceed the player timeout.
func noticeSpeechParts(text string) []string {
	r := []rune(text)
	parts := []string{}
	for len(r) > 0 {
		end := 220
		if len(r) < end {
			end = len(r)
		}
		parts = append(parts, string(r[:end]))
		r = r[end:]
	}
	return parts
}
