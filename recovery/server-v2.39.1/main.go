// Independently reconstructed server using byte-exact recovered web assets.
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

const notice = `<aside class="recovery-banner">격리 테스트판 v1.2 · 기존 자료와 분리 · 외부 연동 꺼짐 <a href="/recovery-status">기능 차이 확인</a> · <a href="/">관리자</a></aside>`

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
	if e := run(); e != nil {
		log.Print(e)
		nativeError(e.Error())
		os.Exit(1)
	}
}
func run() error {
	home, e := os.UserConfigDir()
	if e != nil {
		return e
	}
	defaultDir := filepath.Join(home, testFolderName)
	dir := flag.String("data", defaultDir, "격리 테스트 자료 폴더 (기존 자료 폴더 사용 금지)")
	listen := flag.String("listen", testListen, "테스트 전용 로컬 접속 주소")
	noBrowser := flag.Bool("no-browser", false, "브라우저 자동 열기 생략")
	copyOriginal := flag.Bool("copy-original", false, "기존 자료를 새 테스트 복사본으로 가져오기 (원본은 읽기 전용)")
	headless := flag.Bool("headless", false, "Windows 관리 창과 트레이 생략 (시험용)")
	flag.Parse()
	host, _, e := net.SplitHostPort(*listen)
	if e != nil || host != "127.0.0.1" {
		return errors.New("복구 서버는 127.0.0.1에서만 실행합니다")
	}
	unlockOriginal, e := lockOriginalApp()
	if e != nil {
		return e
	}
	defer unlockOriginal()
	importing := *copyOriginal
	customData := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "data" {
			customData = true
		}
	})
	if importing {
		if customData {
			return errors.New("기존 자료 복사 시 -data를 함께 지정할 수 없습니다")
		}
		source, err := legacyDir()
		if err != nil {
			return err
		}
		copied, err := copyForTest(source, defaultDir, nil)
		if err != nil {
			return err
		}
		*dir = copied
	} else if !customData {
		selected, err := selectedTestDir(defaultDir)
		if err != nil {
			return err
		}
		*dir = selected
	}
	freshTest, e := prepareTestDir(*dir)
	if e != nil {
		return e
	}
	unlock, e := lockData(filepath.Join(*dir, "server.lock"))
	if e != nil {
		return fmt.Errorf("같은 자료 폴더가 사용 중이거나 열 수 없습니다: %w", e)
	}
	defer unlock()

	ln, e := net.Listen("tcp", *listen)
	if e != nil {
		return e
	}
	defer ln.Close()
	a, e := openTestApp(*dir, freshTest)
	if e != nil {
		return e
	}
	if importing {
		if e = a.transaction(func(s *State) error {
			s.Board = token()
			for i := range s.Projects {
				s.Projects[i].rotate()
			}
			return nil
		}); e != nil {
			return e
		}
		if e = selectTestCopy(defaultDir, *dir); e != nil {
			return e
		}
	}
	url := "http://" + ln.Addr().String()
	a.localURL = url
	fmt.Printf("알파 복구 테스트판 v1.2\n관리자: %s\n저장 폴더: %s\n종료: 이 창에서 Ctrl+C\n", url, *dir)
	srv := &http.Server{Handler: a, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 1 << 20}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	desktopStarted := false
	if !*headless {
		closeDesktop, started, err := startDesktop(a, stop)
		if err != nil {
			return err
		}
		defer closeDesktop()
		desktopStarted = started
	}
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
	if !*noBrowser && !desktopStarted {
		openBrowser(url)
	}
	if e = srv.Serve(ln); e != nil && !errors.Is(e, http.ErrServerClosed) {
		return e
	}
	a.mu.Lock()
	publishOffline := a.state.Integration.Connection.Provider == "pages" && a.integration.TunnelSession != ""
	a.mu.Unlock()
	if publishOffline {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		a.publishPointer(ctx, false)
		cancel()
	}

	return nil
}
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if a.testMode && blockedTestRoute(r.URL.Path) {
		http.Error(w, "격리 테스트판입니다. 기존 운영 서비스에 영향을 주지 않도록 휴대폰·지문·외부 연결·원격 백업은 꺼져 있습니다.", http.StatusForbidden)
		return
	}

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

