package main

import (
	"bytes"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/xuri/excelize/v2"
)

func machiningWorkbook(t *testing.T, rows [][]any) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	header := []any{"입고예정일", "발주번호", "거래처", "수주번호", "프로젝트", "로봇품번", "품번", "품명", "수량", "입고수"}
	f.SetSheetRow("Sheet1", "A2", &header)
	for i, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+3)
		if e := f.SetSheetRow("Sheet1", cell, &row); e != nil {
			t.Fatal(e)
		}
	}
	b, e := f.WriteToBuffer()
	if e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func machiningSampleRows() [][]any {
	return [][]any{
		{"2026-11-30", "PO-001", "가공사업부", "ORDER-001", "시험 수주", "ROBOT-A", "PART-A", "BASE", 16, 0},
		{"2026-11-30", "PO-001", "가공사업부", "ORDER-001", "시험 수주", "ROBOT-B", "PART-A", "base", 16, 0},
		{"2026-11-30", "PO-002", "가공사업부", "ORDER-001", "시험 수주", "ROBOT-C", "PART-A", "BASE", 3, 1},
		{"2026-11-30", "PO-001", "가공사업부", "ORDER-001", "시험 수주", "ROBOT-D", "PART-B", "BASE FRAME", 8, 0},
	}
}
func machiningDraft(t *testing.T, rows [][]any) MachiningDraft {
	t.Helper()
	data := machiningWorkbook(t, rows)
	parsed, e := parseMachiningWorkbook(data)
	if e != nil {
		t.Fatal(e)
	}
	return MachiningDraft{Filename: "synthetic.xlsx", Hash: digest(data), Rows: parsed}
}
func machiningWorker(t *testing.T, a *App, id string) (*http.Cookie, Session) {
	t.Helper()
	a.state.AdminAccounts = append(a.state.AdminAccounts, AdminAccount{ID: id, Username: id, Name: "작업자 " + id, Role: "machining", Enabled: true, Revision: 1})
	c := memberInspectorCookie(a, id)
	return c, a.sessions[hashToken(c.Value)]
}
func TestMachiningImportGroupingAndValidation(t *testing.T) {
	d := machiningDraft(t, machiningSampleRows())
	g := machiningGroups(d.Rows)
	if len(d.Rows) != 3 || len(g) != 2 || g[0].Required != 32 || g[1].Required != 2 || len(g[0].Sources) != 2 || g[0].Sources[1].Robot != "ROBOT-B" {
		t.Fatal(g)
	}
	for _, change := range []func([][]any){
		func(r [][]any) { r[0][2] = "다른 업체" }, func(r [][]any) { r[0][8] = 1.5 }, func(r [][]any) { r[0][9] = 17 }, func(r [][]any) { r[0][5] = "" }, func(r [][]any) { r[0][8] = "NaN" },
	} {
		rows := machiningSampleRows()
		change(rows)
		if _, e := parseMachiningWorkbook(machiningWorkbook(t, rows)); e == nil {
			t.Fatal("invalid report accepted")
		}
	}
	if _, e := parseMachiningWorkbook([]byte("not xlsx")); e == nil {
		t.Fatal("invalid xlsx accepted")
	}
}
func TestMachiningImportIdempotenceHistoryAndRestart(t *testing.T) {
	a := fixture(t)
	_, worker := machiningWorker(t, a, "cut01")
	d := machiningDraft(t, machiningSampleRows())
	actor := MachiningActor{Name: "담당 관리자"}
	if e := a.importMachining(d, actor); e != nil {
		t.Fatal(e)
	}
	id := a.state.MachiningJobs[0].ID
	if e := a.actMachining(worker, id, "cutting", "start", "", 1, 10); e != nil {
		t.Fatal(e)
	}
	if e := a.importMachining(d, actor); e != nil {
		t.Fatal(e)
	}
	if len(a.state.MachiningJobs) != 2 || len(a.state.MachiningImports) != 1 || len(a.state.MachiningJobs[0].Cutting) != 1 {
		t.Fatal("duplicate import changed history")
	}
	// Different report omits this order entirely. Work must survive.
	rows := machiningSampleRows()[2:3]
	later := machiningDraft(t, rows)
	if e := a.importMachining(later, actor); e != nil {
		t.Fatal(e)
	}
	j := machiningFind(&a.state, id)
	if !j.Missing || j.Required != 32 || len(j.Cutting) != 1 {
		t.Fatal("missing row removed work")
	}
	restored, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	got := machiningFind(&restored.state, id)
	if got == nil || !got.Missing || got.Cutting[0].StartedBy.Name != "작업자 cut01" {
		t.Fatal("restart lost work")
	}
	if backupEmpty(State{MachiningJobs: restored.state.MachiningJobs}) {
		t.Fatal("machining-only state treated empty")
	}
}
func TestMachiningProcessPartialOvernightAndReceipt(t *testing.T) {
	a := fixture(t)
	_, s := machiningWorker(t, a, "worker01")
	_, other := machiningWorker(t, a, "worker02")
	d := machiningDraft(t, machiningSampleRows())
	if e := a.importMachining(d, MachiningActor{}); e != nil {
		t.Fatal(e)
	}
	id := a.state.MachiningJobs[0].ID
	act := func(phase, action string, qty int) error {
		j := machiningFind(&a.state, id)
		run := ""
		rs := j.Cutting
		if phase == "milling" {
			rs = j.Milling
		}
		if r := machiningActive(rs); r != nil {
			run = r.ID
		}
		return a.actMachining(s, id, phase, action, run, j.Revision, qty)
	}
	if e := act("milling", "start", 1); e == nil {
		t.Fatal("milling before cutting")
	}
	if e := act("cutting", "start", 33); e == nil {
		t.Fatal("over allocated")
	}
	if e := act("cutting", "start", 10); e != nil {
		t.Fatal(e)
	}
	j := machiningFind(&a.state, id)
	run := j.Cutting[0].ID
	rev := j.Revision
	if e := a.actMachining(other, id, "cutting", "complete", run, rev, 10); e == nil {
		t.Fatal("other worker completed")
	}
	// Actual completion time must be now, never the saved yesterday start time.
	yesterday := time.Now().Add(-24 * time.Hour)
	a.state.MachiningJobs[0].Cutting[0].Started = yesterday
	before := time.Now()
	if e := act("cutting", "complete", 0); e != nil {
		t.Fatal(e)
	}
	after := time.Now()
	j = machiningFind(&a.state, id)
	at := j.Cutting[0].Completed
	if at == nil || at.Before(before) || at.After(after) || !j.Cutting[0].Started.Equal(yesterday) {
		t.Fatal("completion timestamp wrong")
	}
	if e := a.actMachining(s, id, "cutting", "complete", run, rev, 0); e == nil {
		t.Fatal("stale duplicate complete")
	}
	if e := act("milling", "start", 11); e == nil {
		t.Fatal("milling exceeded cut quantity")
	}
	if e := act("milling", "start", 10); e != nil {
		t.Fatal(e)
	}
	if e := act("milling", "complete", 0); e != nil {
		t.Fatal(e)
	}
	if machiningUnreceived(&a.state) != 10 {
		t.Fatal("missing receipt queue")
	}
	c := adminCookie(a)
	admin := a.sessions[hashToken(c.Value)]
	j = machiningFind(&a.state, id)
	if e := a.machiningAdminAction(admin, id, "receipt", j.Milling[0].ID, j.Revision, 0); e != nil {
		t.Fatal(e)
	}
	if machiningUnreceived(&a.state) != 0 {
		t.Fatal("receipt acknowledgement missing")
	}
	j = machiningFind(&a.state, id)
	if j.Required != 32 || machiningDone(j.Milling) != 10 {
		t.Fatal("receipt reset work")
	}
	// A later report reflecting the ten received pieces must not shrink work or block it.
	rows := machiningSampleRows()
	rows[0][9] = 10
	later := machiningDraft(t, rows)
	if e := a.importMachining(later, MachiningActor{}); e != nil {
		t.Fatal(e)
	}
	j = machiningFind(&a.state, id)
	if j.SourceChanged || j.Required != 32 || j.ReportOutstanding != 22 {
		t.Fatal("normal ERP receipt changed planned work", j)
	}
	if e := act("cutting", "start", 22); e != nil {
		t.Fatal(e)
	}
}
func TestMachiningPermissionsCSRFAndConflict(t *testing.T) {
	a := fixture(t)
	c, s := machiningWorker(t, a, "machine01")
	owner := adminCookie(a)
	for _, path := range []string{"/", "/mobile-admin"} {
		w := call(a, "GET", path, nil, c)
		if w.Code != 303 || w.Header().Get("Location") != "/machining" {
			t.Fatal(path, w.Code)
		}
	}
	for _, path := range []string{"/admin-accounts", "/machining-admin", "/machining-admin/history.csv", "/work/" + a.state.Board, "/manuals/" + a.state.Board, "/shipping-records/" + a.state.Board, "/native-admin/view?mode=worker"} {
		w := call(a, "GET", path, nil, c)
		if w.Code != 403 && w.Code != 303 {
			t.Fatal(path, w.Code)
		}
	}
	if w := call(a, "GET", "/machining", nil, c); w.Code != 200 || strings.Contains(w.Body.String(), "ERP 엑셀 가져오기") {
		t.Fatal("worker page", w.Code, w.Body.String())
	}
	if w := call(a, "GET", "/machining/status", nil); w.Code != 303 {
		t.Fatal("public data", w.Code)
	}
	if w := call(a, "POST", "/machining/action", url.Values{"csrf": {"forged"}}, c); w.Code != 403 {
		t.Fatal("csrf", w.Code)
	}
	d := machiningDraft(t, machiningSampleRows())
	if e := a.importMachining(d, MachiningActor{}); e != nil {
		t.Fatal(e)
	}
	id := a.state.MachiningJobs[0].ID
	if e := a.actMachining(s, id, "cutting", "start", "", 1, 5); e != nil {
		t.Fatal(e)
	}
	rows := machiningSampleRows()
	rows[0][8] = 20
	if e := a.importMachining(machiningDraft(t, rows), MachiningActor{}); e != nil {
		t.Fatal(e)
	}
	j := machiningFind(&a.state, id)
	if !j.SourceChanged || j.Required != 32 {
		t.Fatal("silent quantity overwrite")
	}
	admin := a.sessions[hashToken(owner.Value)]
	if e := a.machiningAdminAction(admin, id, "quantity", "", j.Revision, 4); e == nil {
		t.Fatal("shrank below started work")
	}
	if e := a.machiningAdminAction(admin, id, "quantity", "", j.Revision, 36); e != nil {
		t.Fatal(e)
	}
	if len(a.state.MachiningAdjustments) != 1 || machiningFind(&a.state, id).Required != 36 {
		t.Fatal("no quantity audit")
	}
	if w := call(a, "GET", "/machining-admin/history.csv", nil, owner); w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte("작업자 machine01")) {
		t.Fatal("history export", w.Code)
	}
}
func TestMachiningPrivateERPReport(t *testing.T) {
	// Local verification only: real uploaded business data is never committed.
	p := os.Getenv("ALPHA_PRIVATE_ERP_XLSX")
	if p == "" {
		t.Skip("private ERP fixture not configured")
	}
	data, e := os.ReadFile(filepath.Clean(p))
	if e != nil {
		t.Fatal(e)
	}
	rows, e := parseMachiningWorkbook(data)
	if e != nil {
		t.Fatal(e)
	}
	groups := machiningGroups(rows)
	units := 0
	for _, g := range groups {
		units += g.Required
	}
	if len(rows) != 88 || len(groups) != 54 || units != 155 {
		t.Fatalf("rows=%d groups=%d units=%d", len(rows), len(groups), units)
	}
}

