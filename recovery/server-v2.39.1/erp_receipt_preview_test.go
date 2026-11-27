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
)

func receiptPreviewFixture(t *testing.T) (*App, string) {
	a, _, _ := cardOutstandingFixture(t)
	order := a.state.Projects[0].Number
	report := func(order string) ERPDBReport {
		return ERPDBReport{Status: "order_outstanding", Connected: true, OutstandingRows: []OrderOutstandingRow{{Order: order, Unit: "TEST", PurchaseDate: "20260101", Purchase: "TEST-PO", Sequence: 1, ItemCode: "TEST-PART-CODE", Robot: "TEST-ROBOT-LONG-PART-NUMBER-AR200-SIDE-COVER-EXAMPLE-2026", Item: "가상 SIDE COVER", Quantity: 2, CustomerCode: "TEST-CUSTOMER", Customer: "가상 거래처", Due: "20261130"}}}
	}
	a.erpDBProbe = func(ctx context.Context, _ string) ERPDBReport {
		r := ERPDBReport{Status: "order_outstanding", Connected: true}
		orders, _ := ctx.Value(erpOrderOutstandingBatchKey{}).([]string)
		for _, o := range orders {
			r.OutstandingRows = append(r.OutstandingRows, report(o).OutstandingRows...)
		}
		return r
	}
	a.erpReceiptProbe = func(ctx context.Context, _ string) ERPDBReport {
		return report(ctx.Value(erpOrderOutstandingKey{}).(string))
	}
	return a, order
}
func TestERPReceiptPreviewOwnerFreshScopeAndNoWrite(t *testing.T) {
	a, order := receiptPreviewFixture(t)
	owner := adminCookie(a)
	worker, _ := machiningWorker(t, a, "preview-worker")
	csrf := a.sessions[hashToken(owner.Value)].CSRF
	selected := []erpReceiptSelection{{Key: ERPReceiptKey{"TEST", "20260101", "TEST-PO", 1}, Quantity: 1}}
	f := url.Values{"csrf": {csrf}, "order": {order}, "lines": {string(jsonRaw(selected))}}
	path := "/machining-admin/receipt-preview"
	if w := call(a, "POST", path, f, worker); w.Code != 403 {
		t.Fatal("worker")
	}
	if w := call(a, "POST", path, url.Values{}, owner); w.Code != 403 {
		t.Fatal("csrf")
	}
	before := string(jsonRaw(a.state))
	w := call(a, "POST", path, f, owner)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"execution_enabled":false`) {
		t.Fatal(w.Code, w.Body.String())
	}
	if before != string(jsonRaw(a.state)) {
		t.Fatal("State changed")
	}
	if len(a.erpReceiptPreviews) != 1 {
		t.Fatal("preview not retained")
	}
}
func TestERPReceiptPreviewBrowser123(t *testing.T) {
	if os.Getenv("ALPHA123_BROWSER") != "1" {
		t.Skip("optional Chromium")
	}
	a, _ := receiptPreviewFixture(t)
	owner := adminCookie(a)
	server := httptest.NewServer(http.HandlerFunc(a.ServeHTTP))
	defer server.Close()
	cmd := exec.Command("node", "audit/browser-receipt-preview-v1.4.123.cjs", server.URL, string(jsonRaw(map[string]string{"name": owner.Name, "value": owner.Value})))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
