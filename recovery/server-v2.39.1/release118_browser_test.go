package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestRelease118ERPLaunchBrowser(t *testing.T) {
	if os.Getenv("ALPHA118_BROWSER") != "1" {
		t.Skip("optional Chromium verification")
	}
	a := fixture(t)
	owner := adminCookie(a)
	cfg := ERPDBSyncConfig{Enabled: true, From: "2026-11-01", To: "2026-12-31"}
	if e := atomicFile(filepath.Join(a.dir, "erp-db-sync.json"), jsonRaw(cfg)); e != nil {
		t.Fatal(e)
	}
	a.erpDBProbe = func(context.Context, string) ERPDBReport { return dbSyncSample() }
	a.syncERPDB(context.Background())
	before := string(jsonRaw(a.state.MachiningJobs))
	var recoverQuery atomic.Bool
	a.erpDBProbe = func(context.Context, string) ERPDBReport {
		if recoverQuery.Load() {
			return dbSyncSample()
		}
		return ERPDBReport{Status: "failed", Stage: "process_start", ErrorCode: 206}
	}
	a.syncERPDB(context.Background())
	if before != string(jsonRaw(a.state.MachiningJobs)) {
		t.Fatal("failed launch changed history")
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.startDBSync(ctx)
	defer func() { cancel(); <-a.erpDBSyncDone }()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This switch belongs only to the injected synthetic probe in this test.
		if r.Method == "POST" && r.URL.Path == "/machining-admin/db-sync" {
			recoverQuery.Store(true)
		}
		a.ServeHTTP(w, r)
	}))
	defer server.Close()
	options := map[string]any{"owner": map[string]string{"name": owner.Name, "value": owner.Value}}
	cmd := exec.Command("node", "audit/browser-erp-launch-v1.4.118.cjs", server.URL, string(jsonRaw(options)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if before != string(jsonRaw(a.state.MachiningJobs)) {
		t.Fatal("retry changed preserved quantities/history")
	}
	if !a.erpDBSyncConfig().Enabled {
		t.Fatal("retry lost saved settings")
	}
}