const statusPage = `<!doctype html><html lang="ko"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><link rel="stylesheet" href="/style.css"><title>복구 기능 확인</title></head><body><header>알파 격리 테스트판 v1.2</header><main><section class="card"><h1>격리 테스트판 v1.2</h1><p><strong>이 버전은 전용 테스트 폴더·포트·트레이·로그인 쿠키를 사용합니다. 기존 운영 자료를 자동으로 읽지 않습니다. 아래 원본 저장 형식 설명은 테스트 폴더 안에서만 적용되며 외부 연동은 차단되어 있습니다.</strong></p><p>원본 메타데이터와 어셈블리·화면 명세로 다시 작성한 코드입니다. 원본과 100% 동일하다고 검증된 프로그램은 아닙니다.</p><h2>원본 직접 저장</h2><p>기존 WanryoTalk의 projects-v2.dat, connection-v1.dat, phone-push-v1.dat, designers-v1.dat, client-notes-v1.dat, backup-v1.dat를 직접 갱신합니다. 기존 파일은 최초 .bak으로 보존하며 기존 백업은 덮어쓰지 않습니다. 수주·기록·담당자·업체·설정은 원본 JSON 필드로 저장하고 사진은 기존 project-photos 폴더의 DPAPI JPEG로 저장합니다. 복구판 추가 상태는 projects-v2.dat의 _RecoveredV1 확장 필드에 보관하며 별도 reconstructed-state.bin은 새로 쓰지 않습니다.</p><p>여러 파일 저장 중 종료되면 복구 저널로 미완료 저장을 되돌립니다. 외부 변경이나 손상이 있으면 자료를 덮어쓰지 않고 중단합니다. 원본과 복구판은 동시에 실행하지 마세요. .bak은 최초 상태이며 이후 자료는 암호화 GitHub 백업도 설정하세요.</p><h2>Windows 창·알림</h2><p>Win32 관리 창, 트레이 메뉴, 자체 팝업, 미확인 숫자 배지를 구현했습니다. 창을 닫으면 트레이에 남고 ‘종료’로 끝납니다. 팝업은 잠시 뒤 사라지며 마우스를 올리면 멈춥니다. 미확인 숫자를 누르면 다시 볼 수 있습니다. 관리자 화면 버튼은 PC가 만든 일회용 티켓으로 연결합니다.</p><h2>확인된 범위와 남은 차이</h2><p>Go 서버 시험·저장 중 종료 및 복구·원본 스키마 읽기·브라우저 업무 흐름·Windows 교차 빌드로 검증합니다. Windows GUI/DPAPI의 실기기 실행, 실제 회사 자료·삼성 인터넷 지문·잠금 알림 및 실제 외부 계정 연동은 이 환경에서 확인하지 못했습니다. 창의 픽셀·배치·애니메이션·소리가 원본과 동일하다는 보장은 없습니다. ngrok 실행, 수주 엑셀 일괄등록, 원본 엑셀 직접 병합·사진 자동 청소는 여전히 미구현입니다.</p><p>시세·일정·Web Push·WebAuthn·Cloudflare·GitHub 백업은 v0.2 기능을 유지합니다. 백업은 복구판 별도 Release에 age 암호화하며 원본 백업을 삭제하지 않습니다. 백업 상한은 256MB입니다. Windows는 현재 계정의 DPAPI를 쓰며 Linux는 암호화 없는 검증용입니다. 실행 전 전체 자료 폴더를 별도 보관하고 복구키는 PC 밖에 보관하세요.</p><a class="button primary" href="/">관리자 화면</a></section></main></body></html>`
