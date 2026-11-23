package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestShippingWorkersBrowser(t *testing.T) {
	if os.Getenv("SHIPPING_WORKERS_BROWSER") != "1" {
		t.Skip("optional shipping workers browser")
	}
	a := exportFixture(t)
	cookie := adminCookie(a)
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg, _ := json.Marshal(map[string]any{"cookie": map[string]string{"name": cookie.Name, "value": cookie.Value}, "record": a.state.ShippingRecords[0].ID, "board": a.state.Board})
	cmd := exec.Command("node", "audit/browser-shipping-workers-v1.4.94.cjs", srv.URL, string(cfg))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
