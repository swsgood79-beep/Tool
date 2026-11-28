package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func TestERPMSAAReadOnlyBoundaryAndProtection(t *testing.T) {
	source := erpUIReceiptMSAASource
	for _, name := range []string{"AccessibleObjectFromWindow", "AccessibleChildren", "get_accState", "get_accName", "get_accValue", "0x20000000u", "expectedProcess", "elements.Count < 800", "entry.Depth >= 16", "1500"} {
		if !strings.Contains(source, name) {
			t.Fatal("missing scope/bound/reader", name)
		}
	}
	if regexp.MustCompile(`(?i)\b(accSelect|accDoDefaultAction|set_accValue|set_accName|SendKeys|mouse_event|SendInput)\b`).MatchString(source) {
		t.Fatal("MSAA reader contains action method")
	}
	if strings.Index(source, "if ((state & 0x20000000u)") > strings.Index(source, "accessible.get_accName") {
		t.Fatal("password guard follows text read")
	}
	if strings.Contains(erpUIReceiptScript, "[System.Windows.Automation.LegacyIAccessiblePattern]::Pattern") {
		t.Fatal("unsupported managed type access returned")
	}
	for _, required := range []string{"RegisterClientSideProviderAssembly", "$legacyType -and", "# MSAA_READER_SOURCE", "[AlphaReceiptMsaa]::Read"} {
		if !strings.Contains(erpUIReceiptScript, required) {
			t.Fatal(required)
		}
	}
	report := &ERPUIReceiptReport{Saved: true, Reason: "captured", Elements: []ERPUIReceiptElement{{Type: "ControlType.Window"}}, MSAA: &ERPMSAAReport{Reason: "captured", Elements: make([]ERPMSAAElement, 801)}}
	sanitizeUIReceiptReport(report)
	if len(report.MSAA.Elements) != 800 || !report.MSAA.Truncated {
		t.Fatal("MSAA bounds")
	}
	if !strings.Contains(uiReceiptMessage(report), "구형 접근성 800개") || !strings.Contains(uiReceiptMessage(report), "일부") {
		t.Fatal("MSAA status missing")
	}
}

func TestERPMSAABrowser126(t *testing.T) {
	if os.Getenv("ALPHA126_BROWSER") != "1" {
		t.Skip("optional Chromium")
	}
	app := fixture(t)
	owner := adminCookie(app)
	app.erpDBProbe = func(context.Context, string) ERPDBReport {
		report := syntheticUIReceipt()
		report.UIReceipt.ProxyStatus = "registered"
		report.UIReceipt.MSAA = &ERPMSAAReport{Reason: "captured", Elements: []ERPMSAAElement{
			{Index: 0, Parent: -1, Role: 10, Name: "가상 입고 검색"},
			{Index: 1, Parent: 0, Role: 28, Name: "TEST-PO / 12행"},
			{Index: 2, Parent: 1, Role: 29, Name: "가상 SIDE COVER", Value: "1"},
			{Index: 3, Parent: 1, Role: 44, State: 16, Name: "가상 선택"},
		}}
		return report
	}
	server := httptest.NewServer(http.HandlerFunc(app.ServeHTTP))
	defer server.Close()
	command := exec.Command("node", "audit/browser-ui-msaa-v1.4.126.cjs", server.URL, string(jsonRaw(map[string]string{"name": owner.Name, "value": owner.Value})))
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		t.Fatal(err)
	}
}
