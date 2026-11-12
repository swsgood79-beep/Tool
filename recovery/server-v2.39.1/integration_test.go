package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"filippo.io/age"
	"fmt"
	"github.com/SherClockHolmes/webpush-go"
	"github.com/fxamacker/cbor/v2"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/http/httptest"

	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(r *http.Request, status int, v any) *http.Response {
	b, _ := json.Marshal(v)
	return &http.Response{StatusCode: status, Body: io.NopCloser(bytes.NewReader(b)), Header: http.Header{}, Request: r}
}
func TestLegacyImportPreservesOriginalFilesAndBoard(t *testing.T) {
	d := t.TempDir()
	pw, _ := bcrypt.GenerateFromPassword([]byte("old-password-1234"), bcrypt.MinCost)
	h := LegacyHost{Version: 14, BoardToken: token(), Projects: []LegacyOrder{{ID: 5, Revision: 8, InfoRevision: 4, Number: "OLD-001", Name: "원본 호환 시험", Active: true, Setting: LegacyStage{Worker: "작업자", Completed: time.Now().Format(time.RFC3339)}}}}
	h.MobileAdmin.PasswordHash = string(pw)
	h.MobileAdmin.Enabled = true
	b, _ := json.Marshal(h)
	cipher, _ := protect(b)
	path := filepath.Join(d, "projects-v2.dat")
	os.WriteFile(path, cipher, 0600)
	orig := digest(cipher)
	a, e := newApp(d)
	if e != nil {
		t.Fatal(e)
	}
	if a.state.Board != h.BoardToken || len(a.state.Projects) != 1 || a.state.Projects[0].Setting.Completed == nil || !a.passwordMatches("old-password-1234") {
		t.Fatal("legacy conversion failed")
	}
	if e = a.transaction(func(s *State) error { s.Projects[0].Name = "복구판 수정"; return nil }); e != nil {
		t.Fatal(e)
	}
	after, _ := os.ReadFile(path)
	if digest(after) != orig {
		t.Fatal("legacy original overwritten")
	}
	os.WriteFile(path, append(cipher, ' '), 0600)
	if e = a.transaction(func(s *State) error { return nil }); e == nil {
		t.Fatal("concurrent original write was ignored")
	}
}
func scheduleFixture(t *testing.T) []byte {
	t.Helper()
	b := excelize.NewFile()
	b.SetSheetName("Sheet1", "조립일정")
	values := [][]any{{"수주NO", "발송예정일"}, {"T-001", "2026-11-20"}, {"ASAP-1", "ASAP"}, {"CONFLICT", "2026-11-21"}, {"CONFLICT", "2026-11-22"}}
	for i, row := range values {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		b.SetSheetRow("조립일정", cell, &row)
	}
	buf, e := b.WriteToBuffer()
	b.Close()
	if e != nil {
		t.Fatal(e)
	}
	return buf.Bytes()
}
func TestScheduleDownloadApplyAndConflicts(t *testing.T) {
	a := fixture(t)
	data := scheduleFixture(t)
	dates, conflicts, e := parseSchedule(data)
	if e != nil || dates["ASAP-1"] != "" || !conflicts["CONFLICT"] {
		t.Fatal(e, dates, conflicts)
	}
	path := filepath.Join(a.dir, "schedule.xlsx")
	os.WriteFile(path, data, 0600)
	a.state.Integration.Schedule = ScheduleConfig{Enabled: true, Mode: "file", Path: path, Revision: 1}
	a.syncSchedule(context.Background(), true)
	if a.state.Projects[0].ShippingDueDate != "2026-11-20" {
		t.Fatal("schedule not applied")
	}
	s := a.scheduleJSON()
	if _, ok := s["status"].(map[string]any)["Changed"]; !ok {
		t.Fatal("original schedule JSON shape mismatch")
	}
	os.WriteFile(path, []byte("broken"), 0600)
	a.syncSchedule(context.Background(), true)
	if a.state.Projects[0].ShippingDueDate != "2026-11-20" {
		t.Fatal("failed schedule erased dates")
	}
	for _, raw := range []string{"http://onedrive.live.com/x", "https://onedrive.live.com.evil.example/x", "https://127.0.0.1/x"} {
		if scheduleURLAllowed(raw) {
			t.Fatal("unsafe schedule URL accepted", raw)
		}
	}
}
func TestUploadedScheduleIfProvided(t *testing.T) {
	path := os.Getenv("RECOVERY_SCHEDULE_SAMPLE")
	if path == "" {
		t.Skip("optional user supplied schedule is kept outside source")
	}
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	dates, conflicts, e := parseSchedule(b)
	if e != nil || len(dates) == 0 {
		t.Fatal("uploaded schedule headers not parsed", e)
	}
	t.Logf("parsed orders=%d conflicts=%d (contents withheld)", len(dates), len(conflicts))
}
func TestMarketAPIAndPreviousCloseCache(t *testing.T) {
	a := fixture(t)
	a.state.Integration.Market = MarketConfig{Enabled: true, ClientID: "test-client", ClientSecret: "test-secret", Revision: 1}
	calls := map[string]int{}
	a.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls[r.URL.Path]++
		switch {
		case r.URL.Host == "api.upbit.com":
			return response(r, 200, []any{map[string]any{"market": "KRW-BTC", "trade_price": 100000000, "signed_change_rate": 0.02, "timestamp": 1794440000000}}), nil
		case r.URL.Path == "/oauth2/token":
			return response(r, 200, map[string]any{"access_token": "not-a-real-token", "expires_in": 3600}), nil
		case strings.HasSuffix(r.URL.Path, "/prices"):
			return response(r, 200, []any{map[string]any{"symbol": "KOSPI", "lastPrice": "2100", "timestamp": "2026-11-12T10:00:00+09:00"}, map[string]any{"symbol": "KOSDAQ", "lastPrice": "900", "basePrice": 1000, "timestamp": "2026-11-12T10:00:00+09:00"}}), nil
		case strings.Contains(r.URL.Path, "/candles"):
			return response(r, 200, []any{map[string]any{"timestamp": "2026-11-11T09:00:00+09:00", "closePrice": "2000"}, map[string]any{"timestamp": "2026-11-12T09:00:00+09:00", "closePrice": "2100"}}), nil
		case strings.Contains(r.URL.Path, "exchange-rate"):
			return response(r, 200, map[string]any{"midRate": "1400", "validFrom": "2026-11-12T10:00:00+09:00", "validUntil": "2026-11-12T10:01:00+09:00"}), nil
		}
		return nil, fmt.Errorf("unexpected market URL %s", r.URL)
	})}
	a.refreshMarkets(context.Background())
	q := a.integration.MarketResult["quotes"].([]any)
	kospi := symbolRecord(q, "KOSPI")
	if pct, _ := numeric(kospi["percent"]); pct != 5 {
		t.Fatal("incorrect daily percent", kospi)
	}
	a.integration.MarketChecked = time.Time{}
	a.refreshMarkets(context.Background())
	if calls["/api/v1/market-indicators/KOSPI/candles"] != 1 || calls["/oauth2/token"] != 1 {
		t.Fatal("daily close/token not cached", calls)
	}
}
func TestEncryptedBackupRoundTripAndWrongKey(t *testing.T) {
	a := fixture(t)
	identity, _ := age.GenerateX25519Identity()
	key := RecoveryKey{1, repository, token(), identity.String()}
	a.state.Integration.Backup.Key = key
	a.state.Integration.Connection.GitHubToken = "TEST-ONLY-NOT-REAL"
	encrypted, e := a.portable(a.state)
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Contains(encrypted, []byte("TEST-ONLY")) {
		t.Fatal("backup plaintext credential leak")
	}
	m, entries, e := decodePortable(encrypted, key)
	if e != nil || len(m.State.Projects) != 1 || len(entries) == 0 {
		t.Fatal(e)
	}
	wrong, _ := age.GenerateX25519Identity()
	wrongKey := key
	wrongKey.Identity = wrong.String()
	if _, _, e = decodePortable(encrypted, wrongKey); e == nil {
		t.Fatal("wrong recovery key accepted")
	}
	fresh, e := newApp(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if e = fresh.restorePortable(key, encrypted); e != nil {
		t.Fatal(e)
	}
	if fresh.state.Board != a.state.Board {
		t.Fatal("restore changed worker QR")
	}
	if e = fresh.restorePortable(key, encrypted); e == nil {
		t.Fatal("restore overwrote live records")
	}
	encrypted[len(encrypted)/2] ^= 1
	if _, _, e = decodePortable(encrypted, key); e == nil {
		t.Fatal("corrupt encrypted backup accepted")
	}
}
func TestOriginalBackupManifestCompatibility(t *testing.T) {
	identity, _ := age.GenerateX25519Identity()
	key := RecoveryKey{1, repository, token(), identity.String()}
	ph := Photo{ID: token(), Name: "test.jpg", Width: 2, Height: 2}
	var pic bytes.Buffer
	jpeg.Encode(&pic, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil)
	ph.Size = pic.Len()
	h := LegacyHost{Version: 14, BoardToken: token(), Projects: []LegacyOrder{{ID: 3, Number: "OLD-1", Name: "원본 백업 시험", Active: true, Photos: []Photo{ph}}}}
	hb, _ := json.Marshal(h)
	old := OriginalManifest{Version: 2, BoxID: key.BoxID, Created: time.Now().Format(time.RFC3339), Host: hb}
	raw, _ := json.Marshal(old)
	var plain bytes.Buffer
	z := zip.NewWriter(&plain)
	f, _ := z.Create("manifest.json")
	f.Write(raw)
	f, _ = z.Create("photos/" + ph.ID + ".jpg")
	f.Write(pic.Bytes())
	z.Close()
	var encrypted bytes.Buffer
	w, _ := age.Encrypt(&encrypted, identity.Recipient())
	w.Write(plain.Bytes())
	w.Close()
	m, entries, e := decodePortable(encrypted.Bytes(), key)
	if e != nil || len(m.State.Projects) != 1 || len(entries["photos/"+ph.ID]) == 0 {
		t.Fatal("original manifest fixture failed", e)
	}
}
func jsonRequest(a *App, path string, body any, origin string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791"+path, bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", origin)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func decodeBody(t *testing.T, r *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var v map[string]any
	if e := json.Unmarshal(r.Body.Bytes(), &v); e != nil {
		t.Fatal(r.Code, r.Body.String())
	}
	return v
}
func authData(key *ecdsa.PrivateKey, id []byte, flags byte) []byte {
	rp := sha256.Sum256([]byte("swsgood79-beep.github.io"))
	b := append([]byte{}, rp[:]...)
	b = append(b, flags)
	counter := make([]byte, 4)
	counterValue := uint32(1)
	if flags&0x40 == 0 {
		counterValue = 2
	}
	binary.BigEndian.PutUint32(counter, counterValue)
	b = append(b, counter...)
	if flags&0x40 != 0 {
		b = append(b, make([]byte, 16)...)
		length := make([]byte, 2)
		binary.BigEndian.PutUint16(length, uint16(len(id)))
		b = append(b, length...)
		b = append(b, id...)
		cose, _ := cbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: key.X.FillBytes(make([]byte, 32)), -3: key.Y.FillBytes(make([]byte, 32))})
		b = append(b, cose...)
	}
	return b
}
func TestWebAuthnRealCryptoFlowAndReplay(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	a.state.Integration.MobileEnabled = true
	a.state.Integration.Passkeys.UserID = token()
	owner := hashToken(cookie.Value)
	start := token()
	a.integration.Starts[hashToken(start)] = PasskeyFlow{Mode: "register", Owner: owner, Revision: 0, Expires: time.Now().Add(time.Minute)}
	begin := jsonRequest(a, "/passkeys/register/begin", map[string]any{"token": start}, pagesOrigin)
	if begin.Code != 200 {
		t.Fatal(begin.Code, begin.Body.String())
	}
	d := decodeBody(t, begin)
	public := d["options"].(map[string]any)["publicKey"].(map[string]any)
	challenge := public["challenge"].(string)
	client, _ := json.Marshal(map[string]any{"type": "webauthn.create", "challenge": challenge, "origin": pagesOrigin, "crossOrigin": false})
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	id := []byte("test-credential-id-real-crypto")
	att, _ := cbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": authData(key, id, 0x45)})
	enc := base64.RawURLEncoding.EncodeToString
	credential := map[string]any{"id": enc(id), "rawId": enc(id), "type": "public-key", "response": map[string]any{"attestationObject": enc(att), "clientDataJSON": enc(client)}, "clientExtensionResults": map[string]any{}}
	finish := jsonRequest(a, "/passkeys/register/finish", map[string]any{"flow": d["flow"], "credential": credential}, pagesOrigin)
	if finish.Code != 200 {
		t.Fatal(finish.Code, finish.Body.String())
	}
	if len(a.state.Integration.Passkeys.Credentials) != 1 {
		t.Fatal("credential not saved")
	}
	if replay := jsonRequest(a, "/passkeys/register/finish", map[string]any{"flow": d["flow"], "credential": credential}, pagesOrigin); replay.Code != 403 {
		t.Fatal("registration replay accepted")
	}
	start = token()
	a.integration.Starts[hashToken(start)] = PasskeyFlow{Mode: "auth", Revision: 0, Expires: time.Now().Add(time.Minute), Next: "/projects/1/inspection"}
	begin = jsonRequest(a, "/passkeys/auth/begin", map[string]any{"token": start}, pagesOrigin)
	if begin.Code != 200 {
		t.Fatal(begin.Code, begin.Body.String())
	}
	d = decodeBody(t, begin)
	public = d["options"].(map[string]any)["publicKey"].(map[string]any)
	client, _ = json.Marshal(map[string]any{"type": "webauthn.get", "challenge": public["challenge"], "origin": pagesOrigin, "crossOrigin": false})
	auth := authData(key, id, 0x05)
	hash := sha256.Sum256(client)
	signed := append(append([]byte{}, auth...), hash[:]...)
	msgHash := sha256.Sum256(signed)
	signature, _ := ecdsa.SignASN1(rand.Reader, key, msgHash[:])
	credential["response"] = map[string]any{"clientDataJSON": enc(client), "authenticatorData": enc(auth), "signature": enc(signature), "userHandle": nil}
	// A cryptographically signed assertion without UV must be rejected.
	badFlow := token()
	a.integration.Flows[hashToken(badFlow)] = a.integration.Flows[hashToken(d["flow"].(string))]
	badAuth := authData(key, id, 0x01)
	badSigned := append(append([]byte{}, badAuth...), hash[:]...)
	badHash := sha256.Sum256(badSigned)
	badSignature, _ := ecdsa.SignASN1(rand.Reader, key, badHash[:])
	badCredential := map[string]any{"id": enc(id), "rawId": enc(id), "type": "public-key", "response": map[string]any{"clientDataJSON": enc(client), "authenticatorData": enc(badAuth), "signature": enc(badSignature), "userHandle": nil}, "clientExtensionResults": map[string]any{}}
	if bad := jsonRequest(a, "/passkeys/auth/finish", map[string]any{"flow": badFlow, "credential": badCredential}, pagesOrigin); bad.Code != 403 {
		t.Fatal("signed assertion without user verification accepted", bad.Code)
	}
	finish = jsonRequest(a, "/passkeys/auth/finish", map[string]any{"flow": d["flow"], "credential": credential}, pagesOrigin)
	if finish.Code != 200 {
		t.Fatal(finish.Code, finish.Body.String())
	}
	ticket := decodeBody(t, finish)["ticket"].(string)
	handoff := jsonRequest(a, "/mobile-admin/passkey/finish", map[string]any{"token": ticket}, "http://127.0.0.1:18791")
	if handoff.Code != 200 || len(handoff.Result().Cookies()) == 0 {
		t.Fatal("handoff failed", handoff.Code, handoff.Body.String())
	}
	if again := jsonRequest(a, "/mobile-admin/passkey/finish", map[string]any{"token": ticket}, "http://127.0.0.1:18791"); again.Code != 403 {
		t.Fatal("handoff replay accepted")
	}
	if bad := jsonRequest(a, "/passkeys/auth/begin", map[string]any{"token": "invalid"}, "https://evil.example"); bad.Code != 403 {
		t.Fatal("wrong origin accepted")
	}
}
func TestPushPayloadAndHostWhitelist(t *testing.T) {
	a := fixture(t)
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	pub := elliptic.Marshal(elliptic.P256(), key.X, key.Y)
	auth := make([]byte, 16)
	rand.Read(auth)
	private, public, e := webpush.GenerateVAPIDKeys()
	if e != nil {
		t.Fatal(e)
	}
	sub := webpush.Subscription{Endpoint: "https://fcm.googleapis.com/fcm/send/test", Keys: webpush.Keys{P256dh: base64.RawURLEncoding.EncodeToString(pub), Auth: base64.RawURLEncoding.EncodeToString(auth)}}
	if !validSubscription(sub) {
		t.Fatal("valid fixture subscription rejected")
	}
	bad := sub
	bad.Endpoint = "https://127.0.0.1/private"
	if validSubscription(bad) {
		t.Fatal("SSRF endpoint accepted")
	}
	a.state.Integration.Phone = PhoneData{Version: 1, PublicKey: public, PrivateKey: private, Subscription: &sub, Generation: 1}
	p := a.state.Projects[0]
	a.requestInspection(p.ID, "combined", "시험 작업자", p.SettingForm, p.CombinedForm)
	sent := 0
	a.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "fcm.googleapis.com" {
			t.Fatal("unexpected push target")
		}
		b, _ := io.ReadAll(r.Body)
		if bytes.Contains(b, []byte("시험 작업자")) || len(b) < 100 || r.Header.Get("Authorization") == "" {
			t.Fatal("push was not encrypted/authenticated")
		}
		sent++
		return response(r, 201, map[string]any{}), nil
	})}
	a.pumpNotifications(context.Background())
	if sent != 1 || a.state.Integration.Phone.Delivered == 0 {
		t.Fatal("combined notification duplicate/missing", sent)
	}
	a.pumpNotifications(context.Background())
	if sent != 1 {
		t.Fatal("notification replayed")
	}
}
func TestPublicHostAndDeepLink(t *testing.T) {
	a := fixture(t)
	a.state.Integration.Connection.PublicURL = "https://test-live.trycloudflare.com"
	r := httptest.NewRequest("GET", "https://test-live.trycloudflare.com/work/"+a.state.Board, nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, p := range []string{"https://evil.example", "//evil.example", "/projects/1/inspection?request=2"} {
		n := safeNext(p)
		if p[0] == '/' && !strings.HasPrefix(p, "//") {
			if n != p {
				t.Fatal("valid review link lost")
			}
		} else if n != "/" {
			t.Fatal("open redirect allowed")
		}
	}
	if s := call(a, "GET", "/mobile-admin/passkey/start", nil); s.Code != 403 {
		t.Fatal("unregistered passkey button logged in", s.Code)
	}
}

func TestGitHubUploadVerificationAndPointerOwnership(t *testing.T) {
	a := fixture(t)
	identity, _ := age.GenerateX25519Identity()
	a.state.Integration.Backup.Key = RecoveryKey{1, repository, token(), identity.String()}
	a.state.Integration.Connection.GitHubToken = "TEST-ONLY"
	a.state.Integration.Connection.Provider = "pages"
	a.state.Integration.Connection.PublicURL = "https://new-test.trycloudflare.com"
	a.integration.TunnelSession = token()
	var uploaded []byte
	var uploadedName string
	puts := 0
	deletes := 0
	a.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "uploads.github.com" {
			uploaded, _ = io.ReadAll(r.Body)
			uploadedName = r.URL.Query().Get("name")
			if bytes.Contains(uploaded, []byte("TEST-ONLY")) {
				t.Fatal("unencrypted backup upload")
			}
			return response(r, 201, map[string]any{"id": 77}), nil
		}
		if strings.Contains(r.URL.Path, "/releases/tags/") {
			assets := []any{}
			if uploaded != nil {
				assets = append(assets, map[string]any{"id": 77, "name": uploadedName, "label": "recovered-v2", "created_at": "2026-11-12T12:00:00Z"})
			}
			return response(r, 200, map[string]any{"id": 22, "body": "wanryotalk-reconstructed-v2", "assets": assets}), nil
		}
		if strings.HasSuffix(r.URL.Path, "/releases/assets/77") {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(uploaded)), Header: http.Header{}, Request: r}, nil
		}
		if strings.HasSuffix(r.URL.Path, "/qr-endpoint.json") {
			if r.Method == "GET" {
				old, _ := json.Marshal(map[string]any{"session": "other-process"})
				return response(r, 200, map[string]any{"sha": "abc", "content": base64.StdEncoding.EncodeToString(old)}), nil
			}
			if r.Method == "PUT" {
				puts++
				b, _ := io.ReadAll(r.Body)
				var payload map[string]any
				json.Unmarshal(b, &payload)
				plain, _ := base64.StdEncoding.DecodeString(text(payload["content"]))
				if bytes.Contains(plain, []byte(a.state.Board)) {
					t.Fatal("worker capability leaked into pointer")
				}
				return response(r, 200, map[string]any{}), nil
			}
		}
		if r.Method == "DELETE" {
			deletes++
			return response(r, 204, map[string]any{}), nil
		}
		return nil, fmt.Errorf("unexpected GitHub operation %s %s", r.Method, r.URL)
	})}
	a.runBackup(context.Background())
	if a.state.Integration.Backup.LastSuccess == "" || len(uploaded) == 0 {
		t.Fatal(a.integration.BackupStatus)
	}
	if e := a.publishPointer(context.Background(), false); e != nil {
		t.Fatal(e)
	}
	if puts != 0 {
		t.Fatal("offline update clobbered another session")
	}
	if e := a.publishPointer(context.Background(), true); e != nil || puts != 1 {
		t.Fatal("online publication failed", e)
	}
	var rel Release
	data := `{"id":1,"body":"wanryotalk-reconstructed-v2","assets":[{"id":9,"name":"alpha-new.age","label":"recovered-v2"},{"id":8,"name":"alpha-8.age","label":"recovered-v2"},{"id":7,"name":"alpha-7.age","label":"recovered-v2"},{"id":6,"name":"alpha-6.age","label":"recovered-v2"},{"id":5,"name":"original.age","label":"alpha-verified-v1"}]}`
	json.Unmarshal([]byte(data), &rel)
	if e := a.pruneBackups(context.Background(), rel, "TEST-ONLY", "alpha-new.age"); e != nil || deletes != 1 {
		t.Fatal("retention touched wrong assets", e, deletes)
	}
}

func TestMinuteFallbackCacheAndRetryAfter(t *testing.T) {
	a := fixture(t)
	calls := 0
	a.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return response(r, 200, map[string]any{"result": map[string]any{"candles": []any{map[string]any{"timestamp": "2026-11-11T15:30:00+09:00", "closePrice": "2105.50"}}}}), nil
	})}
	p, at, e := a.minuteIndex(context.Background(), "KOSPI", "test-token")
	if e != nil || p != 2105.5 || at == "" {
		t.Fatal(e, p, at)
	}
	a.minuteIndex(context.Background(), "KOSPI", "test-token")
	if calls != 1 {
		t.Fatal("minute fallback not cached")
	}
	a.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		res := response(r, 429, map[string]any{})
		res.Header.Set("Retry-After", "30")
		return res, nil
	})}
	if _, e = a.getJSON(context.Background(), "GET", "https://openapi.tossinvest.com/api/v1/market-indicators/prices", "test-token", nil); e == nil || time.Until(a.integration.MarketNext) < 29*time.Second {
		t.Fatal("429 backoff missing", e)
	}
}
