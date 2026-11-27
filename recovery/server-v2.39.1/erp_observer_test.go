package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const observerPath = "/machining-admin/receipt-observer"

func observerWait(t *testing.T, a *App, phase string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		a.mu.Lock()
		p := a.erpObservation.Phase
		a.mu.Unlock()
		if p == phase {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("phase", p, "want", phase)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
func TestERPObserverLifecycle(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	worker, _ := machiningWorker(t, a, "observer-life")
	csrf := a.sessions[hashToken(owner.Value)].CSRF
	before := string(jsonRaw(a.state))
	for _, p := range []string{observerPath, observerPath + "/download"} {
		if w := call(a, "GET", p, nil, worker); w.Code != 403 {
			t.Fatal("worker")
		}
	}
	if w := call(a, "POST", observerPath, url.Values{"action": {"start"}}, owner); w.Code != 403 {
		t.Fatal("csrf")
	}
	a.erpObservationProbe = func(ctx context.Context, _ string) ERPDBReport {
		o := ctx.Value(erpObservationKey{}).(erpObservationOptions)
		os.WriteFile(filepath.Join(o.Directory, "phase"), []byte("ready"), 0600)
		for {
			if _, e := os.Stat(filepath.Join(o.Directory, "stop")); e == nil {
				break
			}
			select {
			case <-ctx.Done():
				return ERPDBReport{}
			case <-time.After(10 * time.Millisecond):
			}
		}
		return ERPDBReport{Status: "observation", Observation: &ERPObservationReport{Ready: true, Cleanup: true, Reason: "captured", Events: []ERPObservationEvent{{Text: "exec SYNTH_RECEIPT @qty=10"}}}}
	}
	f := url.Values{"csrf": {csrf}, "action": {"start"}}
	if w := call(a, "POST", observerPath, f, owner); w.Code != 202 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := call(a, "POST", observerPath, f, owner); w.Code != 409 {
		t.Fatal("duplicate")
	}
	if w := call(a, "GET", observerPath+"/download", nil, owner); w.Code != 409 {
		t.Fatal("early download")
	}
	observerWait(t, a, "recording")
	f.Set("action", "stop")
	if w := call(a, "POST", observerPath, f, owner); w.Code != 200 {
		t.Fatal("stop")
	}
	observerWait(t, a, "finished")
	w := call(a, "GET", observerPath+"/download", nil, owner)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "SYNTH_RECEIPT") {
		t.Fatal("download")
	}
	a.mu.Lock()
	backup := a.erpObservation.Backup
	a.mu.Unlock()
	data, e := os.ReadFile(backup)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := unprotect(data)
	if e != nil || !strings.Contains(string(raw), "SYNTH_RECEIPT") {
		t.Fatal("protected backup")
	}
	if _, e := os.Stat(filepath.Join(a.dir, "erp-observer-pending.dat")); !os.IsNotExist(e) {
		t.Fatal("journal retained after clean stop")
	}
	if before != string(jsonRaw(a.state)) {
		t.Fatal("operating state changed")
	}
}
func TestERPObserverRecoveryAndEmptyResult(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	csrf := a.sessions[hashToken(owner.Value)].CSRF
	f := url.Values{"csrf": {csrf}, "action": {"start"}}
	a.erpObservationProbe = func(ctx context.Context, _ string) ERPDBReport {
		return ERPDBReport{Status: "observation", Observation: &ERPObservationReport{Ready: true, Reason: "cleanup_failed"}}
	}
	call(a, "POST", observerPath, f, owner)
	observerWait(t, a, "finished")
	if w := call(a, "POST", observerPath, f, owner); w.Code != 409 {
		t.Fatal("unclean start accepted")
	}
	a.erpObservationProbe = func(ctx context.Context, _ string) ERPDBReport {
		o := ctx.Value(erpObservationKey{}).(erpObservationOptions)
		if !o.Recovery {
			t.Error("not recovery")
		}
		return ERPDBReport{Status: "observation", Observation: &ERPObservationReport{Cleanup: true, Reason: "recovered"}}
	}
	f.Set("action", "recover")
	if w := call(a, "POST", observerPath, f, owner); w.Code != 202 {
		t.Fatal(w.Code)
	}
	observerWait(t, a, "finished")
	if _, e := os.Stat(filepath.Join(a.dir, "erp-observer-pending.dat")); !os.IsNotExist(e) {
		t.Fatal("recovery journal")
	}
	a.erpObservationProbe = func(ctx context.Context, _ string) ERPDBReport {
		return ERPDBReport{Status: "observation", Observation: &ERPObservationReport{Ready: true, Cleanup: true, Reason: "captured", Truncated: true}}
	}
	f.Set("action", "start")
	call(a, "POST", observerPath, f, owner)
	observerWait(t, a, "finished")
	w := call(a, "GET", observerPath, nil, owner)
	if !strings.Contains(w.Body.String(), "0건") || !strings.Contains(w.Body.String(), "잘림") {
		t.Fatal("false success", w.Body.String())
	}
}
func TestERPObserverScriptScopeAndTransport(t *testing.T) {
	for _, s := range []string{"client_hostname", "client_pid", "database_id", "AddSeconds(90)", "finally", "DROP EVENT SESSION", "STARTUP_STATE=OFF", "max_memory=2048", "trace_permission_denied", "2000", "USER_PW"} {
		if !strings.Contains(erpObservationScript, s) {
			t.Fatal(s)
		}
	}
	cmd := erpDBPowerShellCommand(context.WithValue(context.Background(), erpObservationKey{}, erpObservationOptions{Directory: "C:\\local'folder", Session: "AlphaReceipt_012345678901234567890123"}), "")
	b, _ := io.ReadAll(cmd.Stdin)
	if strings.Contains(string(b), "# OBSERVATION_SCRIPT") || !strings.Contains(string(b), "C:\\local''folder") {
		t.Fatal("script transport")
	}
}
func TestERPObserverBrowser121(t *testing.T) {
	if os.Getenv("ALPHA121_BROWSER") != "1" {
		t.Skip("optional Chromium")
	}
	a := fixture(t)
	owner := adminCookie(a)
	a.erpObservationProbe = func(ctx context.Context, _ string) ERPDBReport {
		o := ctx.Value(erpObservationKey{}).(erpObservationOptions)
		os.WriteFile(filepath.Join(o.Directory, "phase"), []byte("ready"), 0600)
		for {
			if _, e := os.Stat(filepath.Join(o.Directory, "stop")); e == nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		return ERPDBReport{Status: "observation", Observation: &ERPObservationReport{Ready: true, Cleanup: true, Reason: "captured", Events: []ERPObservationEvent{{Text: "exec SYNTH_RECEIPT @qty=10"}}}}
	}
	server := httptest.NewServer(http.HandlerFunc(a.ServeHTTP))
	defer server.Close()
	cmd := exec.Command("node", "audit/browser-observer-v1.4.121.cjs", server.URL, string(jsonRaw(map[string]string{"name": owner.Name, "value": owner.Value})))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
