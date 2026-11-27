package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type receiptDriverFunc func(context.Context, ERPReceiptPlan) (ERPReceiptOutcome, error)

func (f receiptDriverFunc) ExecuteVerifiedTransaction(c context.Context, p ERPReceiptPlan) (ERPReceiptOutcome, error) {
	return f(c, p)
}
func syntheticReceiptPlan() ERPReceiptPlan {
	p, _ := buildERPReceiptPlan([]ERPReceiptLine{{Key: ERPReceiptKey{"TEST", "20260101", "TEST-PO", 1}, Item: "SAME-PART", CustomerCode: "TEST-CUSTOMER", Remaining: 2, Quantity: 1}})
	return p
}
func TestERPReceiptDurableIdempotency(t *testing.T) {
	dir := t.TempDir()
	p := syntheticReceiptPlan()
	calls := 0
	driver := receiptDriverFunc(func(context.Context, ERPReceiptPlan) (ERPReceiptOutcome, error) {
		calls++
		return ERPReceiptOutcome{Status: "committed", Document: "TEST-DOC"}, nil
	})
	for i := 0; i < 2; i++ {
		o, e := executeERPReceipt(context.Background(), dir, "TEST-ID", "TEST-ACTOR", p, driver)
		if e != nil || o.Document != "TEST-DOC" {
			t.Fatal(o, e)
		}
	}
	if calls != 1 {
		t.Fatal("duplicate execution")
	}
	p.Lines[0].Quantity = 2
	if _, e := executeERPReceipt(context.Background(), dir, "TEST-ID", "TEST-ACTOR", p, driver); e == nil {
		t.Fatal("changed request accepted")
	}
}
func TestERPReceiptUncertainNeverBlindRetries(t *testing.T) {
	dir := t.TempDir()
	calls := 0
	driver := receiptDriverFunc(func(context.Context, ERPReceiptPlan) (ERPReceiptOutcome, error) {
		calls++
		return ERPReceiptOutcome{}, fmt.Errorf("synthetic transport interruption")
	})
	for i := 0; i < 2; i++ {
		o, _ := executeERPReceipt(context.Background(), dir, "TEST-ID", "TEST-ACTOR", syntheticReceiptPlan(), driver)
		if o.Status != "uncertain" {
			t.Fatal(o)
		}
	}
	if calls != 1 {
		t.Fatal("uncertain request retried")
	}
}
func TestERPReceiptJournalFailurePreventsWrite(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(receiptExecutionFile(dir, "TEST-ID"), 0700)
	calls := 0
	driver := receiptDriverFunc(func(context.Context, ERPReceiptPlan) (ERPReceiptOutcome, error) {
		calls++
		return ERPReceiptOutcome{}, nil
	})
	executeERPReceipt(context.Background(), dir, "TEST-ID", "TEST-ACTOR", syntheticReceiptPlan(), driver)
	if calls != 0 {
		t.Fatal("write after journal failure")
	}
	os.WriteFile(filepath.Join(dir, "erp-receipt-execution.lock"), nil, 0600)
	if _, e := executeERPReceipt(context.Background(), dir, "OTHER-ID", "TEST-ACTOR", syntheticReceiptPlan(), driver); e == nil {
		t.Fatal("lock bypass")
	}
}
func TestERPReceiptPlanFreshSnapshot(t *testing.T) {
	r := ERPDBReport{Status: "order_outstanding", Connected: true, OutstandingRows: []OrderOutstandingRow{{Unit: "TEST", Order: "TEST-ORDER", PurchaseDate: "20260101", Purchase: "TEST-PO", Sequence: 1, ItemCode: "TEST-PART", CustomerCode: "TEST-CUSTOMER", Quantity: 2}}}
	s := []erpReceiptSelection{{Key: receiptRowKey(r.OutstandingRows[0]), Quantity: 1}}
	if _, e := receiptPlanFromReport(r, "TEST-ORDER", s); e != nil {
		t.Fatal(e)
	}
	r.OutstandingRows[0].Quantity = 0
	if _, e := receiptPlanFromReport(r, "TEST-ORDER", s); e == nil {
		t.Fatal("changed remainder accepted")
	}
	r.OutstandingRows[0].Quantity = 2
	r.Truncated = true
	if _, e := receiptPlanFromReport(r, "TEST-ORDER", s); e == nil {
		t.Fatal("partial data accepted")
	}
}
