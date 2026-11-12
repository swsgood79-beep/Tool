package main

import (
	"errors"
	"fmt"
	"github.com/xuri/excelize/v2"
	"math"
	"strconv"
	"strings"
	"time"
)

type WorkbookCheck struct {
	ExcelRows, ValidRows, InvalidRows, ProgramRecords, Matched, ExcelUnmatched, ProgramUnmatched int
	UnmatchedExamples                                                                            []string
}
type workbookCandidate struct {
	Binding    WorkbookBinding
	Values     []string
	MinuteOnly bool
}

func normalizedReportKind(s string) string {
	switch importNormal(s) {
	case "세팅", "셋팅", "세팅검사", "셋팅검사", "setting", "setup":
		return "setting"
	case "출하", "출하검사", "shipping":
		return "shipping"
	}
	return ""
}
func normalizedReportDay(raw string, date1904 bool) (string, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{"2006-01-02", "2006-1-2", "2006.01.02", "2006.1.2", "2006/01/02", "2006/1/2", "20060102", "2006년 1월 2일", "2006-01-02 15:04:05", "2006/1/2 15:04:05", time.RFC3339Nano} {
		if t, e := time.ParseInLocation(layout, raw, kst); e == nil {
			return t.In(kst).Format("2006-01-02"), nil
		}
	}
	compact := strings.ReplaceAll(strings.Join(strings.Fields(raw), ""), "\u00a0", "")
	compact = strings.TrimSuffix(compact, ".")
	for _, layout := range []string{"2006.1.2", "2006-1-2", "2006/1/2", "2006년1월2일"} {
		if t, err := time.ParseInLocation(layout, compact, kst); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}
	if n, e := strconv.ParseFloat(raw, 64); e == nil && n > 0 && n < 3000000 {
		if t, e := excelize.ExcelDateToTime(n, date1904); e == nil {
			return t.Format("2006-01-02"), nil
		}
	}
	return "", errors.New("완료일자를 읽지 못했습니다")
}
func normalizedReportTime(raw string) (string, bool, error) {
	raw = strings.TrimSpace(raw)
	if n, e := strconv.ParseFloat(raw, 64); e == nil && n >= 0 && n < 1 {
		seconds := int(math.Round(n * 86400))
		if seconds >= 86400 {
			return "", false, errors.New("완료시간 범위 오류")
		}
		return fmt.Sprintf("%02d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60), false, nil
	}
	if strings.HasPrefix(raw, "오전 ") || strings.HasPrefix(raw, "오후 ") {
		afternoon := strings.HasPrefix(raw, "오후 ")
		value := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(raw, "오전 "), "오후 "))
		parts := strings.Split(value, ":")
		if len(parts) >= 2 && len(parts) <= 3 {
			hour, e := strconv.Atoi(parts[0])
			if e == nil && hour >= 1 && hour <= 12 {
				hour %= 12
				if afternoon {
					hour += 12
				}
				raw = fmt.Sprintf("%02d:%s", hour, strings.Join(parts[1:], ":"))
			}
		}
	}
	for _, layout := range []string{"15:04:05.999999999", "15:04:05", "15:04", "3:04:05 PM", "3:04 PM", "03:04:05 PM", "03:04 PM"} {
		if t, e := time.Parse(layout, strings.ToUpper(raw)); e == nil {
			return t.Format("15:04:05"), !strings.Contains(layout, ":05"), nil
		}
	}
	return "", false, errors.New("완료시간을 읽지 못했습니다")
}
func normalizedReportIdentity(values []string, date1904 bool) (string, bool, error) {
	if len(values) != 6 {
		return "", false, errors.New("검사 기록 열 오류")
	}
	order := strings.TrimSpace(values[0])
	kind := normalizedReportKind(values[2])
	if order == "" || kind == "" {
		return "", false, errors.New("수주번호 또는 검사구분을 읽지 못했습니다")
	}
	date, e := normalizedReportDay(values[4], date1904)
	if e != nil {
		return "", false, e
	}
	timeValue := strings.TrimSpace(values[5])
	if n, err := strconv.ParseFloat(timeValue, 64); err == nil && n >= 1 && n < 3000000 {
		combined, err := excelize.ExcelDateToTime(n, date1904)
		if err != nil || combined.Format("2006-01-02") != date {
			return "", false, errors.New("완료시간 셀의 날짜가 완료일자와 다릅니다")
		}
		timeValue = strconv.FormatFloat(n-math.Floor(n), 'f', 12, 64)
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05", "2006/1/2 15:04:05"} {
		if combined, err := time.ParseInLocation(layout, timeValue, kst); err == nil {
			if combined.In(kst).Format("2006-01-02") != date {
				return "", false, errors.New("완료시간에 포함된 날짜가 완료일자와 다릅니다")
			}
			timeValue = combined.In(kst).Format("15:04:05")
			break
		}
	}
	clock, minute, e := normalizedReportTime(timeValue)
	if e != nil {
		return "", false, e
	}
	if minute {
		clock = clock[:5]
	}
	return strings.Join([]string{order, kind, date, clock}, "\x00"), minute, nil
}
func reportMinuteKey(exact string) string {
	parts := strings.Split(exact, "\x00")
	if len(parts) == 4 && len(parts[3]) >= 5 {
		parts[3] = parts[3][:5]
	}
	return strings.Join(parts, "\x00")
}
func analyzeExistingWorkbook(data []byte, s State) (map[int64]WorkbookBinding, WorkbookCheck, error) {
	check := WorkbookCheck{ProgramRecords: len(completedEvents(s))}
	book, e := openImportBook(data)
	if e != nil {
		return nil, check, errors.New("기존 엑셀을 열지 못했습니다. 암호 없는 .xlsx 파일인지 확인하세요")
	}
	defer book.Close()
	props, e := book.GetWorkbookProps()
	if e != nil {
		return nil, check, e
	}
	date1904 := props.Date1904 != nil && *props.Date1904
	candidates := map[string][]workbookCandidate{}
	headers := 0
	for _, sheet := range book.GetSheetList() {
		rows, e := book.GetRows(sheet, excelize.Options{RawCellValue: true})
		if e != nil {
			return nil, check, e
		}
		if len(rows) > 30000 {
			return nil, check, errors.New("엑셀 시트는 30,000행 이하여야 합니다")
		}
		for i, row := range rows {
			if i >= 30 {
				break
			}
			cols, ok := reportHeaderMap(row)
			if !ok {
				continue
			}
			headers++
			for ri := i + 1; ri < len(rows); ri++ {
				values := make([]string, 6)
				for j, col := range cols {
					if col <= len(rows[ri]) {
						values[j] = rows[ri][col-1]
					}
				}
				if strings.TrimSpace(values[0]) == "" || strings.TrimSpace(values[2]) == "" {
					continue
				}
				check.ExcelRows++
				key, minute, err := normalizedReportIdentity(values, date1904)
				if err != nil {
					check.InvalidRows++
					if len(check.UnmatchedExamples) < 5 {
						check.UnmatchedExamples = append(check.UnmatchedExamples, fmt.Sprintf("%s %d행 · %s · %s", sheet, ri+1, values[0], err.Error()))
					}
					continue
				}
				check.ValidRows++
				candidates[key] = append(candidates[key], workbookCandidate{WorkbookBinding{Sheet: sheet, Row: ri + 1, Columns: cols}, values, minute})
			}
			break
		}
	}
	if headers == 0 {
		return nil, check, errors.New("기존 엑셀에 수주번호·프로젝트명·검사구분·작업자·완료일자·완료시간 제목 열이 필요합니다")
	}
	check.ExcelUnmatched = check.ExcelRows
	check.ProgramUnmatched = check.ProgramRecords
	events := completedEvents(s)
	eventKeys := map[int64]string{}
	minuteCounts := map[string]int{}
	for _, event := range events {
		key, _, err := normalizedReportIdentity(eventRow(s, event), false)
		if err != nil {
			return nil, check, err
		}
		eventKeys[event.ID] = key
		minuteCounts[reportMinuteKey(key)]++
	}
	result := map[int64]WorkbookBinding{}
	used := map[string]bool{}
	for _, event := range events {
		key := eventKeys[event.ID]
		matches := append([]workbookCandidate{}, candidates[key]...)
		minuteKey := reportMinuteKey(key)
		if len(candidates[minuteKey]) > 0 {
			if minuteCounts[minuteKey] > 1 {
				return nil, check, errors.New("엑셀에는 분까지만 적혀 있는데 같은 수주·검사구분의 기록이 같은 분에 여러 건 있습니다. 완료시간에 초까지 표시해 주세요")
			}
			matches = append(matches, candidates[minuteKey]...)
		}
		if len(matches) > 1 {
			return nil, check, fmt.Errorf("동일한 검사 기록이 엑셀의 여러 행과 일치합니다: %s / %s. 중복 행을 확인해 주세요", event.Order, stageLabel(event.Inspection))
		}
		if len(matches) == 0 {
			continue
		}
		b := matches[0].Binding
		location := fmt.Sprintf("%s\x00%d", b.Sheet, b.Row)
		if used[location] {
			return nil, check, errors.New("하나의 엑셀 행에 여러 검사 기록이 연결됩니다. 수주·검사구분·완료시각을 확인해 주세요")
		}
		used[location] = true
		b.Baseline = eventRow(s, event)
		result[event.ID] = b
	}
	check.Matched = len(result)
	check.ExcelUnmatched = check.ExcelRows - check.Matched
	check.ProgramUnmatched = check.ProgramRecords - check.Matched
	return result, check, nil
}
func recheckWorkbookBindings(s *State) error {
	fresh, _, e := analyzeExistingWorkbook(s.WorkbookBase, *s)
	if e != nil {
		return e
	}
	if s.WorkbookBindings == nil {
		s.WorkbookBindings = map[int64]WorkbookBinding{}
	}
	occupied := map[string]bool{}
	for _, b := range s.WorkbookBindings {
		occupied[fmt.Sprintf("%s\x00%d", b.Sheet, b.Row)] = true
	}
	for id, b := range fresh {
		if _, exists := s.WorkbookBindings[id]; exists {
			continue
		}
		location := fmt.Sprintf("%s\x00%d", b.Sheet, b.Row)
		if occupied[location] {
			continue
		}
		s.WorkbookBindings[id] = b
		occupied[location] = true
	}
	return nil
}
