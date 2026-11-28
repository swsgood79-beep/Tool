package main

import (
	"context"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestERPReceiptContractReadOnlyTransport(t *testing.T) {
	q := regexp.MustCompile(`(?s)\$sourceSql=@'\s*(.*?)\s*'@`).FindStringSubmatch(erpReceiptContractScript)
	if len(q) != 2 || !strings.HasPrefix(q[1], "SELECT TOP (3)") {
		t.Fatal("source query")
	}
	if regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE|EXEC|EXECUTE|CREATE|DROP|ALTER|GRANT)\b`).MatchString(q[1]) {
		t.Fatal("not read only")
	}
	for _, s := range []string{"sys.sql_modules", "PC_PDTWHSMST_UPDATE2", "PC_PDTWHSHIS_UPDATE", "PC_PDTWHSMST_AMT_UPDATE"} {
		if !strings.Contains(q[1], s) {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"USER_PW", "password", "200000", "redacted"} {
		if !strings.Contains(erpReceiptContractScript, s) {
			t.Fatal("missing bound/redaction", s)
		}
	}
	cmd := erpDBPowerShellCommand(context.WithValue(context.Background(), erpReceiptContractKey{}, true), "")
	b, _ := io.ReadAll(cmd.Stdin)
	if !strings.Contains(string(b), "$ReceiptContract=$true") || strings.Contains(string(b), "# RECEIPT_CONTRACT_SCRIPT") {
		t.Fatal("transport")
	}
}
func TestERPReceiptContractOwnerDownload(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	worker, _ := machiningWorker(t, a, "contract-worker")
	csrf := a.sessions[hashToken(owner.Value)].CSRF
	f := url.Values{"csrf": {csrf}, "mode": {"receipt-contract"}}
	p := "/machining-admin/db-check"
	if w := call(a, "POST", p, f, worker); w.Code != 403 {
		t.Fatal("worker")
	}
	if w := call(a, "POST", p, url.Values{"mode": {"receipt-contract"}}, owner); w.Code != 403 {
		t.Fatal("csrf")
	}
	before := string(jsonRaw(a.state))
	a.erpDBProbe = func(ctx context.Context, _ string) ERPDBReport {
		if on, _ := ctx.Value(erpReceiptContractKey{}).(bool); !on {
			t.Error("mode")
		}
		return ERPDBReport{Status: "receipt_contract", Connected: true, ReceiptDefinitions: []ERPReceiptDefinition{{Name: "SYNTH-HEADER", Definition: "SELECT 1", Visible: true}, {Name: "SYNTH-LINE", Definition: "SELECT 2", Visible: true}, {Name: "SYNTH-TOTAL", Definition: "SELECT 3", Visible: true}}}
	}
	if w := call(a, "POST", p, f, owner); w.Code != 202 {
		t.Fatal(w.Code)
	}
	end := time.Now().Add(time.Second)
	for {
		a.mu.Lock()
		busy := a.erpDBBusy
		a.mu.Unlock()
		if !busy {
			break
		}
		if time.Now().After(end) {
			t.Fatal("stalled")
		}
		time.Sleep(time.Millisecond)
	}
	w := call(a, "GET", p+"/download", nil, owner)
	if w.Code != 200 || !strings.Contains(w.Header().Get("Content-Disposition"), "ERP-receipt-contract.json") {
		t.Fatal(w.Code)
	}
	if string(jsonRaw(a.state)) != before {
		t.Fatal("State changed")
	}
}
func TestERPReceiptContractBrowser123(t *testing.T) {
	if os.Getenv("ALPHA_CONTRACT_BROWSER") != "1" {
		t.Skip("browser separate harness")
	}
}
