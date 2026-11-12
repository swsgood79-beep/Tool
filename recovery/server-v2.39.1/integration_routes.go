package main

import (
	"context"
	"fmt"
	"golang.org/x/crypto/bcrypt"

	"encoding/json"
	"errors"
	"filippo.io/age"
	"github.com/SherClockHolmes/webpush-go"
	"github.com/skip2/go-qrcode"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

func timeValue(s string) *time.Time { t, _ := legacyTime(s); return t }
func (a *App) backupView() map[string]any {
	b := a.state.Integration.Backup
	return map[string]any{"Enabled": b.Enabled, "Status": a.integration.BackupStatus, "LastSuccess": timeValue(b.LastSuccess), "SnapshotAt": timeValue(b.LastSnapshot), "LastSize": b.LastSize, "PendingChanges": snapshotHash(a.state) != b.LastHash, "PendingRestart": false}
}
func (a *App) integrationGET(w http.ResponseWriter, r *http.Request, v map[string]any) bool {
	csrf := v["CSRF"].(string)
	c := a.state.Integration.Connection
	b := a.state.Integration.Backup
	if !localRequest(r) && !strings.HasPrefix(r.URL.Path, "/mobile-admin/passkey") && r.URL.Path != "/mobile-admin/session" {
		for _, p := range []string{"/settings", "/phone", "/backup", "/market-settings", "/schedule", "/mobile-access", "/connection"} {
			if strings.HasPrefix(r.URL.Path, p) {
				http.Error(w, "연결·백업 설정은 회사 PC에서 변경하세요.", 403)
				return true
			}
		}
	}
	switch r.URL.Path {
	case "/settings":
		v["Config"] = map[string]any{"Ready": c.Provider != "", "Provider": c.Provider, "Domain": c.Domain, "PagesURL": pagesOrigin + "/Tool/", "HasGitHubToken": c.GitHubToken != "", "HasToken": c.AuthToken != ""}
		a.render(w, "settings.html", v)
		return true
	case "/connection":
		a.renderTemplate(w, template.Must(template.New("connect").Parse(connectionPage)), "connect", map[string]any{"CSRF": csrf, "Status": a.integration.TunnelStatus})
		return true
	case "/qr.png":
		raw := c.PublicURL + "/work/" + a.state.Board
		if c.Provider == "pages" {
			raw = pagesOrigin + "/Tool/#k=" + a.state.Board
		}
		png, e := qrcode.Encode(raw, qrcode.Medium, 320)
		if e != nil {
			http.Error(w, "QR 생성 실패", 500)
		} else {
			w.Header().Set("Content-Type", "image/png")
			w.Write(png)
		}
		return true
	case "/phone", "/phone/pair.png":
		link := ""
		if a.integration.Pair != "" && a.integration.PairExpiry.After(time.Now()) {
			link = pagesOrigin + "/Tool/phone/#" + url.Values{"endpoint": {c.PublicURL}, "token": {a.integration.Pair}, "key": {a.state.Integration.Phone.PublicKey}}.Encode()
		}
		if r.URL.Path == "/phone/pair.png" {
			png, e := qrcode.Encode(link, qrcode.Medium, 320)
			if e != nil || link == "" {
				http.Error(w, "연결 QR을 먼저 만드세요.", 409)
			} else {
				w.Header().Set("Content-Type", "image/png")
				w.Write(png)
			}
			return true
		}
		v["Phone"] = map[string]any{"Status": a.integration.PhoneStatus, "Connected": a.state.Integration.Phone.Subscription != nil, "Available": c.PublicURL != "", "Link": link}
		a.render(w, "phone.html", v)
		return true
	case "/market-settings":
		m := a.state.Integration.Market
		v["Market"] = map[string]any{"CSRF": csrf, "Enabled": m.Enabled, "HasKeys": m.ClientID != "", "Error": a.integration.MarketStatus}
		a.render(w, "market-settings.html", v)
		return true
	case "/market-settings/status":
		jsonReply(w, map[string]any{"status": a.integration.MarketStatus})
		return true
	case "/schedule":
		status := a.integration.ScheduleStatus
		upper := map[string]any{"Label": status["label"], "Error": status["error"], "Success": timeValue(text(status["success"])), "Next": timeValue(text(status["next"])), "SourceOrders": status["source_orders"], "Matched": status["matched"], "Changed": status["changed"], "Unmatched": status["unmatched"], "Conflicts": status["conflicts"]}
		cfg := a.state.Integration.Schedule
		v["Mode"] = cfg.Mode
		v["Auto"] = cfg.Enabled
		v["HasSource"] = cfg.URL != "" || cfg.Path != ""
		v["Status"] = upper
		a.render(w, "schedule.html", v)
		return true
	case "/schedule/status.json":
		jsonReply(w, a.scheduleJSON())
		return true

	case "/backup":
		v["Backup"] = a.backupView()
		a.render(w, "backup.html", v)
		return true
	case "/backup/status":
		jsonReply(w, map[string]any{"Enabled": b.Enabled, "Status": a.integration.BackupStatus, "LastSuccess": b.LastSuccess, "SnapshotAt": b.LastSnapshot, "PendingChanges": snapshotHash(a.state) != b.LastHash, "LastSize": b.LastSize, "PendingRestart": false})
		return true
	case "/mobile-access":
		v["MobileAccess"] = map[string]any{"Enabled": a.state.Integration.MobileEnabled, "Sessions": len(a.sessions), "Revision": a.state.Integration.Revision, "HasPassword": a.state.PasswordHash != "", "Passkeys": len(a.state.Integration.Passkeys.Credentials)}
		a.render(w, "mobile-access.html", v)
		return true
	case "/mobile-admin/passkey":
		v["PasskeyCount"] = len(a.state.Integration.Passkeys.Credentials)
		a.render(w, "passkey-settings.html", v)
		return true
	case "/mobile-admin/session":
		jsonReply(w, map[string]any{"authenticated": true, "ok": true})
		return true
	}
	return false
}
func (a *App) integrationPOST(w http.ResponseWriter, r *http.Request, s Session) bool {
	path := r.URL.Path
	if !localRequest(r) && !strings.HasPrefix(path, "/mobile-admin/passkey") {
		for _, p := range []string{"/settings", "/phone", "/backup", "/market-settings", "/schedule", "/mobile-access", "/connection"} {
			if strings.HasPrefix(path, p) {
				http.Error(w, "회사 PC에서 설정하세요.", 403)
				return true
			}
		}
	}
	var err error
	handled := true
	redirect := path
	switch path {
	case "/settings":
		err = a.transaction(func(st *State) error {
			c := &st.Integration.Connection
			p := r.FormValue("provider")
			if p != "pages" && p != "quick" {
				return errors.New("복구판은 현재 GitHub Pages·Cloudflare 임시 연결을 지원합니다. 기존 ngrok 설정은 보존됩니다")
			}
			c.Provider = p
			if val := strings.TrimSpace(r.FormValue("github_token")); val != "" {
				c.GitHubToken = val
			}
			if val := r.FormValue("token"); val != "" {
				c.AuthToken = val
			}
			c.Domain = r.FormValue("domain")
			return nil
		})
	case "/connection":
		switch r.FormValue("action") {
		case "start":
			go a.startTunnel()
		case "stop":
			if a.integration.CancelTunnel != nil {
				a.integration.CancelTunnel()
			}
		case "test":
			nativeNotify("알파 알림 테스트", "PC 알림 기능을 확인하세요.", a.localURL+"/unread")
		default:
			err = errors.New("연결 작업 구분 오류")
		}
	case "/phone":
		switch r.FormValue("action") {
		case "pair":
			if a.state.Integration.Connection.PublicURL == "" {
				err = errors.New("먼저 인터넷 연결을 시작하세요.")
				break
			}
			if a.state.Integration.Phone.PublicKey == "" {
				private, public, e := webpush.GenerateVAPIDKeys()
				if e != nil {
					err = e
					break
				}
				err = a.transaction(func(st *State) error {
					st.Integration.Phone.PublicKey = public
					st.Integration.Phone.PrivateKey = private
					st.Integration.Phone.Version = 1
					return nil
				})
			}
			if err == nil {
				a.integration.Pair = token()
				a.integration.PairExpiry = time.Now().Add(10 * time.Minute)
			}
		case "disconnect":
			err = a.transaction(func(st *State) error {
				st.Integration.Phone.Subscription = nil
				st.Integration.Phone.Generation++
				return nil
			})
			a.integration.Pair = ""
			a.integration.PhoneStatus = "휴대폰 연결 해제"
		case "test":
			phone := a.state.Integration.Phone
			if phone.Subscription == nil {
				err = errors.New("연결된 휴대폰이 없습니다")
				break
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				payload, _ := json.Marshal(map[string]any{"title": "알파 휴대폰 알림 테스트", "body": "휴대폰 알림 연결을 확인했습니다.", "tag": "alpha-test"})
				res, e := webpush.SendNotificationWithContext(ctx, payload, phone.Subscription, &webpush.Options{Subscriber: "mailto:admin@example.invalid", VAPIDPublicKey: phone.PublicKey, VAPIDPrivateKey: phone.PrivateKey, TTL: 300, HTTPClient: a.httpClient()})
				a.mu.Lock()
				defer a.mu.Unlock()
				if e != nil {
					a.integration.PhoneStatus = "테스트 알림 전송 실패"
				} else {
					if res.StatusCode >= 200 && res.StatusCode < 300 {
						a.integration.PhoneStatus = "테스트 알림 전송 접수 완료"
					} else {
						a.integration.PhoneStatus = "알림 서비스가 전송을 거부했습니다"
					}
					ioClose(res)
				}
			}()
		default:
			err = errors.New("휴대폰 작업 구분 오류")
		}
	case "/market-settings":
		err = a.transaction(func(st *State) error {
			m := &st.Integration.Market
			m.Enabled = r.FormValue("enabled") != ""
			id, secret := strings.TrimSpace(r.FormValue("client_id")), strings.TrimSpace(r.FormValue("client_secret"))
			if r.FormValue("clear_keys") != "" {
				m.ClientID = ""
				m.ClientSecret = ""
			} else if id != "" || secret != "" {
				if id == "" || secret == "" {
					return errors.New("Client ID와 Secret을 함께 입력하세요")
				}
				m.ClientID = id
				m.ClientSecret = secret
			}
			m.Revision++
			return nil
		})
		a.integration.AccessToken = ""
		a.integration.Bases = map[string]float64{}
		a.integration.Minutes = map[string]MinuteQuote{}
		a.integration.FXCache = nil
		a.integration.FXUntil = time.Time{}
		go a.refreshMarkets(context.Background())
	case "/schedule":
		if r.FormValue("action") == "save" {
			err = a.transaction(func(st *State) error {
				c := &st.Integration.Schedule
				c.Mode = r.FormValue("mode")
				c.Enabled = r.FormValue("enabled") != ""
				if v := strings.TrimSpace(r.FormValue("source_url")); v != "" {
					if !scheduleURLAllowed(v) {
						return errors.New("OneDrive 공유 URL을 확인하세요")
					}
					c.URL = v
				}
				if v := r.FormValue("source_path"); v != "" {
					if !filepath.IsAbs(v) {
						return errors.New("절대 파일 경로를 입력하세요")
					}
					c.Path = v
				}
				if c.Mode != "url" && c.Mode != "file" || c.Mode == "url" && c.URL == "" || c.Mode == "file" && c.Path == "" {
					return errors.New("납기 연결 정보를 확인하세요")
				}
				c.Revision++
				return nil
			})
		}
		if err == nil {
			go a.syncSchedule(context.Background(), true)
		}
	case "/backup/key":
		err = a.transaction(func(st *State) error {
			if st.Integration.Backup.Key.Identity != "" {
				return nil
			}
			id, e := age.GenerateX25519Identity()
			if e != nil {
				return e
			}
			st.Integration.Backup.Key = RecoveryKey{1, repository, token(), id.String()}
			return nil
		})
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", `attachment; filename="Alpha-Recovered-Recovery-Key.json"`)
			json.NewEncoder(w).Encode(a.state.Integration.Backup.Key)
			return true
		}
	case "/backup/config":
		if r.FormValue("enable") == "1" && (r.FormValue("saved_key") != "1" || a.state.Integration.Backup.Key.Identity == "") {
			err = errors.New("복구키 파일을 먼저 PC 밖에 보관하세요")
		} else {
			err = a.transaction(func(st *State) error { st.Integration.Backup.Enabled = r.FormValue("enable") == "1"; return nil })
		}
		redirect = "/backup"
	case "/backup/now":
		go a.runBackup(context.Background())
		redirect = "/backup"
	case "/backup/restore":
		if r.FormValue("confirm") != "1" {
			err = errors.New("복원 확인이 필요합니다")
			break
		}
		f, _, e := r.FormFile("recovery")
		if e != nil {
			err = e
			break
		}
		data, e := io.ReadAll(io.LimitReader(f, 16385))
		f.Close()
		var key RecoveryKey
		if e != nil || len(data) > 16384 || json.Unmarshal(data, &key) != nil {
			err = errors.New("복구키 파일 오류")
			break
		}
		generation := int(number(r.FormValue("generation")))
		bearer := r.FormValue("github_token")
		if bearer == "" {
			bearer = a.state.Integration.Connection.GitHubToken
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			encrypted, e := a.downloadBackup(ctx, key, bearer, generation)
			a.mu.Lock()
			defer a.mu.Unlock()
			if e == nil {
				e = a.restorePortable(key, encrypted)
			}
			if e != nil {
				a.integration.BackupStatus = "복원 실패: " + e.Error()
			} else {
				a.integration.BackupStatus = "복원 완료 · 다시 로그인하세요"
			}
		}()
		redirect = "/backup"
	case "/mobile-access":
		if number(r.FormValue("revision")) != a.state.Integration.Revision {
			err = errors.New("접속 설정이 변경되었습니다")
			break
		}
		action := r.FormValue("action")
		err = a.transaction(func(st *State) error {
			if action == "save" {
				st.Integration.MobileEnabled = r.FormValue("enabled") != ""
				if pw := r.FormValue("password"); pw != "" {
					if utf8.RuneCountInString(pw) < 12 || utf8.RuneCountInString(pw) > 72 || pw != r.FormValue("confirm") {
						return errors.New("비밀번호 확인이 다릅니다")
					}
					salt := token()
					bytes, e := bcrypt.GenerateFromPassword([]byte(pw), 12)
					hash := string(bytes)
					if e != nil {
						return e
					}
					st.Salt = salt
					st.PasswordHash = hash
					st.Integration.Passkeys = PasskeyData{}
				}
			} else if action == "clear-passkeys" {
				if r.FormValue("confirm") != "1" {
					return errors.New("해제 확인이 필요합니다")
				}
				st.Integration.Passkeys = PasskeyData{}
			} else if action != "revoke" {
				return errors.New("요청 구분 오류")
			}
			st.Integration.Revision++
			return nil
		})
		if err == nil {
			a.sessions = map[string]Session{}
			a.integration.Flows = map[string]PasskeyFlow{}
			a.integration.Starts = map[string]PasskeyFlow{}
			a.integration.Handoffs = map[string]Handoff{}
			redirect = "/login"
		}
	case "/mobile-admin/passkey/enroll":
		if !a.passwordMatches(r.FormValue("password")) {
			err = errors.New("관리자 비밀번호가 맞지 않습니다")
			break
		}
		if a.state.Integration.Passkeys.UserID == "" {
			err = a.transaction(func(st *State) error { st.Integration.Passkeys.UserID = token(); return nil })
		}
		if err == nil {
			cookie, e := r.Cookie("aw_recovered_session")
			if e != nil {
				err = e
				break
			}
			start := token()
			a.pruneFlows()
			a.integration.Starts[hashToken(start)] = PasskeyFlow{Mode: "register", Owner: hashToken(cookie.Value), Revision: a.state.Integration.Revision, Expires: time.Now().Add(3 * time.Minute), Next: safeNext(r.FormValue("next"))}
			http.Redirect(w, r, pagesOrigin+"/Tool/admin/?passkey=register#enroll="+start, 303)
			return true
		}
	default:
		handled = false
	}
	if !handled {
		return false
	}
	if err != nil {
		http.Error(w, err.Error(), 409)
		return true
	}
	http.Redirect(w, r, redirect, 303)
	return true
}
func (a *App) passkeyStart(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == "GET" && r.URL.Path == "/native-admin" {
		if !localRequest(r) {
			http.Error(w, "PC에서만 사용할 수 있습니다", 403)
			return true
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<html lang="ko"><meta charset="utf-8"><title>PC 관리자 연결</title><script defer src="/native-enter.js"></script><p id="native-result">PC 관리자 화면에 연결합니다.</p><a href="/login">비밀번호 로그인</a></html>`)
		return true
	}
	if r.Method == "GET" && r.URL.Path == "/native-enter.js" {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		fmt.Fprint(w, nativeEnterJS)
		return true
	}

	if r.URL.Path == "/mobile-admin/passkey/enter" {
		a.render(w, "passkey-enter.html", map[string]any{})
		return true
	}
	if r.URL.Path != "/mobile-admin/passkey/start" {
		return false
	}
	if !a.state.Integration.MobileEnabled || len(a.state.Integration.Passkeys.Credentials) == 0 {
		http.Error(w, "비밀번호 로그인 후 이 기기의 패스키를 먼저 등록하세요.", 403)
		return true
	}
	a.pruneFlows()
	if len(a.integration.Starts) > 128 {
		http.Error(w, "잠시 후 다시 시도하세요", 429)
		return true
	}
	start := token()
	a.integration.Starts[hashToken(start)] = PasskeyFlow{Mode: "auth", Revision: a.state.Integration.Revision, Next: safeNext(r.URL.Query().Get("next")), Expires: time.Now().Add(3 * time.Minute)}
	http.Redirect(w, r, pagesOrigin+"/Tool/admin/?passkey=login#start="+start, 303)
	return true
}

const connectionPage = `<!doctype html><html lang="ko"><head><meta charset="utf-8"><link rel="stylesheet" href="/style.css"></head><body><main><section class="card"><h1>인터넷 연결 · PC 알림</h1><p>{{.Status}}</p><form method="post"><input type="hidden" name="csrf" value="{{.CSRF}}"><button class="primary" name="action" value="start">인터넷 연결</button> <button class="quiet" name="action" value="stop">인터넷 끊기</button> <button class="quiet" name="action" value="test">PC 알림 테스트</button></form><p>인터넷 연결은 검증된 Cloudflare 도구를 사용합니다. 기존 회사 프로그램을 종료한 뒤 연결하세요. GitHub Pages 방식은 연결된 주소만 main의 qr-endpoint.json에 게시합니다.</p><a href="/">관리자 화면</a></section></main></body></html>`
