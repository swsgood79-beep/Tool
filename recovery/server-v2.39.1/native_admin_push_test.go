package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func nativeTestConfig(t *testing.T) (NativeAdminPushConfig, []byte, []byte, *rsa.PrivateKey) {
	t.Helper()
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	der, e := x509.MarshalPKCS8PrivateKey(key)
	if e != nil {
		t.Fatal(e)
	}
	rawKey := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	client := jsonRaw(map[string]any{"project_info": map[string]string{"project_id": "alpha-push-test", "project_number": "12345"}, "client": []any{map[string]any{"client_info": map[string]any{"mobilesdk_app_id": "1:12345:android:abcdef1234", "android_client_info": map[string]string{"package_name": nativeAndroidPackage}}, "api_key": []any{map[string]string{"current_key": strings.Repeat("A", 32)}}}}})
	service := jsonRaw(map[string]string{"type": "service_account", "project_id": "alpha-push-test", "client_email": "alpha@alpha-push-test.iam.gserviceaccount.com", "private_key": rawKey})
	cfg, e := nativeFirebaseFiles(client, service)
	if e != nil {
		t.Fatal(e)
	}
	cfg.Revision = 1
	return cfg, client, service, key
}
func nativePushFixture(t *testing.T) (*App, *http.Cookie, *rsa.PrivateKey) {
	t.Helper()
	a := accountFixture(t)
	a.state.Integration.MobileEnabled = true
	cfg, _, _, key := nativeTestConfig(t)
	a.state.NativeAdminPush = &cfg
	return a, adminCookie(a), key
}
func nativeCall(a *App, c *http.Cookie, path string, f url.Values) *httptest.ResponseRecorder {
	if f == nil {
		f = url.Values{}
	}
	f.Set("csrf", a.sessions[hashToken(c.Value)].CSRF)
	return call(a, "POST", path, f, c)
}
func nativeEnroll(t *testing.T, a *App, c *http.Cookie, id, value string) string {
	t.Helper()
	w := nativeCall(a, c, "/native-admin/register", url.Values{"device": {id}, "token": {value}, "name": {"검사팀 휴대폰"}})
	if w.Code != 200 {
		t.Fatal("enroll", w.Code, w.Body.String())
	}
	var j map[string]any
	json.Unmarshal(w.Body.Bytes(), &j)
	return j["secret"].(string)
}
func nativeConfigUpload(t *testing.T, a *App, c *http.Cookie, client, service []byte) *httptest.ResponseRecorder {
	t.Helper()
	var raw bytes.Buffer
	m := multipart.NewWriter(&raw)
	m.WriteField("csrf", a.sessions[hashToken(c.Value)].CSRF)
	m.WriteField("action", "configure")
	for name, value := range map[string][]byte{"google_services": client, "service_account": service} {
		part, _ := m.CreateFormFile(name, name+".json")
		part.Write(value)
	}
	m.Close()
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791/android-admin-settings", &raw)
	r.Header.Set("Content-Type", m.FormDataContentType())
	r.AddCookie(c)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func TestNativeAdminSetupPermissionsAndPersistence(t *testing.T) {
	a := accountFixture(t)
	a.state.Integration.MobileEnabled = true
	owner := adminCookie(a)
	_, client, service, _ := nativeTestConfig(t)
	before := string(jsonRaw(a.state.Projects))
	phone := string(jsonRaw(a.state.Integration.Phone))
	if w := call(a, "GET", "/native-admin/bootstrap", nil); w.Code != 303 || strings.Contains(w.Body.String(), "APIKey") {
		t.Fatal("unauthenticated bootstrap")
	}
	if w := nativeConfigUpload(t, a, owner, client, service); w.Code != 303 {
		t.Fatal("config", w.Code, w.Body.String())
	}
	if !nativePushReady(a.state.NativeAdminPush) {
		t.Fatal("config not ready")
	}
	createAccount(t, a, owner, "qa02")
	member, _ := accountLogin(t, a, "qa02", accountMemberPassword)
	if w := call(a, "GET", "/android-admin-settings", nil, member); w.Code != 403 {
		t.Fatal("member changed push config")
	}
	if w := call(a, "POST", "/native-admin/register", url.Values{"csrf": {"wrong"}}, owner); w.Code != 403 {
		t.Fatal("CSRF bypass")
	}
	id, tokenValue := token(), strings.Repeat("B", 80)
	secret := nativeEnroll(t, a, member, id, tokenValue)
	if len(secret) != 43 || strings.Contains(string(jsonRaw(a.state.NativeAdminPush)), secret) {
		t.Fatal("raw registration secret persisted")
	}
	for _, p := range []string{"/native-admin/bootstrap?device=" + id, "/android-admin-settings"} {
		w := call(a, "GET", p, nil, owner)
		body := w.Body.String()
		if strings.Contains(body, "PRIVATE KEY") || strings.Contains(body, tokenValue) || strings.Contains(body, secret) || strings.Contains(body, a.state.NativeAdminPush.Devices[0].SecretHash) {
			t.Fatal("private push information leaked", p)
		}
	}
	b := call(a, "GET", "/native-admin/bootstrap?device="+id, nil, member)
	if !strings.Contains(b.Body.String(), `"registered":true`) {
		t.Fatal("own registration absent")
	}
	b = call(a, "GET", "/native-admin/bootstrap?device="+id, nil, owner)
	if !strings.Contains(b.Body.String(), `"registered":false`) {
		t.Fatal("another account registration exposed")
	}
	fresh, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	if fresh.state.NativeAdminPush == nil || fresh.state.NativeAdminPush.PrivateKey == "" || len(fresh.state.NativeAdminPush.Devices) != 1 {
		t.Fatal("push settings not restored")
	}
	if string(jsonRaw(a.state.Projects)) != before || string(jsonRaw(a.state.Integration.Phone)) != phone {
		t.Fatal("push setup changed business or browser notification data")
	}
}
func TestNativeAdminAccountRevocationAndLogout(t *testing.T) {
	a, owner, _ := nativePushFixture(t)
	createAccount(t, a, owner, "qa02")
	member, _ := accountLogin(t, a, "qa02", accountMemberPassword)
	id := token()
	secret := nativeEnroll(t, a, member, id, strings.Repeat("C", 80))
	f := url.Values{"device": {id}, "secret": {secret}, "token": {strings.Repeat("D", 80)}}
	if w := call(a, "POST", "/native-admin/token-refresh", f); w.Code != 200 {
		t.Fatal("refresh failed", w.Code)
	}
	bad := url.Values{"device": {id}, "secret": {"wrong"}, "token": {strings.Repeat("D", 80)}}
	if w := call(a, "POST", "/native-admin/token-refresh", bad); w.Code != 403 {
		t.Fatal("refresh secret bypass")
	}
	account := adminAccountByUsername(&a.state, "qa02")
	if w := accountManageCall(a, owner, url.Values{"action": {"disable"}, "id": {account.ID}, "revision": {"1"}}); w.Code != 303 {
		t.Fatal(w.Code)
	}
	if w := call(a, "POST", "/native-admin/token-refresh", f); w.Code != 403 {
		t.Fatal("disabled member refreshed")
	}
	ownerDevice := token()
	ownerSecret := nativeEnroll(t, a, owner, ownerDevice, strings.Repeat("E", 80))
	if w := nativeCall(a, owner, "/logout", nil); w.Code != 303 {
		t.Fatal("logout", w.Code)
	}
	if nativeDevice(&a.state, ownerDevice).Enabled {
		t.Fatal("logout did not revoke its push device")
	}
	if w := call(a, "POST", "/native-admin/token-refresh", url.Values{"device": {ownerDevice}, "secret": {ownerSecret}, "token": {strings.Repeat("F", 80)}}); w.Code != 403 {
		t.Fatal("logged out device refreshed")
	}
	owner = adminCookie(a)
	id2 := token()
	nativeEnroll(t, a, owner, id2, strings.Repeat("G", 80))
	if w := nativeCall(a, owner, "/android-admin-settings", url.Values{"action": {"remove"}, "device": {id2}}); w.Code != 303 {
		t.Fatal(w.Code)
	}
	if w := nativeCall(a, owner, "/native-admin/register", url.Values{"device": {id2}, "token": {strings.Repeat("G", 80)}}); w.Code != 403 {
		t.Fatal("revoked device re-enrolled with same login")
	}
}

type nativeRoundTrip func(*http.Request) (*http.Response, error)

func (f nativeRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func nativeResponse(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
func TestNativeFCMDeliveryJWTFailureRetryAndDedup(t *testing.T) {
	a, owner, key := nativePushFixture(t)
	id := token()
	nativeEnroll(t, a, owner, id, strings.Repeat("H", 80))
	oauth, calls := 0, 0
	fail := true
	a.client = &http.Client{Transport: nativeRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "oauth2.googleapis.com" {
			oauth++
			r.ParseForm()
			jwt := strings.Split(r.FormValue("assertion"), ".")
			if len(jwt) != 3 {
				t.Fatal("invalid JWT")
			}
			raw := jwt[0] + "." + jwt[1]
			digest := sha256.Sum256([]byte(raw))
			sig, _ := base64.RawURLEncoding.DecodeString(jwt[2])
			if rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], sig) != nil {
				t.Fatal("JWT signature invalid")
			}
			claims, _ := base64.RawURLEncoding.DecodeString(jwt[1])
			if !bytes.Contains(claims, []byte("https://oauth2.googleapis.com/token")) {
				t.Fatal("bad audience")
			}
			return nativeResponse(200, `{"access_token":"mock-access","expires_in":3600}`), nil
		}
		if r.URL.Host != "fcm.googleapis.com" || r.Header.Get("Authorization") != "Bearer mock-access" {
			t.Fatal("unexpected FCM endpoint/auth")
		}
		raw, _ := io.ReadAll(r.Body)
		if len(raw) > 4096 {
			t.Fatal("push payload too large")
		}
		var payload map[string]any
		json.Unmarshal(raw, &payload)
		if payload["message"].(map[string]any)["notification"] != nil {
			t.Fatal("notification bypasses app logout suppression")
		}
		calls++
		if fail {
			return nativeResponse(503, `{}`), nil
		}
		return nativeResponse(200, `{"name":"projects/test/messages/1"}`), nil
	})}
	p := a.state.Projects[0]
	if e := a.requestInspection(p.ID, "combined", "작업자", p.SettingForm, p.CombinedForm); e != nil {
		t.Fatal(e)
	}
	before := nativeDevice(&a.state, id).Delivered
	a.pumpNativeAdmin(context.Background())
	if nativeDevice(&a.state, id).Delivered != before {
		t.Fatal("failed event marked delivered")
	}
	fail = false
	a.pumpNativeAdmin(context.Background())
	if calls != 2 || oauth != 1 || nativeDevice(&a.state, id).Delivered <= before {
		t.Fatal("delivery retry or batch dedup", calls, oauth)
	}
	a.pumpNativeAdmin(context.Background())
	if calls != 2 {
		t.Fatal("event replayed")
	}
	if e := a.commitAnnouncement(func(s *State) error {
		s.Events = append(s.Events, Event{ID: s.next(), ProjectID: p.ID, IsNote: true, Unread: true, Note: strings.Repeat("<", 3000)})
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	a.pumpNativeAdmin(context.Background())
	if calls != 3 {
		t.Fatal("note notification missing")
	}
}
func TestNativeFirebaseRejectsMismatchedFiles(t *testing.T) {
	_, client, service, _ := nativeTestConfig(t)
	for _, data := range [][]byte{[]byte(`{}`), bytes.ReplaceAll(service, []byte("alpha-push-test"), []byte("another-project")), bytes.ReplaceAll(service, []byte("service_account"), []byte("user"))} {
		if _, e := nativeFirebaseFiles(client, data); e == nil {
			t.Fatal("bad service account accepted")
		}
	}
	client = bytes.ReplaceAll(client, []byte(nativeAndroidPackage), []byte("wrong.package"))
	if _, e := nativeFirebaseFiles(client, service); e == nil {
		t.Fatal("wrong Android package accepted")
	}
}
