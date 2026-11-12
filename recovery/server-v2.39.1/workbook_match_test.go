package main

import (
	"bytes"
	"fmt"
	"github.com/xuri/excelize/v2"
	"net/url"
	"strings"
	"testing"
	"time"
)

func reportMatchState() State {
	t := time.Date(2026, 11, 13, 9, 30, 17, 0, kst)
	return State{Version: 1, Projects: []Project{{ID: 1, Number: "A-001", Name: "매칭 시험"}}, Events: []Event{{ID: 2, ProjectID: 1, Order: "A-001", Project: "매칭 시험", Worker: "원래 이름", Inspection: "setting", Completed: &t}}}
}
func matchBook(t *testing.T, date, clock any, kind string, date1904 bool) []byte {
	t.Helper()
	b := excelize.NewFile()
	defer b.Close()
	b.SetWorkbookProps(&excelize.WorkbookPropsOptions{Date1904: &date1904})
	headers := []any{"수주번호", "프로젝트명", "검사구분", "작업자", "완료일자", "완료시간"}
	b.SetSheetRow("Sheet1", "A1", &headers)
	row := []any{" A-001 ", "매칭 시험", kind, "엑셀에서 수정", date, clock}
	b.SetSheetRow("Sheet1", "A2", &row)
	buf, e := b.WriteToBuffer()
	if e != nil {
		t.Fatal(e)
	}
	return buf.Bytes()
}
func TestWorkbookMatchingNormalizesDateTimeAndStage(t *testing.T) {
	s := reportMatchState()
	cases := []struct {
		day, clock any
		kind       string
	}{{"2026.11.13", "9:30", "세팅 검사"}, {"2026/11/13", "09:30:17", "세팅"}, {"2026년 11월 13일", "오전 9:30:17", "SETTING"}, {"20261113", "9:30 AM", "셋팅검사"}}
	for _, c := range cases {
		t.Run(fmt.Sprint(c.day, c.clock), func(t *testing.T) {
			raw := matchBook(t, c.day, c.clock, c.kind, false)
			bound, check, e := analyzeExistingWorkbook(raw, s)
			if e != nil || len(bound) != 1 || check.ExcelRows != 1 || check.Matched != 1 || check.InvalidRows != 0 {
				t.Fatal(bound, check, e)
			}
			if bound[2].Baseline[5] != "09:30:17" {
				t.Fatal("server baseline precision lost")
			}
		})
	}
}
func TestWorkbookMatchingExcelSerialAnd1904Epoch(t *testing.T) {
	for _, epoch := range []struct {
		date1904 bool
		base     time.Time
	}{{false, time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)}, {true, time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)}} {
		date := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
		serial := date.Sub(epoch.base).Hours() / 24
		clock := float64(9*3600+30*60+17) / 86400
		full := matchBook(t, serial, serial+clock, "세팅검사", epoch.date1904)
		if bound, _, err := analyzeExistingWorkbook(full, reportMatchState()); err != nil || len(bound) != 1 {
			t.Fatal("full datetime serial in time cell did not match", err)
		}
		raw := matchBook(t, serial, clock, "세팅검사", epoch.date1904)
		b, e := openImportBook(raw)
		if e != nil {
			t.Fatal(e)
		}
		st, _ := b.NewStyle(&excelize.Style{CustomNumFmt: ptr("yyyy.mm.dd")})
		b.SetCellStyle("Sheet1", "E2", "E2", st)
		st, _ = b.NewStyle(&excelize.Style{CustomNumFmt: ptr("hh:mm")})
		b.SetCellStyle("Sheet1", "F2", "F2", st)
		buf, _ := b.WriteToBuffer()
		b.Close()
		bound, check, e := analyzeExistingWorkbook(buf.Bytes(), reportMatchState())
		if e != nil || len(bound) != 1 || check.InvalidRows != 0 {
			t.Fatal(epoch.date1904, serial, bound, check, e)
		}
	}
}
func ptr(s string) *string { return &s }
func TestWorkbookMinutePrecisionDoesNotGuessMultipleEvents(t *testing.T) {
	s := reportMatchState()
	next := *s.Events[0].Completed
	next = next.Add(20 * time.Second)
	e := s.Events[0]
	e.ID = 3
	e.Completed = &next
	s.Events = append(s.Events, e)
	if _, _, err := analyzeExistingWorkbook(matchBook(t, "2026-11-13", "09:30", "세팅검사", false), s); err == nil {
		t.Fatal("minute precision guessed multiple records")
	}
	bound, _, err := analyzeExistingWorkbook(matchBook(t, "2026-11-13", "09:30:17", "세팅검사", false), s)
	if err != nil || len(bound) != 1 {
		t.Fatal("exact seconds should still resolve one event", err)
	}
	bound, _, err = analyzeExistingWorkbook(matchBook(t, "2026-11-13", "09:30:18", "세팅검사", false), s)
	if err != nil || len(bound) != 0 {
		t.Fatal("explicit different second incorrectly matched", err)
	}
}
func TestWorkbookZeroMatchExplainsCountsAndDoesNotImportRows(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	data := matchBook(t, "2026-11-13", "09:30", "세팅검사", false)
	r := postWorkbook(a, "/report-settings", "workbook", data, cookie, map[string]string{"csrf": "admin-csrf", "action": "attach"})
	if r.Code != 303 {
		t.Fatal(r.Code, r.Body.String())
	}
	if len(a.state.Events) != 0 || len(a.state.Projects) != 1 {
		t.Fatal("attaching workbook imported business state")
	}
	r = call(a, "GET", "/report-settings", nil, cookie)
	body := r.Body.String()
	if r.Code != 200 || !strings.Contains(body, "완료된 검사 기록이 없어") || !strings.Contains(body, "프로그램 화면으로 가져오는 기능은 아닙니다") || !strings.Contains(body, "엑셀 검사 행") {
		t.Fatal("zero-match diagnostics absent", r.Code)
	}
	if !bytes.Equal(a.state.WorkbookBase, data) {
		t.Fatal("original base not preserved")
	}
}
func TestWorkbookRecheckAddsNormalizedLinksAndKeepsOldBaseline(t *testing.T) {
	a := fixture(t)
	s := reportMatchState()
	s.Board = a.state.Board
	s.LegacyHashes = a.state.LegacyHashes
	s.LegacySourceHash = a.state.LegacySourceHash
	s.LegacySnapshots = a.state.LegacySnapshots
	s.WorkbookBase = matchBook(t, "2026.11.13", "09:30", "세팅 검사", false)
	a.state = s
	cookie := adminCookie(a)
	if e := a.transaction(func(s *State) error { return nil }); e != nil {
		t.Fatal(e)
	}
	r := call(a, "POST", "/report-settings", url.Values{"csrf": {"admin-csrf"}, "action": {"recheck"}}, cookie)
	if r.Code != 303 || len(a.state.WorkbookBindings) != 1 {
		t.Fatal("recheck did not add links", r.Code, r.Body.String())
	}
	old := a.state.WorkbookBindings[2]
	a.transaction(func(s *State) error {
		s.Events[0].Worker = "프로그램 수정"
		s.Projects[0].Number = "A-RENAMED"
		return nil
	})
	r = call(a, "POST", "/report-settings", url.Values{"csrf": {"admin-csrf"}, "action": {"recheck"}}, cookie)
	if r.Code != 303 || a.state.WorkbookBindings[2].Baseline[3] != old.Baseline[3] {
		t.Fatal("recheck overwrote baseline")
	}
	merged, e := workbook(a.state)
	if e != nil {
		t.Fatal(e)
	}
	book, _ := openImportBook(merged)
	defer book.Close()
	if value, _ := book.GetCellValue("Sheet1", "D2"); value != "프로그램 수정" {
		t.Fatal("recheck lost pending update")
	}
	r = call(a, "GET", "/report-settings", nil, cookie)
	if strings.Contains(r.Body.String(), "일치하는 기록을 찾지 못했습니다") {
		t.Fatal("saved links incorrectly reported as failed after rename")
	}
	other := call(a, "POST", "/report-settings", url.Values{"action": {"recheck"}}, cookie)
	if other.Code != 403 {
		t.Fatal("recheck bypassed csrf")
	}
}
