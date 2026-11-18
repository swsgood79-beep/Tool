package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
)

const workerAppOrigin = "https://swsgood79-beep.github.io"

// Restrict embedding to the existing fixed worker-app host, never administrator pages.
func workerFramePath(path, board string) bool {
	p := strings.Split(strings.Trim(path, "/"), "/")
	if len(p) < 2 || p[1] != board {
		return false
	}
	switch p[0] {
	case "work", "shipping-records", "community", "client-directory", "project-info", "notices", "worker-app-entry":
		return true
	}
	return false
}
func (a *App) workerAppMarkup(s, name string, value any) string {
	root, _ := value.(map[string]any)
	if !workerAlarmPage(name, root) || a.testMode {
		return s
	}
	config := `<div id="worker-app-config" hidden data-board="` + template.HTMLEscapeString(a.state.Board) + `" data-release="` + releaseVersion + `"></div>`
	assets := `<link rel="stylesheet" href="/worker-app-v1.4.63.css"><script defer src="/worker-app-v1.4.63.js"></script>`
	s = strings.Replace(s, "<head>", `<head><script src="/worker-app-theme-bootstrap-v1.4.63.js"></script>`, 1)
	s = strings.Replace(s, "</head>", assets+"</head>", 1)
	return strings.Replace(s, "</body>", config+"</body>", 1)
}
func communityPWACookie(r *http.Request) bool {
	return (r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https") && r.Header.Get("Sec-Fetch-Dest") == "iframe"
}

func (a *App) workerAppHTTP(w http.ResponseWriter, r *http.Request) bool {
	p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(p) != 2 || (p[0] != "worker-app-entry" && p[0] != "worker-app-session") {
		return false
	}
	if !equal(p[1], a.state.Board) {
		http.NotFound(w, r)
		return true
	}
	if p[0] == "worker-app-entry" && r.Method == "GET" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<!doctype html><html lang="ko"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><script defer src="/worker-app-entry-v1.4.63.js"></script><title>알파 작업장 연결</title></head><body><div id="app-entry" hidden data-board="` + template.HTMLEscapeString(a.state.Board) + `"></div><p>알파 작업장에 연결하고 있습니다…</p></body></html>`))
		return true
	}
	if p[0] == "worker-app-session" && r.Method == "GET" {
		c, err := r.Cookie("alpha-community-owner-pwa-v1")
		reader := ""
		if rc, e := r.Cookie("alpha-talk-reader-pwa-v1"); e == nil && validPhotoID.MatchString(rc.Value) {
			reader = rc.Value
		}
		jsonReply(w, map[string]any{"ok": err == nil && validPhotoID.MatchString(c.Value), "reader": reader})
		return true
	}
	if p[0] == "worker-app-session" && r.Method == "POST" {
		owner := r.FormValue("owner")
		reader := r.FormValue("reader")
		if !equal(r.FormValue("csrf"), a.state.Board) || (!validPhotoID.MatchString(owner) || !validPhotoID.MatchString(reader)) {
			http.Error(w, "앱 연결 검증 실패", 403)
			return true
		}
		secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
		if !secure {
			http.Error(w, "앱은 HTTPS 연결이 필요합니다.", 403)
			return true
		}
		http.SetCookie(w, &http.Cookie{Name: "alpha-community-owner-pwa-v1", Value: owner, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteNoneMode, Partitioned: true, MaxAge: 31536000})
		http.SetCookie(w, &http.Cookie{Name: "alpha-talk-reader-pwa-v1", Value: reader, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteNoneMode, Partitioned: true, MaxAge: 31536000})
		if theme := r.FormValue("theme"); theme == "system" || theme == "light" || theme == "dark" {
			http.SetCookie(w, &http.Cookie{Name: "alpha-app-theme-pwa-v1", Value: theme, Path: "/", Secure: true, SameSite: http.SameSiteNoneMode, Partitioned: true, MaxAge: 31536000})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		return true
	}
	http.Error(w, "지원하지 않는 요청", 405)
	return true
}
