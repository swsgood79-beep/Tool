package main

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func dbSyncSample() ERPDBReport {
	return ERPDBReport{Connected: true, Status: "purchase_preview", PurchaseRows: []ERPDBPurchase{
		{Unit: "100", PurchaseDate: "2026-11-03", Purchase: "TEST-PO", Sequence: 1, Order: "TEST-ORDER", Project: "시험 프로젝트", Item: "TEST-BASE", ItemName: "BASE", Robot: "TEST-ROBOT-A", RobotMatches: 1, Due: "2026-11-30", Quantity: 16, WarehouseStatus: "0"},
		{Unit: "100", PurchaseDate: "2026-11-03", Purchase: "TEST-PO", Sequence: 2, Order: "TEST-ORDER", Project: "시험 프로젝트", Item: "TEST-BASE", ItemName: "BASE", Robot: "TEST-ROBOT-B", RobotMatches: 1, Due: "2026-12-02", Quantity: 16, WarehouseStatus: "0"},
	}}
}
func TestDBSyncValidationAndDuplicates(t *testing.T) {
	r := dbSyncSample()
	rows, e := machiningDBRows(r)
	if e != nil {
		t.Fatal(e)
	}
	g := machiningGroups(rows)
	if len(g) != 1 || g[0].Required != 32 || len(g[0].Sources) != 2 {
		t.Fatal("distinct purchase lines collapsed or duplicate quantities lost")
	}
	for _, change := range []func(*ERPDBReport){
		func(r *ERPDBReport) { r.Truncated = true }, func(r *ERPDBReport) { r.Connected = false },
		func(r *ERPDBReport) { r.PurchaseRows[0].RobotMatches = 2 }, func(r *ERPDBReport) { r.PurchaseRows[0].Robot = "" },
		func(r *ERPDBReport) { r.PurchaseRows[0].Received = 17 }, func(r *ERPDBReport) { r.PurchaseRows[0].WarehouseStatus = "?" },
		func(r *ERPDBReport) { r.PurchaseRows[1].Sequence = 1 },
	} {
		r := dbSyncSample()
		change(&r)
		if _, e := machiningDBRows(r); e == nil {
			t.Fatal("unsafe result accepted", r)
		}
	}
}
func TestDBSyncPersistsGuardsAndHistory(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	worker, _ := machiningWorker(t, a, "dbsync-worker")
	csrf := a.sessions[hashToken(owner.Value)].CSRF
	if w := call(a, "POST", "/machining-admin/db-sync", url.Values{}, owner); w.Code != 403 {
		t.Fatal("csrf bypass")
	}
	if w := call(a, "GET", "/machining-admin/db-sync", nil, worker); w.Code != 403 {
		t.Fatal("worker access")
	}
	result := dbSyncSample()
	a.erpDBProbe = func(ctx context.Context, _ string) ERPDBReport {
		if sync, _ := ctx.Value(erpDBSyncKey{}).(bool); !sync {
			t.Error("not sync scoped")
		}
		if d, ok := erpDBPreviewOptions(ctx); !ok || d.From != "2026-11-01" || d.To != "2026-12-31" {
			t.Error("range lost")
		}
		return result
	}
	f := url.Values{"csrf": {csrf}, "action": {"enable"}, "from": {"2026-11-01"}, "to": {"2026-12-31"}}
	if w := call(a, "POST", "/machining-admin/db-sync", f, owner); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if e := atomicFile(filepath.Join(a.dir, "erp-bridge-enabled"), []byte("enabled")); e != nil {
		t.Fatal(e)
	}
	if a.machiningBridgeEnabled() {
		t.Fatal("GUI capture still active")
	}
	if w := call(a, "POST", "/machining-admin/bridge", url.Values{"csrf": {csrf}, "enabled": {"1"}}, owner); w.Code != 409 {
		t.Fatal("GUI/DB can race")
	}
	a.syncERPDB(context.Background())
	if len(a.state.MachiningJobs) != 1 || a.state.MachiningJobs[0].Required != 32 {
		t.Fatal("initial sync failed", a.erpDBSyncMessage)
	}
	// Restart uses durable settings; identical snapshots don't append audit imports.
	if !a.erpDBSyncConfig().Enabled {
		t.Fatal("settings lost")
	}
	a.syncERPDB(context.Background())
	if len(a.state.MachiningImports) != 1 {
		t.Fatal("duplicate sync")
	}
	now := time.Now()
	a.state.MachiningJobs[0].Cutting = []MachiningRun{{ID: "preserve", Quantity: 1, Started: now}}
	result.PurchaseRows[0].Received = 1
	a.syncERPDB(context.Background())
	j := a.state.MachiningJobs[0]
	if j.Required != 32 || j.ReportOutstanding != 31 || !j.SourceChanged || j.Cutting[0].ID != "preserve" {
		t.Fatal("work history overwritten")
	}
	// A -> B -> A must restore source quantities; historic hash is not a permanent skip.
	result = dbSyncSample()
	a.syncERPDB(context.Background())
	if a.state.MachiningJobs[0].ReportOutstanding != 32 {
		t.Fatal("returning snapshot ignored")
	}
	before := string(jsonRaw(a.state.MachiningJobs))
	result.Truncated = true
	a.syncERPDB(context.Background())
	if before != string(jsonRaw(a.state.MachiningJobs)) {
		t.Fatal("partial query mutated state")
	}
	b, e := os.ReadFile(filepath.Join(a.dir, "erp-db-sync.json"))
	if e != nil || strings.Contains(string(b), "password") {
		t.Fatal("settings invalid")
	}
}
func TestDBSyncDisabledInFlightDoesNotApply(t *testing.T) {
	a := fixture(t)
	cfg := ERPDBSyncConfig{Enabled: true, From: "2026-11-01", To: "2026-12-31"}
	atomicFile(filepath.Join(a.dir, "erp-db-sync.json"), jsonRaw(cfg))
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	a.erpDBProbe = func(context.Context, string) ERPDBReport { close(entered); <-release; return dbSyncSample() }
	go func() { a.syncERPDB(context.Background()); close(done) }()
	<-entered
	a.mu.Lock()
	cfg.Enabled = false
	atomicFile(filepath.Join(a.dir, "erp-db-sync.json"), jsonRaw(cfg))
	a.erpDBSyncGeneration++
	a.mu.Unlock()
	close(release)
	<-done
	if len(a.state.MachiningJobs) != 0 {
		t.Fatal("disabled result applied")
	}
}

func TestDBSyncRobotJoinScope(t *testing.T) {
	b, e := os.ReadFile("erp-db-probe.ps1")
	if e != nil {
		t.Fatal(e)
	}
	q := string(b)
	for _, s := range []string{"d.UNT_CD=h.UNT_CD", "d.DEMREQ_DATE=h.DEMREQ_DATE", "d.DEMREQ_NO=h.DEMREQ_NO", "d.DEMREQ_SEQ=h.DEMREQ_SEQ", "d.TITM_CD=h.ITM_CD", "d.RCVORD_NO=m.RCVORD_NO", "MIN(d.MITM_CD)", "COUNT(*) AS matches", "@BaseOnly", "@Customer"} {
		if !strings.Contains(q, s) {
			t.Fatal("unsafe robot join", s)
		}
	}
}
