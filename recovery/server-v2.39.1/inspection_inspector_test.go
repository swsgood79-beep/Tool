package main

import (
	"bytes"
	"context"
	"filippo.io/age"
	"github.com/xuri/excelize/v2"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func memberInspectorCookie(a *App, id string) *http.Cookie {
	value := token()
	u := adminAccount(&a.state, id)
	a.sessions[hashToken(value)] = Session{CSRF: "member-csrf-" + id, Expires: time.Now().Add(time.Hour), AccountID: id, AccountRevision: u.Revision}
	return &http.Cookie{Name: "aw_recovered_session", Value: value}
}
func approveThroughHTTP(t *testing.T, a *App, cookie *http.Cookie, kind string) {
	t.Helper()
	p := a.state.Projects[0]
	event := p.Setting.EventID
	if kind == "shipping" {
		event = p.Shipping.EventID
	}
	csrf := a.sessions[hashToken(cookie.Value)].CSRF
	w := call(a, "POST", "/projects/"+fmtID(p.ID)+"/inspection", url.Values{"csrf": {csrf}, "inspection": {kind}, "request": {fmtID(event)}, "inspector_name": {"FORGED"}}, cookie)
	if w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestInspectorAutoUsesSessionAndPreservesSnapshot(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	a.state.OwnerName = "대표 실명"
	colleague := createAccount(t, a, owner, "qa90")
	member := memberInspectorCookie(a, colleague.ID)
	p := a.state.Projects[0]
	if e := a.requestInspection(p.ID, "combined", "작업자A", p.SettingForm, p.CombinedForm); e != nil {
		t.Fatal(e)
	}
	approveThroughHTTP(t, a, member, "setting")
	if got := a.state.Projects[0].Setting.Inspector; got == nil || got.Name != colleague.Name || got.Username != "qa90" || got.Source != "auto" {
		t.Fatal("session attribution missing", got)
	}
	approveThroughHTTP(t, a, owner, "shipping")
	rec := a.state.ShippingRecords[0]
	if rec.Project.Setting.Inspector.Name != colleague.Name || rec.Project.Shipping.Inspector.Name != "대표 실명" {
		t.Fatal("mixed inspectors lost")
	}
	if len(a.state.InspectionAudits) != 2 {
		t.Fatal("approval audit absent")
	}
	old := rec.Project.Shipping.Inspector.Name
	w := call(a, "POST", "/admin-accounts", url.Values{"csrf": {"admin-csrf"}, "action": {"owner-name"}, "name": {"대표 새이름"}, "owner_original": {"대표 실명"}}, owner)
	if w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	if a.state.ShippingRecords[0].Project.Shipping.Inspector.Name != old {
		t.Fatal("profile rename rewrote old inspector")
	}
	restarted, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	if restarted.state.OwnerName != "대표 새이름" || restarted.state.ShippingRecords[0].Project.Shipping.Inspector.Name != old || restarted.state.Events[0].Inspector.Name != colleague.Name {
		t.Fatal("direct format restart lost metadata")
	}
	book, e := workbook(restarted.state)
	if e != nil {
		t.Fatal(e)
	}
	f, e := excelize.OpenReader(bytes.NewReader(book))
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	found := false
	for _, sheet := range f.GetSheetList() {
		rows, _ := f.GetRows(sheet)
		text := string(jsonRaw(rows))
		if strings.Contains(text, "대표 실명") && strings.Contains(text, "검사자") {
			found = true
		}
	}
	if !found {
		t.Fatal("inspector workbook ledger missing")
	}
}
func backfillForm(a *App, name string) url.Values {
	return url.Values{"csrf": {"admin-csrf"}, "scope": {"all"}, "inspector_name": {name}, "confirm": {"1"}, "revision": {inspectorRevision(inspectorMissing(&a.state, ""))}}
}
func TestInspectorBackfillBulkDatesPermissionsAndBackup(t *testing.T) {
	a := exportFixture(t)
	owner := adminCookie(a)
	a.state.OwnerName = "과거 검사자"
	before := a.state.ShippingRecords[0]
	setting := *before.Project.Setting.Completed
	shipping := *before.Project.Shipping.Completed
	archived := *before.ArchivedAt
	if len(inspectorMissing(&a.state, "")) != 2 {
		t.Fatal("duplicate snapshot counted twice")
	}
	member := createAccount(t, a, owner, "qa91")
	cookie := memberInspectorCookie(a, member.ID)
	if w := call(a, "GET", "/inspection-inspectors", nil, cookie); w.Code != 403 {
		t.Fatal("member can backfill", w.Code)
	}
	if w := call(a, "POST", "/inspection-inspectors", url.Values{"csrf": {"wrong"}}, owner); w.Code != 403 {
		t.Fatal("CSRF", w.Code)
	}
	form := backfillForm(a, "과거 검사자")
	stale := cloneValues(form)
	form.Set("confirm", "")
	if w := call(a, "POST", "/inspection-inspectors", form, owner); w.Code != 200 || !strings.Contains(w.Body.String(), "체크") {
		t.Fatal("unconfirmed backfill accepted", w.Code)
	}
	form.Set("confirm", "1")
	w := call(a, "POST", "/inspection-inspectors", form, owner)
	if w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	r := a.state.ShippingRecords[0]
	for _, stage := range []Stage{r.Project.Setting, r.Project.Shipping, a.state.Projects[0].Setting, a.state.Projects[0].Shipping} {
		if stage.Inspector == nil || stage.Inspector.Name != "과거 검사자" || stage.Inspector.Source != "backfill" || stage.Inspector.RecordedByUsername != ownerUsername {
			t.Fatal("historical inspector not synchronized")
		}
	}
	if !r.Project.Setting.Completed.Equal(setting) || !r.Project.Shipping.Completed.Equal(shipping) || !r.ArchivedAt.Equal(archived) {
		t.Fatal("backfill changed original dates")
	}
	if len(a.state.InspectionAudits) != 1 || len(a.state.InspectionAudits[0].Targets) != 2 {
		t.Fatal("backfill audit targets missing")
	}
	if len(inspectorMissing(&a.state, "")) != 0 {
		t.Fatal("already filled record still editable")
	}
	if w = call(a, "POST", "/inspection-inspectors", stale, owner); w.Code != 200 || !strings.Contains(w.Body.String(), "変更") && !strings.Contains(w.Body.String(), "변경") {
		t.Fatal("stale backfill did not reject", w.Code)
	}
	data, e := a.portable(a.state)
	if e != nil { // fixture has no recovery key until explicitly configured.
		identity, _ := age.GenerateX25519Identity()
		a.state.Integration.Backup.Key = RecoveryKey{1, repository, token(), identity.String()}
		data, e = a.portable(a.state)
	}
	if e != nil {
		t.Fatal(e)
	}
	fresh, e := newApp(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if e = fresh.restorePortable(a.state.Integration.Backup.Key, data); e != nil {
		t.Fatal(e)
	}
	if fresh.state.OwnerName != "과거 검사자" || fresh.state.ShippingRecords[0].Project.Shipping.Inspector.Name != "과거 검사자" {
		t.Fatal("backup lost inspector")
	}
	dir := t.TempDir()
	_, _, _, e = buildShippingExport(context.Background(), dir, []ShippingRecord{r}, time.Now(), func(p Photo) ([]byte, error) { return shippingExportPhoto(a.dir, p) }, nil, shippingExportLimits{PhotoBytes: 24 << 20, Records: 40, Single: true})
	if e != nil {
		t.Fatal(e)
	}
	f, e := excelize.OpenFile(filepath.Join(dir, "export.xlsx"))
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	rows, _ := f.GetRows(f.GetSheetName(1))
	if text := string(jsonRaw(rows)); !strings.Contains(text, "과거 검사자") || strings.Contains(text, "과거 기록 보완") {
		t.Fatal("shipping Excel missing inspector")
	}
	if os.Getenv("INSPECTOR_XLSX_SAMPLE") == "1" {
		raw, e := os.ReadFile(filepath.Join(dir, "export.xlsx"))
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile("/tmp/alpha-inspector-v1.4.90.xlsx", raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
}
func cloneValues(v url.Values) url.Values {
	out := url.Values{}
	for k, values := range v {
		out[k] = append([]string{}, values...)
	}
	return out
}
func TestInspectorOwnerNameDoesNotChangeCredentials(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	hash, revision := a.state.PasswordHash, a.state.Integration.Revision
	w := call(a, "POST", "/admin-accounts", url.Values{"csrf": {"admin-csrf"}, "action": {"owner-name"}, "name": {"실명 등록"}, "owner_original": {""}}, owner)
	if w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	if a.state.PasswordHash != hash || a.state.Integration.Revision != revision || !validAccountSession(&a.state, a.sessions[hashToken(owner.Value)]) {
		t.Fatal("name change revoked login or password")
	}
	w = call(a, "GET", "/admin-accounts", nil, owner)
	if !strings.Contains(w.Body.String(), "실명 등록") {
		t.Fatal("owner name not shown")
	}
}

func TestInspectorCombinedAndManualNeverOverwritesAuto(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	a.state.OwnerName = "실제 검사자"
	p := a.state.Projects[0]
	if e := a.requestInspection(p.ID, "combined", "신청 작업자", p.SettingForm, p.CombinedForm); e != nil {
		t.Fatal(e)
	}
	approveThroughHTTP(t, a, owner, "combined")
	if len(inspectorMissing(&a.state, "")) != 0 {
		t.Fatal("automatic inspector is editable as missing")
	}
	before := string(jsonRaw(a.state.ShippingRecords))
	form := backfillForm(a, "다른 이름")
	form.Set("scope", "selected")
	form.Add("record", inspectionKey(p.ID, "setting", a.state.Projects[0].Setting.EventID, a.state.Projects[0].Setting.Completed))
	w := call(a, "POST", "/inspection-inspectors", form, owner)
	if w.Code != 200 || string(jsonRaw(a.state.ShippingRecords)) != before {
		t.Fatal("manual input changed automatic inspector")
	}
	for _, e := range a.state.Events {
		if e.Inspector == nil || e.Inspector.Name != "실제 검사자" || e.Worker != "신청 작업자" {
			t.Fatal("combined attribution confused worker and inspector")
		}
	}
}
