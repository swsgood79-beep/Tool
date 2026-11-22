package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

func TestShippingExportCustomerNamingConvention(t *testing.T) {
	for _, tc := range []struct{ name, stored, want string }{
		{"비엔에스 4축 (선진행, 예가)", "", "비엔에스"},
		{"은일 2축", "", "은일"},
		{"영우DSP 1축 (예가)", "이전 이름", "영우DSP"},
		{"  에이 비 회사 100축  ", "", "에이 비 회사"},
		{"DNC(하나마이크론) 12축", "", "DNC(하나마이크론)"},
		{"3M 코리아\t2축", "", "3M 코리아"},
		{"A261105-001", "기존 고객사", "기존 고객사"},
		{"테스트", "", ""},
		{"테스트 2축제어기", "원본 업체", "원본 업체"},
	} {
		r := ShippingRecord{Project: Project{Name: tc.name}, ClientName: tc.stored}
		if got := shippingExportClientName(r); got != tc.want {
			t.Errorf("%q: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestShippingExportCustomerAndNoDetailNavigation(t *testing.T) {
	a := exportFixture(t)
	stored := &a.state.ShippingRecords[0]
	stored.Project.Name = "비엔에스 4축 (선진행, 예가)"
	stored.ClientName = ""
	j, records, _, err := a.newShippingExportLocked("owner", token(), stored.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(j.Dir)
	if records[0].ClientName != "비엔에스" || j.MissingMetadata != 0 || stored.ClientName != "" {
		t.Fatal("export customer missing or archive modified", records[0].ClientName, j.MissingMetadata, stored.ClientName)
	}
	for _, single := range []bool{true, false} {
		dir := t.TempDir()
		_, _, _, err = buildShippingExport(context.Background(), dir, records, time.Now(), func(p Photo) ([]byte, error) { return shippingExportPhoto(a.dir, p) }, nil, shippingExportLimits{PhotoBytes: 24 << 20, Records: 40, Single: single})
		if err != nil {
			t.Fatal(err)
		}
		books := map[string][]byte{}
		if single {
			books["single"] = mustRead(t, filepath.Join(dir, "export.xlsx"))
			if os.Getenv("EXPORT_LAYOUT_SAMPLE85") == "1" {
				if err := os.WriteFile("/tmp/alpha-excel-layout85.xlsx", books["single"], 0600); err != nil {
					t.Fatal(err)
				}
			}
		} else {
			books = exportArchive(t, filepath.Join(dir, "export.zip"))
		}
		details := 0
		for name, raw := range books {
			if name != "single" && !strings.HasSuffix(name, ".xlsx") {
				continue
			}
			f, err := excelize.OpenReader(bytes.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}
			for _, sheet := range f.GetSheetList() {
				label, _ := f.GetCellValue(sheet, "A8")
				if label != "고객사" {
					continue
				}
				details++
				got, _ := f.GetCellValue(sheet, "C8")
				if got != "비엔에스" {
					t.Fatal("incorrect workbook customer", name, sheet, got)
				}
				for _, cell := range []string{"A1", "J1"} {
					has, target, err := f.GetCellHyperLink(sheet, cell)
					if err != nil || has || target != "" {
						t.Fatal("detail navigation remains", name, cell, target, err)
					}
				}
				merges, _ := f.GetMergeCells(sheet)
				if len(merges) == 0 || merges[0].GetStartAxis() != "A1" || merges[0].GetEndAxis() != "L2" {
					t.Fatal("title does not span the full width")
				}
			}
			f.Close()
		}
		if details != 1 {
			t.Fatal("expected exactly one detail sheet", single, details)
		}
	}
}
