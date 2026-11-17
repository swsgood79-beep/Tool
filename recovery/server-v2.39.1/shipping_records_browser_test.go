package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestShippingRecordsBrowser(t *testing.T) {
	if os.Getenv("SHIPPING_BROWSER") != "1" {
		t.Skip("optional actual Go server and Chromium")
	}
	a := fixture(t)
	a.state.Projects[0].Name = "지아이텍(로보에테크놀로지) 1축"
	a.state.Projects[0].DesignerName = "김재환 계장"
	a.state.Projects[0].DesignerPhone = "01012345678"
	a.state.Projects[0].SuppliedMaterials = "사급품 수령"
	a.state.Projects[0].SuppliedMaterialsColor = "#00B050"
	a.state.Projects[0].SpecialInstructions = "센서 위치와 케이블 방향 확인"
	a.state.MaterialReceipts = map[string][]MaterialReceipt{"T-001": {{Kind: "모터", Manufacturer: "PANASONIC", PartNumber: "MSMF012L1T1", Quantity: "2"}}}
	pending := []string{}
	for i := 0; i < 4; i++ {
		record := approveShipment(t, a)
		pending = append(pending, record.ID)
		if i < 3 {
			p := Project{ID: a.state.next(), Number: "A261116-00" + fmtID(int64(i+2)), Name: []string{"엔에스 1축", "SMDC 56축, F타입", "DNC(하나마이크론) 12축"}[i], Revision: 1, InfoRevision: 1}
			p.rotate()
			a.state.Projects = []Project{p}
		}
	}
	// Source removal must not hide a pending upload page or archived detail.
	if err := a.transaction(func(s *State) error { s.Projects = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	a.state.Integration.Connection.Provider = "pages"
	a.state.Integration.Connection.PublicURL = "https://old-worker.trycloudflare.com"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()
		a.ServeHTTP(rec, r)
		for k, vs := range rec.Header() {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		body := rec.Body.Bytes()
		if strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
			body = bytes.ReplaceAll(body, []byte(`data-origin="https://old-worker.trycloudflare.com"`), []byte(`data-origin="http://`+r.Host+`"`))
		}
		w.WriteHeader(rec.Code)
		w.Write(body)
	}))
	defer server.Close()
	cmd := exec.Command("node", "audit/browser-shipping-records-v1.4.51.cjs", server.URL, a.state.Board, string(jsonRaw(pending)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	for _, record := range a.state.ShippingRecords {
		if record.ArchivedAt == nil || len(record.Photos) != 20 {
			t.Fatal("browser upload did not archive all 20 photos")
		}
	}
}
