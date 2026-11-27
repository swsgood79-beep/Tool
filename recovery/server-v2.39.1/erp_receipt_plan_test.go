package main

import (
	"math"
	"testing"
)

func TestERPReceiptPlanUsesExactPurchaseRows(t *testing.T) {
	a := ERPReceiptLine{Key: ERPReceiptKey{"TEST", "20260101", "TEST-PO", 1}, Item: "SAME-PART", CustomerCode: "TEST-CUSTOMER", Remaining: 2, Quantity: 1}
	b := a
	b.Key.Sequence = 2
	p, e := buildERPReceiptPlan([]ERPReceiptLine{a, b})
	if e != nil || len(p.Lines) != 2 {
		t.Fatal("distinct purchase rows merged", e)
	}
	for _, change := range []func(*ERPReceiptLine){func(x *ERPReceiptLine) { x.Key = a.Key }, func(x *ERPReceiptLine) { x.Quantity = 3 }, func(x *ERPReceiptLine) { x.Quantity = math.NaN() }, func(x *ERPReceiptLine) { x.CustomerCode = "OTHER" }, func(x *ERPReceiptLine) { x.Quantity = 0 }, func(x *ERPReceiptLine) { x.Quantity = 0.5 }} {
		c := b
		change(&c)
		if _, e := buildERPReceiptPlan([]ERPReceiptLine{a, c}); e == nil {
			t.Fatal("invalid plan accepted")
		}
	}
}
