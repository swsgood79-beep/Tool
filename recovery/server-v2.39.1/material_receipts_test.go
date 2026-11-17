package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/xuri/excelize/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func receiptBook(t *testing.T) []byte {
	t.Helper()
	b, e := excelize.OpenReader(bytes.NewReader(materialBook(t, "사급품 수령", true, false)))
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	b.NewSheet("사급품 수령 내역")
	rows := [][]any{{"수주번호", "종류", "제조사", "품번", "수량"}, {"T-001", "모터", "PANASONIC", "MSMF012L1T1", 2}, {"T-001", "모터", "PANASONIC", "MSMF022L1S1", 1}, {"T-002", "드라이브", "PANASONIC", "MBDLT25SM", 1}, {nil, "연결하면 안됨", "wrong", "wrong", 99}, {"OTHER", "모터", "secret", "other", 3}}
	for i, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if e := b.SetSheetRow("사급품 수령 내역", cell, &row); e != nil {
			t.Fatal(e)
		}
	}
	buf, e := b.WriteToBuffer()
	if e != nil {
		t.Fatal(e)
	}
	return buf.Bytes()
}
func TestMaterialReceiptParsingSyncPersistenceAndAccess(t *testing.T) {
	data := receiptBook(t)
	rows, available, e := parseMaterialReceipts(data)
	if e != nil || !available || len(rows["T-001"]) != 2 || rows["T-001"][0].PartNumber != "MSMF012L1T1" || rows["T-001"][1].Quantity != "1" || len(rows["T-002"]) != 1 {
		t.Fatal(rows, available, e)
	}
	a := fixture(t)
	path := filepath.Join(a.dir, "schedule.xlsx")
	a.state.Integration.Schedule = ScheduleConfig{Enabled: true, Mode: "file", Path: path}
	os.WriteFile(path, data, 0600)
	a.syncSchedule(context.Background(), true)
	if len(a.state.MaterialReceipts["T-001"]) != 2 || len(a.state.MaterialReceipts["OTHER"]) != 0 || a.state.MaterialReceiptsUpdated == "" {
		t.Fatal("sync mapping or source boundary", a.state.MaterialReceipts)
	}
	again, e := newApp(a.dir)
	if e != nil || len(again.state.MaterialReceipts["T-001"]) != 2 {
		t.Fatal("restart", e)
	}
	url := "/material-receipts/" + a.state.Board + "?number=T-001"
	w := call(a, "GET", url, nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "MSMF022L1S1") {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := call(a, "GET", "/material-receipts/wrong?number=T-001", nil); w.Code != 404 {
		t.Fatal("board bypass")
	}
	if w := call(a, "GET", "/material-receipts/"+a.state.Board+"?number=OTHER", nil); w.Code != 404 {
		t.Fatal("historical order exposed")
	}
	a.state.Projects[0].SuppliedMaterials = "사급품 확인중"
	w = call(a, "GET", url, nil)
	if strings.Contains(w.Body.String(), "MSMF") {
		t.Fatal("non-received exposed receipt rows")
	}
	a.state.Projects[0].SuppliedMaterials = "사급품 수령"
	os.WriteFile(path, []byte("broken"), 0600)
	a.syncSchedule(context.Background(), true)
	if len(a.state.MaterialReceipts["T-001"]) != 2 {
		t.Fatal("failed download cleared good data")
	}
	os.WriteFile(path, materialBook(t, "사급품 수령", true, false), 0600)
	a.syncSchedule(context.Background(), true)
	if len(a.state.MaterialReceipts) != 0 || a.state.MaterialReceiptsError == "" {
		t.Fatal("missing sheet retained misleading receipts")
	}
}
func TestMaterialReceiptMergedOrderAndMissingColumns(t *testing.T) {
	b, e := excelize.OpenReader(bytes.NewReader(receiptBook(t)))
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	b.MergeCell("사급품 수령 내역", "A2", "A3")
	buf, _ := b.WriteToBuffer()
	rows, _, e := parseMaterialReceipts(buf.Bytes())
	if e != nil || len(rows["T-001"]) != 2 {
		t.Fatal("explicit merged order", e)
	}
	b.SetCellValue("사급품 수령 내역", "C1", "unknown")
	buf, _ = b.WriteToBuffer()
	_, available, e := parseMaterialReceipts(buf.Bytes())
	if e == nil || !available {
		t.Fatal("malformed header accepted")
	}
}
func TestMaterialReceiptBrowserFixture(t *testing.T) {
	dir := os.Getenv("MATERIAL_RECEIPT_HTML")
	if dir == "" {
		t.Skip("optional browser fixture")
	}
	os.MkdirAll(dir, 0700)
	a := fixture(t)
	a.state.Projects[0].SuppliedMaterials = "사급품 수령"
	a.state.Projects[0].SuppliedMaterialsColor = "#00B050"
	a.state.Projects[0].ShippingDueManaged = true
	for name, path := range map[string]string{"work.html": "/work/" + a.state.Board + "?entry=skip", "admin.html": "/", "info.html": "/project-info/" + a.state.Board + "/1", "admin-info.html": "/projects/1/info"} {
		w := call(a, "GET", path, nil, adminCookie(a))
		if w.Code != 200 {
			t.Fatal(name, w.Code)
		}
		os.WriteFile(filepath.Join(dir, name), w.Body.Bytes(), 0600)
	}
	b, _ := json.Marshal(map[string]any{"board": a.state.Board, "items": []MaterialReceipt{{"모터", "PANASONIC", "MSMF012L1T1", "2"}, {"모터", "PANASONIC", "MSMF022L1S1", "1"}}})
	os.WriteFile(filepath.Join(dir, "fixture.json"), b, 0600)
}
func TestMaterialReceiptUploadedWorkbook(t *testing.T) {
	path := os.Getenv("MATERIAL_RECEIPT_WORKBOOK")
	if path == "" {
		t.Skip("optional read-only uploaded workbook test")
	}
	data, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	rows, available, e := parseMaterialReceipts(data)
	if e != nil || !available {
		t.Fatal(e)
	}
	items := rows["A261026-001"]
	if len(items) != 2 || items[0].PartNumber != "MSMF012L1T1" || items[0].Quantity != "2" || items[1].PartNumber != "MSMF022L1S1" || items[1].Quantity != "1" {
		t.Fatal("uploaded workbook sample differs")
	}
	t.Log("Uploaded workbook: two expected Panasonic rows and quantities verified with production parser; workbook read only")
}
func TestMaterialReceiptNextSchedulePollReplacesChangedAndRemovedRows(t *testing.T) {
	a := fixture(t)
	path := filepath.Join(a.dir, "schedule.xlsx")
	a.state.Integration.Schedule = ScheduleConfig{Enabled: true, Mode: "file", Path: path}
	data := receiptBook(t)
	os.WriteFile(path, data, 0600)
	a.syncSchedule(context.Background(), true)
	b, e := excelize.OpenReader(bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	b.SetCellValue("사급품 수령 내역", "E2", 7)
	b.RemoveRow("사급품 수령 내역", 3)
	buf, _ := b.WriteToBuffer()
	os.WriteFile(path, buf.Bytes(), 0600)
	a.syncSchedule(context.Background(), false)
	rows := a.state.MaterialReceipts["T-001"]
	if len(rows) != 1 || rows[0].Quantity != "7" {
		t.Fatal("next automatic poll did not replace receipt values", rows)
	}
	w := call(a, "GET", "/material-receipts/"+a.state.Board+"?number=T-001", nil)
	if strings.Contains(w.Body.String(), "MSMF022L1S1") || !strings.Contains(w.Body.String(), `"quantity":"7"`) {
		t.Fatal("stale receipt rows returned", w.Body.String())
	}
}
