package main

import (
	"errors"
	"fmt"
	"github.com/xuri/excelize/v2"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
)

var reportHeaders = []string{"수주번호", "프로젝트명", "검사구분", "작업자", "완료일자", "완료시간"}

type WorkbookBinding struct {
	Sheet    string
	Row      int
	Columns  [6]int
	Baseline []string
}

func reportHeaderMap(row []string) ([6]int, bool) {
	cols := [6]int{}
	seen := map[int]bool{}
	for i, want := range reportHeaders {
		found := false
		for j, value := range row {
			if importNormal(value) == importNormal(want) {
				if seen[j] {
					return cols, false
				}
				cols[i] = j + 1
				seen[j] = true
				found = true
				break
			}
		}
		if !found {
			return cols, false
		}
	}
	return cols, true
}
func reportKey(values []string) string {
	if len(values) != 6 {
		return ""
	}
	return strings.Join([]string{values[0], values[2], values[4], values[5]}, "\x00")
}
func bindExistingWorkbook(data []byte, s State) (map[int64]WorkbookBinding, error) {
	b, _, e := analyzeExistingWorkbook(data, s)
	return b, e
}
func mergedWorkbook(s State) ([]byte, error) {
	book, e := openImportBook(s.WorkbookBase)
	if e != nil {
		return nil, e
	}
	defer book.Close()
	current := map[int64]Event{}
	for _, event := range completedEvents(s) {
		current[event.ID] = event
	}
	ids := []int64{}
	for id := range s.WorkbookBindings {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		b := s.WorkbookBindings[id]
		event, exists := current[id]
		values := make([]string, 6)
		if exists {
			values = eventRow(s, event)
		}
		if len(b.Baseline) != 6 {
			return nil, errors.New("엑셀 연결 기준이 손상됐습니다")
		}
		for j, col := range b.Columns {
			if col < 1 || col > 16384 || b.Row < 1 {
				return nil, errors.New("엑셀 연결 위치가 올바르지 않습니다")
			}
			if exists && values[j] == b.Baseline[j] {
				continue
			}
			axis, _ := excelize.CoordinatesToCellName(col, b.Row)
			if e = book.SetCellStr(b.Sheet, axis, values[j]); e != nil {
				return nil, e
			}
		}
	}
	styleGreen, err := book.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F0FDF4"}}})
	if err != nil {
		return nil, err
	}
	styleYellow, err := book.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFF8DD"}}})
	if err != nil {
		return nil, err
	}
	type destination struct {
		last  string
		tone  int
		sheet string
		cols  [6]int
		next  int
	}
	destinations := map[string]destination{}
	for _, event := range completedEvents(s) {
		if _, ok := s.WorkbookBindings[event.ID]; ok {
			continue
		}
		month := event.Completed.In(kst).Format("2006-01")
		dest, ok := destinations[month]
		if !ok {
			sheet := month
			has := false
			for _, name := range book.GetSheetList() {
				if name == sheet {
					has = true
				}
			}
			cols := [6]int{1, 2, 3, 4, 5, 6}
			next := 2
			validHeader := false
			if has {
				rows, err := book.GetRows(sheet)
				if err != nil {
					return nil, err
				}
				for i, row := range rows {
					if i >= 30 {
						break
					}
					if c, yes := reportHeaderMap(row); yes {
						cols = c
						validHeader = true
						break
					}
				}
				if validHeader {
					next = len(rows) + 1
				}
			}
			if has && !validHeader {
				sheet = month + " (검사)"
				for i := 2; ; i++ {
					exists := false
					for _, name := range book.GetSheetList() {
						if name == sheet {
							exists = true
						}
					}
					if !exists {
						break
					}
					sheet = fmt.Sprintf("%s (검사 %d)", month, i)
				}
				has = false
			}
			if !has {
				if _, e = book.NewSheet(sheet); e != nil {
					return nil, e
				}
				for j, h := range reportHeaders {
					axis, _ := excelize.CoordinatesToCellName(j+1, 1)
					if e = book.SetCellStr(sheet, axis, h); e != nil {
						return nil, e
					}
				}
				book.SetColWidth(sheet, "A", "F", 22)
			}
			dest = destination{sheet: sheet, cols: cols, next: next}
		}
		values := eventRow(s, event)
		if dest.last != "" && dest.last != values[0] {
			dest.tone = 1 - dest.tone
		}
		dest.last = values[0]
		style := styleGreen
		if dest.tone == 1 {
			style = styleYellow
		}
		for j, value := range values {
			axis, _ := excelize.CoordinatesToCellName(dest.cols[j], dest.next)
			if e = book.SetCellStr(dest.sheet, axis, value); e != nil {
				return nil, e
			}
			if e = book.SetCellStyle(dest.sheet, axis, axis, style); e != nil {
				return nil, e
			}
		}
		dest.next++
		destinations[month] = dest
	}
	buf, e := book.WriteToBuffer()
	if e != nil {
		return nil, e
	}
	return buf.Bytes(), nil
}
func (a *App) reportSettingsGET(w http.ResponseWriter, r *http.Request, v map[string]any) bool {
	if r.URL.Path != "/report-settings" {
		return false
	}
	v["WorkbookBaseName"] = a.state.WorkbookBaseName
	v["WorkbookBoundCount"] = len(a.state.WorkbookBindings)
	active := 0
	for _, event := range completedEvents(a.state) {
		if _, ok := a.state.WorkbookBindings[event.ID]; ok {
			active++
		}
	}
	v["WorkbookActiveBoundCount"] = active
	if len(a.state.WorkbookBase) > 0 {
		_, check, e := analyzeExistingWorkbook(a.state.WorkbookBase, a.state)
		v["WorkbookCheck"] = check
		if e != nil {
			v["WorkbookCheckError"] = e.Error()
		}
	}

	a.render(w, "report-settings.html", v)
	return true
}
func (a *App) reportSettingsPOST(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/report-settings" {
		return false
	}
	fail := func(e error) { http.Error(w, e.Error(), 409) }
	switch r.FormValue("action") {
	case "recheck":
		if len(a.state.WorkbookBase) == 0 {
			fail(errors.New("먼저 기존 엑셀 파일을 연결하세요"))
			return true
		}
		if e := a.transaction(func(s *State) error { return recheckWorkbookBindings(s) }); e != nil {
			fail(e)
			return true
		}

	case "reset":
		if e := a.transaction(func(s *State) error {
			s.WorkbookBase = nil
			s.WorkbookBaseName = ""
			s.WorkbookBindings = nil
			return nil
		}); e != nil {
			fail(e)
			return true
		}
	case "attach":
		f, h, e := r.FormFile("workbook")
		if e != nil {
			fail(errors.New("기존 엑셀 파일을 선택하세요"))
			return true
		}
		defer f.Close()
		if strings.ToLower(filepath.Ext(h.Filename)) != ".xlsx" {
			fail(errors.New(".xlsx 파일만 사용할 수 있습니다"))
			return true
		}
		data, e := io.ReadAll(io.LimitReader(f, (5<<20)+1))
		if e != nil || len(data) > 5<<20 {
			fail(errors.New("기존 엑셀은 5MB 이하여야 합니다"))
			return true
		}
		bindings, e := bindExistingWorkbook(data, a.state)
		if e != nil {
			fail(e)
			return true
		}
		e = a.transaction(func(s *State) error {
			s.WorkbookBase = append([]byte{}, data...)
			s.WorkbookBaseName = filepath.Base(h.Filename)
			s.WorkbookBindings = bindings
			_, err := mergedWorkbook(*s)
			return err
		})
		if e != nil {
			fail(e)
			return true
		}
	default:
		fail(errors.New("엑셀 저장 설정 요청을 확인하세요"))
		return true
	}
	http.Redirect(w, r, "/report-settings", 303)
	return true
}
