package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestShipmentPhotoViewerBrowser(t *testing.T) {
	if os.Getenv("SHIPMENT_VIEWER_BROWSER") != "1" {
		t.Skip("optional full photo viewer browser flow")
	}
	a := fixture(t)
	record := approveShipment(t, a)
	if w := shippingUpload(t, a, record.ID, 3, false); w.Code != 303 {
		t.Fatal(w.Code)
	}
	a.state.ShippingRecords[0].Project.Photos = []Photo{a.state.ShippingRecords[0].Photos[0]}
	before := string(jsonRaw(a.state.ShippingRecords))
	owner := adminCookie(a)
	srv := httptest.NewServer(a)
	defer srv.Close()
	cmd := exec.Command("node", "audit/browser-shipment-viewer-v1.4.62.cjs", srv.URL, a.state.Board, record.ID, string(jsonRaw(map[string]string{"name": owner.Name, "value": owner.Value})))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
	if string(jsonRaw(a.state.ShippingRecords)) != before {
		t.Fatal("viewer changed stored records")
	}
}
