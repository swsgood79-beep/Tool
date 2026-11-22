package main

import (
	"context"
	"github.com/xuri/excelize/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestShippingExportMetadataOnlyFillsTrustedMissingFields(t *testing.T) {
	a := exportFixture(t)
	r := a.state.ShippingRecords[0]
	r.ClientName = ""
	r.Project.ShippingDueDate = ""
	c := ClientNote{ID: token(), Name: "은일", Revision: 1}
	a.state.ClientNotes = append(a.state.ClientNotes, c)
	p := &a.state.Projects[0]
	p.ClientID = c.ID
	p.ShippingDueDate = "2026-11-19"
	got := shippingExportMetadata(&a.state, r)
	if got.ClientName != "은일" || got.Project.ShippingDueDate != "2026-11-19" || !strings.Contains(got.ExportSourceNote, "현재 동일 수주") {
		t.Fatal("trusted source not filled", got.ClientName, got.Project.ShippingDueDate, got.ExportSourceNote)
	}
	r.ClientName = "보관 당시 고객사"
	r.Project.ShippingDueDate = "2026-11-18"
	got = shippingExportMetadata(&a.state, r)
	if got.ClientName != r.ClientName || got.Project.ShippingDueDate != r.Project.ShippingDueDate {
		t.Fatal("overwrote archived truth")
	}
	r.ClientName = ""
	r.Project.ShippingDueDate = ""
	r.Project.Shipping.EventID++
	got = shippingExportMetadata(&a.state, r)
	if got.ClientName != "" || got.Project.ShippingDueDate != "" {
		t.Fatal("used a different inspection cycle")
	}
	r.Project.Shipping.EventID = p.Shipping.EventID
	r.Project.Number = "OTHER"
	got = shippingExportMetadata(&a.state, r)
	if got.ClientName != "" || got.Project.ShippingDueDate != "" {
		t.Fatal("used a different order")
	}
	// A customer-looking project name is not a trusted customer source.
	r.Project.Number = p.Number
	p.ClientID = ""
	p.ShippingDueDate = ""
	got = shippingExportMetadata(&a.state, r)
	if got.ClientName != "" || !strings.Contains(got.ExportSourceNote, "원자료 미등록") {
		t.Fatal("guessed a customer")
	}
}
func TestShippingExportCompactGridAndMissingMarkers(t *testing.T) {
	a := exportFixture(t)
	r := a.state.ShippingRecords[0]
	b := newExportBook("목차")
	defer b.File.Close()
	entry := exportEntry{Record: r, Sheet: "상세"}
	b.detail(entry, nil)
	if b.Err != nil {
		t.Fatal(b.Err)
	}
	for cell, want := range map[string]string{"C4": r.Project.Number, "C5": "2026.11.21", "C8": "MOT", "G4": "설계담당자", "I4": "세팅 작업자", "K4": "출하 작업자", "G5": "김설계", "I5": "박세팅", "K5": "이출하"} {
		got, e := b.File.GetCellValue("상세", cell)
		if e != nil || got != want {
			t.Fatal(cell, got, want, e)
		}
	}
	covered := map[string]bool{}
	merges, e := b.File.GetMergeCells("상세")
	if e != nil {
		t.Fatal(e)
	}
	for _, merge := range merges {
		x1, y1, e := excelize.CellNameToCoordinates(merge.GetStartAxis())
		if e != nil {
			t.Fatal(e)
		}
		x2, y2, e := excelize.CellNameToCoordinates(merge.GetEndAxis())
		if e != nil {
			t.Fatal(e)
		}
		for x := x1; x <= x2; x++ {
			for y := y1; y <= y2; y++ {
				cell, _ := excelize.CoordinatesToCellName(x, y)
				if covered[cell] {
					t.Fatal("overlapping merged cells", cell)
				}
				covered[cell] = true
			}
		}
	}
	for _, cell := range []string{"G5", "I5", "K5"} {
		id, _ := b.File.GetCellStyle("상세", cell)
		style, e := b.File.GetStyle(id)
		if e != nil || style.Alignment.Horizontal != "center" {
			t.Fatal("worker names misaligned", cell)
		}
	}
	r.ClientName = ""
	r.Project.ShippingDueDate = ""
	empty := newExportBook("목차")
	defer empty.File.Close()
	empty.detail(exportEntry{Record: r, Sheet: "상세"}, nil)
	for _, cell := range []string{"C5", "C8"} {
		got, _ := empty.File.GetCellValue("상세", cell)
		if got != "미등록" {
			t.Fatal("silent blank metadata", cell, got)
		}
	}
}
func TestShippingExportCompactLayoutSample(t *testing.T) {
	if os.Getenv("EXPORT_LAYOUT_SAMPLE84") != "1" {
		t.Skip("optional real XLSX layout sample")
	}
	a := exportFixture(t)
	r := a.state.ShippingRecords[0]
	dir := t.TempDir()
	_, _, _, err := buildShippingExport(context.Background(), dir, []ShippingRecord{r}, time.Now(), func(p Photo) ([]byte, error) { return shippingExportPhoto(a.dir, p) }, nil, shippingExportLimits{PhotoBytes: 24 << 20, Records: 40, Single: true})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "export.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile("/tmp/alpha-excel-layout84.xlsx", raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestShippingFinalArchivePreservesLateKnownMetadata(t *testing.T) {
	a := fixture(t)
	if err := a.transaction(func(s *State) error { s.Projects[0].ShippingDueDate = ""; s.Projects[0].ClientID = ""; return nil }); err != nil {
		t.Fatal(err)
	}
	rec := approveShipment(t, a)
	if rec.ClientName != "" || rec.Project.ShippingDueDate != "" {
		t.Fatal("fixture should start with missing metadata")
	}
	c := ClientNote{ID: token(), Name: "확정 고객사", Text: "공통 안내", Revision: 1}
	if err := a.transaction(func(s *State) error {
		s.ClientNotes = append(s.ClientNotes, c)
		s.Projects[0].ClientID = c.ID
		s.Projects[0].ShippingDueDate = "2026-11-21"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if w := shippingUpload(t, a, rec.ID, 1, false); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	stored := a.state.ShippingRecords[0]
	if stored.ClientName != c.Name || stored.Project.ShippingDueDate != "2026-11-21" {
		t.Fatal("final archive lost confirmed metadata", stored.ClientName, stored.Project.ShippingDueDate)
	}
	if err := a.transaction(func(s *State) error {
		s.Projects[0].ShippingDueDate = "2026-12-31"
		s.Projects[0].ClientID = ""
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	got := shippingExportMetadata(&a.state, stored)
	if got.ClientName != c.Name || got.Project.ShippingDueDate != "2026-11-21" {
		t.Fatal("later order change rewrote archive")
	}
}

func TestShippingExportSnapshotMetadataIsReadOnly(t *testing.T) {
	a := exportFixture(t)
	r := &a.state.ShippingRecords[0]
	r.ClientName = ""
	r.Project.ShippingDueDate = ""
	c := ClientNote{ID: token(), Name: "확정 고객사", Text: "공통 안내", Revision: 1}
	a.state.ClientNotes = append(a.state.ClientNotes, c)
	a.state.Projects[0].ClientID = c.ID
	a.state.Projects[0].ShippingDueDate = "2026-11-21"
	a.mu.Lock()
	job, records, _, err := a.newShippingExportLocked("", token(), r.ID)
	a.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(job.Dir)
	if records[0].ClientName != c.Name || records[0].Project.ShippingDueDate != "2026-11-21" || job.MissingMetadata != 0 {
		t.Fatal("snapshot did not include trusted metadata")
	}
	if r.ClientName != "" || r.Project.ShippingDueDate != "" {
		t.Fatal("export silently rewrote archive")
	}
}
