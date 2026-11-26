package main

import (
	"bytes"
	"image"
	"image/png"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestSplitDeliveryBrowser(t *testing.T) {
	if os.Getenv("SPLIT_DELIVERY_BROWSER") != "1" {
		t.Skip("optional Chromium split flow")
	}
	a := splitFixture(t)
	owner := adminCookie(a)
	srv := httptest.NewServer(a)
	defer srv.Close()
	var photo bytes.Buffer
	png.Encode(&photo, image.NewRGBA(image.Rect(0, 0, 8, 6)))
	cfg := map[string]any{"png": photo.Bytes(), "board": a.state.Board, "project": a.state.Projects[0].ID, "cookie": map[string]string{"name": owner.Name, "value": owner.Value}}
	cmd := exec.Command("node", "audit/browser-split-delivery-v1.4.105.cjs", srv.URL, string(jsonRaw(cfg)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
	if a.state.Projects[0].Shipping.Completed == nil || len(a.state.Projects) != 1 || len(a.state.ShippingRecords) != 2 {
		t.Fatal("browser flow did not finish one order/two lots")
	}
}
