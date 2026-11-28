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
	"regexp"
	"strings"
	"testing"
	"time"
)

func syntheticUIReceipt() ERPDBReport {
	return ERPDBReport{Status: "ui_receipt", Stage: "done", UIReceipt: &ERPUIReceiptReport{Reason: "captured", Window: "PG03F2120", Elements: []ERPUIReceiptElement{
		{Index: 0, Parent: -1, Type: "ControlType.Window", Name: "가상 입고 발주 검색", Row: -1, Column: -1},
		{Index: 1, Parent: 0, Type: "ControlType.DataItem", Name: "TEST-PO / 12행 / TEST-PART", Row: 0, Column: 1, Patterns: []string{"GridItemPatternIdentifiers.Pattern"}},
		{Index: 2, Parent: 1, Type: "ControlType.CheckBox", Toggle: "Off", Patterns: []string{"TogglePatternIdentifiers.Pattern"}},
	}}}
}

func TestERPUIReceiptReadOnlySTAAndNoDatabase(t *testing.T) {
	script := strings.Replace(erpUIReceiptScript, "# MSAA_READER_SOURCE", erpUIReceiptMSAASource, 1)
	for _, required := range []string{"PG03F2120", "ProcessId", "IsPassword", "password_excluded", "GridItemPattern", "TogglePattern", "SelectionItemPattern", "1200", "IsOffscreen"} {
		if !strings.Contains(script, required) {
			t.Fatal(required)
		}
	}
	if regexp.MustCompile(`(?i)\.(Invoke|SetValue|Toggle|Select|SetFocus|Move|Click|Scroll|Realize)\s*\(`).MatchString(script) {
		t.Fatal("UI mutation in inspection")
	}
	if strings.Contains(script, "SqlConnection") || strings.Contains(script, "DB_Information") || strings.Contains(script, "SendKeys") {
		t.Fatal("unrelated database/input access")
	}
	if strings.Index(script, "if($cur.IsPassword)") > strings.Index(script, "$item.name=") {
		t.Fatal("password checked after text read")
	}
	cmd := erpDBPowerShellCommand(context.WithValue(context.Background(), erpUIReceiptKey{}, true), "C:\\not-used")
	raw, e := io.ReadAll(cmd.Stdin)
	if e != nil {
		t.Fatal(e)
	}
	if string(raw) != script || !strings.Contains(strings.Join(cmd.Args, " "), "-STA") || len(strings.Join(cmd.Args, " ")) > 2048 {
		t.Fatal("UI script not isolated or command length")
	}
}

func TestERPUIReceiptOwnerCaptureProtection(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	worker, _ := machiningWorker(t, a, "uia-worker")
	csrf := a.sessions[hashToken(owner.Value)].CSRF
	form := url.Values{"csrf": {csrf}, "mode": {"ui-receipt"}}
	if w := call(a, "POST", "/machining-admin/db-check", form, worker); w.Code != 403 {
		t.Fatal("worker capture")
	}
	if w := call(a, "POST", "/machining-admin/db-check", url.Values{"mode": {"ui-receipt"}}, owner); w.Code != 403 {
		t.Fatal("CSRF")
	}
	before := string(jsonRaw(a.state))
	a.erpDBProbe = func(ctx context.Context, _ string) ERPDBReport {
		if enabled, _ := ctx.Value(erpUIReceiptKey{}).(bool); !enabled {
			t.Error("UI mode missing")
		}
		return syntheticUIReceipt()
	}
	if w := call(a, "POST", "/machining-admin/db-check", form, owner); w.Code != 202 {
		t.Fatal(w.Code, w.Body.String())
	}
	until := time.Now().Add(2 * time.Second)
	for {
		a.mu.Lock()
		busy := a.erpDBBusy
		a.mu.Unlock()
		if !busy {
			break
		}
		if time.Now().After(until) {
			t.Fatal("stalled")
		}
		time.Sleep(time.Millisecond)
	}
	w := call(a, "GET", "/machining-admin/db-check/download", nil, owner)
	if w.Code != 200 || !strings.Contains(w.Header().Get("Content-Disposition"), "ERP-ui-receipt-check.json") || !strings.Contains(w.Body.String(), `"saved":true`) {
		t.Fatal("download", w.Code, w.Body.String())
	}
	paths, _ := filepath.Glob(filepath.Join(a.dir, "ERP-ui-receipt-check-*.dat"))
	if len(paths) != 1 {
		t.Fatal("backup not stored")
	}
	sealed, e := os.ReadFile(paths[0])
	if e != nil {
		t.Fatal(e)
	}
	plain, e := unprotect(sealed)
	if e != nil || string(plain) != w.Body.String()[:len(w.Body.String())-1] {
		t.Fatal("backup roundtrip mismatch")
	}
	if string(jsonRaw(a.state)) != before {
		t.Fatal("operating state changed")
	}
}

func TestERPUIReceiptPartialAndSaveFailure(t *testing.T) {
	r := syntheticUIReceipt()
	r.UIReceipt.Elements = make([]ERPUIReceiptElement, 1201)
	decoded := decodeDBProbe(jsonRaw(r))
	if len(decoded.UIReceipt.Elements) != 1200 || !decoded.UIReceipt.Truncated {
		t.Fatal("element bound")
	}
	decoded.UIReceipt.Saved = true
	if !strings.Contains(uiReceiptMessage(decoded.UIReceipt), "일부") {
		t.Fatal("partial capture hidden")
	}
	r = syntheticUIReceipt()
	saveUIReceiptReport(filepath.Join(t.TempDir(), "missing-parent"), &r)
	if r.UIReceipt.Saved || !strings.Contains(uiReceiptMessage(r.UIReceipt), "저장에 실패") {
		t.Fatal("failed backup presented as saved")
	}
	a := fixture(t)
	owner := adminCookie(a)
	a.erpDBReport = &r
	if w := call(a, "GET", "/machining-admin/db-check/download", nil, owner); w.Code != 409 {
		t.Fatal("unprotected result downloadable")
	}
	for _, reason := range []string{"erp_missing", "erp_ambiguous", "window_missing", "window_ambiguous", "timeout", "unsupported"} {
		if message := uiReceiptMessage(&ERPUIReceiptReport{Reason: reason, Saved: true}); strings.Contains(message, "확인 완료") {
			t.Fatal("failure presented as recognition", reason)
		}
	}
}

func TestERPUIReceiptBrowser124(t *testing.T) {
	if os.Getenv("ALPHA124_BROWSER") != "1" {
		t.Skip("optional actual Chromium")
	}
	a := fixture(t)
	owner := adminCookie(a)
	a.erpDBProbe = func(context.Context, string) ERPDBReport { return syntheticUIReceipt() }
	server := httptest.NewServer(http.HandlerFunc(a.ServeHTTP))
	defer server.Close()
	command := exec.Command("node", "audit/browser-ui-receipt-v1.4.124.cjs", server.URL, string(jsonRaw(map[string]string{"name": owner.Name, "value": owner.Value})))
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if e := command.Run(); e != nil {
		t.Fatal(e)
	}
}
