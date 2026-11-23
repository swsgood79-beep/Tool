package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestShippingCustomerBrowser(t *testing.T) {
	if os.Getenv("SHIPPING_CUSTOMER_BROWSER") != "1" {
		t.Skip("optional shipping customer browser")
	}
	a := exportFixture(t)
	cookie := adminCookie(a)
	if e := a.transaction(func(s *State) error {
		s.ShippingRecords[0].Project.Name = "은일 12축"
		s.ShippingRecords[0].ClientName = ""
		s.ShippingRecords[0].ClientSourceID = ""
		r := s.ShippingRecords[0]
		r.ID = token()
		r.Project.ID += 100
		r.Project.Number = "A261109-004"
		r.Project.Name = "쿼드메디슨 3축, 긴급건"
		r.ClientName = "쿼드메디슨"
		s.ShippingRecords = append(s.ShippingRecords, r)
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg, _ := json.Marshal(map[string]any{"cookie": map[string]string{"name": cookie.Name, "value": cookie.Value}, "board": a.state.Board, "records": []map[string]string{{"id": a.state.ShippingRecords[0].ID, "customer": "은일"}, {"id": a.state.ShippingRecords[1].ID, "customer": "쿼드메디슨"}}})
	cmd := exec.Command("node", "audit/browser-shipping-customer-v1.4.93.cjs", srv.URL, string(cfg))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
