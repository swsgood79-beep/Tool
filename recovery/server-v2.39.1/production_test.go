package main

import (
	"bytes"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionCopiesOriginalAndKeepsSettingsAndBoard(t *testing.T) {
	original := fixture(t)
	original.transaction(func(s *State) error {
		s.Integration.Connection = ConnectionConfig{Provider: "pages", GitHubToken: "fixture-token"}
		s.Integration.Market.Enabled = true
		return nil
	})
	before := originalBytes(t, original.dir)
	base := t.TempDir()
	app, e := startProductionApp(base, original.dir, false, original.dir)
	if e != nil {
		t.Fatal(e)
	}
	if app.testMode || app.state.Board != original.state.Board || app.state.Integration.Connection.GitHubToken != "fixture-token" || !app.state.Integration.Market.Enabled {
		t.Fatal("production settings reset or test mode retained")
	}
	assertOriginalBytes(t, original.dir, before)
	if app.dir == original.dir {
		t.Fatal("production used original folder")
	}
	if _, e := prepareTestDir(app.dir); e == nil {
		t.Fatal("test mode accepted production folder")
	}
	app.transaction(func(s *State) error { s.Projects[0].Name = "운영 수정"; return nil })
	next, e := startProductionApp(base, "missing-original", false, "missing-original")
	if e != nil || next.dir != app.dir || next.state.Projects[0].Name != "운영 수정" {
		t.Fatal("production did not resume", e)
	}
	assertOriginalBytes(t, original.dir, before)
}
func TestProductionRejectsOriginalTestAndUnsafeSelection(t *testing.T) {
	source := fixture(t)
	if _, e := prepareProductionDir(source.dir); e == nil {
		t.Fatal("unmarked original accepted")
	}
	test, e := newTestApp(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if _, e := prepareProductionDir(test.dir); e == nil {
		t.Fatal("test folder accepted as production")
	}
	base := t.TempDir()
	prepareProductionDir(base)
	os.WriteFile(filepath.Join(base, productionSelection), []byte(`{"Version":1,"Folder":"../../WanryoTalk"}`), 0600)
	if _, e := selectedProductionDir(base); e == nil {
		t.Fatal("production path escape accepted")
	}
	if _, e := prepareProductionDir(filepath.Join(t.TempDir(), "WanryoTalk")); e == nil {
		t.Fatal("reserved original folder accepted")
	}
}
func TestPromoteTestKeepsBusinessButRestoresOriginalConnections(t *testing.T) {
	source := fixture(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("original-password-123"), bcrypt.MinCost)
	source.transaction(func(s *State) error {
		s.PasswordHash = string(hash)
		s.Integration.Connection.GitHubToken = "original-fixture-key"
		s.Integration.MobileEnabled = true
		return nil
	})
	test, e := newTestApp(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	test.transaction(func(s *State) error { s.Projects[0].Name = "테스트에서 확인한 수정"; return nil })
	before := originalBytes(t, source.dir)
	testBefore := originalBytes(t, test.dir)
	base := t.TempDir()
	app, e := startProductionApp(base, test.dir, true, source.dir)
	if e != nil {
		t.Fatal(e)
	}
	if len(app.state.Projects) != 2 || app.state.Projects[0].Name != "테스트에서 확인한 수정" || app.state.Board != source.state.Board || app.state.PasswordHash != string(hash) || app.state.Integration.Connection.GitHubToken != "original-fixture-key" {
		t.Fatal("promotion lost selected business data or original settings")
	}
	if app.state.Integration.Phone.Delivered != app.state.NextID {
		t.Fatal("promotion would resend history or skip new push IDs")
	}
	assertOriginalBytes(t, source.dir, before)
	assertOriginalBytes(t, test.dir, testBefore)
	if _, e = startProductionApp(base, test.dir, true, source.dir); e == nil {
		t.Fatal("promoting again overwrote active production")
	}
}
func TestProductionEmptyRestoreStartDoesNotReplaceExistingData(t *testing.T) {
	base := t.TempDir()
	app, e := startEmptyProductionApp(base)
	if e != nil || len(app.state.Projects) != 0 || app.testMode {
		t.Fatal("empty restoration start failed", e)
	}
	if _, e = startEmptyProductionApp(base); e == nil {
		t.Fatal("empty start replaced production")
	}
	before := originalBytes(t, app.dir)
	if _, e = startProductionApp(base, "missing", true, "missing"); e == nil {
		t.Fatal("promotion overwrote empty-but-selected data")
	}
	assertOriginalBytes(t, app.dir, before)
}
func TestProductionEndpointsOpenButTestStillBlocksServices(t *testing.T) {
	app := fixture(t)
	cookie := adminCookie(app)
	for _, route := range []string{"/connection", "/phone", "/backup", "/mobile-access", "/mobile-admin/passkey"} {
		if r := call(app, "GET", route, nil, cookie); r.Code != 200 {
			t.Fatal("production route blocked", route, r.Code)
		}
	}
	if strings.Contains(app.modeNotice(), "外部") || strings.Contains(app.modeNotice(), "외부 연동 꺼짐") {
		t.Fatal("production banner says services blocked")
	}
	test, e := newTestApp(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	for _, route := range []string{"/connection", "/phone", "/backup", "/mobile-access"} {
		if r := call(test, "GET", route, nil); r.Code != 403 {
			t.Fatal("test route no longer blocked", route)
		}
	}
	a, b := desktopIdentity(true)
	c, d := desktopIdentity(false)
	if a == c || b == d || c != `Local\WanryoTalkNotifier-v1` || d != "WanryoTalkMain" {
		t.Fatal("desktop identities not separated")
	}
}
func TestTestNativeHandoffWorksWithoutOpeningExternalPasskeyLogin(t *testing.T) {
	a, e := newTestApp(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("native-test-password"), bcrypt.MinCost)
	a.state.PasswordHash = string(hash)
	a.localURL = "http://127.0.0.1:28791"
	u, _ := url.Parse(a.nativeAdminURL("/"))
	ticket := strings.TrimPrefix(u.Fragment, "ticket=")
	r := httptest.NewRequest("POST", a.localURL+"/mobile-admin/passkey/finish", bytes.NewReader(jsonRaw(map[string]string{"token": ticket})))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != 200 || len(w.Result().Cookies()) == 0 || w.Result().Cookies()[0].Name != a.sessionCookie() {
		t.Fatal("valid native ticket blocked in test mode", w.Code, w.Body.String())
	}
	if r := call(a, "GET", "/mobile-admin/passkey/start", nil); r.Code != http.StatusForbidden {
		t.Fatal("external passkey route opened in test mode")
	}
}

func TestTestCopiesUseLatestProductionAfterCutover(t *testing.T) {
	home := t.TempDir()
	source := fixture(t)
	app, e := startProductionApp(filepath.Join(home, productionFolderName), source.dir, false, source.dir)
	if e != nil {
		t.Fatal(e)
	}
	selected, e := latestBusinessSource(home)
	if e != nil || selected != app.dir {
		t.Fatal("test copy would use stale original after cutover", e)
	}
	if _, e := latestBusinessSource(t.TempDir()); e != nil {
		t.Fatal(e)
	}
}
