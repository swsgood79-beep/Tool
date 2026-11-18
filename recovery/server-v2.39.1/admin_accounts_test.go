package main

import (
	"bytes"
	"filippo.io/age"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

const accountOwnerPassword = "owner-password-123"
const accountMemberPassword = "member-password-123"

func accountFixture(t *testing.T) *App {
	t.Helper()
	a := fixture(t)
	hash, err := bcrypt.GenerateFromPassword([]byte(accountOwnerPassword), 12)
	if err != nil {
		t.Fatal(err)
	}
	if err = a.commitAnnouncement(func(s *State) error { s.PasswordHash = string(hash); return nil }); err != nil {
		t.Fatal(err)
	}
	return a
}
func accountManageCall(a *App, owner *http.Cookie, v url.Values) *httptest.ResponseRecorder {
	v.Set("csrf", a.sessions[hashToken(owner.Value)].CSRF)
	return call(a, "POST", "/admin-accounts", v, owner)
}
func createAccount(t *testing.T, a *App, owner *http.Cookie, username string) *AdminAccount {
	t.Helper()
	w := accountManageCall(a, owner, url.Values{"action": {"create"}, "name": {"검사자 " + username}, "username": {username}, "password": {accountMemberPassword}, "confirm": {accountMemberPassword}})
	if w.Code != 303 {
		t.Fatal("create", w.Code, w.Body.String())
	}
	return adminAccountByUsername(&a.state, strings.ToLower(username))
}
func accountLogin(t *testing.T, a *App, username, password string) (*http.Cookie, *httptest.ResponseRecorder) {
	t.Helper()
	w := call(a, "GET", "/login", nil)
	m := regexp.MustCompile(`name="csrf" value="([^"]+)"`).FindStringSubmatch(w.Body.String())
	if len(m) != 2 {
		t.Fatal("login nonce missing")
	}
	var nonce *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == a.nonceCookie() {
			nonce = c
		}
	}
	form := url.Values{"csrf": {m[1]}, "password": {password}}
	if username != "" {
		form.Set("username", username)
		form.Set("login_mode", "account")
	}
	w = call(a, "POST", "/login", form, nonce)
	for _, c := range w.Result().Cookies() {
		if c.Name == a.sessionCookie() {
			return c, w
		}
	}
	return nil, w
}
func TestSeparateAdminAccountsLoginAndOwnerCompatibility(t *testing.T) {
	a := accountFixture(t)
	owner := adminCookie(a)
	ownerHash := a.state.PasswordHash
	projects, events := jsonRaw(a.state.Projects), jsonRaw(a.state.Events)
	book, _ := os.ReadFile(filepath.Join(a.dir, "검사완료기록-복구서버.xlsx"))
	u := createAccount(t, a, owner, "qa02")
	member, w := accountLogin(t, a, "QA02", accountMemberPassword)
	if member == nil || w.Code != 303 {
		t.Fatal("member login", w.Code)
	}
	session := a.sessions[hashToken(member.Value)]
	if session.AccountID != u.ID || session.AccountRevision != 1 {
		t.Fatal("member became owner")
	}
	if w = call(a, "GET", "/", nil, member); w.Code != 200 || !strings.Contains(w.Body.String(), "검사자 qa02") {
		t.Fatal("shared admin view", w.Code)
	}
	legacy, w := accountLogin(t, a, "", accountOwnerPassword)
	if legacy == nil || w.Code != 303 || a.sessions[hashToken(legacy.Value)].AccountID != "" {
		t.Fatal("legacy owner login broken")
	}
	for _, pair := range [][2]string{{"qa02", accountOwnerPassword}, {"missing", accountMemberPassword}, {"qa02", "wrong-password-123"}} {
		cookie, result := accountLogin(t, a, pair[0], pair[1])
		if cookie != nil || !strings.Contains(result.Body.String(), "아이디 또는 비밀번호") {
			t.Fatal("wrong account accepted or disclosed")
		}
	}
	if a.state.PasswordHash != ownerHash || !bytes.Equal(projects, jsonRaw(a.state.Projects)) || !bytes.Equal(events, jsonRaw(a.state.Events)) {
		t.Fatal("account changed business data or owner password")
	}
	after, _ := os.ReadFile(filepath.Join(a.dir, "검사완료기록-복구서버.xlsx"))
	if !bytes.Equal(book, after) {
		t.Fatal("account management rewrote inspection workbook")
	}
	if raw, _ := os.ReadFile(filepath.Join(a.dir, "projects-v2.dat")); bytes.Contains(raw, []byte(accountMemberPassword)) {
		t.Fatal("plaintext password persisted")
	}
	if !strings.HasPrefix(a.state.AdminAccounts[0].PasswordHash, "$2") {
		t.Fatal("member password unhashed")
	}
}
func TestAccountResetAndDisableOnlyTargetSessions(t *testing.T) {
	a := accountFixture(t)
	owner := adminCookie(a)
	u := createAccount(t, a, owner, "qa02")
	id := u.ID
	createAccount(t, a, owner, "qa03")
	one, _ := accountLogin(t, a, "qa02", accountMemberPassword)
	two, _ := accountLogin(t, a, "qa02", accountMemberPassword)
	other, _ := accountLogin(t, a, "qa03", accountMemberPassword)
	w := accountManageCall(a, owner, url.Values{"action": {"reset"}, "id": {id}, "revision": {"1"}, "password": {"new-password-123"}, "confirm": {"new-password-123"}})
	if w.Code != 303 {
		t.Fatal("reset", w.Code, w.Body.String())
	}
	for _, c := range []*http.Cookie{one, two} {
		if call(a, "GET", "/", nil, c).Code != 303 {
			t.Fatal("old member session still active")
		}
	}
	for _, c := range []*http.Cookie{owner, other} {
		if call(a, "GET", "/", nil, c).Code != 200 {
			t.Fatal("reset logged out another account")
		}
	}
	fresh, _ := accountLogin(t, a, "qa02", "new-password-123")
	if fresh == nil {
		t.Fatal("new password failed")
	}
	w = accountManageCall(a, owner, url.Values{"action": {"disable"}, "id": {id}, "revision": {"2"}})
	if w.Code != 303 || call(a, "GET", "/", nil, fresh).Code != 303 {
		t.Fatal("disable did not revoke")
	}
	if c, _ := accountLogin(t, a, "qa02", "new-password-123"); c != nil {
		t.Fatal("disabled account login")
	}
	if call(a, "GET", "/", nil, other).Code != 200 {
		t.Fatal("disable affected other account")
	}
	w = accountManageCall(a, owner, url.Values{"action": {"enable"}, "id": {id}, "revision": {"3"}})
	if w.Code != 303 {
		t.Fatal("enable")
	}
	if c, _ := accountLogin(t, a, "qa02", "new-password-123"); c == nil {
		t.Fatal("re-enabled account cannot login")
	}
}
func TestAccountOwnerBoundariesAndSelfPassword(t *testing.T) {
	a := accountFixture(t)
	owner := adminCookie(a)
	createAccount(t, a, owner, "qa02")
	createAccount(t, a, owner, "qa03")
	member, _ := accountLogin(t, a, "qa02", accountMemberPassword)
	second, _ := accountLogin(t, a, "qa02", accountMemberPassword)
	other, _ := accountLogin(t, a, "qa03", accountMemberPassword)
	s := a.sessions[hashToken(member.Value)]
	if w := call(a, "GET", "/projects/new", nil, member); w.Code != 200 || !strings.Contains(w.Body.String(), `data-owner="false"`) {
		t.Fatal("named identity lost on order form")
	}
	for _, path := range []string{"/admin-accounts", "/mobile-access", "/mobile-admin/passkey", "/mobile-admin/passkey/enroll", "/settings", "/backup", "/connection", "/data-storage"} {
		for _, method := range []string{"GET", "POST"} {
			w := call(a, method, path, url.Values{"csrf": {s.CSRF}}, member)
			if w.Code != 403 {
				t.Fatal("member owner-path bypass", method, path, w.Code)
			}
		}
	}
	if w := call(a, "POST", "/my-password", url.Values{"csrf": {"wrong"}}, member); w.Code != 403 {
		t.Fatal("self password CSRF missing")
	}
	w := call(a, "POST", "/my-password", url.Values{"csrf": {s.CSRF}, "current_password": {accountMemberPassword}, "password": {"self-password-123"}, "confirm": {"self-password-123"}}, member)
	if w.Code != 303 {
		t.Fatal("self password", w.Code, w.Body.String())
	}
	var newCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == a.sessionCookie() {
			newCookie = c
		}
	}
	if newCookie == nil || call(a, "GET", "/my-password", nil, newCookie).Code != 200 {
		t.Fatal("self password lost active session")
	}
	if call(a, "GET", "/", nil, second).Code != 303 {
		t.Fatal("other device of same account remains")
	}
	if call(a, "GET", "/", nil, other).Code != 200 || call(a, "GET", "/", nil, owner).Code != 200 {
		t.Fatal("self password revoked another account")
	}
	// A named session can never turn into the legacy owner via a handoff.
	principal := a.sessions[hashToken(newCookie.Value)]
	ticket := token()
	a.integration.Handoffs[hashToken(ticket)] = Handoff{Owner: hashToken(newCookie.Value), Expires: time.Now().Add(time.Minute), Next: "/"}
	body := strings.NewReader(string(jsonRaw(map[string]string{"token": ticket})))
	request := httptest.NewRequest("POST", "http://127.0.0.1:18791/mobile-admin/passkey/finish", body)
	request.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, request)
	if rec.Code == 200 {
		for _, c := range rec.Result().Cookies() {
			if c.Name == a.sessionCookie() && a.sessions[hashToken(c.Value)].AccountID != principal.AccountID {
				t.Fatal("handoff elevated named account")
			}
		}
	}
}
func TestAccountPersistenceBackupAndLegacyHashIsolation(t *testing.T) {
	a := accountFixture(t)
	owner := adminCookie(a)
	createAccount(t, a, owner, "qa02")
	// Named login must never migrate the legacy owner's password to the member's.
	legacyHash, err := passwordHash(accountOwnerPassword, "old-salt")
	if err != nil {
		t.Fatal(err)
	}
	if err = a.commitAnnouncement(func(s *State) error { s.PasswordHash = legacyHash; s.Salt = "old-salt"; return nil }); err != nil {
		t.Fatal(err)
	}
	if c, _ := accountLogin(t, a, "qa02", accountMemberPassword); c == nil {
		t.Fatal("member login")
	}
	if a.state.PasswordHash != legacyHash {
		t.Fatal("member login overwrote owner password")
	}
	fresh, err := newApp(a.dir)
	if err != nil {
		t.Fatal(err)
	}
	if c, _ := accountLogin(t, fresh, "qa02", accountMemberPassword); c == nil {
		t.Fatal("restart account lost")
	}
	if c, _ := accountLogin(t, fresh, "", accountOwnerPassword); c == nil {
		t.Fatal("owner legacy migration broken")
	}
	identity, _ := age.GenerateX25519Identity()
	key := RecoveryKey{1, repository, token(), identity.String()}
	fresh.state.Integration.Backup.Key = key
	encrypted, err := fresh.portable(fresh.state)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := newApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = restored.restorePortable(key, encrypted); err != nil {
		t.Fatal(err)
	}
	if c, _ := accountLogin(t, restored, "qa02", accountMemberPassword); c == nil {
		t.Fatal("backup account lost")
	}
}
