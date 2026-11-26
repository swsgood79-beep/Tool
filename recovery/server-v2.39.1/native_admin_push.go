package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

const nativeAndroidPackage = "com.alpharobotics.admin"

type NativeFirebasePublic struct{ ProjectID, ApplicationID, SenderID, APIKey string }
type NativeAdminDevice struct {
	ID, Name, AccountID, Token, SecretHash, SessionHash string
	Role                                                string `json:",omitempty"`
	AccountRevision, OwnerRevision, Delivered           int64
	Enabled, TestPending                                bool
	LastStatus                                          string
}
type NativeAdminRevocation struct {
	ID, SessionHash string
	Expires         time.Time
}
type NativeAdminPushConfig struct {
	Public                   NativeFirebasePublic
	ServiceEmail, PrivateKey string
	Enabled                  bool
	Revision                 int64
	Devices                  []NativeAdminDevice
	Revoked                  []NativeAdminRevocation
}

var nativeFCMToken = regexp.MustCompile(`^[A-Za-z0-9_:\-]+$`)

func validNativeFCMToken(value string) bool {
	return len(value) >= 20 && len(value) <= 4096 && nativeFCMToken.MatchString(value)
}
func nativePushReady(c *NativeAdminPushConfig) bool {
	return c != nil && c.Enabled && c.Public.ProjectID != "" && c.PrivateKey != ""
}
func nativeDevice(s *State, id string) *NativeAdminDevice {
	if s.NativeAdminPush != nil {
		for i := range s.NativeAdminPush.Devices {
			if s.NativeAdminPush.Devices[i].ID == id {
				return &s.NativeAdminPush.Devices[i]
			}
		}
	}
	return nil
}
func nativeAccountValid(s *State, d NativeAdminDevice) bool {
	if !s.Integration.MobileEnabled || !d.Enabled {
		return false
	}
	if d.AccountID == "" {
		return d.Role != "purchasing" && d.Role != "machining" && s.PasswordHash != "" && d.OwnerRevision == s.Integration.Revision
	}
	u := adminAccount(s, d.AccountID)
	return u != nil && u.Enabled && u.Revision == d.AccountRevision && (u.Role == "purchasing") == (d.Role == "purchasing") && (u.Role == "machining") == (d.Role == "machining")
}
func nativeAccountLabel(s *State, id string) string {
	if id == "" {
		if s.OwnerName != "" {
			return s.OwnerName
		}
		return ownerUsername
	}
	if u := adminAccount(s, id); u != nil {
		return u.Name
	}
	return "사용 중지 계정"
}
func nativeRSA(raw string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, errors.New("서비스 계정의 비밀키 형식을 확인해 주세요.")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		if k, ok := key.(*rsa.PrivateKey); ok && k.N.BitLen() >= 2048 {
			return k, nil
		}
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil && k.N.BitLen() >= 2048 {
		return k, nil
	}
	return nil, errors.New("2048비트 이상의 RSA 서비스 계정 키가 필요합니다.")
}
func nativeFirebaseFiles(clientJSON, serviceJSON []byte) (NativeAdminPushConfig, error) {

	var g struct {
		ProjectInfo struct {
			ProjectID     string `json:"project_id"`
			ProjectNumber string `json:"project_number"`
		} `json:"project_info"`
		Client []struct {
			ClientInfo struct {
				AppID   string `json:"mobilesdk_app_id"`
				Android struct {
					Package string `json:"package_name"`
				} `json:"android_client_info"`
			} `json:"client_info"`
			APIKeys []struct {
				Key string `json:"current_key"`
			} `json:"api_key"`
		} `json:"client"`
	}
	var service struct{ Type, ProjectID, Email, Key string }
	var raw struct {
		Type      string `json:"type"`
		ProjectID string `json:"project_id"`
		Email     string `json:"client_email"`
		Key       string `json:"private_key"`
	}
	if json.Unmarshal(clientJSON, &g) != nil || json.Unmarshal(serviceJSON, &raw) != nil {
		return NativeAdminPushConfig{}, errors.New("Firebase JSON 파일 형식을 확인해 주세요.")
	}
	service.Type, service.ProjectID, service.Email, service.Key = raw.Type, raw.ProjectID, raw.Email, raw.Key
	if service.Type != "service_account" || service.ProjectID == "" || service.ProjectID != g.ProjectInfo.ProjectID || !regexp.MustCompile(`^[a-z][a-z0-9-]{4,62}$`).MatchString(service.ProjectID) || !strings.HasSuffix(service.Email, ".iam.gserviceaccount.com") {
		return NativeAdminPushConfig{}, errors.New("같은 Firebase 프로젝트의 서비스 계정 파일이 필요합니다.")
	}
	if _, err := nativeRSA(service.Key); err != nil {
		return NativeAdminPushConfig{}, err
	}
	for _, c := range g.Client {
		if c.ClientInfo.Android.Package != nativeAndroidPackage {
			continue
		}
		if len(c.APIKeys) == 0 || !regexp.MustCompile(`^[A-Za-z0-9_-]{20,200}$`).MatchString(c.APIKeys[0].Key) || !regexp.MustCompile(`^[0-9]+$`).MatchString(g.ProjectInfo.ProjectNumber) || !strings.HasPrefix(c.ClientInfo.AppID, "1:"+g.ProjectInfo.ProjectNumber+":android:") {
			return NativeAdminPushConfig{}, errors.New("Android 앱의 Firebase 등록 정보를 확인해 주세요.")
		}
		return NativeAdminPushConfig{Public: NativeFirebasePublic{g.ProjectInfo.ProjectID, c.ClientInfo.AppID, g.ProjectInfo.ProjectNumber, c.APIKeys[0].Key}, ServiceEmail: service.Email, PrivateKey: service.Key, Enabled: true}, nil
	}
	return NativeAdminPushConfig{}, errors.New("Firebase Android 앱의 패키지명은 " + nativeAndroidPackage + "이어야 합니다.")
}
func nativeJSONFile(r *http.Request, name string) ([]byte, error) {
	f, h, e := r.FormFile(name)
	if e != nil {
		return nil, errors.New("두 Firebase JSON 파일을 모두 선택해 주세요.")
	}
	defer f.Close()
	if h.Size > 1<<20 {
		return nil, errors.New("Firebase 설정 파일은 1MB 이하만 등록할 수 있습니다.")
	}
	return io.ReadAll(io.LimitReader(f, 1<<20))
}
func (a *App) nativeAdminPublicHTTP(w http.ResponseWriter, r *http.Request) bool {
	if a.nativeLoginPublicHTTP(w, r) {
		return true
	}
	if r.URL.Path != "/native-admin/token-refresh" && r.URL.Path != "/native-admin/device-disable" {
		return false
	}
	if r.Method != "POST" || a.testMode {
		http.Error(w, "지원하지 않는 요청", 403)
		return true
	}
	d := nativeDevice(&a.state, r.FormValue("device"))
	if d == nil || !nativeAccountValid(&a.state, *d) || !equal(d.SecretHash, hashToken(r.FormValue("secret"))) {
		http.Error(w, "앱에 다시 로그인해 주세요.", 403)
		return true
	}
	id := d.ID
	disable := strings.HasSuffix(r.URL.Path, "device-disable")
	value := r.FormValue("token")
	if !disable && !validNativeFCMToken(value) {
		http.Error(w, "앱 알림 토큰 형식 오류", 400)
		return true
	}
	if err := a.commitAnnouncement(func(s *State) error {
		v := nativeDevice(s, id)
		if disable {
			v.Enabled = false
			v.SecretHash = ""
			v.Token = ""
			v.LastStatus = "앱에서 연결 해제"
		} else {
			v.Token = value
			v.LastStatus = "앱 알림 토큰 갱신"
		}
		return nil
	}); err != nil {
		http.Error(w, "앱 연결을 저장하지 못했습니다.", 500)
		return true
	}
	jsonReply(w, map[string]bool{"ok": true})
	return true
}
func (a *App) nativeAdminHTTP(w http.ResponseWriter, r *http.Request, session Session) bool {
	if a.nativeLoginEnroll(w, r, session) {
		return true
	}
	path := r.URL.Path
	if !strings.HasPrefix(path, "/native-admin/") && path != "/android-admin-settings" {
		return false
	}
	if a.testMode {
		http.Error(w, "격리 모드에서는 앱 푸시를 설정할 수 없습니다.", 403)
		return true
	}
	purchase := purchasingSession(&a.state, session)
	machine := machiningSession(&a.state, session)
	if path == "/native-admin/bootstrap" && r.Method == "GET" {
		var public *NativeFirebasePublic
		if nativePushReady(a.state.NativeAdminPush) {
			copy := a.state.NativeAdminPush.Public
			public = &copy
		}
		d := nativeDevice(&a.state, r.URL.Query().Get("device"))
		registered := d != nil && d.AccountID == session.AccountID && nativeAccountValid(&a.state, *d)
		grant := nativeLogin(&a.state, session.NativeGrantID)
		persistent := grant != nil && grant.DeviceID == r.URL.Query().Get("device") && nativeLoginValid(&a.state, *grant)
		username := ownerUsername
		if account := adminAccount(&a.state, session.AccountID); account != nil {
			username = account.Username
		}
		jsonReply(w, map[string]any{"account_role": nativeSessionRole(&a.state, session), "worker_view_supported": !purchase && !machine, "worker_view": nativeWorkerView(r, a.sessionCookie()), "account_username": username, "persistent_login": persistent, "cookie_name": a.sessionCookie(), "registered": registered, "ready": public != nil, "config": public, "csrf": session.CSRF, "release": releaseVersion, "account_key": session.AccountID, "account_label": nativeAccountLabel(&a.state, session.AccountID), "owner": session.AccountID == ""})
		return true
	}
	if path == "/native-admin/register" && r.Method == "POST" {
		if !nativePushReady(a.state.NativeAdminPush) {
			http.Error(w, "회사 PC에서 Firebase 연결을 먼저 설정해 주세요.", 409)
			return true
		}
		id, value := r.FormValue("device"), r.FormValue("token")
		if !validPhotoID.MatchString(id) || !validNativeFCMToken(value) {
			http.Error(w, "앱 등록 정보 오류", 400)
			return true
		}
		for _, rev := range a.state.NativeAdminPush.Revoked {
			if rev.ID == id && rev.Expires.After(time.Now()) && equal(rev.SessionHash, hashToken(session.CSRF)) {
				http.Error(w, "앱에서 로그아웃한 뒤 다시 로그인해 주세요.", 403)
				return true
			}
		}
		if nativeDevice(&a.state, id) == nil && len(a.state.NativeAdminPush.Devices) >= 20 {
			http.Error(w, "앱 연결은 최대20대입니다. 기존 연결을 정리해 주세요.", 400)
			return true
		}
		secret := token()
		name := strings.TrimSpace(r.FormValue("name"))
		if len([]rune(name)) > 60 {
			name = string([]rune(name)[:60])
		}
		err := a.commitAnnouncement(func(s *State) error {
			d := nativeDevice(s, id)
			delivered := s.NextID
			if d != nil && d.AccountID == session.AccountID && d.AccountRevision == session.AccountRevision {
				delivered = d.Delivered
			}
			appKind := ""
			if machine {
				appKind = "machining"
			}
			if purchase {
				appKind = "purchasing"
			}
			next := NativeAdminDevice{Role: appKind, ID: id, Name: name, AccountID: session.AccountID, AccountRevision: session.AccountRevision, OwnerRevision: s.Integration.Revision, Token: value, SecretHash: hashToken(secret), SessionHash: hashToken(session.CSRF), Enabled: true, Delivered: delivered, LastStatus: "앱 연결 완료"}
			for i := range s.NativeAdminPush.Devices {
				if s.NativeAdminPush.Devices[i].Token == value && s.NativeAdminPush.Devices[i].ID != id {
					s.NativeAdminPush.Devices[i].Enabled = false
					s.NativeAdminPush.Devices[i].Token = ""
					s.NativeAdminPush.Devices[i].SecretHash = ""
				}
			}
			if d == nil {
				s.NativeAdminPush.Devices = append(s.NativeAdminPush.Devices, next)
			} else {
				*d = next
			}
			return nil
		})
		if err != nil {
			http.Error(w, "앱 연결을 저장하지 못했습니다.", 500)
			return true
		}
		jsonReply(w, map[string]any{"ok": true, "secret": secret})
		return true
	}
	if path == "/native-admin/test" && r.Method == "POST" {
		if !nativePushReady(a.state.NativeAdminPush) {
			http.Error(w, "Firebase 연결 설정이 필요합니다.", 409)
			return true
		}
		d := nativeDevice(&a.state, r.FormValue("device"))
		if d == nil || d.AccountID != session.AccountID || !nativeAccountValid(&a.state, *d) {
			http.Error(w, "본인 앱 연결이 필요합니다.", 403)
			return true
		}
		id := d.ID
		if err := a.commitAnnouncement(func(s *State) error { nativeDevice(s, id).TestPending = true; return nil }); err != nil {
			http.Error(w, "테스트 요청 저장 실패", 500)
			return true
		}
		jsonReply(w, map[string]bool{"ok": true})
		return true
	}
	if path == "/android-admin-settings" {
		if session.AccountID != "" {
			http.Error(w, "대표 관리자만 설정할 수 있습니다.", 403)
			return true
		}
		view := map[string]any{"CSRF": session.CSRF, "WorkerURL": "/work/" + a.state.Board, "Ready": nativePushReady(a.state.NativeAdminPush), "Configured": a.state.NativeAdminPush != nil && a.state.NativeAdminPush.PrivateKey != "", "Saved": r.URL.Query().Get("saved") == "1", "Package": nativeAndroidPackage}
		if r.Method == "POST" {
			var err error
			switch r.FormValue("action") {
			case "configure":
				var b, c []byte
				b, err = nativeJSONFile(r, "google_services")
				if err == nil {
					c, err = nativeJSONFile(r, "service_account")
				}
				if err == nil {
					var config NativeAdminPushConfig
					config, err = nativeFirebaseFiles(b, c)
					if err == nil {
						err = a.commitAnnouncement(func(s *State) error {
							config.Revision = 1
							if s.NativeAdminPush != nil {
								config.Revision = s.NativeAdminPush.Revision + 1
								config.Devices = s.NativeAdminPush.Devices
								config.Revoked = s.NativeAdminPush.Revoked
							}
							s.NativeAdminPush = &config
							return nil
						})
						if err == nil {
							a.integration.NativeAdminOAuth = ""
						}
					}
				}
			case "revoke-login":
				id := r.FormValue("login")
				if nativeLogin(&a.state, id) == nil {
					err = errors.New("앱 로그인 연결을 찾을 수 없습니다.")
				} else {
					err = a.commitAnnouncement(func(s *State) error { disableNativeLogin(s, nativeLogin(s, id)); return nil })
					if err == nil {
						a.clearNativeGrantSessions(map[string]bool{id: true})
					}
				}
			case "enable":
				if a.state.NativeAdminPush == nil {
					err = errors.New("Firebase 설정을 먼저 등록하세요.")
				} else {
					err = a.commitAnnouncement(func(s *State) error { s.NativeAdminPush.Enabled = true; s.NativeAdminPush.Revision++; return nil })
				}
			case "disable":
				err = a.commitAnnouncement(func(s *State) error {
					if s.NativeAdminPush != nil {
						s.NativeAdminPush.Enabled = false
						s.NativeAdminPush.Revision++
					}
					return nil
				})
				a.integration.NativeAdminOAuth = ""
			case "remove", "test":
				id := r.FormValue("device")
				if nativeDevice(&a.state, id) == nil {
					err = errors.New("앱 연결을 찾을 수 없습니다.")
				} else {
					action := r.FormValue("action")
					err = a.commitAnnouncement(func(s *State) error {
						if action == "test" {
							nativeDevice(s, id).TestPending = true
						} else {
							old := nativeDevice(s, id)
							if old != nil {
								s.NativeAdminPush.Revoked = append(s.NativeAdminPush.Revoked, NativeAdminRevocation{ID: id, SessionHash: old.SessionHash, Expires: time.Now().Add(8 * time.Hour)})
							}
							list := s.NativeAdminPush.Devices[:0]
							for _, d := range s.NativeAdminPush.Devices {
								if d.ID != id {
									list = append(list, d)
								}
							}
							s.NativeAdminPush.Devices = list
						}
						return nil
					})
				}
			default:
				err = errors.New("지원하지 않는 요청입니다.")
			}
			if err == nil {
				http.Redirect(w, r, "/android-admin-settings?saved=1", 303)
				return true
			}
			view["Error"] = err.Error()
		}
		rows := []map[string]any{}
		if cfg := a.state.NativeAdminPush; cfg != nil {
			view["ProjectID"] = cfg.Public.ProjectID
			for _, d := range cfg.Devices {
				rows = append(rows, map[string]any{"ID": d.ID, "Name": d.Name, "Account": nativeAccountLabel(&a.state, d.AccountID), "Enabled": nativeAccountValid(&a.state, d), "Status": d.LastStatus})
			}
		}
		view["Devices"] = rows
		logins := []map[string]any{}
		for _, g := range a.state.NativeAdminLogins {
			if g.Disabled {
				continue
			}
			logins = append(logins, map[string]any{"ID": g.ID, "Name": g.Name, "Account": nativeAccountLabel(&a.state, g.AccountID), "Valid": nativeLoginValid(&a.state, g)})
		}
		view["Logins"] = logins
		a.render(w, "android-admin-settings.html", view)
		return true
	}
	http.NotFound(w, r)
	return true
}
func (a *App) nativeOAuth(ctx context.Context, c NativeAdminPushConfig) (string, error) {
	a.mu.Lock()
	if a.integration.NativeAdminOAuth != "" && a.integration.NativeAdminOAuthRevision == c.Revision && a.integration.NativeAdminOAuthUntil.After(time.Now().Add(time.Minute)) {
		v := a.integration.NativeAdminOAuth
		a.mu.Unlock()
		return v, nil
	}
	a.mu.Unlock()
	key, err := nativeRSA(c.PrivateKey)
	if err != nil {
		return "", err
	}
	now := time.Now().Unix()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims, _ := json.Marshal(map[string]any{"iss": c.ServiceEmail, "scope": "https://www.googleapis.com/auth/firebase.messaging", "aud": "https://oauth2.googleapis.com/token", "iat": now, "exp": now + 3600})
	body := header + "." + base64.RawURLEncoding.EncodeToString(claims)
	sum := sha256.Sum256([]byte(body))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", errors.New("Firebase 인증 생성 실패")
	}
	assertion := body + "." + base64.RawURLEncoding.EncodeToString(signature)
	r, _ := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"}, "assertion": {assertion}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := a.httpClient().Do(r)
	if err != nil {
		return "", errors.New("Firebase 인증 서버에 연결하지 못했습니다.")
	}
	defer res.Body.Close()
	var out struct {
		AccessToken string `json:"access_token"`
		Expires     int    `json:"expires_in"`
	}
	if res.StatusCode != 200 || json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&out) != nil || out.AccessToken == "" {
		return "", fmt.Errorf("Firebase 인증 HTTP %d · 서비스 계정 권한을 확인하세요", res.StatusCode)
	}
	a.mu.Lock()
	if a.state.NativeAdminPush != nil && a.state.NativeAdminPush.Revision == c.Revision {
		a.integration.NativeAdminOAuth = out.AccessToken
		a.integration.NativeAdminOAuthRevision = c.Revision
		a.integration.NativeAdminOAuthUntil = time.Now().Add(time.Duration(min(max(out.Expires, 60), 3600)) * time.Second)
	}
	a.mu.Unlock()
	return out.AccessToken, nil
}
func (a *App) sendNativeAdmin(ctx context.Context, c NativeAdminPushConfig, d NativeAdminDevice, e Event, test bool) (bool, error) {
	payload := notification(e)
	title, body := "알파 관리자", nativePreview(text(payload["body"]))
	if test {
		title, body = "알파 관리자", "앱 알림 연결 테스트입니다."
	}
	data := map[string]string{"title": title, "body": body, "project_id": fmtID(e.ProjectID), "request_id": fmtID(e.ID), "tag": "alpha-admin-" + fmtID(e.ID)}
	if test {
		data["tag"] = "alpha-admin-test"
	}
	return a.sendNativeData(ctx, c, d, data)
}
func (a *App) sendNativeData(ctx context.Context, c NativeAdminPushConfig, d NativeAdminDevice, data map[string]string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	auth, err := a.nativeOAuth(ctx, c)
	if err != nil {
		return false, err
	}
	data["account_key"] = d.AccountID
	android := map[string]any{"priority": "HIGH", "ttl": "3600s"}
	if d.Role == "purchasing" {
		android["restricted_package_name"] = nativeAndroidPackage
	}
	raw, _ := json.Marshal(map[string]any{"message": map[string]any{"token": d.Token, "data": data, "android": android}})
	r, _ := http.NewRequestWithContext(ctx, "POST", "https://fcm.googleapis.com/v1/projects/"+c.Public.ProjectID+"/messages:send", strings.NewReader(string(raw)))
	r.Header.Set("Authorization", "Bearer "+auth)
	r.Header.Set("Content-Type", "application/json")
	res, err := a.httpClient().Do(r)
	if err != nil {
		return false, errors.New("앱 푸시 서버에 연결하지 못했습니다.")
	}
	defer res.Body.Close()
	io.Copy(io.Discard, io.LimitReader(res.Body, 1<<20))
	if res.StatusCode == 404 {
		return true, errors.New("앱 알림 등록이 만료됐습니다. 앱에 다시 로그인해 주세요.")
	}
	if res.StatusCode == 401 {
		a.mu.Lock()
		a.integration.NativeAdminOAuth = ""
		a.mu.Unlock()
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return false, fmt.Errorf("앱 알림 전송 HTTP %d", res.StatusCode)
	}
	return false, nil
}
func (a *App) pumpNativeAdmin(ctx context.Context) {
	a.mu.Lock()
	if a.testMode || !nativePushReady(a.state.NativeAdminPush) {
		a.mu.Unlock()
		return
	}
	cfg := *a.state.NativeAdminPush
	cfg.Devices = append([]NativeAdminDevice(nil), cfg.Devices...)
	events := append([]Event(nil), a.state.Events...)
	epoch := a.epoch
	a.mu.Unlock()
	sort.Slice(events, func(i, j int) bool { return events[i].ID < events[j].ID })
	for _, d := range cfg.Devices {
		if d.Role == "purchasing" || d.Role == "machining" {
			continue
		}
		a.mu.Lock()
		valid := nativeAccountValid(&a.state, d)
		a.mu.Unlock()
		if !valid {
			continue
		}
		delivered := d.Delivered
		lastStatus := ""
		disable := false
		testDone := false
		if d.TestPending {
			bad, err := a.sendNativeAdmin(ctx, cfg, d, Event{}, true)
			disable = bad
			if err != nil {
				lastStatus = err.Error()
			} else {
				lastStatus = "테스트 알림 전송 완료"
				testDone = true
			}
		}
		if lastStatus == "" || testDone {
			sent := 0
			for _, e := range events {
				if e.ID <= delivered {
					continue
				}
				if notifyEvent(e) && (e.BatchID == 0 || e.BatchID == e.ID) {
					bad, err := a.sendNativeAdmin(ctx, cfg, d, e, false)
					if err != nil {
						disable = bad
						lastStatus = err.Error()
						break
					}
					lastStatus = "새 검사·메시지 알림 전송 완료"
					sent++
				}
				delivered = e.ID
				if sent >= 32 {
					break
				}
			}
		}
		if delivered == d.Delivered && lastStatus == "" {
			continue
		}
		a.mu.Lock()
		current := nativeDevice(&a.state, d.ID)
		if current != nil && equal(current.SecretHash, d.SecretHash) && a.epoch == epoch && a.state.NativeAdminPush.Revision == cfg.Revision {
			a.commitAnnouncement(func(s *State) error {
				v := nativeDevice(s, d.ID)
				v.Delivered = max(v.Delivered, delivered)
				if lastStatus != "" {
					v.LastStatus = lastStatus
				}
				if testDone {
					v.TestPending = false
				}
				if disable {
					v.Enabled = false
				}
				return nil
			})
		}
		a.mu.Unlock()
	}
}

