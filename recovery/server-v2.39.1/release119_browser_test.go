package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"sync/atomic"
	"testing"
)

func TestRelease119BasicOrderCardsBrowser(t *testing.T) {
	if os.Getenv("ALPHA119_BROWSER") != "1" {
		t.Skip("optional Chromium verification")
	}
	a, first, second := cardOutstandingFixture(t)
	owner := adminCookie(a)
	if e := a.commitAnnouncement(func(s *State) error {
		p := &s.Projects[0]
		p.Name = "가상 로봇 설비 13축"
		p.ShippingDueDate = "2026-12-02"
		p.DesignerName = "가상 설계자"
		p.DesignerPhone = "01000000000"
		p.SuppliedMaterials = "사급품 12/01 발송 예정"
		p.SuppliedMaterialsColor = "#D9F2D0"
		p.SpecialInstructions = "가상 시험용 주의사항"
		p.InfoRevision++
		s.Projects[1].Name = "가상 이송 설비"
		s.Projects[1].ShippingDueDate = "2026-12-05"
		s.AdminAccounts = append(s.AdminAccounts, AdminAccount{ID: "synthetic-purchase", Name: "가상 구매 담당", Username: "synthetic-purchase", Role: "purchasing", Enabled: true, Revision: 1})
		s.PurchasingRecipient = "synthetic-purchase"
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	before := string(jsonRaw(a.state))
	var mode atomic.Value
	mode.Store("ready")
	a.erpDBProbe = func(ctx context.Context, _ string) ERPDBReport {
		if mode.Load().(string) == "error" {
			return ERPDBReport{Status: "order_outstanding", Connected: true, Truncated: true}
		}
		report := ERPDBReport{Status: "order_outstanding", Connected: true}
		if mode.Load().(string) == "zero" {
			return report
		}
		orders, _ := ctx.Value(erpOrderOutstandingBatchKey{}).([]string)
		for _, order := range orders {
			if order == "TEST-001" {
				report.OutstandingRows = append(report.OutstandingRows,
					OrderOutstandingRow{Order: order, Robot: "TEST-ROBOT-PR2-400(530)-20N-N-EL-V3-LONG-ROBOT-PART-NUMBER-2026", Item: "가상 BASE", Quantity: 10, Customer: "가상 업체 A", Due: "20261130"},
					OrderOutstandingRow{Order: order, Robot: "TEST-ROBOT-PR2-400(530)-20N-N-EL-V3-LONG-ROBOT-PART-NUMBER-2026", Item: "가상 BRACKET", Quantity: 1.5, Customer: "가상 업체 B", Due: "20261201"},
				)
			}
		}
		return report
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/synthetic-outstanding-mode" {
			switch r.URL.Query().Get("value") {
			case "ready", "error", "zero":
				mode.Store(r.URL.Query().Get("value"))
			default:
				http.Error(w, "invalid synthetic mode", 400)
				return
			}
			w.WriteHeader(204)
			return
		}
		a.ServeHTTP(w, r)
	}))
	defer server.Close()
	options := map[string]any{"owner": map[string]string{"name": owner.Name, "value": owner.Value}, "board": a.state.Board, "first": first, "second": second}
	cmd := exec.Command("node", "audit/browser-basic-cards-v1.4.119.cjs", server.URL, string(jsonRaw(options)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if before != string(jsonRaw(a.state)) {
		t.Fatal("viewing/refreshing popup changed operating state")
	}
}
