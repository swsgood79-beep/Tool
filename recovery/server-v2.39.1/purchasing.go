package main

import (
	"errors"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"
)

type PurchasingCall struct {
	ID                                                 int64
	Key, RecipientID, Worker, OrderNumber, ProjectName string
	ProjectID                                          int64
	CreatedAt                                          time.Time
	CompletedAt                                        *time.Time `json:",omitempty"`
	CompletedBy                                        string     `json:",omitempty"`
	RetryKeys                                          []string   `json:",omitempty"`
}

func purchasingSession(s *State, session Session) bool {
	u := adminAccount(s, session.AccountID)
	return u != nil && u.Role == "purchasing"
}
func purchasingAllowedPath(path string) bool {
	if path == "/login" || path == "/logout" || path == "/mobile-admin/logout" || path == "/purchasing" || path == "/purchasing/status" || path == "/purchasing/complete" {
		return true
	}
	for _, p := range []string{"/native-admin/bootstrap", "/native-admin/register", "/native-admin/test", "/native-admin/session/enroll", "/native-admin/session/restore", "/native-admin/session/revoke", "/native-admin/device-disable", "/native-admin/token-refresh"} {
		if path == p {
			return true
		}
	}
	return false
}
func (a *App) purchasingGate(w http.ResponseWriter, r *http.Request) bool {
	s, ok := a.session(r)
	if !ok || !purchasingSession(&a.state, s) || purchasingAllowedPath(r.URL.Path) {
		return false
	}
	if r.Method == "GET" && (r.URL.Path == "/" || r.URL.Path == "/mobile-admin") {
		http.Redirect(w, r, "/purchasing", 303)
	} else {
		http.Error(w, "구매팀 계정은 호출 업무만 이용할 수 있습니다.", 403)
	}
	return true
}
func purchasingRecipient(s *State) *AdminAccount {
	u := adminAccount(s, s.PurchasingRecipient)
	if u == nil || !u.Enabled || u.Role != "purchasing" {
		return nil
	}
	return u
}
func purchasingFind(s *State, id int64) *PurchasingCall {
	for i := range s.PurchasingCalls {
		if s.PurchasingCalls[i].ID == id {
			return &s.PurchasingCalls[i]
		}
	}
	return nil
}
func (a *App) createPurchasingCall(project int64, worker, key string) (*PurchasingCall, bool, error) {
	if !validText(worker, 40) || !validPhotoID.MatchString(key) {
		return nil, false, errors.New("작업자 이름을 40자 이내로 입력해 주세요.")
	}
	for i := range a.state.PurchasingCalls {
		c := &a.state.PurchasingCalls[i]
		used := equal(c.Key, key)
		for _, alias := range c.RetryKeys {
			used = used || equal(alias, key)
		}
		if used {
			if c.ProjectID != project || c.Worker != worker {
				return nil, false, errors.New("호출 정보를 다시 확인해 주세요.")
			}
			return c, true, nil
		}
	}
	recipient := purchasingRecipient(&a.state)
	if recipient == nil {
		return nil, false, errors.New("구매 담당자가 지정되지 않았습니다. 관리자에게 문의해 주세요.")
	}
	p := a.state.project(project)
	if p == nil || p.Hidden {
		return nil, false, errors.New("현재 작업 수주를 찾을 수 없습니다.")
	}
	pending := 0
	for i := range a.state.PurchasingCalls {
		c := &a.state.PurchasingCalls[i]
		if c.CompletedAt != nil {
			continue
		}
		pending++
		if c.ProjectID == project && c.Worker == worker && c.RecipientID == recipient.ID {
			if len(c.RetryKeys) >= 32 {
				return nil, false, errors.New("이미 접수된 호출이 있습니다. 구매 담당자의 처리를 기다려 주세요.")
			}
			id := c.ID
			if err := a.commitAnnouncement(func(s *State) error { v := purchasingFind(s, id); v.RetryKeys = append(v.RetryKeys, key); return nil }); err != nil {
				return nil, false, err
			}
			return purchasingFind(&a.state, id), true, nil
		}
	}
	if pending >= 500 || len(a.state.PurchasingCalls) >= 100000 {
		return nil, false, errors.New("호출 기록이 많습니다. 관리자에게 문의해 주세요.")
	}
	value := PurchasingCall{Key: key, RecipientID: recipient.ID, Worker: worker, ProjectID: p.ID, OrderNumber: p.Number, ProjectName: p.Name, CreatedAt: time.Now()}
	err := a.commitAnnouncement(func(s *State) error {
		value.ID = s.next()
		s.PurchasingCalls = append(s.PurchasingCalls, value)
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return purchasingFind(&a.state, value.ID), false, nil
}
func purchasingList(s *State, account string, done bool) []PurchasingCall {
	out := []PurchasingCall{}
	for _, c := range s.PurchasingCalls {
		if c.RecipientID == account && (c.CompletedAt != nil) == done {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if done {
			return out[i].ID > out[j].ID
		}
		return out[i].ID < out[j].ID
	})
	return out
}
func (a *App) completePurchasingCall(account string, id int64) (int64, error) {
	c := purchasingFind(&a.state, id)
	if c == nil || c.RecipientID != account {
		return 0, errors.New("본인에게 접수된 호출만 완료할 수 있습니다.")
	}
	if c.CompletedAt == nil {
		err := a.commitAnnouncement(func(s *State) error {
			v := purchasingFind(s, id)
			now := time.Now()
			v.CompletedAt = &now
			v.CompletedBy = account
			return nil
		})
		if err != nil {
			return 0, err
		}
	}
	pending := purchasingList(&a.state, account, false)
	if len(pending) > 0 {
		return pending[0].ID, nil
	}
	return 0, nil
}
func (a *App) purchasingHTTP(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	if strings.HasPrefix(path, "/purchasing-receipts/") {
		if path != "/purchasing-receipts/"+a.state.Board {
			http.NotFound(w, r)
			return true
		}
		if r.Method != "POST" {
			http.Error(w, "POST 요청이 필요합니다.", 405)
			return true
		}
		r.Body = http.MaxBytesReader(w, r.Body, 32768)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "요청이 너무 큽니다.", 400)
			return true
		}
		if !equal(r.FormValue("csrf"), a.state.Board) {
			http.Error(w, "요청 검증 실패", 403)
			return true
		}
		keys := r.Form["key"]
		if len(keys) > 500 {
			http.Error(w, "요청이 너무 많습니다.", 400)
			return true
		}
		for _, key := range keys {
			if !validPhotoID.MatchString(key) {
				http.Error(w, "요청 검증 실패", 400)
				return true
			}
		}
		ownedKeys := make(map[string]bool, len(keys))
		for _, key := range keys {
			ownedKeys[key] = true
		}
		now := time.Now()
		rows := []map[string]any{}
		for _, c := range a.state.PurchasingCalls {
			if c.CompletedAt != nil && now.Sub(*c.CompletedAt) >= time.Minute {
				continue
			}
			owned := ownedKeys[c.Key]
			for _, alias := range c.RetryKeys {
				owned = owned || ownedKeys[alias]
			}
			if owned {
				rows = append(rows, map[string]any{"id": c.ID, "project": c.ProjectID, "worker": c.Worker, "created": c.CreatedAt, "completed": c.CompletedAt})
			}
		}
		w.Header().Set("Cache-Control", "no-store")
		jsonReply(w, map[string]any{"ok": true, "now": now, "calls": rows})
		return true
	}
	if strings.HasPrefix(path, "/purchasing-call/") {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) != 3 || !equal(parts[1], a.state.Board) {
			http.NotFound(w, r)
			return true
		}
		if r.Method != "POST" {
			http.Error(w, "POST 요청이 필요합니다.", 405)
			return true
		}
		if !equal(r.FormValue("csrf"), a.state.Board) {
			http.Error(w, "요청 검증 실패", 403)
			return true
		}
		call, duplicate, err := a.createPurchasingCall(number(parts[2]), strings.TrimSpace(r.FormValue("worker")), r.FormValue("key"))
		if err != nil {
			http.Error(w, err.Error(), 400)
			return true
		}
		jsonReply(w, map[string]any{"ok": true, "duplicate": duplicate, "id": call.ID})
		return true
	}
	if path == "/purchasing-settings" {
		return a.purchasingSettingsHTTP(w, r)
	}
	if path != "/purchasing" && path != "/purchasing/status" && path != "/purchasing/complete" {
		return false
	}
	session, ok := a.admin(w, r)
	if !ok {
		return true
	}
	if !purchasingSession(&a.state, session) {
		http.Error(w, "구매팀 계정으로 로그인해 주세요.", 403)
		return true
	}
	if path == "/purchasing/complete" {
		if r.Method != "POST" {
			http.Error(w, "POST 요청이 필요합니다.", 405)
			return true
		}
		next, err := a.completePurchasingCall(session.AccountID, number(r.FormValue("id")))
		if err != nil {
			http.Error(w, err.Error(), 400)
			return true
		}
		jsonReply(w, map[string]any{"ok": true, "next": next})
		return true
	}
	if r.Method != "GET" {
		http.Error(w, "GET 요청이 필요합니다.", 405)
		return true
	}
	if path == "/purchasing/status" {
		done := r.URL.Query().Get("done") == "1"
		list := purchasingList(&a.state, session.AccountID, done)
		page := int(number(r.URL.Query().Get("page")))
		if page < 1 {
			page = 1
		}
		total := len(list)
		page = min(page, max(1, (total+19)/20))
		start := min((page-1)*20, total)
		end := min(start+20, total)
		selected := purchasingFind(&a.state, number(r.URL.Query().Get("call")))
		if r.URL.Query().Get("call") != "" && (selected == nil || selected.RecipientID != session.AccountID) {
			http.NotFound(w, r)
			return true
		}
		pending := purchasingList(&a.state, session.AccountID, false)
		position := 0
		var next *PurchasingCall
		if selected != nil {
			for i, c := range pending {
				if c.ID == selected.ID {
					position = i + 1
					break
				}
			}
			for i := range pending {
				if pending[i].ID != selected.ID {
					next = &pending[i]
					break
				}
			}
		}
		jsonReply(w, map[string]any{"selected": purchasingCallView(selected), "position": position, "next": purchasingCallView(next), "calls": purchasingCallViews(list[start:end]), "pending_count": len(purchasingList(&a.state, session.AccountID, false)), "done_count": len(purchasingList(&a.state, session.AccountID, true)), "page": page, "pages": max(1, (total+19)/20)})
		return true
	}
	cfg := map[string]any{"CSRF": session.CSRF, "Name": nativeAccountLabel(&a.state, session.AccountID), "Username": adminAccount(&a.state, session.AccountID).Username, "NativeApp": nativeAdminAgent(r)}
	a.renderTemplate(w, templates, "purchasing.html", cfg)
	return true
}
func (a *App) purchasingWorkerMarkup(s, name string) string {
	if name != "work.html" || purchasingRecipient(&a.state) == nil {
		return s
	}
	u := purchasingRecipient(&a.state)
	marker := `<div id="purchasing-worker-config" hidden data-board="` + template.HTMLEscapeString(a.state.Board) + `" data-recipient="` + template.HTMLEscapeString(u.Name) + `"></div>`
	s = strings.Replace(s, "</head>", `<link rel="stylesheet" href="/purchasing-worker-v1.4.80.css"><script defer src="/purchasing-worker-v1.4.80.js"></script></head>`, 1)
	return strings.Replace(s, "</body>", marker+"</body>", 1)
}

type PurchasingCallView struct {
	ID                               int64
	Worker, OrderNumber, ProjectName string
	CreatedAt                        time.Time
	CompletedAt                      *time.Time
}

func purchasingCallView(c *PurchasingCall) *PurchasingCallView {
	if c == nil {
		return nil
	}
	return &PurchasingCallView{c.ID, c.Worker, c.OrderNumber, c.ProjectName, c.CreatedAt, c.CompletedAt}
}
func purchasingCallViews(list []PurchasingCall) []PurchasingCallView {
	out := []PurchasingCallView{}
	for _, c := range list {
		out = append(out, *purchasingCallView(&c))
	}
	return out
}
