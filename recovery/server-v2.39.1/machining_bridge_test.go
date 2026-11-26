package main

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMachiningBridgeFiltersAndPreservesWork(t *testing.T) {
	a := fixture(t)
	_, worker := machiningWorker(t, a, "bridge-worker")
	path := filepath.Join(t.TempDir(), "current.xlsx")
	data := machiningWorkbook(t, machiningSampleRows())
	os.WriteFile(path, data, 0600)
	if e := a.importBridgeFile(path); e != nil {
		t.Fatal(e)
	}
	if len(a.state.MachiningJobs) != 0 {
		t.Fatal("disabled bridge imported")
	}
	if e := atomicFile(filepath.Join(a.dir, "erp-bridge-enabled"), []byte("enabled")); e != nil {
		t.Fatal(e)
	}
	if e := a.importBridgeFile(path); e != nil {
		t.Fatal(e)
	}
	if len(a.state.MachiningJobs) != 2 || a.state.MachiningJobs[0].Required != 32 {
		t.Fatal("BASE selection or quantity aggregation failed")
	}
	j := a.state.MachiningJobs[0]
	if e := a.actMachining(worker, j.ID, "cutting", "start", "", j.Revision, 8); e != nil {
		t.Fatal(e)
	}
	if e := a.importBridgeFile(path); e != nil {
		t.Fatal(e)
	}
	if len(a.state.MachiningImports) != 1 {
		t.Fatal("repeat import duplicated")
	}
	rows := machiningSampleRows()
	rows = rows[2:]
	os.WriteFile(path, machiningWorkbook(t, rows), 0600)
	if e := a.importBridgeFile(path); e != nil {
		t.Fatal(e)
	}
	got := machiningFind(&a.state, j.ID)
	if !got.Missing || len(got.Cutting) != 1 || got.Required != 32 {
		t.Fatal("source omission erased work")
	}
	os.WriteFile(path, []byte("invalid"), 0600)
	if e := a.importBridgeFile(path); e == nil {
		t.Fatal("invalid report accepted")
	}
	if len(a.state.MachiningImports) != 2 || len(machiningFind(&a.state, j.ID).Cutting) != 1 {
		t.Fatal("invalid report changed work")
	}
	restored, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	if !restored.machiningBridgeEnabled() || len(machiningFind(&restored.state, j.ID).Cutting) != 1 {
		t.Fatal("restart lost configuration/history")
	}
}

func TestMachiningBridgeSettingsRequireOwnerAndCSRF(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	worker, _ := machiningWorker(t, a, "bridge-user")
	for _, c := range []*http.Cookie{nil, worker, owner} {
		var cookies []*http.Cookie
		if c != nil {
			cookies = append(cookies, c)
		}
		w := call(a, "POST", "/machining-admin/bridge", url.Values{"enabled": {"1"}}, cookies...)
		if w.Code == 303 && c == owner {
			t.Fatal("missing CSRF allowed")
		}
		if a.machiningBridgeEnabled() {
			t.Fatal("unverified request enabled bridge")
		}
	}
	csrf := a.sessions[hashToken(owner.Value)].CSRF
	w := call(a, "POST", "/machining-admin/bridge", url.Values{"enabled": {"1"}, "csrf": {csrf}}, owner)
	if w.Code != 303 || !a.machiningBridgeEnabled() {
		t.Fatal("owner enable", w.Code, w.Body.String())
	}
	w = call(a, "GET", "/machining-admin", nil, owner)
	if !strings.Contains(w.Body.String(), "ERP 자동 연결") || !strings.Contains(w.Body.String(), "자동 반영 끄기") {
		t.Fatal("owner bridge controls missing")
	}
	w = call(a, "POST", "/machining-admin/bridge", url.Values{"enabled": {"0"}, "csrf": {csrf}}, owner)
	if w.Code != 303 || a.machiningBridgeEnabled() {
		t.Fatal("disable failed")
	}
}
