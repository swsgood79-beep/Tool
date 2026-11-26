package main

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/base64"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

func passwordHash(password, salt string) (string, error) {
	b, e := pbkdf2.Key(sha256.New, password, []byte(salt), 600000, 32)
	return base64.RawURLEncoding.EncodeToString(b), e
}
func (a *App) session(r *http.Request) (Session, bool) {
	c, e := r.Cookie(a.sessionCookie())
	if e != nil {
		return Session{}, false
	}
	s, ok := a.sessions[hashToken(c.Value)]
	return s, ok && validAccountSession(&a.state, s)
}
func (a *App) admin(w http.ResponseWriter, r *http.Request) (Session, bool) {
	s, ok := a.session(r)
	if !localRequest(r) && !a.state.Integration.MobileEnabled {
		http.Error(w, "휴대폰 관리자 접속이 꺼져 있습니다", 403)
		return s, false
	}
	if !ok {
		if r.Method == "GET" {
			http.Redirect(w, r, "/login?next="+url.QueryEscape(safeNext(r.URL.RequestURI())), http.StatusSeeOther)
		} else {
			http.Error(w, "관리자 로그인이 필요합니다.", 401)
		}
		return s, false
	}
	if machiningSession(&a.state, s) && !machiningAllowedPath(r.URL.Path) {
		http.Error(w, "가공팀 계정은 가공 업무만 이용할 수 있습니다", 403)
		return s, false
	}
	if purchasingSession(&a.state, s) && !purchasingAllowedPath(r.URL.Path) {
		http.Error(w, "구매팀 계정은 호출 업무만 이용할 수 있습니다.", 403)
		return s, false
	}
	if s.AccountID != "" && ownerOnlyAdminPath(r.URL.Path) {
		http.Error(w, "이 설정은 대표 관리자 계정에서 관리합니다.", 403)
		return s, false
	}
	if r.Method == "POST" && !equal(r.FormValue("csrf"), s.CSRF) {
		http.Error(w, "관리자 요청 검증 실패", 403)
		return s, false
	}
	return s, true
}
func localRequest(r *http.Request) bool {
	host := r.Host
	if h, _, e := net.SplitHostPort(host); e == nil {
		host = h
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, e := url.Parse(origin)
		if e != nil || !strings.EqualFold(u.Host, r.Host) || (u.Scheme != "http" && u.Scheme != "https") {
			return false
		}
	}
	return true
}

