package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"image"
	"image/jpeg"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func readOriginal(t *testing.T, dir string) LegacyHost {
	t.Helper()
	var h LegacyHost
	if e := legacyJSON(filepath.Join(dir, "projects-v2.dat"), &h); e != nil {
		t.Fatal(e)
	}
	return h
}
func originalBytes(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	for _, n := range originalFiles {
		b, e := os.ReadFile(filepath.Join(dir, n))
		if e != nil {
			t.Fatal(e)
		}
		out[n] = b
	}
	return out
}
func assertOriginalBytes(t *testing.T, dir string, expected map[string][]byte) {
	t.Helper()
	for n, want := range expected {
		got, e := os.ReadFile(filepath.Join(dir, n))
		if e != nil || !bytes.Equal(got, want) {
			t.Fatalf("original %s changed after rollback: %v", n, e)
		}
	}
}
func crashWrite(a *App, phase, name string) {
	a.directHook = func(p, n string) error {
		if phase == p && (name == "" || n == name) {
			panic("simulated process exit")
		}
		return nil
	}
	defer func() { recover() }()
	_ = a.transaction(func(s *State) error { s.Projects[0].Name = "after crash"; return nil })
}
func TestDirectJournalCrashRollbackAtEveryFile(t *testing.T) {
	for _, n := range originalFiles {
		t.Run(n, func(t *testing.T) {
			a := fixture(t)
			before := originalBytes(t, a.dir)
			crashWrite(a, "written", n)
			if e := recoverDirect(a.dir); e != nil {
				t.Fatal(e)
			}
			assertOriginalBytes(t, a.dir, before)
			b, e := newApp(a.dir)
			if e != nil {
				t.Fatal(e)
			}
			if b.state.Projects[0].Name != "복원 서버 자동시험" {
				t.Fatal("uncommitted state escaped")
			}
			if _, e = os.Stat(filepath.Join(a.dir, journalName)); !os.IsNotExist(e) {
				t.Fatal("journal not cleaned")
			}
		})
	}
}
func TestDirectJournalCrashAfterCommitKeepsNewState(t *testing.T) {
	a := fixture(t)
	crashWrite(a, "committed", "")
	b, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	if b.state.Projects[0].Name != "after crash" || readOriginal(t, a.dir).Projects[0].Name != "after crash" {
		t.Fatal("committed state lost")
	}
}
func TestDirectJournalErrorRollbackAndRetry(t *testing.T) {
	a := fixture(t)
	before := originalBytes(t, a.dir)
	a.directHook = func(p, n string) error {
		if p == "written" && n == "designers-v1.dat" {
			return errors.New("simulated write failure")
		}
		return nil
	}
	if e := a.transaction(func(s *State) error { s.Projects[0].Name = "failed"; return nil }); e == nil {
		t.Fatal("injected error ignored")
	}
	assertOriginalBytes(t, a.dir, before)
	if a.state.Projects[0].Name == "failed" {
		t.Fatal("failed state visible")
	}
	a.directHook = nil
	if e := a.transaction(func(s *State) error { s.Projects[0].Name = "retry"; return nil }); e != nil {
		t.Fatal(e)
	}
}
func TestDirectJournalRejectsForeignEditsAndInvalidPaths(t *testing.T) {
	a := fixture(t)
	crashWrite(a, "written", "phone-push-v1.dat")
	outside := filepath.Join(a.dir, "connection-v1.dat")
	foreign := []byte(`{"Provider":"independently-modified"}`)
	if e := os.WriteFile(outside, foreign, 0600); e != nil {
		t.Fatal(e)
	}
	if e := recoverDirect(a.dir); e == nil {
		t.Fatal("foreign edit overwritten")
	}
	got, _ := os.ReadFile(outside)
	if !bytes.Equal(got, foreign) {
		t.Fatal("foreign edit lost")
	}
	if _, e := os.Stat(filepath.Join(a.dir, journalName)); e != nil {
		t.Fatal("evidence journal removed")
	}
	for _, d := range []string{"../outside", "/tmp/other", ".alpha-direct-short"} {
		j := writeJournal{Version: 1, Directory: d}
		if validateJournal(j) == nil {
			t.Fatal("unsafe stage accepted", d)
		}
	}
}
func TestDirectJournalPreflightsAllRollbackBytes(t *testing.T) {
	a := fixture(t)
	crashWrite(a, "written", "projects-v2.dat")
	var j writeJournal
	if e := legacyJSON(filepath.Join(a.dir, journalName), &j); e != nil {
		t.Fatal(e)
	}
	before := originalBytes(t, a.dir)
	p := filepath.Join(a.dir, j.Directory, fmt.Sprintf("old-%d", len(j.Entries)-1))
	if e := os.WriteFile(p, []byte("corrupt"), 0600); e != nil {
		t.Fatal(e)
	}
	if recoverDirect(a.dir) == nil {
		t.Fatal("corrupted rollback accepted")
	}
	assertOriginalBytes(t, a.dir, before)
}
func TestDirectProjectionOriginalReaderAndUnknownFields(t *testing.T) {
	dir := t.TempDir()
	id := int64(9007199254740995)
	board := token()
	raw := map[string]any{"Version": 14, "BoardToken": board, "SigningKey": "preserved-signing-key", "FutureHost": map[string]any{"x": true}, "Projects": []map[string]any{{"ID": id, "Number": "ORIG-1", "Name": "원본", "Active": true, "Created": "2026-01-01T00:00:00Z", "FutureOrder": "keep"}}, "Events": []map[string]any{{"id": id, "project_id": id, "inspection": "note", "completed_at": "2026-01-01T00:00:00Z", "FutureEvent": "keep-event"}}}
	b, _ := protect(jsonRaw(raw))
	if e := os.WriteFile(filepath.Join(dir, "projects-v2.dat"), b, 0600); e != nil {
		t.Fatal(e)
	}
	a, e := newApp(dir)
	if e != nil {
		t.Fatal(e)
	}
	if e = a.transaction(func(s *State) error {
		s.Projects[0].Name = "직접 수정"
		s.Designers = append(s.Designers, Designer{token(), "설계 시험", "01000000000", 1})
		s.ClientNotes = append(s.ClientNotes, ClientNote{ID: token(), Name: "업체 시험", Text: "주의", Revision: 1})
		s.Projects[0].DesignerID = s.Designers[0].ID
		s.Projects[0].ClientID = s.ClientNotes[0].ID
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	h := readOriginal(t, dir)
	if h.Projects[0].Name != "직접 수정" || h.BoardToken != board || h.SigningKey != "preserved-signing-key" || h.Projects[0].SpecialInstructions != "주의" {
		t.Fatal("projection missing", h)
	}
	var obj map[string]json.RawMessage
	if e = legacyJSON(filepath.Join(dir, "projects-v2.dat"), &obj); e != nil {
		t.Fatal(e)
	}
	var ps, es []map[string]json.RawMessage
	json.Unmarshal(obj["Projects"], &ps)
	json.Unmarshal(obj["Events"], &es)
	if string(obj["FutureHost"]) != `{"x":true}` || string(ps[0]["FutureOrder"]) != `"keep"` || string(es[0]["FutureEvent"]) != `"keep-event"` {
		t.Fatal("unknown fields discarded")
	}
	bak, _ := os.ReadFile(filepath.Join(dir, "projects-v2.dat.bak"))
	if !bytes.Equal(bak, b) {
		t.Fatal("initial original backup replaced")
	}
	for i := 0; i < 2; i++ {
		a, e = newApp(dir)
		if e != nil {
			t.Fatal(e)
		}
	}
	bak, _ = os.ReadFile(filepath.Join(dir, "projects-v2.dat.bak"))
	if !bytes.Equal(bak, b) {
		t.Fatal("restart replaced initial backup")
	}
	if _, e = os.Stat(filepath.Join(dir, "reconstructed-state.bin")); !os.IsNotExist(e) {
		t.Fatal("shadow state created")
	}
	legacy, e := importLegacy(dir)
	if e != nil || legacy.Projects[0].Name != "직접 수정" || len(legacy.Designers) != 1 || len(legacy.ClientNotes) != 1 {
		t.Fatal("original typed reader failed", e)
	}
	if a.state.Projects[0].SettingForm == "" {
		t.Fatal("recovery form state lost")
	}
}
func TestDirectReimportsOriginalChangesOnRestart(t *testing.T) {
	a := fixture(t)
	var h map[string]json.RawMessage
	if e := legacyJSON(filepath.Join(a.dir, "projects-v2.dat"), &h); e != nil {
		t.Fatal(e)
	}
	delete(h, extensionKey)
	var ps []map[string]json.RawMessage
	json.Unmarshal(h["Projects"], &ps)
	ps[0]["Name"] = jsonRaw("원본 프로그램 수정")
	h["Projects"] = jsonRaw(ps)
	b, _ := protect(jsonRaw(h))
	os.WriteFile(filepath.Join(a.dir, "projects-v2.dat"), b, 0600)
	if a.transaction(func(s *State) error { return nil }) == nil {
		t.Fatal("external write not detected")
	}
	next, e := newApp(a.dir)
	if e != nil || next.state.Projects[0].Name != "원본 프로그램 수정" {
		t.Fatal("original edits ignored", e)
	}
}
func TestDirectBlockedBackupNeverWritesOriginals(t *testing.T) {
	a := fixture(t)
	before := originalBytes(t, a.dir)
	path := filepath.Join(a.dir, "projects-v2.dat.bak")
	if e := os.WriteFile(path, []byte("broken backup"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := a.transaction(func(s *State) error { s.Projects[0].Name = "blocked"; return nil }); e == nil {
		t.Fatal("bad backup ignored")
	}
	assertOriginalBytes(t, a.dir, before)
}
func TestDirectSourcePhotosMaterializedAsIndependentJPEGs(t *testing.T) {
	a := fixture(t)
	id := token()
	var b bytes.Buffer
	if e := jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, 5, 7)), nil); e != nil {
		t.Fatal(e)
	}
	if e := a.createCanonicalPhoto(id, b.Bytes()); e != nil {
		t.Fatal(e)
	}
	if e := a.transaction(func(s *State) error {
		c := ClientNote{ID: token(), Name: "업체", Text: "원본 사진", Revision: 1, Photos: []Photo{{ID: id, Name: "source.jpg", MIME: "image/jpeg", Width: 5, Height: 7, Legacy: true, Size: b.Len()}}}
		s.ClientNotes = append(s.ClientNotes, c)
		s.Projects[0].ClientID = c.ID
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	h := readOriginal(t, a.dir)
	copy := h.Projects[0].Photos[0]
	if copy.ID == id || copy.Size != b.Len() || copy.Width != 5 || copy.Height != 7 {
		t.Fatal("source photograph not independently materialized", copy)
	}
	copy.Legacy = true
	data, e := a.legacyPhoto(copy)
	if e != nil || !bytes.Equal(data, b.Bytes()) {
		t.Fatal("materialized JPEG invalid", e)
	}
	imp, e := importLegacy(a.dir)
	if e != nil || len(imp.Projects[0].Photos) != 0 {
		t.Fatal("linked source duplicated", e)
	}
}
func TestNativeTicketOneUseAndAdminCookie(t *testing.T) {
	a := fixture(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("native-test-password"), bcrypt.MinCost)
	a.state.PasswordHash = string(hash)
	a.localURL = "http://127.0.0.1:18791"
	for _, next := range []string{"/", "/unread", "/settings", "/records/2/edit", "/projects/1/inspection?request=2"} {
		u, _ := url.Parse(a.nativeAdminURL(next))
		ticket := strings.TrimPrefix(u.Fragment, "ticket=")
		if ticket == "" {
			t.Fatal("native ticket missing")
		}
		r := httptest.NewRequest("POST", a.localURL+"/mobile-admin/passkey/finish", strings.NewReader(string(jsonRaw(map[string]string{"token": ticket}))))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		if w.Code != 200 || len(w.Result().Cookies()) != 1 {
			t.Fatal("native handoff failed", w.Code, w.Body.String())
		}
		var result map[string]any
		json.Unmarshal(w.Body.Bytes(), &result)
		if result["next"] != next {
			t.Fatal("native target lost", result)
		}
		cookie := w.Result().Cookies()[0]
		if got := call(a, "GET", next, nil, cookie); got.Code == 303 && strings.HasPrefix(got.Header().Get("Location"), "/login") {
			t.Fatal("native cookie did not grant session")
		}
		r = httptest.NewRequest("POST", a.localURL+"/mobile-admin/passkey/finish", strings.NewReader(string(jsonRaw(map[string]string{"token": ticket}))))
		w = httptest.NewRecorder()
		a.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal("reused ticket accepted")
		}
	}
	for _, next := range []string{"https://evil.example/", "//evil.example/", "/\\evil.example/"} {
		if safeNext(next) != "/" {
			t.Fatal("unsafe redirect accepted")
		}
	}
	if r := call(a, "GET", "/native-admin", nil); r.Code != 200 {
		t.Fatal("native handoff page unavailable", r.Code)
	}
}
func TestPopupTimeoutAndUnreadAreIndependent(t *testing.T) {
	now := time.Now()
	p := popupClock{Remaining: 12 * time.Second, Last: now}
	if p.tick(now.Add(time.Minute), true) {
		t.Fatal("hover prematurely expired")
	}
	if p.Remaining != 12*time.Second {
		t.Fatal("hover consumed time")
	}
	if p.tick(now.Add(time.Minute+11*time.Second), false) {
		t.Fatal("early expiration")
	}
	if !p.tick(now.Add(time.Minute+12*time.Second), false) {
		t.Fatal("timeout did not expire")
	}
	a := fixture(t)
	proj := a.state.Projects[0]
	if e := a.requestInspection(proj.ID, "combined", "작업자", proj.SettingForm, proj.CombinedForm); e != nil {
		t.Fatal(e)
	}
	v := a.desktopSnapshot()
	if v.Unread != 1 || len(v.Events) != 1 || !strings.Contains(v.Events[0].Body, "통합") {
		t.Fatal("combined notification not grouped", v)
	}
	if a.desktopSnapshot().Unread != 1 {
		t.Fatal("snapshot marked messages read")
	}
}

func TestDirectUpgradeFromV02PreservesShadowAndMigratesPassword(t *testing.T) {
	dir := t.TempDir()
	s := State{Version: 1, Board: token(), Salt: token(), Projects: []Project{{ID: 1, Number: "UPGRADE", Name: "이전 검증판", Revision: 1, InfoRevision: 1}}, NextID: 1}
	hash, e := passwordHash("previous-password-123", s.Salt)
	if e != nil {
		t.Fatal(e)
	}
	s.PasswordHash = hash
	s.Projects[0].rotate()
	raw, _ := protect(jsonRaw(s))
	shadow := filepath.Join(dir, "reconstructed-state.bin")
	if e = os.WriteFile(shadow, raw, 0600); e != nil {
		t.Fatal(e)
	}
	a, e := newApp(dir)
	if e != nil {
		t.Fatal(e)
	}
	if a.state.Projects[0].Name != "이전 검증판" || !a.passwordMatches("previous-password-123") {
		t.Fatal("old state lost")
	}
	if h := readOriginal(t, dir); h.MobileAdmin.PasswordHash != "" {
		t.Fatal("PBKDF2 placed into bcrypt slot")
	}
	login := call(a, "GET", "/login", nil)
	cookies := login.Result().Cookies()
	nonce := cookies[0].Value
	post := call(a, "POST", "/login", url.Values{"csrf": {nonce}, "password": {"previous-password-123"}}, cookies...)
	if post.Code != 303 {
		t.Fatal("password upgrade login failed", post.Code, post.Body.String())
	}
	if h := readOriginal(t, dir); !strings.HasPrefix(h.MobileAdmin.PasswordHash, "$2") || bcrypt.CompareHashAndPassword([]byte(h.MobileAdmin.PasswordHash), []byte("previous-password-123")) != nil {
		t.Fatal("bcrypt upgrade not original-readable")
	}
	next, e := newApp(dir)
	if e != nil || !next.passwordMatches("previous-password-123") {
		t.Fatal("upgraded restart failed", e)
	}
	after, _ := os.ReadFile(shadow)
	bak, _ := os.ReadFile(shadow + ".bak")
	if !bytes.Equal(raw, after) || !bytes.Equal(raw, bak) {
		t.Fatal("old shadow was changed or not backed up")
	}
}

func TestDirectInitialCrashRemovesOnlyNewFiles(t *testing.T) {
	dir := t.TempDir()
	a := &App{dir: dir, state: State{Version: 1, Board: token(), LegacyHashes: map[string]string{}}}
	sentinel := filepath.Join(dir, "untouched.txt")
	if e := os.WriteFile(sentinel, []byte("keep"), 0600); e != nil {
		t.Fatal(e)
	}
	func() {
		defer func() { recover() }()
		a.directHook = func(phase, name string) error {
			if phase == "written" && name == "designers-v1.dat" {
				panic("exit")
			}
			return nil
		}
		n, _ := clone(a.state)
		_ = a.persist(&n)
	}()
	if e := recoverDirect(dir); e != nil {
		t.Fatal(e)
	}
	for _, name := range originalFiles {
		if _, e := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(e) {
			t.Fatal("new uncommitted file remains", name, e)
		}
	}
	b, _ := os.ReadFile(sentinel)
	if string(b) != "keep" {
		t.Fatal("unrelated file touched")
	}
}
