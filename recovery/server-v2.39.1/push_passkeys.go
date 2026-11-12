package main

import (
	"context"
	"crypto/ecdh"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SherClockHolmes/webpush-go"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type credentialUser struct{ Data PasskeyData }

func (u credentialUser) WebAuthnID() []byte {
	b, _ := base64.RawURLEncoding.DecodeString(u.Data.UserID)
	return b
}
func (u credentialUser) WebAuthnName() string        { return "alpha-admin" }
func (u credentialUser) WebAuthnDisplayName() string { return "알파 관리자" }
func (u credentialUser) WebAuthnCredentials() []webauthn.Credential {
	out := []webauthn.Credential{}
	for _, c := range u.Data.Credentials {
		out = append(out, c.Credential)
	}
	return out
}
func verifier() (*webauthn.WebAuthn, error) {
	return webauthn.New(&webauthn.Config{RPDisplayName: "알파 관리자", RPID: "swsgood79-beep.github.io", RPOrigins: []string{pagesOrigin}, AuthenticatorSelection: protocol.AuthenticatorSelection{UserVerification: protocol.VerificationRequired, ResidentKey: protocol.ResidentKeyRequirementPreferred}})
}
func safeNext(raw string) string {
	u, e := url.Parse(raw)
	if e != nil || u.IsAbs() || u.Host != "" || strings.HasPrefix(raw, "//") {
		return "/"
	}
	if u.Path == "/inspection-requests" || strings.HasPrefix(u.Path, "/projects/") && strings.HasSuffix(u.Path, "/inspection") {
		return u.RequestURI()
	}
	return "/"
}
func (a *App) newSession(w http.ResponseWriter, r *http.Request, next string) {
	value := token()
	a.sessions[hashToken(value)] = Session{CSRF: token(), Expires: time.Now().Add(8 * time.Hour)}
	http.SetCookie(w, &http.Cookie{Name: "aw_recovered_session", Value: value, Path: "/", MaxAge: 8 * 3600, HttpOnly: true, Secure: secureRequest(r), SameSite: http.SameSiteLaxMode})
	jsonReply(w, map[string]any{"ok": true, "next": safeNext(next)})
}
func passkeyFailure(w http.ResponseWriter, code string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"code": code})
}
func (a *App) pruneFlows() {
	now := time.Now()
	for k, v := range a.integration.Starts {
		if now.After(v.Expires) {
			delete(a.integration.Starts, k)
		}
	}
	for k, v := range a.integration.Flows {
		if now.After(v.Expires) {
			delete(a.integration.Flows, k)
		}
	}
	for k, v := range a.integration.Handoffs {
		if now.After(v.Expires) {
			delete(a.integration.Handoffs, k)
		}
	}
}
func (a *App) specialHTTP(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	isAPI := strings.HasPrefix(path, "/passkeys/") || path == "/phone/register"
	if !isAPI && path != "/mobile-admin/passkey/finish" {
		return false
	}
	origin := r.Header.Get("Origin")
	if isAPI {
		if origin != pagesOrigin {
			passkeyFailure(w, "ORIGIN", 403)
			return true
		}
		w.Header().Set("Access-Control-Allow-Origin", pagesOrigin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return true
		}
	} else if !a.allowedRequest(r) {
		passkeyFailure(w, "ORIGIN", 403)
		return true
	}
	if r.Method != "POST" {
		w.WriteHeader(405)
		return true
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pruneFlows()
	if path == "/phone/register" {
		bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if bearer == "" || !equal(bearer, a.integration.Pair) || !a.integration.PairExpiry.After(time.Now()) {
			http.Error(w, "만료된 연결 QR", 403)
			return true
		}
		var sub webpush.Subscription
		if e := readJSON(r, &sub); e != nil || !validSubscription(sub) {
			http.Error(w, "유효하지 않은 알림 등록", 400)
			return true
		}
		e := a.transaction(func(s *State) error {
			s.Integration.Phone.Subscription = &sub
			s.Integration.Phone.Generation++
			s.Integration.Phone.Delivered = s.NextID
			return nil
		})
		if e != nil {
			http.Error(w, "연결 저장 실패", 500)
			return true
		}
		a.integration.Pair = ""
		a.integration.PhoneStatus = "휴대폰 연결 완료"
		jsonReply(w, map[string]any{"ok": true})
		return true
	}
	if path == "/mobile-admin/passkey/finish" {
		var req struct{ Token string }
		if readJSON(r, &req) != nil {
			passkeyFailure(w, "RESPONSE", 400)
			return true
		}
		ticket, ok := a.integration.Handoffs[hashToken(req.Token)]
		delete(a.integration.Handoffs, hashToken(req.Token))
		if !ok || !ticket.Expires.After(time.Now()) {
			passkeyFailure(w, "EXPIRED", 403)
			return true
		}
		if ticket.Owner != "" {
			owner, ok := a.sessions[ticket.Owner]
			if !ok || !owner.Expires.After(time.Now()) {
				passkeyFailure(w, "OWNER", 403)
				return true
			}
		}
		a.newSession(w, r, ticket.Next)
		return true
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || (parts[1] != "register" && parts[1] != "auth") {
		http.NotFound(w, r)
		return true
	}
	mode := parts[1]
	var req struct {
		Token, Flow string
		Credential  json.RawMessage
	}
	if readJSON(r, &req) != nil {
		passkeyFailure(w, "RESPONSE", 400)
		return true
	}
	v, e := verifier()
	if e != nil {
		passkeyFailure(w, "CONFIG", 500)
		return true
	}
	user := credentialUser{a.state.Integration.Passkeys}
	if parts[2] == "begin" {
		start, ok := a.integration.Starts[hashToken(req.Token)]
		delete(a.integration.Starts, hashToken(req.Token))
		if !ok || start.Mode != mode || !start.Expires.After(time.Now()) || start.Revision != a.state.Integration.Revision {
			passkeyFailure(w, "FLOW", 403)
			return true
		}
		if mode == "register" {
			owner, ok := a.sessions[start.Owner]
			if !ok || !owner.Expires.After(time.Now()) {
				passkeyFailure(w, "OWNER", 403)
				return true
			}
		} else if !a.state.Integration.MobileEnabled || len(user.Data.Credentials) == 0 {
			passkeyFailure(w, "CONFIG", 403)
			return true
		}
		if len(a.integration.Flows) > 128 {
			passkeyFailure(w, "FLOW", 429)
			return true
		}
		var options any
		var session *webauthn.SessionData
		if mode == "register" {
			options, session, e = v.BeginRegistration(user)
		} else {
			options, session, e = v.BeginLogin(user)
		}
		if e != nil {
			passkeyFailure(w, "PROTOCOL", 400)
			return true
		}
		flow := token()
		start.Session = session
		start.Expires = time.Now().Add(3 * time.Minute)
		a.integration.Flows[hashToken(flow)] = start
		jsonReply(w, map[string]any{"flow": flow, "options": options})
		return true
	}
	if parts[2] == "finish" {
		flow, ok := a.integration.Flows[hashToken(req.Flow)]
		delete(a.integration.Flows, hashToken(req.Flow))
		if !ok || flow.Mode != mode || flow.Revision != a.state.Integration.Revision || !flow.Expires.After(time.Now()) {
			passkeyFailure(w, "FLOW", 403)
			return true
		}
		if mode == "register" {
			owner, ok := a.sessions[flow.Owner]
			if !ok || !owner.Expires.After(time.Now()) {
				passkeyFailure(w, "OWNER", 403)
				return true
			}
		}
		var credential *webauthn.Credential
		if mode == "register" {
			parsed, err := protocol.ParseCredentialCreationResponseBytes(req.Credential)
			if err != nil {
				passkeyFailure(w, "RESPONSE", 400)
				return true
			}
			credential, e = v.CreateCredential(user, *flow.Session, parsed)
		} else {
			parsed, err := protocol.ParseCredentialRequestResponseBytes(req.Credential)
			if err != nil {
				passkeyFailure(w, "RESPONSE", 400)
				return true
			}
			credential, e = v.ValidateLogin(user, *flow.Session, parsed)
		}
		if e != nil || credential == nil || !credential.Flags.UserVerified || credential.Authenticator.CloneWarning {
			passkeyFailure(w, "SIGNATURE", 403)
			return true
		}
		e = a.transaction(func(s *State) error {
			if mode == "register" {
				if len(s.Integration.Passkeys.Credentials) >= 20 {
					return errors.New("패스키 등록 수 제한")
				}
				for _, old := range s.Integration.Passkeys.Credentials {
					if subtle.ConstantTimeCompare(old.Credential.ID, credential.ID) == 1 {
						return errors.New("이미 등록된 패스키")
					}
				}
				s.Integration.Passkeys.Credentials = append(s.Integration.Passkeys.Credentials, StoredCredential{*credential, time.Now().Format(time.RFC3339)})
			} else {
				found := false
				for i, c := range s.Integration.Passkeys.Credentials {
					if subtle.ConstantTimeCompare(c.Credential.ID, credential.ID) == 1 {
						s.Integration.Passkeys.Credentials[i].Credential = *credential
						found = true
					}
				}
				if !found {
					return errors.New("저장되지 않은 패스키")
				}
			}
			return nil
		})
		if e != nil {
			passkeyFailure(w, "CONFIG", 409)
			return true
		}
		ticket := token()
		a.integration.Handoffs[hashToken(ticket)] = Handoff{Owner: flow.Owner, Next: flow.Next, Expires: time.Now().Add(time.Minute)}
		jsonReply(w, map[string]any{"ticket": ticket})
		return true
	}
	http.NotFound(w, r)
	return true
}
func validSubscription(s webpush.Subscription) bool {
	u, e := url.Parse(s.Endpoint)
	if e != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return false
	}
	h := u.Hostname()
	allowed := h == "fcm.googleapis.com" || h == "updates.push.services.mozilla.com" || strings.HasSuffix(h, ".notify.windows.com")
	auth, e := base64.RawURLEncoding.DecodeString(s.Keys.Auth)
	if e != nil || len(auth) != 16 {
		return false
	}
	key, e := base64.RawURLEncoding.DecodeString(s.Keys.P256dh)
	if e != nil || !allowed || len(key) != 65 || key[0] != 4 {
		return false
	}
	_, e = ecdh.P256().NewPublicKey(key)
	return e == nil
}
func notification(e Event) map[string]any {
	title := stageLabel(e.Inspection)
	if e.IsNote {
		title = "비고 메시지"
	} else if e.Pending {
		if e.BatchID == e.ID {
			title = "세팅·출하 통합검사 신청"
		} else {
			title += " 신청"
		}
	} else {
		title += " 완료"
	}
	body := fmt.Sprintf("%s\n수주 %s\n%s\n작업자 %s", title, e.Order, e.Project, e.Worker)
	if e.Note != "" {
		body += "\n" + e.Note
	}
	return map[string]any{"title": "알파작업완료확인 프로그램", "body": body, "tag": "alpha-" + fmtID(e.ID), "pending_approval": e.Pending, "project_id": e.ProjectID, "request_id": e.ID}
}
func (a *App) pumpNotifications(ctx context.Context) {
	a.mu.Lock()
	phone := a.state.Integration.Phone
	epoch := a.epoch
	local := a.localURL
	if local == "" {
		local = "http://127.0.0.1:18791"
	}
	events := []Event{}
	for _, e := range a.state.Events {
		if e.ID > phone.Delivered && (e.BatchID == 0 || e.BatchID == e.ID) {
			events = append(events, e)
		}
	}
	a.mu.Unlock()
	sort.Slice(events, func(i, j int) bool { return events[i].ID < events[j].ID })
	for _, e := range events {
		payload := notification(e)
		a.mu.Lock()
		seen := a.integration.NativeSent[e.ID]
		if !seen && a.epoch == epoch {
			a.integration.NativeSent[e.ID] = true
		}
		a.mu.Unlock()
		if !seen && e.Unread {
			nativeNotify(text(payload["title"]), text(payload["body"]), local+"/projects/"+fmtID(e.ProjectID)+"/inspection")
		}
		if phone.Subscription != nil {
			b, _ := json.Marshal(payload)
			res, err := webpush.SendNotificationWithContext(ctx, b, phone.Subscription, &webpush.Options{Subscriber: "mailto:admin@example.invalid", VAPIDPublicKey: phone.PublicKey, VAPIDPrivateKey: phone.PrivateKey, TTL: 3600, HTTPClient: a.httpClient()})
			if err != nil {
				return
			}
			status := res.StatusCode
			ioClose(res)
			if status == 410 || status == 404 {
				a.mu.Lock()
				if a.epoch == epoch && a.state.Integration.Phone.Generation == phone.Generation {
					a.transaction(func(s *State) error {
						s.Integration.Phone.Subscription = nil
						s.Integration.Phone.Generation++
						return nil
					})
					a.integration.PhoneStatus = "휴대폰 등록이 만료되었습니다. 다시 연결하세요"
				}
				a.mu.Unlock()
				return
			}
			if status < 200 || status >= 300 {
				return
			}
		}
		a.mu.Lock()
		if a.epoch == epoch && a.state.Integration.Phone.Generation == phone.Generation {
			a.transaction(func(s *State) error {
				s.Integration.Phone.Delivered = max(s.Integration.Phone.Delivered, e.ID)
				return nil
			})
		}
		a.mu.Unlock()
	}
}
func ioClose(r *http.Response) {
	if r != nil && r.Body != nil {
		r.Body.Close()
	}
}
