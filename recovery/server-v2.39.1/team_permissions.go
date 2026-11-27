package main

import (
	"html/template"
	"net/http"
	"strings"
)

type AdminMenus struct{ Purchasing, Production, Machining bool }

func managerAccount(s *State, session Session) bool {
	if session.AccountID == "" {
		return true
	}
	u := adminAccount(s, session.AccountID)
	return u != nil && (u.Role == "" || u.Role == "admin")
}
func adminMenus(s *State, session Session) AdminMenus {
	if !managerAccount(s, session) {
		return AdminMenus{}
	}
	if session.AccountID == "" {
		return AdminMenus{true, true, true}
	}
	u := adminAccount(s, session.AccountID)
	if u.Menus != nil {
		return *u.Menus
	}
	// Preserve existing access without migrating or rewriting an account.
	return AdminMenus{Production: true, Machining: true}
}
func (a *App) teamPermissionCheck(w http.ResponseWriter, r *http.Request, s Session) bool {
	if !managerAccount(&a.state, s) {
		return true
	}
	p := r.URL.Path
	m := adminMenus(&a.state, s)
	if strings.HasPrefix(p, "/machining") {
		if m.Machining {
			return true
		}
	} else if strings.HasPrefix(p, "/purchasing-admin") {
		if m.Purchasing {
			return true
		}
	} else {
		for _, prefix := range []string{"/team-menu", "/team-alerts", "/native-admin/"} {
			if strings.HasPrefix(p, prefix) {
				return true
			}
		}
		for _, path := range []string{"/my-password", "/logout", "/mobile-admin/logout"} {
			if p == path {
				return true
			}
		}
		if m.Production {
			return true
		}
		if r.Method == "GET" && (p == "/" || p == "/mobile-admin") {
			http.Redirect(w, r, "/team-menu", 303)
			return false
		}
	}
	http.Error(w, "이 메뉴의 접근 권한이 없습니다. 대표 관리자에게 확인해 주세요", 403)
	return false
}
func (a *App) teamHTTP(w http.ResponseWriter, r *http.Request, s Session) bool {
	if r.URL.Path != "/team-menu" && r.URL.Path != "/purchasing-admin" {
		return false
	}
	if !managerAccount(&a.state, s) {
		http.Error(w, "관리자 계정이 필요합니다", 403)
		return true
	}
	if r.Method != "GET" {
		http.Error(w, "GET 요청이 필요합니다", 405)
		return true
	}
	m := adminMenus(&a.state, s)
	if r.URL.Path == "/purchasing-admin" && !m.Purchasing {
		http.Error(w, "구매팀 메뉴 권한이 없습니다", 403)
		return true
	}
	v := map[string]any{"CSRF": s.CSRF, "Menus": m, "Name": nativeAccountLabel(&a.state, s.AccountID), "Purchasing": r.URL.Path == "/purchasing-admin", "Calls": a.state.PurchasingCalls}
	a.renderTemplate(w, templates, "team-menu.html", v)
	return true
}
func (a *App) teamMarkup(html string, session Session) string {
	if !managerAccount(&a.state, session) {
		return html
	}
	m := adminMenus(&a.state, session)
	cfg := `<div id="team-admin-config" hidden data-csrf="` + template.HTMLEscapeString(session.CSRF) + `" data-account="` + template.HTMLEscapeString(session.AccountID) + `" data-purchasing="` + fmtBool(m.Purchasing) + `" data-production="` + fmtBool(m.Production) + `" data-machining="` + fmtBool(m.Machining) + `"></div><script defer src="/team-admin-v1.4.115.js"></script>`
	html = strings.Replace(html, "</head>", `<link rel="stylesheet" href="/team-admin-v1.4.115.css"></head>`, 1)
	return strings.Replace(html, "</body>", cfg+"</body>", 1)
}
func fmtBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
