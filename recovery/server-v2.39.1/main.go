// Independently reconstructed partial server using byte-exact recovered web assets.
// NOT a recovered copy of the original Go backend. See README.md and /recovery-status.
package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

//go:embed web/*
var web embed.FS
var templates *template.Template

func init() {
	templates = template.Must(template.New("recovered").Funcs(template.FuncMap{"kst": func(v any) string {
		switch t := v.(type) {
		case *time.Time:
			if t != nil {
				return t.In(kst).Format("2006.01.02 15:04")
			}
		case time.Time:
			return t.In(kst).Format("2006.01.02 15:04")
		}
		return ""
	}}).ParseFS(web, "web/*.html"))
}

const notice = `<aside class="recovery-banner">복구 서버 · 부분 구현 · 기존 자료 호환 읽기 · 추가 기능 검증판 <a href="/recovery-status">기능 차이 확인</a> · <a href="/">관리자</a></aside>`

func (a *App) renderTemplate(w http.ResponseWriter, t *template.Template, name string, v any) {
	var b bytes.Buffer
	if err := t.ExecuteTemplate(&b, name, v); err != nil {
		log.Printf("template %s: %v", name, err)
		http.Error(w, "화면 생성 실패", 500)
		return
	}
	s := b.String()
	s = strings.Replace(s, `<a href="/market-settings">`, `<a href="/connection">인터넷 연결 · PC 알림</a><a href="/market-settings">`, 1)

	s = strings.ReplaceAll(s, "수주를 등록하고 공용 QR을 작업자에게 전달하세요.", "수주를 등록하고 상단의 작업자 화면 버튼으로 검사를 신청하세요.")
	s = strings.ReplaceAll(s, "‘고정 QR 설정’에서 주소와 인증키를 한 번 저장한 뒤 PC 프로그램에서 ‘인터넷 연결’을 눌러 주세요. 연결되면 이 페이지를 새로고침해 QR을 확인할 수 있습니다.", "설정 메뉴에서 인터넷 연결을 시작하면 기존 고정 QR과 휴대폰 접속 주소를 사용할 수 있습니다.")
	s = strings.ReplaceAll(s, "Excel에서 파일을 열어 잠겨 있으면 닫은 뒤 자동으로 다시 저장합니다.", "Excel에서 파일을 열어 저장이 막히면 닫아 주세요. 다음 자료 변경 때 다시 저장하며 엑셀 내려받기도 사용할 수 있습니다.")
	if root, ok := v.(map[string]any); ok && root["WorkerURL"] != nil {
		if csrf, ok := root["CSRF"].(string); ok && csrf != a.state.Board {
			logout := `<form method="post" action="/logout"><input type="hidden" name="csrf" value="` + template.HTMLEscapeString(csrf) + `"><button class="quiet">관리자 로그아웃</button></form>`
			s = strings.Replace(s, "</main>", logout+"</main>", 1)
		}
	}

	s = strings.Replace(s, "</head>", `<link rel="stylesheet" href="/recovery.css"></head>`, 1)
	i := strings.Index(s, "<body")
	if i >= 0 {
		j := strings.Index(s[i:], ">")
		if j >= 0 {
			s = s[:i+j+1] + notice + s[i+j+1:]
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(s))
}
func (a *App) render(w http.ResponseWriter, name string, v any) {
	if name == "project-create.html" || name == "project-edit.html" {
		var b bytes.Buffer
		if err := templates.ExecuteTemplate(&b, name, v); err != nil {
			http.Error(w, "화면 생성 실패", 500)
			return
		}
		due := ""
		if root, ok := v.(map[string]any); ok {
			if ps, ok := root["Projects"].([]ProjectView); ok && name == "project-edit.html" && len(ps) > 0 {
				due = ps[0].ShippingDueDate
			}
		}
		field := `<label>출고 예정일 (수동)<input type="date" name="shipping_due_date" value="` + template.HTMLEscapeString(due) + `"></label>`
		html := strings.Replace(b.String(), `<div class="project-edit-grid">`, field+`<div class="project-edit-grid">`, 1)
		t := template.Must(template.New("form").Parse(`{{.}}`))
		a.renderTemplate(w, t, "form", template.HTML(html))
		return
	}
	a.renderTemplate(w, templates, name, v)
}
func jsonReply(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}
func main() {
	home, e := os.UserConfigDir()
	if e != nil {
		log.Fatal(e)
	}
	defaultDir := filepath.Join(home, "AlphaWorkRecoveredServer")
	if old, e := legacyDir(); e == nil {
		if _, e = os.Stat(filepath.Join(old, "projects-v2.dat")); e == nil {
			defaultDir = old
		}
	}
	dir := flag.String("data", defaultDir, "복구 서버 전용 저장 폴더")
	listen := flag.String("listen", "127.0.0.1:18791", "로컬 접속 주소")
	noBrowser := flag.Bool("no-browser", false, "브라우저 자동 열기 생략")
	flag.Parse()
	host, _, e := net.SplitHostPort(*listen)
	if e != nil || host != "127.0.0.1" {
		log.Fatal("이 부분 복구판은 127.0.0.1에서만 실행합니다.")
	}
	if e = os.MkdirAll(*dir, 0700); e != nil {
		log.Fatal(e)
	}
	unlock, e := lockData(filepath.Join(*dir, "server.lock"))
	if e != nil {
		log.Fatal("같은 자료 폴더가 다른 프로세스에서 사용 중이거나 열 수 없습니다: ", e)
	}
	defer unlock()
	if _, err := os.Stat(filepath.Join(*dir, "projects-v2.dat")); err == nil && originalRunning() {
		log.Fatal("원본 알파 프로그램을 먼저 종료한 뒤 복구 서버를 실행하세요. 기존 자료는 변경하지 않았습니다.")
	}
	a, e := newApp(*dir)
	if e != nil {
		log.Fatal(e)
	}
	ln, e := net.Listen("tcp", *listen)
	if e != nil {
		log.Fatal(e)
	}
	url := "http://" + ln.Addr().String()
	a.localURL = url
	fmt.Printf("알파 검사 복구 서버 (부분 구현)\n관리자: %s\n저장 폴더: %s\n종료: 이 창에서 Ctrl+C\n", url, *dir)
	srv := &http.Server{Handler: a, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 1 << 20}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	a.startPumps(ctx)
	go func() {
		<-ctx.Done()
		a.mu.Lock()
		cancelTunnel := a.integration.CancelTunnel
		a.mu.Unlock()
		if cancelTunnel != nil {
			cancelTunnel()
		}
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()
	if !*noBrowser {
		openBrowser(url)
	}
	if e = srv.Serve(ln); e != nil && !errors.Is(e, http.ErrServerClosed) {
		log.Fatal(e)
	}
	a.mu.Lock()
	publishOffline := a.state.Integration.Connection.Provider == "pages" && a.integration.TunnelSession != ""
	a.mu.Unlock()
	if publishOffline {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		a.publishPointer(ctx, false)
		cancel()
	}

}
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
	if a.specialHTTP(w, r) {
		return
	}
	if !a.allowedRequest(r) {
		http.Error(w, "허용하지 않는 접속 주소 또는 출처", 403)
		return
	}
	if r.Method != "GET" && r.Method != "POST" {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "지원하지 않는 요청 방식", 405)
		return
	}
	if r.URL.Path == "/favicon.ico" && r.Method == "GET" {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.URL.Path == "/recovery.css" && r.Method == "GET" {
		w.Header().Set("Content-Type", "text/css")
		fmt.Fprint(w, `.recovery-banner{padding:10px 16px;background:#fff1c2;color:#473700;text-align:center;font:13px sans-serif}.recovery-banner a{color:#173d79;text-decoration:underline;margin:0 4px}`)
		return
	}
	if r.URL.Path == "/recovery-status" && r.Method == "GET" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, statusPage)
		return
	}
	if r.Method == "GET" && !strings.Contains(strings.TrimPrefix(r.URL.Path, "/"), "/") {
		asset := strings.TrimPrefix(r.URL.Path, "/")
		if !strings.HasSuffix(asset, ".html") && asset != "" {
			if data, e := web.ReadFile("web/" + asset); e == nil {
				w.Header().Set("Cache-Control", "public, max-age=3600")
				http.ServeContent(w, r, asset, time.Time{}, bytes.NewReader(data))
				return
			}
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	r.Body = http.MaxBytesReader(w, r.Body, 48<<20)
	if r.Method == "POST" {
		var err error
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			err = r.ParseMultipartForm(2 << 20)
			if r.MultipartForm != nil {
				defer r.MultipartForm.RemoveAll()
			}
		} else {
			err = r.ParseForm()
		}
		if err != nil {
			http.Error(w, "요청 내용 또는 파일이 너무 큽니다.", 400)
			return
		}
	}
	if a.passkeyStart(w, r) {
		return
	}
	if strings.HasPrefix(r.URL.Path, "/ready/") {
		if equal(strings.TrimPrefix(r.URL.Path, "/ready/"), a.state.Board) {
			jsonReply(w, map[string]any{"application": "wanryotalk", "session": a.integration.TunnelSession})
		} else {
			http.NotFound(w, r)
		}
		return
	}
	if r.URL.Path == "/login" || r.URL.Path == "/mobile-admin/login" {
		if !localRequest(r) && !a.state.Integration.MobileEnabled {
			http.Error(w, "휴대폰 접속 꺼짐", 403)
			return
		}
		a.login(w, r)
		return
	}
	if a.worker(w, r) {
		return
	}
	s, ok := a.admin(w, r)
	if !ok {
		return
	}
	v := a.view(s.CSRF)
	if !localRequest(r) {
		v["MobileAdmin"] = true
	}
	if r.Method == "GET" {
		if a.integrationGET(w, r, v) {
			return
		}
		if a.adminGET(w, r, v) {
			return
		}
	} else {
		if a.integrationPOST(w, r, s) {
			return
		}
		if a.adminPOST(w, r, s) {
			return
		}
	}
	if pathLooksOriginal(r.URL.Path) {
		w.WriteHeader(http.StatusNotImplemented)
		a.renderTemplate(w, template.Must(template.New("status").Parse(statusPage)), "status", nil)
		return
	}
	http.NotFound(w, r)
}
func pathLooksOriginal(p string) bool {
	for _, prefix := range []string{"/backup", "/settings", "/phone", "/mobile", "/market", "/schedule", "/orders/import", "/passkey", "/qr.", "/connection", "/api/"} {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

const statusPage = `<!doctype html><html lang="ko"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><link rel="stylesheet" href="/style.css"><title>복구 기능 확인</title></head><body><header>알파 복구 서버 v0.2 · 호환·연동 검증판</header><main><section class="card"><h1>구현 범위와 확인 상태</h1><p>원본과 100% 동일하다고 검증된 프로그램은 아닙니다. 코드는 원본 메타데이터와 화면·API 명세를 참고해 다시 작성했습니다.</p><h2>이번에 연결한 기능</h2><p>기존 %AppData%\WanryoTalk 자료 자동 감지·DPAPI 읽기, 수주·사진·담당자·설정 가져오기, 기존 bcrypt 비밀번호와 등록 패스키 읽기, WebAuthn 서명/UV/일회용 흐름 검증, PC Windows 알림과 암호화 Web Push, 토스 지수·코인·환율, Cloudflare 연결·고정 QR 주소 갱신, OneDrive/동기화 파일 일정, age 암호화 GitHub 백업과 새 PC 복원.</p><h2>원본 자료를 보호하는 방식</h2><p>원본 .dat와 사진은 보존합니다. 복구판 변경 사항은 같은 폴더의 reconstructed-state.bin에 저장합니다. 원본 프로그램이 자료를 바꾸면 추가 저장을 중지합니다. 변경 내용이 원본 프로그램에 역으로 반영되는 동작은 아직 구현되지 않았습니다.</p><h2>아직 동일하지 않거나 실제 확인이 필요한 부분</h2><p>원본 Windows 관리 창·트레이·떠 있는 미확인 숫자, ngrok, 엑셀 일괄등록은 미복원입니다. PC 알림은 원본 자체 창 대신 Windows 토스트를 사용합니다. 실제 회사 DPAPI 파일·백업, 삼성 인터넷의 지문·잠금 알림, 실제 금융 API·공인 IP·Cloudflare·GitHub 연동은 본인 PC와 계정에서 추가 확인이 필요합니다. 자동시험은 원본 스키마 기반 시험 파일·실제 암호 서명·외부 API 대체 응답을 사용합니다.</p><p>백업은 이 복구판의 별도 Release에 보관하며 기존 원본 백업은 보존합니다. 원본 Version 1·2 백업은 형식 호환 읽기를 구현했으나 실제 회사 백업으로 확인한 것은 아닙니다. 메모리 보호를 위해 백업은 256MB로 제한됩니다.</p><p>Windows는 현재 계정의 DPAPI로 상태·사진을 보호합니다. Linux는 검증용이며 파일 암호화가 적용되지 않습니다. 복구키 파일은 PC 밖에 별도로 보관하세요.</p><a class="button primary" href="/">관리자 화면</a></section></main></body></html>`