func TestMachiningAccountLoginAndNativePersistence(t *testing.T) {
	a := accountFixture(t)
	owner := adminCookie(a)
	w := accountManageCall(a, owner, url.Values{"action": {"create"}, "name": {"가공 담당자"}, "username": {"machinist01"}, "role": {"machining"}, "password": {accountMemberPassword}, "confirm": {accountMemberPassword}})
	if w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	c, login := accountLogin(t, a, "machinist01", accountMemberPassword)
	if c == nil || login.Header().Get("Location") != "/machining" {
		t.Fatal("login destination")
	}
	a.state.Integration.MobileEnabled = true
	grant := enrollNativeLogin(t, a, c, token())
	fresh, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	restored := restoredCookie(t, nativeLoginCall(fresh, nil, "/native-admin/session/restore", grant))
	if w := call(fresh, "GET", "/machining", nil, restored); w.Code != 200 || !strings.Contains(w.Body.String(), "가공 담당자") {
		t.Fatal("role restore", w.Code)
	}
	if w := call(fresh, "GET", "/inspection-requests", nil, restored); w.Code != 403 {
		t.Fatal("restored role escalation")
	}
	if w := call(fresh, "GET", "/native-admin/bootstrap", nil, restored); w.Code != 200 || !strings.Contains(w.Body.String(), `"account_role":"machining"`) || !strings.Contains(w.Body.String(), `"worker_view_supported":false`) {
		t.Fatal("native role metadata", w.Code, w.Body.String())
	}
	u := adminAccountByUsername(&fresh.state, "machinist01")
	u.Enabled = false
	if w := call(fresh, "GET", "/machining/status", nil, restored); w.Code != 303 {
		t.Fatal("disabled account remained active")
	}
}

