package main

import (
	"html/template"
	"net/http"
	"strings"
)

// Presentation is selected only for an authenticated APK session. A user agent
// selects layout, never permissions; every existing admin action still authenticates.
func nativeAdminAgent(r *http.Request) bool {
	return strings.Contains(r.UserAgent(), "AlphaAdminAndroid/")
}
func nativeAdminWorkerPath(path string) bool {
	for _, p := range []string{"/work/", "/community/", "/notices/", "/client-directory/", "/shipping-records/", "/project-info/"} {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
func (a *App) nativeAdminPresentation(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, bool) {
	session, ok := a.session(r)
	app := nativeAdminAgent(r) || (ok && session.NativeGrantID != "")
	if !app {
		return w, false
	}
	if nativeAdminWorkerPath(r.URL.Path) && !ok {
		// Native administrator navigation must not silently become an anonymous worker.
		if r.Method == "GET" {
			http.Redirect(w, r, "/login?next="+template.URLQueryEscaper(safeNext(r.URL.RequestURI())), 303)
		} else {
			http.Error(w, "관리자 로그인이 필요합니다.", 401)
		}
		return w, true
	}
	if !ok {
		return w, false
	}
	if r.Method == "GET" && r.URL.Path == "/work/"+a.state.Board {
		q := r.URL.Query()
		tab := q.Get("design-tab")
		if tab != "calendar" && tab != "menu" {
			tab = "orders"
		}
		q.Set("design-tab", tab)
		q.Del("entry")
		http.Redirect(w, r, "/mobile-admin?"+q.Encode(), 303)
		return w, true
	}
	reader := ""
	if g := nativeLogin(&a.state, session.NativeGrantID); g != nil {
		reader = hashToken("native-talk-reader:" + g.DeviceID + ":" + g.AccountID)
	}
	return &nativeAdminPageWriter{ResponseWriter: w, session: session, label: nativeAccountLabel(&a.state, session.AccountID), board: a.state.Board, reader: reader, tab: r.URL.Query().Get("design-tab"), controls: nativeAdminAgent(r)}, false
}

type nativeAdminPageWriter struct {
	http.ResponseWriter
	session                   Session
	label, board, reader, tab string
	controls                  bool
}

func (w *nativeAdminPageWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *nativeAdminPageWriter) Write(data []byte) (int, error) {
	if !strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		return w.ResponseWriter.Write(data)
	}
	s := string(data)
	if strings.Contains(s, `id="alpha-design"`) && strings.Contains(s, "</body>") && strings.Contains(s, "</head>") {
		esc := template.HTMLEscapeString
		marker := `<div id="alpha-admin-app" hidden data-csrf="` + esc(w.session.CSRF) + `" data-owner="` + map[bool]string{true: "true", false: "false"}[w.session.AccountID == ""] + `" data-label="` + esc(w.label) + `" data-board="` + esc(w.board) + `" data-reader="` + esc(w.reader) + `" data-native-controls="` + map[bool]string{true: "true", false: "false"}[w.controls] + `"></div>`
		s = strings.Replace(s, `<html `, `<html data-admin-app="true" `, 1)
		if strings.Contains(s, `data-page="admin.html"`) {
			tab := w.tab
			if tab != "calendar" && tab != "inspection" && tab != "menu" {
				tab = "orders"
			}
			// Initial layout matches its destination before deferred scripts execute.
			s = strings.Replace(s, `<body `, `<body data-design-tab="`+tab+`" `, 1)
		}
		s = strings.ReplaceAll(s, "/mobile-design-v1.4.64.js", "/mobile-design-app-v1.4.65.js")
		s = strings.ReplaceAll(s, "/talk-unread-v1.4.61.js", "/talk-unread-app-v1.4.65.js")
		s = strings.ReplaceAll(s, "/worker-reconnect-v1.4.61.js", "/worker-reconnect-app-v1.4.65.js")
		for _, script := range []string{`<script src="/worker-app-theme-bootstrap-v1.4.63.js"></script>`, `<script defer src="/worker-app-v1.4.63.js"></script>`} {
			s = strings.ReplaceAll(s, script, "")
		}
		s = strings.Replace(s, "</head>", `<link rel="stylesheet" href="/native-admin-app-v1.4.65.css"></head>`, 1)
		s = strings.Replace(s, "</body>", marker+`<script defer src="/native-admin-app-v1.4.65.js"></script></body>`, 1)
		w.Header().Del("Content-Length")
		_, err := w.ResponseWriter.Write([]byte(s))
		return len(data), err
	}
	return w.ResponseWriter.Write(data)
}
