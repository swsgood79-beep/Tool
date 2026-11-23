package main

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func workerEditValues(r *ShippingRecord, setting, shipping string) url.Values {
	return url.Values{"csrf": {"admin-csrf"}, "workers_revision": {shippingWorkersRevision(r)}, "setting_worker": {setting}, "shipping_worker": {shipping}, "actor_name": {"FORGED ACTOR"}, "completed": {"2099-01-01"}}
}
func TestShippingWorkersCorrectionPreservesEvidenceAndExport(t *testing.T) {
	a := exportFixture(t)
	cookie := adminCookie(a)
	r := a.state.ShippingRecords[0]
	r.Project.Setting.Inspector = &InspectionInspector{Name: "세팅 검사자", Username: "inspector", RecordedAt: *r.Project.Setting.Completed}
	r.Project.Shipping.Inspector = &InspectionInspector{Name: "출하 검사자", Username: "inspector", RecordedAt: *r.Project.Shipping.Completed}
	a.state.ShippingRecords[0] = r
	beforeProjects, beforeEvents := string(jsonRaw(a.state.Projects)), string(jsonRaw(a.state.Events))
	monthly, _ := os.ReadFile(filepath.Join(a.dir, "검사완료기록-복구서버.xlsx"))
	w := call(a, "POST", "/shipping-archive-admin/"+r.ID+"/edit-workers", workerEditValues(&r, "  수정 세팅 A, B  ", "수정 출하 C"), cookie)
	if w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	out := *shippingRecord(&a.state, r.ID)
	if out.Project.Setting.Worker != "수정 세팅 A, B" || out.Project.Shipping.Worker != "수정 출하 C" || out.WorkerRevision != 1 || len(out.WorkerEdits) != 1 || out.WorkerEdits[0].ActorUsername != "wssong" || out.WorkerEdits[0].ActorName == "FORGED ACTOR" {
		t.Fatal("worker correction not attributed", out)
	}
	preserved := out
	preserved.Project.Setting.Worker = r.Project.Setting.Worker
	preserved.Project.Shipping.Worker = r.Project.Shipping.Worker
	preserved.WorkerEdits = nil
	preserved.WorkerRevision = 0
	if string(jsonRaw(preserved)) != string(jsonRaw(r)) {
		t.Fatal("correction changed other archived evidence")
	}
	if beforeProjects != string(jsonRaw(a.state.Projects)) || beforeEvents != string(jsonRaw(a.state.Events)) {
		t.Fatal("correction changed original requests")
	}
	afterMonthly, _ := os.ReadFile(filepath.Join(a.dir, "검사완료기록-복구서버.xlsx"))
	if string(monthly) != string(afterMonthly) {
		t.Fatal("archive correction rewrote original monthly workbook")
	}
	public := call(a, "GET", "/shipping-records/"+a.state.Board+"/"+r.ID, nil)
	if public.Code != 200 || !strings.Contains(public.Body.String(), "수정 세팅 A, B") || !strings.Contains(public.Body.String(), "수정 출하 C") || strings.Contains(public.Body.String(), "shipment-worker-editor") {
		t.Fatal("worker view mismatch", public.Code)
	}
	job, records, _, e := a.newShippingExportLocked(hashToken(cookie.Value), token(), r.ID)
	if e != nil {
		t.Fatal(e)
	}
	a.runShippingExport(job, records)
	if job.State != "ready" {
		t.Fatal(job.Message)
	}
	f, e := excelize.OpenFile(filepath.Join(job.Dir, "export.xlsx"))
	if e != nil {
		t.Fatal(e)
	}
	for cell, want := range map[string]string{"I5": "수정 세팅 A, B", "K5": "수정 출하 C"} {
		value, _ := f.GetCellValue(f.GetSheetName(1), cell)
		if value != want {
			t.Fatal("XLSX worker mismatch", cell, value)
		}
	}
	f.Close()
	fresh, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	stored := shippingRecord(&fresh.state, r.ID)
	if stored.Project.Setting.Worker != out.Project.Setting.Worker || len(stored.WorkerEdits) != 1 {
		t.Fatal("restart lost correction")
	}
}
func TestShippingWorkersPermissionsValidationStaleAndNoop(t *testing.T) {
	a := exportFixture(t)
	cookie := adminCookie(a)
	r := a.state.ShippingRecords[0]
	path := "/shipping-archive-admin/" + r.ID + "/edit-workers"
	initial := string(jsonRaw(a.state.ShippingRecords))
	w := call(a, "POST", path, workerEditValues(&r, "A", "B"))
	if w.Code != 401 {
		t.Fatal("anonymous accepted", w.Code)
	}
	bad := workerEditValues(&r, "A", "B")
	bad.Set("csrf", "forged")
	if w = call(a, "POST", path, bad, cookie); w.Code != 403 {
		t.Fatal("forged CSRF accepted", w.Code)
	}
	public := workerEditValues(&r, "A", "B")
	public.Set("csrf", a.state.Board)
	if w = call(a, "POST", "/shipping-records/"+a.state.Board+"/"+r.ID+"/edit-workers", public); w.Code != 403 {
		t.Fatal("worker mutation accepted", w.Code)
	}
	if w = call(a, "POST", path, workerEditValues(&r, strings.Repeat("가", 41), "B"), cookie); w.Code != 409 || !strings.Contains(w.Body.String(), "40자") {
		t.Fatal("validation not shown in form", w.Code)
	}
	if initial != string(jsonRaw(a.state.ShippingRecords)) {
		t.Fatal("rejected request changed archive")
	}
	stale := workerEditValues(&r, "이전 수정", "B")
	if w = call(a, "POST", path, workerEditValues(&r, "현재 수정", "B"), cookie); w.Code != 303 {
		t.Fatal(w.Code)
	}
	if w = call(a, "POST", path, stale, cookie); w.Code != 409 || !strings.Contains(w.Body.String(), "다른 화면") || !strings.Contains(w.Body.String(), `value="이전 수정"`) {
		t.Fatal("stale edit/error lost", w.Code)
	}
	r = *shippingRecord(&a.state, r.ID)
	revision := r.WorkerRevision
	if w = call(a, "POST", path, workerEditValues(&r, r.Project.Setting.Worker, r.Project.Shipping.Worker), cookie); w.Code != 303 || shippingRecord(&a.state, r.ID).WorkerRevision != revision {
		t.Fatal("no-op made new correction", w.Code)
	}
	// An authenticated colleague is attributed to their own server-side account.
	member := createAccount(t, a, cookie, "worker-correction")
	memberCookie := memberInspectorCookie(a, member.ID)
	values := workerEditValues(&r, "동료 수정", "")
	values.Set("csrf", a.sessions[hashToken(memberCookie.Value)].CSRF)
	if w = call(a, "POST", path, values, memberCookie); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	edits := shippingRecord(&a.state, r.ID).WorkerEdits
	if edits[len(edits)-1].ActorAccountID != member.ID {
		t.Fatal("colleague actor lost")
	}
	// Same account with purchasing-only privileges cannot edit the archive.
	for i := range a.state.AdminAccounts {
		if a.state.AdminAccounts[i].ID == member.ID {
			a.state.AdminAccounts[i].Role = "purchasing"
		}
	}
	before := string(jsonRaw(a.state.ShippingRecords))
	call(a, "POST", path, values, memberCookie)
	if before != string(jsonRaw(a.state.ShippingRecords)) {
		t.Fatal("purchasing account edited archive")
	}
}
func TestShippingWorkersPendingAndSaveFailureDoNotMutate(t *testing.T) {
	a := exportFixture(t)
	actor := InspectionInspector{Username: "wssong", Name: "대표"}
	r := &a.state.ShippingRecords[0]
	at := r.ArchivedAt
	r.ArchivedAt = nil
	if e := a.saveShippingWorkers(r.ID, shippingWorkersRevision(r), "A", "B", actor); e == nil {
		t.Fatal("pending record editable")
	}
	r.ArchivedAt = at
	before := string(jsonRaw(a.state.ShippingRecords))
	a.state.LegacySourceHash = "not-current"
	if e := a.saveShippingWorkers(r.ID, shippingWorkersRevision(r), "A", "B", actor); e == nil {
		t.Fatal("save failure ignored")
	}
	if before != string(jsonRaw(a.state.ShippingRecords)) {
		t.Fatal("save failure changed names or audit")
	}
}
