package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func purchaseDisplayFixtureRows() []OrderOutstandingRow {
	robot := "TEST-ROBOT-LONG-ABC-1234567890123456789012345678901234567890"
	return []OrderOutstandingRow{
		{Order: "TEST-001", Robot: robot, Item: "가상 BASE", Quantity: 4, Customer: "가상 공급사 A", Due: "20261208", Purchase: "TEST-PO-0001-LONG-123456789012345678901234567890", Sequence: 1, SequenceKnown: true},
		{Order: "TEST-001", Robot: robot, Item: "가상 BASE", Quantity: 2, Customer: "가상 공급사 B", Due: "20261209", Purchase: "TEST-PO-0002", Sequence: 0, SequenceKnown: true},
		{Order: "TEST-001", Robot: robot, Item: "가상 COVER", Quantity: 1.5, Customer: "가상 공급사 A", Due: "20261210", Purchase: "   "},
	}
}
func purchaseDisplayFixture(t *testing.T) (*App, string, string) {
	a, id, other := cardOutstandingFixture(t)
	a.erpDBProbe = func(ctx context.Context, _ string) ERPDBReport {
		report := ERPDBReport{Status: "order_outstanding", Connected: true}
		orders, _ := ctx.Value(erpOrderOutstandingBatchKey{}).([]string)
		for _, order := range orders {
			if order == "TEST-001" {
				report.OutstandingRows = purchaseDisplayFixtureRows()
			}
		}
		return report
	}
	return a, id, other
}
func TestOutstandingPurchaseVisibleForWorkerAndAdminNoInput132(t *testing.T) {
	a, id, _ := purchaseDisplayFixture(t)
	owner := adminCookie(a)
	before := string(jsonRaw(a.state))
	getCardOutstanding(t, a, id, "")
	view := awaitCardOutstanding(t, a, id)
	if len(view.Rows) != 3 || view.Rows[0].Purchase != "TEST-PO-0001-LONG-123456789012345678901234567890" || view.Rows[1].Purchase != "TEST-PO-0002" || !view.Rows[1].SequenceKnown || view.Rows[2].Purchase != "" {
		t.Fatal("per-row purchase identity lost", view)
	}
	response := call(a, "GET", "/order-outstanding/"+a.state.Board+"?projects="+id+"&detail=1", nil, owner)
	if response.Code != 200 || !strings.Contains(response.Body.String(), "TEST-PO-0002") || strings.Contains(response.Body.String(), "snapshot") {
		t.Fatal(response.Code, response.Body.String())
	}
	for _, path := range []string{"/?design-tab=orders", "/work/" + a.state.Board} {
		page := call(a, "GET", path, nil, owner).Body.String()
		if strings.Contains(page, "data-helper-csrf") || strings.Contains(page, "erp-receipt-helper") {
			t.Fatal("input feature still loaded")
		}
	}
	for _, method := range []string{"GET", "POST"} {
		for _, cookie := range []*http.Cookie{nil, owner} {
			var response *httptest.ResponseRecorder
			if cookie == nil {
				response = call(a, method, "/erp-receipt-helper/prepare", url.Values{"launch": {"1"}})
			} else {
				response = call(a, method, "/erp-receipt-helper/prepare", url.Values{"csrf": {"admin-csrf"}, "launch": {"1"}}, cookie)
			}
			if response.Code != 410 {
				t.Fatal("old helper route still usable", method, response.Code)
			}
		}
	}
	for _, path := range []string{"erp_receipt_helper_windows.go", "erp-receipt-helper.ps1", "erp-receipt-filter.cs"} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("native input runtime still present", path)
		}
	}
	for _, asset := range []string{"extras/order-outstanding-cards-v1.4.128.js", "extras/order-outstanding-cards-v1.4.129.js", "extras/order-outstanding-cards-v1.4.130.js"} {
		if _, err := extras.ReadFile(asset); err == nil {
			t.Fatal("old helper UI still embedded", asset)
		}
	}
	if before != string(jsonRaw(a.state)) {
		t.Fatal("read-only purchase view or retired helper mutated data")
	}
}
func TestOutstandingPurchaseBrowser132(t *testing.T) {
	if os.Getenv("ALPHA132_BROWSER") != "1" {
		t.Skip("optional Chromium")
	}
	a, id, other := purchaseDisplayFixture(t)
	owner := adminCookie(a)
	completed := time.Now()
	a.state.Projects[0].Setting.Completed = &completed
	a.state.Projects[0].Setting.Worker = "가상 작업자 가나다 · 가상 작업자 라마바사"
	before := string(jsonRaw(a.state))
	server := httptest.NewServer(http.HandlerFunc(a.ServeHTTP))
	defer server.Close()
	options := map[string]any{"owner": map[string]string{"name": owner.Name, "value": owner.Value}, "id": id, "other": other, "board": a.state.Board}
	command := exec.Command("node", "audit/browser-outstanding-purchase-v1.4.132.cjs", server.URL, string(jsonRaw(options)))
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if before != string(jsonRaw(a.state)) {
		t.Fatal("purchase popup changed operational state")
	}
}
