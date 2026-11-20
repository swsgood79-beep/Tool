package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestPurchasingReceiptBrowser(t *testing.T) {
	if os.Getenv("PURCHASING_BROWSER") != "1" {
		t.Skip("optional actual purchasing queue browser")
	}
	a := accountFixture(t)
	owner := adminCookie(a)
	_, buyer := purchaseAccount(t, a, owner, "buyerbrowser")
	a.state.Integration.MobileEnabled = true
	a.state.Projects[0].Name = "월드에프엘와이 4축 1종"
	a.state.Projects[0].Number = "A261028-002"
	p := a.state.Projects[0]
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg := map[string]any{"cookie": map[string]string{"name": buyer.Name, "value": buyer.Value}, "owner": map[string]string{"name": owner.Name, "value": owner.Value}, "board": a.state.Board, "project": p.ID}
	cmd := exec.Command("node", "audit/browser-purchasing-receipts-v1.4.80.cjs", srv.URL, string(jsonRaw(cfg)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if len(a.state.PurchasingCalls) != 8 || len(purchasingList(&a.state, a.state.PurchasingRecipient, false)) != 0 {
		t.Fatal("queue after browser", len(a.state.PurchasingCalls))
	}
}
