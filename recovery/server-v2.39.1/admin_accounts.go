package main

import (
	"errors"
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

type AdminAccount struct {
	ID, Username, Name, PasswordHash string
	Enabled                          bool
	Revision                         int64
}
type AdminAccountView struct {
	ID, Username, Name string
	Enabled            bool
	Revision           int64
}

var accountUsername = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{2,31}$`)
var accountDummyHash = func() []byte {
	h, e := bcrypt.GenerateFromPassword([]byte(token()), 12)
	if e != nil {
		panic(e)
	}
	return h
}()

func adminAccount(s *State, id string) *AdminAccount {
	for i := range s.AdminAccounts {
		if s.AdminAccounts[i].ID == id {
			return &s.AdminAccounts[i]
		}
	}
	return nil
}
func adminAccountByUsername(s *State, username string) *AdminAccount {
	for i := range s.AdminAccounts {
		if s.AdminAccounts[i].Username == username {
			return &s.AdminAccounts[i]
		}
	}
	return nil
}
func validAccountSession(s *State, session Session) bool {
	if !session.Expires.After(time.Now()) {
		return false
	}
	if session.AccountID == "" {
		return true
	}
	u := adminAccount(s, session.AccountID)
	return u != nil && u.Enabled && u.Revision == session.AccountRevision
}
func ownerOnlyAdminPath(path string) bool {
	for _, prefix := range []string{"/admin-accounts", "/mobile-access", "/mobile-admin/passkey", "/settings", "/connection", "/backup", "/phone", "/data-storage", "/schedule", "/market-settings"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}
func accountPassword(password string) error {
	if utf8.RuneCountInString(password) < 12 {
		return errors.New("비밀번호는 12자 이상 입력해 주세요.")
	}
	if len(password) > 72 {
		return errors.New("비밀번호가 너무 깁니다. 조금 짧게 입력해 주세요.")
	}
	return nil
}
func (a *App) revokeAccountSessions(id string) {
	for key, s := range a.sessions {
		if s.AccountID == id {
			delete(a.sessions, key)
		}
	}
}
func (a *App) issueAccountSession(w http.ResponseWriter, r *http.Request, u *AdminAccount) {
	for key, existing := range a.sessions {
		if !validAccountSession(&a.state, existing) {
			delete(a.sessions, key)
		}
	}
	value := token()
	a.sessions[hashToken(value)] = Session{CSRF: token(), Expires: time.Now().Add(8 * time.Hour), AccountID: u.ID, AccountRevision: u.Revision}
	http.SetCookie(w, &http.Cookie{Name: a.sessionCookie(), Value: value, Path: "/", HttpOnly: true, Secure: secureRequest(r), SameSite: http.SameSiteStrictMode, MaxAge: 8 * 3600})
}
func (a *App) manageAdminAccount(r *http.Request) error {
	action := r.FormValue("action")
	if action == "create" {
		name, username := strings.TrimSpace(r.FormValue("name")), strings.ToLower(strings.TrimSpace(r.FormValue("username")))
		if !validText(name, 40) {
			return errors.New("이름을 40자 이내로 입력해 주세요.")
		}
		if !accountUsername.MatchString(username) || username == "admin" || username == "administrator" || username == "owner" || username == "root" {
			return errors.New("아이디는 영문 소문자·숫자와 . _ - 로 3~32자 입력해 주세요. 관리자 예약 아이디는 사용할 수 없습니다.")
		}
		if adminAccountByUsername(&a.state, username) != nil {
			return errors.New("이미 사용 중인 아이디입니다.")
		}
		pw := r.FormValue("password")
		if err := accountPassword(pw); err != nil {
			return err
		}
		if pw != r.FormValue("confirm") {
			return errors.New("비밀번호 확인이 다릅니다.")
		}
		if len(a.state.AdminAccounts) >= 50 {
			return errors.New("등록 가능한 계정 수를 초과했습니다.")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
		if err != nil {
			return err
		}
		return a.commitAnnouncement(func(s *State) error {
			s.AdminAccounts = append(s.AdminAccounts, AdminAccount{ID: token(), Name: name, Username: username, PasswordHash: string(hash), Enabled: true, Revision: 1})
			return nil
		})
	}
	u := adminAccount(&a.state, r.FormValue("id"))
	if u == nil {
		return errors.New("계정을 찾을 수 없습니다.")
	}
	if u.Revision != number(r.FormValue("revision")) {
		return errors.New("계정 정보가 변경되었습니다. 화면을 다시 확인해 주세요.")
	}
	id := u.ID
	hash := ""
	if action == "reset" {
		pw := r.FormValue("password")
		if err := accountPassword(pw); err != nil {
			return err
		}
		if pw != r.FormValue("confirm") {
			return errors.New("비밀번호 확인이 다릅니다.")
		}
		b, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
		if err != nil {
			return err
		}
		hash = string(b)
	} else if action != "disable" && action != "enable" {
		return errors.New("계정 관리 요청을 확인해 주세요.")
	}
	err := a.commitAnnouncement(func(s *State) error {
		u := adminAccount(s, id)
		if action == "reset" {
			u.PasswordHash = hash
		} else {
			u.Enabled = action == "enable"
		}
		u.Revision++
		return nil
	})
	if err == nil {
		a.revokeAccountSessions(id)
	}
	return err
}
func (a *App) accountHTTP(w http.ResponseWriter, r *http.Request, session Session) bool {
	path := r.URL.Path
	if path != "/admin-accounts" && path != "/my-password" {
		return false
	}
	if path == "/admin-accounts" && session.AccountID != "" {
		http.Error(w, "계정 관리는 기존 관리자만 사용할 수 있습니다.", 403)
		return true
	}
	message := ""
	problem := ""
	if r.Method == "POST" {
		var err error
		if path == "/admin-accounts" {
			err = a.manageAdminAccount(r)
			message = "계정 정보를 저장했습니다."
		} else {
			u := adminAccount(&a.state, session.AccountID)
			if u == nil {
				err = errors.New("기존 관리자 비밀번호는 ‘휴대폰 관리자 접속’ 설정에서 변경해 주세요.")
			} else if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(r.FormValue("current_password"))) != nil {
				err = errors.New("현재 비밀번호가 맞지 않습니다.")
			} else if err = accountPassword(r.FormValue("password")); err == nil {
				if r.FormValue("password") != r.FormValue("confirm") {
					err = errors.New("비밀번호 확인이 다릅니다.")
				} else {
					var hash []byte
					hash, err = bcrypt.GenerateFromPassword([]byte(r.FormValue("password")), 12)
					if err == nil {
						id := u.ID
						err = a.commitAnnouncement(func(s *State) error {
							u := adminAccount(s, id)
							u.PasswordHash = string(hash)
							u.Revision++
							return nil
						})
						if err == nil {
							a.revokeAccountSessions(id)
							a.issueAccountSession(w, r, adminAccount(&a.state, id))
							http.Redirect(w, r, "/my-password?changed=1", 303)
							return true
						}
					}
				}
			}
		}
		if err != nil {
			problem = err.Error()
		} else {
			http.Redirect(w, r, path+"?saved=1", 303)
			return true
		}
	}
	if r.URL.Query().Get("saved") == "1" {
		message = "계정 정보를 저장했습니다."
	}
	if r.URL.Query().Get("changed") == "1" {
		message = "내 비밀번호를 변경했습니다. 이 계정의 다른 기기는 다시 로그인해야 합니다."
	}
	v := a.view(session.CSRF)
	v["AccountError"], v["AccountMessage"], v["PasswordOnly"], v["AccountOwner"] = problem, message, path == "/my-password", session.AccountID == ""
	rows := []AdminAccountView{}
	for _, u := range a.state.AdminAccounts {
		rows = append(rows, AdminAccountView{u.ID, u.Username, u.Name, u.Enabled, u.Revision})
	}
	v["Accounts"] = rows
	v["AccountCount"] = len(rows) + 1
	if u := adminAccount(&a.state, session.AccountID); u != nil {
		v["AccountName"], v["AccountUsername"] = u.Name, u.Username
	}
	if !localRequest(r) || session.AccountID != "" {
		v["MobileAdmin"] = true
	}
	a.render(w, "admin-accounts.html", v)
	return true
}
func (a *App) accountMarkup(s, name string, value any) string {
	root, ok := value.(map[string]any)
	if !ok || !strings.Contains(s, "</body>") {
		return s
	}
	csrf, _ := root["CSRF"].(string)
	if csrf == "" || equal(csrf, a.state.Board) {
		return s
	}
	var session Session
	found := false
	for _, candidate := range a.sessions {
		if equal(candidate.CSRF, csrf) && validAccountSession(&a.state, candidate) {
			session = candidate
			found = true
			break
		}
	}
	if !found {
		return s
	}
	label, username, owner := "기존 관리자", "", true
	if u := adminAccount(&a.state, session.AccountID); u != nil {
		label, username, owner = u.Name, u.Username, false
	}
	marker := `<div id="admin-account-config" hidden data-owner="` + map[bool]string{true: "true", false: "false"}[owner] + `" data-label="` + template.HTMLEscapeString(label) + `" data-username="` + template.HTMLEscapeString(username) + `"></div>`
	s = strings.Replace(s, "</head>", `<link rel="stylesheet" href="/admin-accounts-v1.4.54.css"></head>`, 1)
	return strings.Replace(s, "</body>", marker+"</body>", 1)
}
