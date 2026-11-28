package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
)

func TestERPUIWindowDiscoveryBoundsAndFailure(t *testing.T) {
	report := &ERPUIReceiptReport{Saved: true, Reason: "window_missing", NativeWindows: 17, Discovery: make([]ERPUIWindowDiscovery, 101)}
	sanitizeUIReceiptReport(report)
	if len(report.Discovery) != 100 || !report.DiscoveryTruncated {
		t.Fatal("discovery cap")
	}
	message := uiReceiptMessage(report)
	if !strings.Contains(message, "17") || !strings.Contains(message, "결과 파일") || strings.Contains(message, "열고 다시") {
		t.Fatal("missing-window diagnosis was discarded", message)
	}
	for _, required := range []string{"EnumChildWindows", "GetWindowText", "Select-ReceiptNativeWindow", "FromHandle", "uia_descendants", "NativeWindowHandle", "discovery_truncated"} {
		if !strings.Contains(erpUIReceiptScript, required) {
			t.Fatal(required)
		}
	}
}

func TestERPUIWindowDiscoveryBrowser125(t *testing.T) {
	if os.Getenv("ALPHA125_BROWSER") != "1" {
		t.Skip("optional Chromium")
	}
	a := fixture(t)
	owner := adminCookie(a)
	var missing atomic.Bool
	a.erpDBProbe = func(context.Context, string) ERPDBReport {
		report := syntheticUIReceipt()
		report.UIReceipt.NativeWindows = 17
		report.UIReceipt.DiscoveryMethod = "win32"
		report.UIReceipt.Discovery = []ERPUIWindowDiscovery{{Source: "win32", Handle: "2", Parent: "1", Module: "PG03F2120", Label: "입고등록", Class: "SyntheticMDIChild", Visible: true}}
		if missing.Load() {
			report.UIReceipt.Reason = "window_missing"
			report.UIReceipt.Elements = nil
			report.UIReceipt.DiscoveryMethod = ""
		}
		return report
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/synthetic-ui-missing" {
			missing.Store(true)
			w.WriteHeader(204)
			return
		}
		a.ServeHTTP(w, r)
	}))
	defer server.Close()
	command := exec.Command("node", "audit/browser-ui-discovery-v1.4.125.cjs", server.URL, string(jsonRaw(map[string]string{"name": owner.Name, "value": owner.Value})))
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if e := command.Run(); e != nil {
		t.Fatal(e)
	}
}
