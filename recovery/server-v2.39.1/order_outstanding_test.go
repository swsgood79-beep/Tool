package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestOrderOutstandingReadOnlyAndFailures(t *testing.T) {
	a := fixture(t)
	worker, _ := machiningWorker(t, a, "read-only")
	a.state.MachiningJobs = []MachiningJob{{ID: "one", Order: "ORDER-001", Required: 20}}
	before := string(jsonRaw(a.state))
	called := false
	a.erpDBProbe = func(ctx context.Context, folder string) ERPDBReport {
		called = true
		if ctx.Value(erpOrderOutstandingKey{}) != "ORDER-001" {
			t.Fatal("missing order scope")
		}
		return ERPDBReport{Status: "order_outstanding", Connected: true, OutstandingRows: []OrderOutstandingRow{{Robot: "TEST-ROBOT", Item: "판재", Quantity: 1.5, Customer: "가상 업체", Due: "20261130"}}}
	}
	w := call(a, "GET", "/machining/outstanding?order=ORDER-001", nil, worker)
	var result struct {
		OK   bool                  `json:"ok"`
		Rows []OrderOutstandingRow `json:"rows"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil || !result.OK || result.Rows[0].Quantity != 1.5 || !called {
		t.Fatal(w.Code, w.Body.String())
	}
	if before != string(jsonRaw(a.state)) {
		t.Fatal("read mutated state")
	}
	for _, report := range []ERPDBReport{{Status: "failed"}, {Status: "order_outstanding", Connected: true, Truncated: true}, {Status: "order_outstanding", Connected: false}} {
		a.erpDBProbe = func(context.Context, string) ERPDBReport { return report }
		if w = call(a, "GET", "/machining/outstanding?order=ORDER-001", nil, worker); w.Code != 502 {
			t.Fatal(w.Code)
		}
	}
	called = false
	if w = call(a, "GET", "/machining/outstanding?order=OTHER", nil, worker); w.Code != 400 || called {
		t.Fatal("unscoped query")
	}
	if w = call(a, "POST", "/machining/outstanding", nil, worker); w.Code == http.StatusOK {
		t.Fatal("POST accepted")
	}
}
