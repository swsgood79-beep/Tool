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
	"time"
)

func materialBook(t *testing.T, value string, header bool, merge bool) []byte {
	t.Helper()
	b := excelize.NewFile()
	defer b.Close()
	b.SetSheetName("Sheet1", "조립일정")
	b.SetCellValue("조립일정", "A1", "수주번호")
	b.SetCellValue("조립일정", "B1", "발송예정일")
	if header {
		b.SetCellValue("조립일정", "N1", "사급 관련")
	}
	for i, n := range []string{"T-001", "T-002"} {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		b.SetCellValue("조립일정", cell, n)
		cell, _ = excelize.CoordinatesToCellName(2, i+2)
		b.SetCellValue("조립일정", cell, "2026-11-20")
	}
	b.SetCellValue("조립일정", "N2", value)
	if merge {
		b.MergeCell("조립일정", "N2", "N3")
	}
	buf, e := b.WriteToBuffer()
	if e != nil {
		t.Fatal(e)
	}
	return buf.Bytes()
}
func TestScheduleMaterialsNColumnMergedBlankAndRepeatedOrders(t *testing.T) {
	data := materialBook(t, "사급품 확인 중", true, true)
	m, available, e := parseScheduleMaterials(data)
	if e != nil || !available || m["T-001"] != "사급품 확인 중" || m["T-002"] != "사급품 확인 중" {
		t.Fatal("merged N mapping failed", e)
	}
	b, e := excelize.OpenReader(bytes.NewReader(materialBook(t, "부품 A", true, false)))
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	for row, value := range map[int]string{4: "부품 A", 5: "부품 B"} {
		cell, _ := excelize.CoordinatesToCellName(1, row)
		b.SetCellValue("조립일정", cell, "T-001")
		cell, _ = excelize.CoordinatesToCellName(14, row)
		b.SetCellValue("조립일정", cell, value)
	}
	buf, _ := b.WriteToBuffer()
	m, _, e = parseScheduleMaterials(buf.Bytes())
	if e != nil || m["T-001"] != "부품 A\n부품 B" || m["T-002"] != "" {
		t.Fatal("repeated or blank N lost", e)
	}
	_, available, e = parseScheduleMaterials(materialBook(t, "ignored", false, false))
	if e != nil || available {
		t.Fatal("unidentified N treated as source")
	}
}
func TestScheduleMaterialsSyncPersistsAndDoesNotOverwriteManualNotes(t *testing.T) {
	a := fixture(t)
	a.state.Projects[0].ShippingDueDate = "2026-11-20"
	a.state.Projects[0].SpecialInstructions = "관리자 특이사항"
	path := filepath.Join(a.dir, "schedule.xlsx")
	a.state.Integration.Schedule = ScheduleConfig{Enabled: true, Mode: "file", Path: path}
	write := func(data []byte) {
		t.Helper()
		if e := os.WriteFile(path, data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	write(materialBook(t, "입고 대기", true, false))
	a.syncSchedule(context.Background(), true)
	p := a.state.Projects[0]
	if p.SuppliedMaterials != "입고 대기" || p.SpecialInstructions != "관리자 특이사항" || a.integration.ScheduleStatus["material_changed"] != 1 || a.integration.ScheduleStatus["changed"] != 0 {
		t.Fatal("independent material sync failed")
	}
	again, e := newApp(a.dir)
	if e != nil || again.state.Projects[0].SuppliedMaterials != "입고 대기" {
		t.Fatal("restart lost materials", e)
	}
	legacy, e := importLegacy(a.dir)
	if e != nil || legacy.Projects[0].SuppliedMaterials != "입고 대기" {
		t.Fatal("canonical projection lost materials", e)
	}
	write(materialBook(t, "ignored", false, false))
	a.syncSchedule(context.Background(), true)
	if a.state.Projects[0].SuppliedMaterials != "입고 대기" {
		t.Fatal("missing N header cleared old value")
	}
	write([]byte("broken"))
	a.syncSchedule(context.Background(), true)
	if a.state.Projects[0].SuppliedMaterials != "입고 대기" {
		t.Fatal("failed download cleared old value")
	}
	write(materialBook(t, "", true, false))
	a.syncSchedule(context.Background(), true)
	if a.state.Projects[0].SuppliedMaterials != "" {
		t.Fatal("explicit blank not cleared")
	}
}
func TestSuppliedMaterialsViewsAndCompletedStatusPolling(t *testing.T) {
	a := fixture(t)
	text := "고객 지급 <시험> 부품\n확인 중"
	a.state.Projects[0].SuppliedMaterials = text
	a.state.Projects[0].SpecialInstructions = "특이사항"
	cookie := adminCookie(a)
	for _, path := range []string{"/", "/work/" + a.state.Board + "?entry=skip", "/projects/1/inspection", "/projects/1/info", "/project-info/" + a.state.Board + "/1"} {
		r := call(a, "GET", path, nil, cookie)
		body := r.Body.String()
		if r.Code != 200 || !strings.Contains(body, "supplied-materials-text") || strings.Contains(body, "고객 지급 <시험>") {
			t.Fatal("material view missing or unescaped", path)
		}
		if out := os.Getenv("MATERIAL_HTML_OUT"); out != "" && (path == "/" || strings.HasPrefix(path, "/work/")) {
			name := "admin.html"
			if path != "/" {
				name = "work.html"
			}
			if e := os.WriteFile(filepath.Join(out, name), r.Body.Bytes(), 0600); e != nil {
				t.Fatal(e)
			}
		}
	}
	now := time.Now()
	a.state.Projects[0].Shipping.Completed = &now
	r := call(a, "GET", "/status/"+a.state.Board, nil)
	var data map[string]any
	if json.Unmarshal(r.Body.Bytes(), &data) != nil {
		t.Fatal("bad status")
	}
	shipments := data["shipments"].([]any)
	if len(shipments) != 1 || shipments[0].(map[string]any)["supplied_materials"] != text {
		t.Fatal("poll payload lost supplied materials")
	}
}
