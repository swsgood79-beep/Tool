package main

import (
	"net/http"
	"strings"
)

func (a *App) beltHTTP(w http.ResponseWriter, r *http.Request) bool {
	p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(p) == 0 || p[0] != "belt-tension" {
		return false
	}
	if r.Method != "GET" {
		http.Error(w, "GET 요청이 필요합니다", 405)
		return true
	}
	admin := len(p) == 1
	csrf, back := a.state.Board, "/work/"+a.state.Board+"?entry=skip&design-tab=menu"
	if admin {
		s, ok := a.admin(w, r)
		if !ok {
			return true
		}
		csrf = s.CSRF
		back = "/?design-tab=menu"
		if nativeAdminAgent(r) || s.NativeGrantID != "" {
			back = "/mobile-admin?design-tab=menu"
		}
	} else if len(p) != 2 || !equal(p[1], a.state.Board) {
		http.NotFound(w, r)
		return true
	}
	a.render(w, "belt-tension.html", map[string]any{"Back": back, "WorkerURL": "/work/" + a.state.Board, "Board": a.state.Board, "CSRF": csrf, "ToolWorker": !admin})
	return true
}