func (a *App) revokeNativeSession(session Session) error {
	hash := hashToken(session.CSRF)
	ids := map[string]bool{}
	for _, g := range a.state.NativeAdminLogins {
		if g.ID == session.NativeGrantID || equal(g.SessionHash, hash) {
			ids[g.ID] = true
		}
	}
	found := len(ids) > 0
	if a.state.NativeAdminPush != nil {
		for _, d := range a.state.NativeAdminPush.Devices {
			if d.Enabled && equal(d.SessionHash, hash) {
				found = true
			}
		}
	}
	if !found {
		return nil
	}
	err := a.commitAnnouncement(func(s *State) error {
		for i := range s.NativeAdminLogins {
			g := &s.NativeAdminLogins[i]
			if ids[g.ID] {
				disableNativeLogin(s, g)
			}
		}
		if s.NativeAdminPush != nil {
			for i := range s.NativeAdminPush.Devices {
				d := &s.NativeAdminPush.Devices[i]
				if equal(d.SessionHash, hash) {
					d.Enabled = false
					d.SecretHash = ""
					d.Token = ""
					d.LastStatus = "관리자 로그아웃"
				}
			}
		}
		return nil
	})
	if err == nil {
		a.clearNativeGrantSessions(ids)
	}
	return err
}

func nativePreview(value string) string {
	if len(value) <= 500 {
		return value
	}
	out := ""
	for _, r := range value {
		if len(out)+len(string(r)) > 497 {
			break
		}
		out += string(r)
	}
	return out + "…"
}

func nativeSessionRole(s *State, session Session) string {
	if machiningSession(s, session) {
		return "machining"
	}
	if purchasingSession(s, session) {
		return "purchasing"
	}
	return "admin"
}
