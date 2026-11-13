package main

import (
	"archive/zip"
	"bytes"
	"context"
	"github.com/xuri/excelize/v2"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaterialFirstRowAndFirstLineIncludingBlank(t *testing.T) {
	b, e := excelize.OpenReader(bytes.NewReader(materialBook(t, "선택 첫 줄\n읽지 않을 둘째 줄", true, false)))
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	b.SetCellValue("조립일정", "A3", "T-001")
	b.SetCellValue("조립일정", "N3", "뒤쪽 행 내용")
	buf, _ := b.WriteToBuffer()
	m, _, e := parseScheduleMaterialDetails(buf.Bytes())
	if e != nil || m["T-001"].Text != "선택 첫 줄" {
		t.Fatal("first cell/line not selected", e)
	}
	b.SetCellValue("조립일정", "N2", "")
	buf, _ = b.WriteToBuffer()
	m, _, e = parseScheduleMaterialDetails(buf.Bytes())
	if e != nil || m["T-001"].Text != "" {
		t.Fatal("later nonempty row replaced blank first cell")
	}
}
func TestMaterialRGBThemeAndConditionalColor(t *testing.T) {
	b, e := excelize.OpenReader(bytes.NewReader(materialBook(t, "사급품 수령", true, false)))
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	id, e := b.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"00B050"}}})
	if e != nil {
		t.Fatal(e)
	}
	b.SetCellStyle("조립일정", "N2", "N2", id)
	buf, _ := b.WriteToBuffer()
	m, _, e := parseScheduleMaterialDetails(buf.Bytes())
	if e != nil || m["T-001"].Color != "#00B050" {
		t.Fatal("direct fill differs", e, m)
	}
	rule, e := b.NewConditionalStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFFF00"}}})
	if e != nil {
		t.Fatal(e)
	}
	b.SetConditionalFormat("조립일정", "N1:O1048576", []excelize.ConditionalFormatOptions{{Type: "cell", Criteria: "=", Value: `"사급품 수령"`, Format: &rule}})
	buf, _ = b.WriteToBuffer()
	m, _, e = parseScheduleMaterialDetails(buf.Bytes())
	if e != nil || m["T-001"].Color != "#FFFF00" {
		t.Fatal("conditional fill not applied", e, m)
	}
	b.SetCellValue("조립일정", "N2", "사급품 수령\n부연 내용")
	buf, _ = b.WriteToBuffer()
	m, _, e = parseScheduleMaterialDetails(buf.Bytes())
	if e != nil || m["T-001"].Text != "사급품 수령" || m["T-001"].Color != "#00B050" {
		t.Fatal("condition used truncated text instead of full source cell", e, m)
	}
	if materialColorHex("red; background:url(x)") != "" {
		t.Fatal("unsafe CSS accepted")
	}
}
func TestMaterialOnlyColorChangePersistsAndRefreshes(t *testing.T) {
	a := fixture(t)
	p := filepath.Join(a.dir, "color.xlsx")
	a.state.Integration.Schedule = ScheduleConfig{Enabled: true, Mode: "file", Path: p}
	makeBook := func(color string) {
		t.Helper()
		b, e := excelize.OpenReader(bytes.NewReader(materialBook(t, "입고 대기", true, false)))
		if e != nil {
			t.Fatal(e)
		}
		defer b.Close()
		id, e := b.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{color}}})
		if e != nil {
			t.Fatal(e)
		}
		b.SetCellStyle("조립일정", "N2", "N2", id)
		buf, e := b.WriteToBuffer()
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(p, buf.Bytes(), 0600); e != nil {
			t.Fatal(e)
		}
	}
	makeBook("00B050")
	a.syncSchedule(context.Background(), true)
	rev := a.state.Projects[0].Revision
	makeBook("FFFF00")
	a.syncSchedule(context.Background(), true)
	if a.state.Projects[0].SuppliedMaterialsColor != "#FFFF00" || a.state.Projects[0].Revision <= rev || a.integration.ScheduleStatus["material_changed"] != 1 {
		t.Fatal("color-only change not tracked")
	}
	r, e := newApp(a.dir)
	if e != nil || r.state.Projects[0].SuppliedMaterialsColor != "#FFFF00" {
		t.Fatal("reopen lost color", e)
	}
	legacy, e := importLegacy(a.dir)
	if e != nil || legacy.Projects[0].SuppliedMaterialsColor != "#FFFF00" {
		t.Fatal("projection lost color", e)
	}
	if strings.Contains(a.state.Projects[0].SuppliedMaterials, "\n") {
		t.Fatal("multiple lines persisted")
	}
}

func TestMaterialThemeTintResolved(t *testing.T) {
	b, e := excelize.OpenReader(bytes.NewReader(materialBook(t, "테마 색", true, false)))
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	id, e := b.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"00B050"}}})
	if e != nil {
		t.Fatal(e)
	}
	b.SetCellStyle("조립일정", "N2", "N2", id)
	buf, _ := b.WriteToBuffer()
	reader, e := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if e != nil {
		t.Fatal(e)
	}
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	changed := false
	for _, file := range reader.File {
		r, e := file.Open()
		if e != nil {
			t.Fatal(e)
		}
		data, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			t.Fatal(e)
		}
		if file.Name == "xl/styles.xml" {
			old := data
			data = bytes.ReplaceAll(data, []byte(`rgb="FF00B050"`), []byte(`theme="6" tint="0.5"`))
			changed = !bytes.Equal(data, old)
		}
		w, e := writer.Create(file.Name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = w.Write(data); e != nil {
			t.Fatal(e)
		}
	}
	writer.Close()
	if !changed {
		t.Fatal("theme fixture did not replace RGB")
	}
	values, _, e := parseScheduleMaterialDetails(out.Bytes())
	theme := 6
	expected := materialColorHex(excelize.ThemeColor(b.GetBaseColor("", 0, &theme), 0.5))
	if e != nil || expected == "" || values["T-001"].Color != expected {
		t.Fatal("theme/tint color incorrect", e, expected, values["T-001"].Color)
	}
}
