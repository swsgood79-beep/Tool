package main

import (
	"bytes"
	"github.com/xuri/excelize/v2"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestShippingExportBrowser(t *testing.T) {
	if os.Getenv("SHIPPING_EXPORT_BROWSER") != "1" {
		t.Skip("optional shipment Excel browser")
	}
	a := exportFixture(t)
	record := a.state.ShippingRecords[0]
	if err := a.commitAnnouncement(func(s *State) error {
		other := record
		other.ID = token()
		other.Project.Name = "다른고객사 제품"
		other.Project.Number = "OTHER-CUSTOMER-002"
		other.ClientName = "다른고객사"
		s.ShippingRecords = append(s.ShippingRecords, other)
		pending := record
		pending.ID = token()
		pending.ArchivedAt = nil
		pending.Photos = nil
		s.ShippingRecords = append(s.ShippingRecords, pending)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	owner := adminCookie(a)
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg := map[string]any{"cookie": map[string]string{"name": owner.Name, "value": owner.Value}, "board": a.state.Board, "record": record.ID}
	cmd := exec.Command("node", "audit/browser-shipping-export-v1.4.83.cjs", srv.URL, string(jsonRaw(cfg)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		archive := exportArchive(t, filepath.Join("/tmp", "alpha-excel-global-"+fmtID(int64(i))+".zip"))
		if archive["출하목록.xlsx"] == nil {
			t.Fatal("actual global download missing index")
		}
		raw := mustRead(t, filepath.Join("/tmp", "alpha-excel-single-"+fmtID(int64(i))+".xlsx"))
		validateWorkbookXML(t, raw)
		f, err := excelize.OpenReader(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		if len(f.GetSheetList()) != 2 {
			t.Fatal("single has extra records")
		}
		for _, sheet := range f.GetSheetList() {
			rows, _ := f.GetRows(sheet)
			for _, row := range rows {
				for _, value := range row {
					if strings.Contains(value, "OTHER-CUSTOMER") {
						t.Fatal("single leaked another customer")
					}
				}
			}
		}
		f.Close()
	}
	for _, j := range a.shippingExports {
		os.RemoveAll(j.Dir)
	}
}
