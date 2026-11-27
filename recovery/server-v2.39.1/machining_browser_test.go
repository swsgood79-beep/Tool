package main

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestMachiningBrowser(t *testing.T) {
	if os.Getenv("MACHINING_BROWSER") != "1" {
		t.Skip("optional real Chromium flow")
	}
	a := fixture(t)
	worker, _ := machiningWorker(t, a, "machinebrowser")
	owner := adminCookie(a)
	rows := machiningSampleRows()
	rows[0][5] = "TEST-ROBOT-PR2-400(530)-20N-N-EL-V3"
	rows[0][6] = "TEST-S/P-0001-1234L-V3"
	rows[1][6] = rows[0][6]
	d := machiningDraft(t, rows)
	if e := a.importMachining(d, MachiningActor{Name: "시험 관리자"}); e != nil {
		t.Fatal(e)
	}
	file := filepath.Join(t.TempDir(), "erp-synthetic.xlsx")
	if e := os.WriteFile(file, machiningWorkbook(t, rows), 0600); e != nil {
		t.Fatal(e)
	}
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg := map[string]any{"worker": map[string]string{"name": worker.Name, "value": worker.Value}, "admin": map[string]string{"name": owner.Name, "value": owner.Value}, "file": file}
	cmd := exec.Command("node", "audit/browser-machining-v1.4.113.cjs", srv.URL, string(jsonRaw(cfg)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}

func TestMachiningMissingAndDBBrowser(t *testing.T) {
	if os.Getenv("MACHINING_BROWSER") != "1" {
		t.Skip("optional Chromium")
	}
	a := fixture(t)
	worker, _ := machiningWorker(t, a, "missingbrowser")
	owner := adminCookie(a)
	rows := machiningSampleRows()
	if e := a.importMachining(machiningDraft(t, rows), MachiningActor{}); e != nil {
		t.Fatal(e)
	}
	if e := a.importMachining(machiningDraft(t, rows[:2]), MachiningActor{}); e != nil {
		t.Fatal(e)
	}
	a.erpDBProbe = func(ctx context.Context, folder string) ERPDBReport {
		if _, ok := erpDBPreviewOptions(ctx); ok {
			return ERPDBReport{Status: "purchase_preview", Connected: true, PurchaseRows: []ERPDBPurchase{{Purchase: "TEST-PO", Quantity: 2}}}
		}

		return ERPDBReport{Status: "ok", Connected: true, Columns: []ERPDBColumn{{Schema: "dbo", Object: "TEST_ORDER", Column: "PART", Type: "nvarchar"}, {Schema: "dbo", Object: "TEST_ORDER", Column: "QUANTITY", Type: "int"}}}
	}
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg := map[string]any{"worker": map[string]string{"name": worker.Name, "value": worker.Value}, "admin": map[string]string{"name": owner.Name, "value": owner.Value}}
	cmd := exec.Command("node", "audit/browser-machining-missing-db-v1.4.113.cjs", srv.URL, string(jsonRaw(cfg)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}

func TestMachiningDBSyncBrowser(t *testing.T) {
	if os.Getenv("MACHINING_BROWSER") != "1" {
		t.Skip("optional Chromium")
	}
	a := fixture(t)
	owner := adminCookie(a)
	a.erpDBProbe = func(context.Context, string) ERPDBReport { return dbSyncSample() }
	ctx, cancel := context.WithCancel(context.Background())
	a.startDBSync(ctx)
	defer func() { cancel(); <-a.erpDBSyncDone }()
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg := map[string]any{"admin": map[string]string{"name": owner.Name, "value": owner.Value}}
	cmd := exec.Command("node", "audit/browser-db-sync-v1.4.113.cjs", srv.URL, string(jsonRaw(cfg)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
