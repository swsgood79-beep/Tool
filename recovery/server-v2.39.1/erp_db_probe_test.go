package main

import (
	"context"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestERPDBProbeRedactsUnexpectedFieldsAndRestrictsFolder(t *testing.T) {
	raw := []byte(`{"status":"ok","connected":true,"connection_string":"pwd=fixture-secret","error":"fixture-secret","server":"private-server","columns":[{"schema":"dbo","object":"PO_ITEMS","column":"PART","type":"nvarchar"}]}`)
	r := decodeDBProbe(raw)
	safe := string(jsonRaw(r))
	if strings.Contains(safe, "fixture-secret") || strings.Contains(safe, "private-server") || len(r.Columns) != 1 {
		t.Fatal("unexpected fields leaked")
	}
	for _, path := range []string{"", "C:\\ERP", "D:/ERP (test)"} {
		if !validERPFolder(path) {
			t.Fatal(path)
		}
	}
	for _, path := range []string{"\\\\server\\ERP", "../ERP", "C:\\ERP\npassword", "https://example.test"} {
		if validERPFolder(path) {
			t.Fatal("nonlocal path accepted")
		}
	}
	b, e := os.ReadFile("erp-db-probe.ps1")
	if e != nil {
		t.Fatal(e)
	}
	script := string(b)
	if strings.Contains(script, "Login_Info") || strings.Contains(script, "SELECT *") {
		t.Fatal("login file or business data read")
	}
	queries := regexp.MustCompile(`(?s)\$(?:columnSql|foreignSql)=@'\s*(.*?)\s*'@`).FindAllStringSubmatch(script, -1)
	if len(queries) != 2 {
		t.Fatal("unexpected metadata query count")
	}
	for _, q := range queries {
		if !strings.HasPrefix(q[1], "SELECT TOP") || regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE|EXEC|EXECUTE|CREATE|ALTER|DROP|GRANT|REVOKE|TRUNCATE|MERGE)\b`).MatchString(q[1]) {
			t.Fatal("non-read-only metadata query")
		}
	}
}
func TestERPDBProbeOwnerAsyncAndDownload(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	worker, _ := machiningWorker(t, a, "db-worker")
	csrf := a.sessions[hashToken(owner.Value)].CSRF
	for _, path := range []string{"/machining-admin/db-check", "/machining-admin/db-check/download"} {
		if w := call(a, "GET", path, nil, worker); w.Code != 403 {
			t.Fatal("worker read probe", w.Code)
		}
	}
	release := make(chan struct{})
	defer close(release)
	a.erpDBProbe = func(ctx context.Context, folder string) ERPDBReport {
		select {
		case <-release:
		case <-ctx.Done():
		}
		return ERPDBReport{Status: "ok", Connected: true, Columns: []ERPDBColumn{{Schema: "dbo", Object: "PO_ITEMS", Column: "PART", Type: "nvarchar"}}}
	}
	if w := call(a, "POST", "/machining-admin/db-check", url.Values{}, owner); w.Code != 403 {
		t.Fatal("CSRF bypass")
	}
	f := url.Values{"csrf": {csrf}, "folder": {}}
	if w := call(a, "POST", "/machining-admin/db-check", f, owner); w.Code != 202 {
		t.Fatal("start", w.Code)
	}
	if w := call(a, "POST", "/machining-admin/db-check", f, owner); w.Code != 409 {
		t.Fatal("duplicate probe")
	}
	release <- struct{}{}
	until := time.Now().Add(2 * time.Second)
	for {
		a.mu.Lock()
		busy := a.erpDBBusy
		a.mu.Unlock()
		if !busy {
			break
		}
		if time.Now().After(until) {
			t.Fatal("probe stalled")
		}
		time.Sleep(time.Millisecond)
	}
	w := call(a, "GET", "/machining-admin/db-check/download", nil, owner)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "PO_ITEMS") || strings.Contains(w.Body.String(), "password") {
		t.Fatal("download", w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("diagnostic cache enabled")
	}
}

func TestERPDBOutputLimit(t *testing.T) {
	var b erpDBOutput
	if _, e := b.Write(make([]byte, 8<<20)); e != nil {
		t.Fatal(e)
	}
	if _, e := b.Write([]byte("x")); e == nil || len(b.Bytes()) != 8<<20 {
		t.Fatal("output exceeded cap")
	}
}

func TestERPDBBuilderUsesCLRAccessors(t *testing.T) {
	b, e := os.ReadFile("erp-db-probe.ps1")
	if e != nil {
		t.Fatal(e)
	}
	s := string(b)
	// SqlConnectionStringBuilder is IDictionary: PowerShell dot lookup can treat
	// CLR property names as unsupported connection keywords.
	if regexp.MustCompile(`\$builder\.(ConnectionString|DataSource|InitialCatalog|ConnectTimeout|ApplicationName|PersistSecurityInfo|ApplicationIntent|TrustServerCertificate)\b`).MatchString(s) {
		t.Fatal("ambiguous PowerShell dictionary property access returned")
	}
	for _, required := range []string{".set_ConnectionString($connections[0])", ".get_DataSource()", ".get_InitialCatalog()", ".set_ConnectTimeout(8)", ".get_ConnectionString()"} {
		if !strings.Contains(s, required) {
			t.Fatal("missing CLR accessor", required)
		}
	}
	r := decodeDBProbe([]byte(`{"status":"config_invalid","stage":"config_parse","config_reason":"setting_format","connected":false}`))
	if r.Stage != "config_parse" || r.ConfigReason != "setting_format" || !strings.Contains(dbProbeMessage(r), "문자열") {
		t.Fatal("configuration failure detail lost")
	}
	r = decodeDBProbe([]byte(`{"status":"config_invalid","config_reason":"fixture-secret"}`))
	if strings.Contains(string(jsonRaw(r)), "fixture-secret") {
		t.Fatal("unknown reason leaked")
	}
}