func TestMachiningDurableFailureAndBackup(t *testing.T) {
	a := fixture(t)
	_, worker := machiningWorker(t, a, "backupworker")
	if e := a.importMachining(machiningDraft(t, machiningSampleRows()), MachiningActor{Name: "시험 관리자"}); e != nil {
		t.Fatal(e)
	}
	j := a.state.MachiningJobs[0]
	before := string(jsonRaw(a.state.MachiningJobs))
	savedHash := a.state.LegacySourceHash
	a.state.LegacySourceHash = "stale-source"
	if e := a.actMachining(worker, j.ID, "cutting", "start", "", j.Revision, 4); e == nil {
		t.Fatal("failed persist succeeded")
	}
	if string(jsonRaw(a.state.MachiningJobs)) != before {
		t.Fatal("failed persist mutated work")
	}
	a.state.LegacySourceHash = savedHash
	if e := a.actMachining(worker, j.ID, "cutting", "start", "", j.Revision, 4); e != nil {
		t.Fatal(e)
	}
	identity, e := age.GenerateX25519Identity()
	if e != nil {
		t.Fatal(e)
	}
	a.state.Integration.Backup.Key = RecoveryKey{1, repository, token(), identity.String()}
	data, e := a.portable(a.state)
	if e != nil {
		t.Fatal(e)
	}
	fresh, e := newApp(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if e := fresh.restorePortable(a.state.Integration.Backup.Key, data); e != nil {
		t.Fatal(e)
	}
	got := machiningFind(&fresh.state, j.ID)
	if got == nil || got.Cutting[0].Quantity != 4 || got.Cutting[0].StartedBy.Name != "작업자 backupworker" || len(fresh.state.MachiningImports) != 1 {
		t.Fatal("backup lost machining work")
	}
}

func TestMachiningOwnerWorkerPreviewIsReadOnly(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	w := call(a, "GET", "/machining-admin", nil, owner)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `id="machining-worker-preview"`) {
		t.Fatal("missing preview entry", w.Code)
	}
	w = call(a, "GET", "/machining?preview=1", nil, owner)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `data-readonly="true"`) || !strings.Contains(w.Body.String(), "가공 관리로 돌아가기") || strings.Contains(w.Body.String(), "ERP 엑셀 가져오기") {
		t.Fatal("owner preview layout", w.Code, w.Body.String())
	}
	if e := a.importMachining(machiningDraft(t, machiningSampleRows()), MachiningActor{}); e != nil {
		t.Fatal(e)
	}
	j := a.state.MachiningJobs[0]
	w = call(a, "POST", "/machining/action", url.Values{"csrf": {"admin-csrf"}, "id": {j.ID}, "phase": {"cutting"}, "action": {"start"}, "revision": {fmtID(j.Revision)}, "quantity": {"1"}}, owner)
	if w.Code != 409 || len(a.state.MachiningJobs[0].Cutting) != 0 {
		t.Fatal("owner preview changed work")
	}
}
