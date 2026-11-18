package main

import (
	"net/http"
	"strings"
	"time"
)

// Native grants are independent of FCM device-token capabilities and never store a password.
type NativeAdminLogin struct {
	ID, DeviceID, Name, AccountID, SecretHash, SessionHash string
	AccountRevision, OwnerRevision                         int64
	Disabled                                               bool
}

func nativeLogin(s *State, id string) *NativeAdminLogin {
	for i := range s.NativeAdminLogins {
		if s.NativeAdminLogins[i].ID == id {
			return &s.NativeAdminLogins[i]
		}
	}
	return nil
}
func nativeLoginValid(s *State, g NativeAdminLogin) bool {
	if g.Disabled || !s.Integration.MobileEnabled {
		return false
	}
	if g.AccountID == "" {
		return s.PasswordHash != "" && g.OwnerRevision == s.Integration.Revision
	}
	u := adminAccount(s, g.AccountID)
	return u != nil && u.Enabled && u.Revision == g.AccountRevision
}
func (a *App) clearNativeGrantSessions(ids map[string]bool) {
	for key, v := range a.sessions {
		if ids[v.NativeGrantID] {
			delete(a.sessions, key)
		}
	}
}
func disableNativeLogin(s *State, g *NativeAdminLogin) {
	g.Disabled = true
	g.SecretHash = ""
	if d := nativeDevice(s, g.DeviceID); d != nil && d.AccountID == g.AccountID {
		d.Enabled = false
		d.SecretHash = ""
		d.Token = ""
		d.LastStatus = "관리자 앱 로그아웃"
	}
}
func (a *App) nativeLoginPublicHTTP(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/native-admin/session/restore" && r.URL.Path != "/native-admin/session/revoke" {
		return false
	}
	if r.Method != "POST" || a.testMode || !secureRequest(r) {
		http.Error(w, "HTTPS 앱 요청이 필요합니다.", 403)
		return true
	}
	g := nativeLogin(&a.state, r.FormValue("grant"))
	if g == nil || g.DeviceID != r.FormValue("device") || g.Disabled || !equal(g.SecretHash, hashToken(r.FormValue("secret"))) {
		w.WriteHeader(403)
		jsonReply(w, map[string]string{"code": "LOGIN_REVOKED"})
		return true
	}
	if strings.HasSuffix(r.URL.Path, "/revoke") {
		id := g.ID
		if err := a.commitAnnouncement(func(s *State) error { disableNativeLogin(s, nativeLogin(s, id)); return nil }); err != nil {
			http.Error(w, "로그아웃 저장 실패", 500)
			return true
		}
		a.clearNativeGrantSessions(map[string]bool{id: true})
		jsonReply(w, map[string]bool{"ok": true})
		return true
	}
	if !a.state.Integration.MobileEnabled {
		w.WriteHeader(403)
		jsonReply(w, map[string]string{"code": "REMOTE_DISABLED"})
		return true
	}
	if !nativeLoginValid(&a.state, *g) {
		w.WriteHeader(403)
		jsonReply(w, map[string]string{"code": "LOGIN_REVOKED"})
		return true
	}
	id := g.ID
	value := token()
	csrf := r.FormValue("csrf_hint")
	if !validPhotoID.MatchString(csrf) {
		csrf = token()
	}
	s := Session{CSRF: csrf, Expires: time.Now().Add(365 * 24 * time.Hour), AccountID: g.AccountID, AccountRevision: g.AccountRevision, NativeGrantID: id}
	if err := a.commitAnnouncement(func(st *State) error {
		v := nativeLogin(st, id)
		v.SessionHash = hashToken(s.CSRF)
		if d := nativeDevice(st, v.DeviceID); d != nil && d.AccountID == v.AccountID {
			d.SessionHash = v.SessionHash
		}
		return nil
	}); err != nil {
		http.Error(w, "앱 로그인 연결을 저장하지 못했습니다.", 500)
		return true
	}
	a.clearNativeGrantSessions(map[string]bool{id: true})
	a.sessions[hashToken(value)] = s
	http.SetCookie(w, &http.Cookie{Name: a.sessionCookie(), Value: value, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 365 * 24 * 3600})
	jsonReply(w, map[string]any{"ok": true, "csrf": s.CSRF})
	return true
}
func (a *App) nativeLoginEnroll(w http.ResponseWriter, r *http.Request, session Session) bool {
	if r.URL.Path != "/native-admin/session/enroll" {
		return false
	}
	if r.Method != "POST" || a.testMode || !secureRequest(r) {
		http.Error(w, "HTTPS 앱 요청이 필요합니다.", 403)
		return true
	}
	device := r.FormValue("device")
	if !validPhotoID.MatchString(device) {
		http.Error(w, "앱 설치 식별자 오류", 400)
		return true
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if len([]rune(name)) > 60 {
		name = string([]rune(name)[:60])
	}
	oldIDs := map[string]bool{}
	for _, g := range a.state.NativeAdminLogins {
		if g.DeviceID == device {
			oldIDs[g.ID] = true
		}
	}
	active := 0
	for _, g := range a.state.NativeAdminLogins {
		if !g.Disabled && !oldIDs[g.ID] {
			active++
		}
	}
	if active >= 20 {
		http.Error(w, "관리자 앱 로그인 연결은 최대20대입니다.", 400)
		return true
	}
	grant, secret := token(), token()
	if err := a.commitAnnouncement(func(s *State) error {
		for i := range s.NativeAdminLogins {
			g := &s.NativeAdminLogins[i]
			if oldIDs[g.ID] {
				disableNativeLogin(s, g)
			}
		}
		s.NativeAdminLogins = append(s.NativeAdminLogins, NativeAdminLogin{ID: grant, DeviceID: device, Name: name, AccountID: session.AccountID, AccountRevision: session.AccountRevision, OwnerRevision: s.Integration.Revision, SecretHash: hashToken(secret), SessionHash: hashToken(session.CSRF)})
		return nil
	}); err != nil {
		http.Error(w, "앱 로그인 연결 저장 실패", 500)
		return true
	}
	a.clearNativeGrantSessions(oldIDs)
	if cookie, err := r.Cookie(a.sessionCookie()); err == nil {
		session.NativeGrantID = grant
		session.Expires = time.Now().Add(365 * 24 * time.Hour)
		http.SetCookie(w, &http.Cookie{Name: a.sessionCookie(), Value: cookie.Value, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 365 * 24 * 3600})
		a.sessions[hashToken(cookie.Value)] = session
	}
	jsonReply(w, map[string]string{"grant": grant, "secret": secret, "cookie_name": a.sessionCookie()})
	return true
}
