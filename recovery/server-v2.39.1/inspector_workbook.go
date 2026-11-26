package main

import (
	"bytes"
	"fmt"
	"github.com/xuri/excelize/v2"
	"time"
)

func withInspectorWorkbook(raw []byte, s State) ([]byte, error) {
	if len(s.InspectionAudits) == 0 {
		found := false
		for _, e := range s.Events {
			if e.Inspector != nil {
				found = true
				break
			}
		}
		if !found {
			return raw, nil
		}
	}
	f, e := excelize.OpenReader(bytes.NewReader(raw))
	if e != nil {
		return nil, e
	}
	defer f.Close()
	sheet := ""
	marker := "검사자 기록 · 알파 자동 생성"
	base := "검사자 기록"
	sheet = base
	for i := 0; ; i++ {
		if i > 0 {
			sheet = fmt.Sprintf("%s (%d)", base, i)
		}
		exists := false
		for _, n := range f.GetSheetList() {
			if n == sheet {
				exists = true
				break
			}
		}
		if !exists {
			if _, e = f.NewSheet(sheet); e != nil {
				return nil, e
			}
			break
		}
		text, _ := f.GetCellValue(sheet, "A1")
		if text == marker {
			if len(f.GetSheetList()) == 1 {
				return nil, fmt.Errorf("검사 엑셀 기본 시트가 없습니다")
			}
			if e = f.DeleteSheet(sheet); e != nil {
				return nil, e
			}
			if _, e = f.NewSheet(sheet); e != nil {
				return nil, e
			}
			break
		}
	}
	f.SetCellStr(sheet, "A1", marker)
	headers := []string{"수주번호", "프로젝트명", "검사구분", "작업자", "검사자", "검사 완료일시", "기록 방식", "기록 일시", "보완 처리자", "보완 계정"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		if e = f.SetCellStr(sheet, cell, h); e != nil {
			return nil, e
		}
	}
	style, e := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "FFFFFF"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"173A69"}, Pattern: 1}})
	if e != nil {
		return nil, e
	}
	f.SetCellStyle(sheet, "A3", "J3", style)
	f.SetColWidth(sheet, "A", "J", 22)
	f.SetColWidth(sheet, "B", "B", 36)
	datetime := "yyyy.mm.dd hh:mm"
	dateStyle, e := f.NewStyle(&excelize.Style{CustomNumFmt: &datetime})
	if e != nil {
		return nil, e
	}
	for index, event := range completedEvents(s) {
		row := index + 4
		name, source := inspectionName(event.Inspector), inspectionSource(event.Inspector)
		actor, username := "—", "—"
		if event.Inspector != nil && event.Inspector.Source == "backfill" {
			actor, username = event.Inspector.RecordedByName, event.Inspector.RecordedByUsername
		}
		values := []string{event.Order, event.Project, eventInspectionLabel(event), event.Worker, name, "", source, "", actor, username}
		for i, v := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			if e = f.SetCellStr(sheet, cell, v); e != nil {
				return nil, e
			}
		}
		for cell, at := range map[string]*time.Time{fmt.Sprintf("F%d", row): event.Completed} {
			if at != nil {
				f.SetCellValue(sheet, cell, excelWallTime(*at))
				f.SetCellStyle(sheet, cell, cell, dateStyle)
			}
		}
		if event.Inspector != nil {
			cell := fmt.Sprintf("H%d", row)
			f.SetCellValue(sheet, cell, excelWallTime(event.Inspector.RecordedAt))
			f.SetCellStyle(sheet, cell, cell, dateStyle)
		}
	}
	if e = f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 3, TopLeftCell: "A4", ActivePane: "bottomLeft"}); e != nil {
		return nil, e
	}
	last := max(3, len(completedEvents(s))+3)
	if e = f.AutoFilter(sheet, fmt.Sprintf("A3:J%d", last), nil); e != nil {
		return nil, e
	}
	out, e := f.WriteToBuffer()
	if e != nil {
		return nil, e
	}
	return out.Bytes(), nil
}
