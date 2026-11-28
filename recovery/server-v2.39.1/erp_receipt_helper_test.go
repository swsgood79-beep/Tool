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

func syntheticReceiptRows() []OrderOutstandingRow {
	return []OrderOutstandingRow{
		{Order: "TEST-001", Unit: "SYNTH01", PurchaseDate: "20261120", Purchase: "TEST-PO-1", Sequence: 1, Part: "TEST-LONG-PART-ROBOT-ABC1234567890123456789012345678901234567890", Robot: "TEST-LONG-ROBOT-ABC1234567890123456789012345678901234567890", Item: "가상 BASE", CustomerCode: "TEST-A", Customer: "가상 거래처 A", Quantity: 10, Due: "20261208"},
		{Order: "TEST-001", Unit: "SYNTH01", PurchaseDate: "20261120", Purchase: "TEST-PO-1", Sequence: 2, Part: "TEST-LONG-PART-ROBOT-ABC1234567890123456789012345678901234567890", Robot: "TEST-LONG-ROBOT-ABC1234567890123456789012345678901234567890", Item: "가상 BASE", CustomerCode: "TEST-A", Customer: "가상 거래처 A", Quantity: 2, Due: "20261208"},
		{Order: "TEST-001", Unit: "SYNTH01", PurchaseDate: "20261121", Purchase: "TEST-PO-2", Sequence: 1, Part: "TEST-COVER", Robot: "TEST-LONG-ROBOT-ABC1234567890123456789012345678901234567890", Item: "가상 COVER", CustomerCode: "TEST-B", Customer: "가상 거래처 B", Quantity: 1.5, Due: "20261209"}}
}
func TestReceiptHelperExactRowsAndQuantities(t *testing.T) {
	rows := syntheticReceiptRows()
	choices := []receiptHelperSelection{}
	for _, r := range rows {
		choices = append(choices, receiptHelperSelection{receiptRowKey(r), r.Quantity})
	}
	task, e := buildReceiptHelper("TEST-001", rows, choices)
	if e != nil || len(task.Groups) != 2 || len(task.Groups[0].Lines) != 2 || task.Groups[0].Lines[1].Sequence != 2 {
		t.Fatal("exact rows", e)
	}
	for _, bad := range [][]receiptHelperSelection{{{choices[0].Key, 11}}, {{choices[0].Key, 0}}, {choices[0], choices[0]}, {{"unknown", 1}}} {
		if _, e := buildReceiptHelper("TEST-001", rows, bad); e == nil {
			t.Fatal("invalid choice")
		}
	}
	if _, e := buildReceiptHelper("TEST-001", append(rows, rows[0]), choices[:1]); e == nil {
		t.Fatal("duplicate source key")
	}
	rows[0].Customer = ""
	if _, e := buildReceiptHelper("TEST-001", rows, choices[:1]); e == nil {
		t.Fatal("unknown vendor")
	}
}
func TestReceiptHelperAuthorizationStaleAndNoOperatingMutation(t *testing.T) {
	a, id, _ := cardOutstandingFixture(t)
	owner := adminCookie(a)
	worker, _ := machiningWorker(t, a, "helper-worker")
	rows := syntheticReceiptRows()
	for i := range rows {
		rows[i].Order = ""
		rows[i].Key = receiptRowKey(rows[i])
	}
	entry := orderOutstandingEntry{Scope: a.outstandingScope(), State: "ready", Rows: rows, At: time.Now(), Expires: time.Now().Add(time.Minute)}
	a.outstandingCards = map[string]orderOutstandingEntry{"TEST-001": entry}
	f := url.Values{"csrf": {"admin-csrf"}, "project": {id}, "snapshot": {receiptHelperSnapshot(entry)}, "selections": {string(jsonRaw([]receiptHelperSelection{{rows[0].Key, 5}}))}}
	path := "/erp-receipt-helper/prepare"
	if w := call(a, "POST", path, f); w.Code != 401 {
		t.Fatal("anonymous", w.Code)
	}
	if w := call(a, "POST", path, url.Values{"project": {id}}, owner); w.Code != 403 {
		t.Fatal("csrf", w.Code)
	}
	if w := call(a, "POST", path, f, worker); w.Code != 403 {
		t.Fatal("worker", w.Code)
	}
	before := string(jsonRaw(a.state))
	if w := call(a, "POST", path, f, owner); w.Code != 200 || !strings.Contains(w.Body.String(), `"sequence":1`) {
		t.Fatal("preview", w.Code, w.Body.String())
	}
	launches := 0
	a.receiptHelperLaunch = func(task receiptHelperTask) error { launches++; return nil }
	f.Set("launch", "1")
	if w := call(a, "POST", path, f, owner); w.Code != 200 {
		t.Fatal("launch", w.Code)
	}
	if w := call(a, "POST", path, f, owner); w.Code != 429 || launches != 1 {
		t.Fatal("repeat launch")
	}
	entry.Rows = append([]OrderOutstandingRow{}, rows...)
	entry.Rows[0].Quantity = 4
	a.outstandingCards["TEST-001"] = entry
	f.Set("launch", "0")
	if w := call(a, "POST", path, f, owner); w.Code != 409 {
		t.Fatal("stale source", w.Code)
	}
	f.Set("snapshot", receiptHelperSnapshot(entry))
	entry.Expires = time.Now().Add(-time.Second)
	a.outstandingCards["TEST-001"] = entry
	if w := call(a, "POST", path, f, owner); w.Code != 409 {
		t.Fatal("expiry")
	}
	if before != string(jsonRaw(a.state)) {
		t.Fatal("operating state mutation")
	}
	if strings.Contains(call(a, "GET", "/work/"+a.state.Board, nil).Body.String(), "data-helper-csrf") {
		t.Fatal("worker token leak")
	}
	if !strings.Contains(call(a, "GET", "/?design-tab=orders", nil, owner).Body.String(), "data-helper-csrf") {
		t.Fatal("owner token missing")
	}
}
func TestReceiptHelperBrowser128(t *testing.T) {
	if os.Getenv("ALPHA128_BROWSER") != "1" {
		t.Skip("optional Chromium")
	}
	a, id, _ := cardOutstandingFixture(t)
	owner := adminCookie(a)
	before := string(jsonRaw(a.state))
	a.erpDBProbe = func(ctx context.Context, _ string) ERPDBReport {
		orders, _ := ctx.Value(erpOrderOutstandingBatchKey{}).([]string)
		r := ERPDBReport{Status: "order_outstanding", Connected: true, Customers: []ERPDBCustomer{{Code: "TEST-A", Unit: "SYNTH01", Name: "가상 거래처 A"}, {Code: "TEST-B", Unit: "SYNTH01", Name: "가상 거래처 B"}}}
		for _, order := range orders {
			if order == "TEST-001" {
				r.OutstandingRows = syntheticReceiptRows()
			}
		}
		return r
	}
	a.receiptHelperLaunch = func(task receiptHelperTask) error { return nil }
	server := httptest.NewServer(http.HandlerFunc(a.ServeHTTP))
	defer server.Close()
	options := map[string]any{"owner": map[string]string{"name": owner.Name, "value": owner.Value}, "id": id, "board": a.state.Board}
	command := exec.Command("node", "audit/browser-receipt-helper-v1.4.128.cjs", server.URL, string(jsonRaw(options)))
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if e := command.Run(); e != nil {
		t.Fatal(e)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if before != string(jsonRaw(a.state)) {
		t.Fatal("UI changed operating state")
	}
}

func TestReceiptOptionalDisplayPartAndExplicitZeroSequence129(t *testing.T) {
	rows := syntheticReceiptRows()
	rows[0].Part = ""
	rows[0].ItemCode = "TEST-ERP-ITEM-CODE"
	if receiptRowKey(rows[0]) == "" {
		t.Fatal("optional display part blocks complete purchase identity")
	}
	task, err := buildReceiptHelper("TEST-001", rows, []receiptHelperSelection{{receiptRowKey(rows[0]), 5}})
	if err != nil || task.Groups[0].Lines[0].Part != "" || task.Groups[0].Lines[0].ItemCode != "TEST-ERP-ITEM-CODE" {
		t.Fatal("missing display part invented or lost item code", err)
	}
	row := rows[0]
	row.Sequence = 0
	if receiptRowKey(row) != "" {
		t.Fatal("unknown zero sequence accepted")
	}
	row.SequenceKnown = true
	if receiptRowKey(row) == "" {
		t.Fatal("explicit ERP zero sequence blocked")
	}
	row.Unit = ""
	row.Purchase = ""
	reason := receiptRowBlockReason(row)
	if !strings.Contains(reason, "사업장") || !strings.Contains(reason, "발주번호") || strings.Contains(reason, "품번") {
		t.Fatal(reason)
	}
}
func TestReceiptHelperBrowser129(t *testing.T) {
	if os.Getenv("ALPHA129_BROWSER") != "1" {
		t.Skip("optional Chromium")
	}
	a, id, _ := cardOutstandingFixture(t)
	owner := adminCookie(a)
	before := string(jsonRaw(a.state))
	a.erpDBProbe = func(ctx context.Context, _ string) ERPDBReport {
		orders, _ := ctx.Value(erpOrderOutstandingBatchKey{}).([]string)
		report := ERPDBReport{Status: "order_outstanding", Connected: true, Customers: []ERPDBCustomer{{Code: "TEST-A", Unit: "SYNTH01", Name: "가상 거래처 A"}, {Code: "TEST-B", Unit: "SYNTH01", Name: "가상 거래처 B"}}}
		for _, order := range orders {
			if order == "TEST-001" {
				rows := syntheticReceiptRows()
				rows[0].Part = ""
				rows[0].ItemCode = "TEST-ERP-ITEM-CODE-NO-DISPLAY-PART"
				rows[1].Sequence = 0
				rows[1].SequenceKnown = true
				blocked := rows[2]
				blocked.Unit = ""
				blocked.Purchase = ""
				blocked.Sequence = 0
				blocked.SequenceKnown = false
				blocked.Item = "가상 발주 정보 누락 품목"
				report.OutstandingRows = append(rows, blocked)
			}
		}
		return report
	}
	a.receiptHelperLaunch = func(task receiptHelperTask) error { return nil }
	server := httptest.NewServer(http.HandlerFunc(a.ServeHTTP))
	defer server.Close()
	options := map[string]any{"owner": map[string]string{"name": owner.Name, "value": owner.Value}, "id": id, "board": a.state.Board}
	command := exec.Command("node", "audit/browser-receipt-helper-v1.4.129.cjs", server.URL, string(jsonRaw(options)))
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if before != string(jsonRaw(a.state)) {
		t.Fatal("operating state mutation")
	}
}

func TestReceiptHelperBrowser130(t *testing.T) {
	if os.Getenv("ALPHA130_BROWSER") != "1" {
		t.Skip("optional Chromium")
	}
	a, id, _ := cardOutstandingFixture(t)
	owner := adminCookie(a)
	before := string(jsonRaw(a.state))
	a.erpDBProbe = func(ctx context.Context, _ string) ERPDBReport {
		orders, _ := ctx.Value(erpOrderOutstandingBatchKey{}).([]string)
		report := ERPDBReport{Status: "order_outstanding", Connected: true, Customers: []ERPDBCustomer{{Code: "TEST-A", Unit: "SYNTH01", Name: "가상 거래처 A"}, {Code: "TEST-B", Unit: "SYNTH01", Name: "가상 거래처 B"}}}
		for _, order := range orders {
			if order == "TEST-001" {
				rows := syntheticReceiptRows()
				rows[0].Part = ""
				rows[0].ItemCode = "TEST-ERP-ITEM-CODE-NO-DISPLAY-PART"
				rows[1].Sequence = 0
				rows[1].SequenceKnown = true
				blocked := rows[2]
				blocked.Unit = ""
				blocked.Purchase = ""
				blocked.Sequence = 0
				blocked.SequenceKnown = false
				blocked.Item = "가상 발주 정보 누락 품목"
				report.OutstandingRows = append(rows, blocked)
			}
		}
		return report
	}
	a.receiptHelperLaunch = func(task receiptHelperTask) error { return nil }
	server := httptest.NewServer(http.HandlerFunc(a.ServeHTTP))
	defer server.Close()
	options := map[string]any{"owner": map[string]string{"name": owner.Name, "value": owner.Value}, "id": id, "board": a.state.Board}
	command := exec.Command("node", "audit/browser-receipt-helper-v1.4.130.cjs", server.URL, string(jsonRaw(options)))
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if before != string(jsonRaw(a.state)) {
		t.Fatal("operating state mutation")
	}
}
