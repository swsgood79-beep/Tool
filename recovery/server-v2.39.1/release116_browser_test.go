package main

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestRelease116Browser(t *testing.T) {
	if os.Getenv("ALPHA116_BROWSER") != "1" {
		t.Skip("optional Chromium verification")
	}
	a, owner, worker := resetFixture(t)
	a.erpDBProbe = func(ctx context.Context, folder string) ERPDBReport {
		return ERPDBReport{Status: "order_outstanding", Connected: true, OutstandingRows: []OrderOutstandingRow{
			{Robot: "TEST-ROBOT-PR2-400(530)-20N-N-EL-V3-LONG-ROBOT-PART-NUMBER-2026", Item: "가상 BASE 플레이트", Quantity: 10, Customer: "가상 거래처 A", Due: "20261130"},
			{Robot: "TEST-ROBOT-PR2-400(530)-20N-N-EL-V3-LONG-ROBOT-PART-NUMBER-2026", Item: "가상 브래킷", Quantity: 1.5, Customer: "가상 거래처 B", Due: "20261201"},
			{Item: "가상 구매 품목", Quantity: 3, Customer: "가상 거래처 C", Due: "20261202"},
		}}
	}
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg := map[string]any{"owner": map[string]string{"name": owner.Name, "value": owner.Value}, "worker": map[string]string{"name": worker.Name, "value": worker.Value}}
	cmd := exec.Command("node", "audit/browser-release-v1.4.116.cjs", srv.URL, string(jsonRaw(cfg)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
