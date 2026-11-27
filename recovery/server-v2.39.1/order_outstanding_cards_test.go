package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func cardOutstandingFixture(t *testing.T) (*App, string, string) {
	a := fixture(t)
	if e := a.commitAnnouncement(func(s *State) error {
		s.Projects[0].Number = "TEST-001"
		p := Project{ID: s.next(), Number: "TEST-002", Name: "가상 두 번째 수주", Revision: 1, InfoRevision: 1}
		p.rotate()
		s.Projects = append(s.Projects, p)
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	return a, fmtID(a.state.Projects[0].ID), fmtID(a.state.Projects[1].ID)
}
func getCardOutstanding(t *testing.T, a *App, ids, extra string) (map[string]orderOutstandingView, *httptest.ResponseRecorder) {
	t.Helper()
	r := call(a, "GET", "/order-outstanding/"+a.state.Board+"?projects="+ids+extra, nil)
	var result struct {
		OK       bool                            `json:"ok"`
		Projects map[string]orderOutstandingView `json:"projects"`
	}
	if r.Code != 200 || json.Unmarshal(r.Body.Bytes(), &result) != nil || !result.OK {
		t.Fatalf("card query failed %d", r.Code)
	}
	return result.Projects, r
}
func awaitCardOutstanding(t *testing.T, a *App, id string) orderOutstandingView {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		views, _ := getCardOutstanding(t, a, id, "&detail=1")
		if views[id].State != "pending" {
			return views[id]
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("background query did not complete")
	return orderOutstandingView{}
}
func TestOutstandingCardsScopeCoalescingAndExports(t *testing.T) {
	a, first, second := cardOutstandingFixture(t)
	owner := adminCookie(a)
	old := string(jsonRaw(a.state))
	csv := call(a, "GET", "/export.csv", nil, owner).Body.String()
	book, e := workbook(a.state)
	if e != nil {
		t.Fatal(e)
	}
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	var calls atomic.Int32
	a.erpDBProbe = func(ctx context.Context, _ string) ERPDBReport {
		calls.Add(1)
		orders, ok := ctx.Value(erpOrderOutstandingBatchKey{}).([]string)
		if !ok || len(orders) != 2 {
			t.Error("missing scoped order batch")
		}
		entered <- struct{}{}
		<-release
		return ERPDBReport{Status: "order_outstanding", Connected: true, OutstandingRows: []OrderOutstandingRow{{Order: "TEST-001", Robot: "TEST-ROBOT", Item: "ERP_ONLY_ITEM_MARK", Quantity: 1.5, Customer: "가상 공급사"}}}
	}
	views, _ := getCardOutstanding(t, a, first+","+second, "")
	<-entered
	if views[first].State != "pending" || views[second].State != "pending" {
		t.Fatal("query did not return pending")
	}
	getCardOutstanding(t, a, first+","+second, "")
	if calls.Load() != 1 {
		t.Fatal("concurrent duplicate query")
	}
	close(release)
	one := awaitCardOutstanding(t, a, first)
	two := awaitCardOutstanding(t, a, second)
	if one.State != "ready" || one.Count != 1 || len(one.Rows) != 1 || one.Rows[0].Quantity != 1.5 || one.Rows[0].Order != "" || one.Number != "TEST-001" {
		t.Fatal(one)
	}
	if two.State != "ready" || two.Count != 0 {
		t.Fatal(two)
	}
	getCardOutstanding(t, a, first+","+second, "")
	if calls.Load() != 1 {
		t.Fatal("cache not shared")
	}
	if string(jsonRaw(a.state)) != old || call(a, "GET", "/export.csv", nil, owner).Body.String() != csv {
		t.Fatal("lookup changed state or exported CSV")
	}
	after, e := workbook(a.state)
	if e != nil {
		t.Fatal(e)
	}
	sheets := func(raw []byte) map[string]string {
		z, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
		if e != nil {
			t.Fatal(e)
		}
		out := map[string]string{}
		for _, f := range z.File {
			if strings.HasPrefix(f.Name, "xl/worksheets/") {
				r, e := f.Open()
				if e != nil {
					t.Fatal(e)
				}
				data, e := io.ReadAll(r)
				r.Close()
				if e != nil {
					t.Fatal(e)
				}
				out[f.Name] = string(data)
			}
		}
		return out
	}
	if !reflect.DeepEqual(sheets(book), sheets(after)) {
		t.Fatal("lookup changed workbook sheets")
	}
	record := approveShipment(t, a)
	if strings.Contains(string(jsonRaw(record)), "ERP_ONLY_ITEM_MARK") || strings.Contains(string(jsonRaw(a.state)), "ERP_ONLY_ITEM_MARK") {
		t.Fatal("outstanding item saved in shipping/state")
	}
	fresh, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	if len(fresh.outstandingCards) != 0 {
		t.Fatal("transient ERP results persisted")
	}
	for _, path := range []string{"/shipping-records/" + a.state.Board, "/shipping-archive-admin"} {
		page := call(a, "GET", path, nil, owner)
		if strings.Contains(page.Body.String(), "order-outstanding-config") {
			t.Fatal("outstanding control added to archive")
		}
	}
}
func TestOutstandingCardsAuthorizationAndInactiveOrders(t *testing.T) {
	a, first, _ := cardOutstandingFixture(t)
	var called atomic.Bool
	a.erpDBProbe = func(context.Context, string) ERPDBReport { called.Store(true); return ERPDBReport{} }
	paths := []string{"/order-outstanding/wrong?projects=" + first, "/order-outstanding/" + a.state.Board + "?projects=999999"}
	for _, path := range paths {
		if r := call(a, "GET", path, nil); r.Code != 404 {
			t.Fatal(r.Code)
		}
	}
	if r := call(a, "POST", "/order-outstanding/"+a.state.Board+"?projects="+first, nil); r.Code != 405 {
		t.Fatal(r.Code)
	}
	if r := call(a, "GET", "/order-outstanding/"+a.state.Board+"?projects="+strings.Repeat(first+",", 10)+first, nil); r.Code != 400 {
		t.Fatal(r.Code)
	}
	a.state.Projects[0].Hidden = true
	if r := call(a, "GET", "/order-outstanding/"+a.state.Board+"?projects="+first, nil); r.Code != 404 {
		t.Fatal(r.Code)
	}
	a.state.Projects[0].Hidden = false
	now := time.Now()
	a.state.Projects[0].Shipping.Completed = &now
	if r := call(a, "GET", "/order-outstanding/"+a.state.Board+"?projects="+first, nil); r.Code != 404 {
		t.Fatal(r.Code)
	}
	if called.Load() {
		t.Fatal("unauthorized/inactive request reached ERP")
	}
}
func TestOutstandingCardsFailureIsNotEmpty(t *testing.T) {
	for _, report := range []ERPDBReport{{Status: "failed", Stage: "process_start", ErrorCode: 206}, {Status: "order_outstanding", Connected: true, Truncated: true}, {Status: "order_outstanding", Connected: true, OutstandingRows: []OrderOutstandingRow{{Order: "OTHER", Quantity: 1}}}} {
		a, first, _ := cardOutstandingFixture(t)
		a.erpDBProbe = func(context.Context, string) ERPDBReport { return report }
		getCardOutstanding(t, a, first, "")
		view := awaitCardOutstanding(t, a, first)
		if view.State != "error" || view.Error == "" {
			t.Fatal("incomplete query treated as empty", view)
		}
	}
}
func TestOutstandingCardsDiscardChangedConfig(t *testing.T) {
	a, first, _ := cardOutstandingFixture(t)
	cfg := ERPDBSyncConfig{Enabled: true, From: "2026-11-01", To: "2026-12-31"}
	if e := atomicFile(filepath.Join(a.dir, "erp-db-sync.json"), jsonRaw(cfg)); e != nil {
		t.Fatal(e)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	a.erpDBProbe = func(context.Context, string) ERPDBReport {
		if calls.Add(1) == 1 {
			close(entered)
			<-release
			return ERPDBReport{Status: "order_outstanding", Connected: true}
		}
		return ERPDBReport{Status: "order_outstanding", Connected: true, OutstandingRows: []OrderOutstandingRow{{Order: "TEST-001", Quantity: 2}}}
	}
	getCardOutstanding(t, a, first, "")
	<-entered
	a.mu.Lock()
	cfg.Folder = `C:\SyntheticERP`
	e := atomicFile(filepath.Join(a.dir, "erp-db-sync.json"), jsonRaw(cfg))
	a.mu.Unlock()
	if e != nil {
		t.Fatal(e)
	}
	close(release)
	view := awaitCardOutstanding(t, a, first)
	if view.State != "ready" || view.Count != 1 || calls.Load() != 2 {
		t.Fatal("old scope applied", view, calls.Load())
	}
}
func TestOutstandingCardsForcedRefreshWhileOtherBatchBusy(t *testing.T) {
	a, first, second := cardOutstandingFixture(t)
	var changed atomic.Bool
	entered := make(chan struct{})
	release := make(chan struct{})
	a.erpDBProbe = func(ctx context.Context, _ string) ERPDBReport {
		orders := ctx.Value(erpOrderOutstandingBatchKey{}).([]string)
		if orders[0] == "TEST-002" {
			close(entered)
			<-release
		}
		r := ERPDBReport{Status: "order_outstanding", Connected: true}
		if orders[0] == "TEST-001" && changed.Load() {
			r.OutstandingRows = []OrderOutstandingRow{{Order: "TEST-001", Quantity: 1}}
		}
		return r
	}
	getCardOutstanding(t, a, first, "")
	if v := awaitCardOutstanding(t, a, first); v.State != "ready" || v.Count != 0 {
		t.Fatal(v)
	}
	getCardOutstanding(t, a, second, "")
	<-entered
	changed.Store(true)
	getCardOutstanding(t, a, first, "&refresh=1")
	if views, _ := getCardOutstanding(t, a, first, ""); views[first].State != "pending" {
		t.Fatal("stale zero reused while refresh waits")
	}
	close(release)
	if v := awaitCardOutstanding(t, a, first); v.State != "ready" || v.Count != 1 {
		t.Fatal(v)
	}
}
func TestOutstandingCardsMarkupOnlyBasicLists(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	for _, path := range []string{"/work/" + a.state.Board, "/?design-tab=orders"} {
		page := call(a, "GET", path, nil, owner)
		if !strings.Contains(page.Body.String(), "order-outstanding-config") {
			t.Fatal("missing basic-list control")
		}
	}
	page := call(a, "GET", "/machining-admin", nil, owner)
	if strings.Contains(page.Body.String(), "order-outstanding-config") || strings.Contains(page.Body.String(), "/order-outstanding-v1.4.116.js") || strings.Contains(page.Body.String(), "/machining-v1.4.116.js") {
		t.Fatal("old machining entry remains")
	}
	if _, e := os.Stat("extras/order-outstanding-cards-v1.4.119.js"); e != nil {
		t.Fatal(e)
	}
}
