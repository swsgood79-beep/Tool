package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestReceiptMetadataReadOnlyAndBoundedTransport(t *testing.T) {
	s := erpReceiptMetadataScript
	q := regexp.MustCompile(`(?s)\$receiptSql=@'\s*(.*?)\s*'@`).FindStringSubmatch(s)
	if len(q) != 2 {
		t.Fatal("missing SQL")
	}
	if regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE|EXEC|EXECUTE|CREATE|ALTER|DROP|GRANT|REVOKE|TRUNCATE|MERGE)\b`).MatchString(q[1]) {
		t.Fatal("mutation SQL")
	}
	for _, v := range []string{"TOP (5001)", "sys.sql_expression_dependencies", "sys.parameters", "sys.triggers", "sys.index_columns", "sys.columns", "PDTWHS_HIS", "PURORD_HIS"} {
		if !strings.Contains(q[1], v) {
			t.Fatal(v)
		}
	}
	for _, m := range regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+([\w.]+)`).FindAllStringSubmatch(q[1], -1) {
		if !strings.HasPrefix(m[1], "sys.") && m[1] != "targets" && m[1] != "modules" && m[1] != "records" {
			t.Fatal("noncatalog source", m[1])
		}
	}
	if strings.Contains(s, "ExecuteNonQuery") || strings.Contains(s, "sys.sql_logins") {
		t.Fatal("sensitive executable")
	}
	cmd := erpDBPowerShellCommand(context.WithValue(context.Background(), erpDBReceiptKey{}, true), "")
	b, e := io.ReadAll(cmd.Stdin)
	if e != nil {
		t.Fatal(e)
	}
	if len(strings.Join(cmd.Args, " ")) > 3000 || !strings.Contains(string(b), "$ReceiptMetadata=$true") || !strings.Contains(string(b), q[1]) || strings.Contains(string(b), "# RECEIPT_METADATA_SCRIPT") {
		t.Fatal("transport")
	}
}
func TestReceiptMetadataDecodeAndWarnings(t *testing.T) {
	r := decodeDBProbe([]byte(`{"status":"receipt_metadata","stage":"receipt_metadata","connected":true,"receipt_metadata":[{"kind":"module","object":"SYNTH_RECEIPT","definition":"private-body","password":"private-secret"}],"connection_string":"private-connection"}`))
	if r.Status != "receipt_metadata" || len(r.ReceiptMetadata) != 1 || strings.Contains(string(jsonRaw(r)), "private-") {
		t.Fatal("decode leak")
	}
	r.Truncated = true
	m := dbProbeMessage(r)
	if !strings.Contains(m, "권한 제한") || !strings.Contains(m, "잘림") {
		t.Fatal("partial", m)
	}
	r.ReceiptMetadata = make([]ERPDBReceiptMetadata, 5001)
	r = decodeDBProbe(jsonRaw(r))
	if len(r.ReceiptMetadata) != 5000 || !r.Truncated {
		t.Fatal("cap")
	}
}
func TestReceiptMetadataOwnerScopeAndNoOperatingChanges(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	worker, _ := machiningWorker(t, a, "receipt-fixture")
	csrf := a.sessions[hashToken(owner.Value)].CSRF
	f := url.Values{"mode": {"receipt"}, "csrf": {csrf}}
	if w := call(a, "POST", "/machining-admin/db-check", f, worker); w.Code != 403 {
		t.Fatal("worker", w.Code)
	}
	if w := call(a, "POST", "/machining-admin/db-check", url.Values{"mode": {"receipt"}}, owner); w.Code != 403 {
		t.Fatal("csrf", w.Code)
	}
	before := string(jsonRaw(a.state))
	a.erpDBProbe = func(ctx context.Context, _ string) ERPDBReport {
		if on, _ := ctx.Value(erpDBReceiptKey{}).(bool); !on {
			t.Error("mode")
		}
		return ERPDBReport{Status: "receipt_metadata", Connected: true, ReceiptViewDefinition: true, ReceiptMetadata: []ERPDBReceiptMetadata{{Kind: "module", Object: "SYNTH_RECEIPT"}}}
	}
	if w := call(a, "POST", "/machining-admin/db-check", f, owner); w.Code != 202 {
		t.Fatal("start", w.Code, w.Body.String())
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		a.mu.Lock()
		busy := a.erpDBBusy
		a.mu.Unlock()
		if !busy {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("stalled")
		}
		time.Sleep(time.Millisecond)
	}
	w := call(a, "GET", "/machining-admin/db-check/download", nil, owner)
	if w.Code != 200 || !strings.Contains(w.Header().Get("Content-Disposition"), "ERP-receipt-check.json") || !strings.Contains(w.Body.String(), `"receipt_requested":true`) {
		t.Fatal("download", w.Code, w.Body.String())
	}
	if before != string(jsonRaw(a.state)) {
		t.Fatal("state changed")
	}
}
func TestReceiptMetadataBrowser120(t *testing.T) {
	if os.Getenv("ALPHA120_BROWSER") != "1" {
		t.Skip("optional Chromium")
	}
	a := fixture(t)
	owner := adminCookie(a)
	a.erpDBProbe = func(ctx context.Context, _ string) ERPDBReport {
		return ERPDBReport{Status: "receipt_metadata", Connected: true, ReceiptMetadata: []ERPDBReceiptMetadata{{Kind: "module", Schema: "dbo", Object: "SYNTH_RECEIPT"}}, Truncated: true}
	}
	server := httptest.NewServer(http.HandlerFunc(a.ServeHTTP))
	defer server.Close()
	cmd := exec.Command("node", "audit/browser-receipt-v1.4.120.cjs", server.URL, string(jsonRaw(map[string]string{"name": owner.Name, "value": owner.Value})))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