const ownerUsername = "wssong"
const loginPage = `<!doctype html><html lang="ko"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><link rel="stylesheet" href="/style.css"><link rel="stylesheet" href="/admin-accounts-v1.4.54.css"><title>관리자 로그인 · 알파</title></head><body class="account-login-page"><header><span class="brand"><span class="mark">α</span> 알파 <small>관리자</small></span></header><main><section class="card account-login-card"><h1>{{if .Setup}}관리자 비밀번호 만들기{{else}}관리자 로그인{{end}}</h1><p class="muted">{{if .Setup}}관리자 아이디 wssong의 비밀번호를 설정해 주세요.{{else}}아이디와 비밀번호를 입력해 주세요.{{end}}</p>{{if .Error}}<p class="account-error" role="alert">{{.Error}}</p>{{end}}<form method="post" action="/login"><input type="hidden" name="csrf" value="{{.CSRF}}"><input type="hidden" name="login_mode" value="account"><label>아이디<input name="username" value="{{.Username}}" {{if .Setup}}readonly{{end}} required maxlength="32" autocomplete="username" autocapitalize="none" spellcheck="false" placeholder="아이디"></label><label>비밀번호<input type="password" name="password" minlength="{{if .Setup}}12{{else}}8{{end}}" maxlength="72" required autocomplete="{{if .Setup}}new-password{{else}}current-password{{end}}"></label>{{if .Setup}}<label>비밀번호 확인<input type="password" name="confirm" minlength="12" maxlength="72" required autocomplete="new-password"></label>{{end}}<button class="primary full">{{if .Setup}}비밀번호 설정{{else}}로그인{{end}}</button></form></section></main></body></html>`

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	setup := a.state.PasswordHash == ""

	username := ""
	if setup {
		username = ownerUsername
	}
	var account *AdminAccount
	named := false
	message := ""
	if r.Method == "POST" {
		c, e := r.Cookie(a.nonceCookie())
		nonce := r.FormValue("csrf")
		ticket, ok := a.challenges[nonce]
		delete(a.challenges, nonce)
		if e != nil || !ok || !ticket.Expires.After(time.Now()) || !equal(c.Value, nonce) {
			http.Error(w, "로그인 화면을 다시 열어 주세요.", 403)
			return
		}
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		now := time.Now()
		attempts := []time.Time{}
		for _, t := range a.failures[ip] {
			if now.Sub(t) < time.Minute {
				attempts = append(attempts, t)
			}
		}
		a.failures[ip] = attempts
		if len(attempts) >= 5 {
			http.Error(w, "로그인 시도가 많습니다. 1분 후 다시 시도하세요.", 429)
			return
		}
		username = strings.ToLower(strings.TrimSpace(r.FormValue("username")))
		password := r.FormValue("password")
		named = !setup && (r.FormValue("login_mode") == "account" || username != "") && username != ownerUsername
		// Preserve a pre-existing personal account with the reserved username,
		// without granting it owner permissions or modifying either password.
		if !setup && username == ownerUsername {
			if collision := adminAccountByUsername(&a.state, username); collision != nil {
				ownerValid := a.passwordMatches(password)
				if ownerValid && collision.Enabled && bcrypt.CompareHashAndPassword([]byte(collision.PasswordHash), []byte(password)) == nil {
					http.Error(w, "wssong 아이디가 개인 계정과 중복되어 로그인할 수 없습니다. PC에서 계정 구성을 확인해 주세요.", http.StatusConflict)
					return
				}
				named = !ownerValid
			}
		}
		minimum := 8
		if setup {
			minimum = 12
		}
		if utf8.RuneCountInString(password) < minimum || utf8.RuneCountInString(password) > 72 {
			if !setup {
				message = "비밀번호는 8~72자여야 합니다."
			} else {
				message = "비밀번호는 12~72자여야 합니다."
			}
		} else if setup {
			if password != r.FormValue("confirm") {
				message = "비밀번호 확인이 다릅니다."
			} else {
				salt := token()
				bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
				hash := string(bytes)
				if err == nil {
					err = a.transaction(func(s *State) error { s.Salt = salt; s.PasswordHash = hash; return nil })
				}
				if err != nil {
					http.Error(w, "비밀번호 저장 실패", 500)
					return
				}
			}
		} else if named {
			account = adminAccountByUsername(&a.state, username)
			hash := accountDummyHash
			if account != nil && account.Enabled {
				hash = []byte(account.PasswordHash)
			}
			valid := bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
			if !valid || account == nil || !account.Enabled {
				message = "아이디 또는 비밀번호를 확인해 주세요."
				a.failures[ip] = append(attempts, now)
			}
		} else {
			if !a.passwordMatches(password) {
				message = "아이디 또는 비밀번호를 확인해 주세요."
				a.failures[ip] = append(attempts, now)
			}
		}
		if message == "" && !named && !strings.HasPrefix(a.state.PasswordHash, "$2") {
			hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
			if err != nil {
				http.Error(w, "원본 호환 비밀번호는 72바이트 이하여야 합니다", 400)
				return
			}
			if err = a.transaction(func(s *State) error { s.PasswordHash = string(hash); s.Salt = ""; return nil }); err != nil {
				http.Error(w, "비밀번호 호환 저장 실패", 500)
				return
			}
		}
		if message == "" {
			delete(a.failures, ip)
			if named {
				a.issueAccountSession(w, r, account)
				if account.Role == "machining" {
					http.Redirect(w, r, "/machining", 303)
					return
				}
				if account.Role == "purchasing" {
					http.Redirect(w, r, "/purchasing", 303)
					return
				}
				http.Redirect(w, r, safeNext(ticket.Next), 303)
				return
			}
			value := token()
			csrf := token()
			for k, s := range a.sessions {
				if !s.Expires.After(now) {
					delete(a.sessions, k)
				}
			}
			a.sessions[hashToken(value)] = Session{CSRF: csrf, Expires: now.Add(8 * time.Hour)}
			http.SetCookie(w, &http.Cookie{Name: a.sessionCookie(), Value: value, Path: "/", HttpOnly: true, Secure: secureRequest(r), SameSite: http.SameSiteStrictMode, MaxAge: 8 * 3600})
			http.Redirect(w, r, safeNext(ticket.Next), 303)
			return
		}
	}
	if setup {
		username = ownerUsername
	}
	nonce := token()
	for k, t := range a.challenges {
		if !t.Expires.After(time.Now()) {
			delete(a.challenges, k)
		}
	}
	if len(a.challenges) >= 256 {
		http.Error(w, "잠시 후 다시 시도하세요.", 429)
		return
	}
	a.challenges[nonce] = Ticket{Expires: time.Now().Add(10 * time.Minute), Next: safeNext(r.URL.Query().Get("next"))}
	http.SetCookie(w, &http.Cookie{Name: a.nonceCookie(), Value: nonce, Path: "/login", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 600})
	t := template.Must(template.New("login").Parse(loginPage))
	a.renderTemplate(w, t, "login", map[string]any{"Setup": setup, "Error": message, "CSRF": nonce, "Username": username, "Next": safeNext(r.URL.Query().Get("next"))})
}

func (a *App) passwordMatches(pw string) bool {
	if strings.HasPrefix(a.state.PasswordHash, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(a.state.PasswordHash), []byte(pw)) == nil
	}
	hash, e := passwordHash(pw, a.state.Salt)
	return e == nil && equal(hash, a.state.PasswordHash)
}
