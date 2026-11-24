package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestShippingDateBrowser(t *testing.T) {
	if os.Getenv("SHIPPING_DATE_BROWSER") != "1" {
		t.Skip("optional browser")
	}
	a := exportFixture(t)
	a.state.ShippingRecords[0].Project.ShippingDueDate = ""
	cookie := adminCookie(a)
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg, _ := json.Marshal(map[string]any{"cookie": map[string]string{"name": cookie.Name, "value": cookie.Value}, "record": a.state.ShippingRecords[0].ID, "board": a.state.Board})
	cmd := exec.Command("node", "audit/browser-shipping-date-v1.4.96.cjs", srv.URL, string(cfg))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
