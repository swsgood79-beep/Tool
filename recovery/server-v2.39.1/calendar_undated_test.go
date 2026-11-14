package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

func TestScheduleMissingOrderRemovesCalendarDateAndExplicitUnknownIsLinked(t *testing.T) {
	a := fixture(t)
	p := &a.state.Projects[0]
	p.ShippingDueDate = "2026-12-21"
	p.ShippingDueManaged = true
	now := time.Now()
	p.Setting.Completed, p.Shipping.Completed = &now, &now
	file := filepath.Join(t.TempDir(), "schedule.xlsx")
	a.state.Integration.Schedule = ScheduleConfig{Mode: "file", Path: file, Enabled: true}
	write := func(order, value string) {
		t.Helper()
		book := excelize.NewFile()
		defer book.Close()
		book.SetSheetName("Sheet1", "조립일정")
		book.SetCellValue("조립일정", "A1", "수주번호")
		book.SetCellValue("조립일정", "B1", "발송예정일")
		book.SetCellValue("조립일정", "A2", order)
		book.SetCellValue("조립일정", "B2", value)
		if err := book.SaveAs(file); err != nil {
			t.Fatal(err)
		}
	}
	write("OTHER-001", "2026-12-22")
	a.syncSchedule(context.Background(), true)
	if text(a.integration.ScheduleStatus["error"]) != "" || a.state.Projects[0].ShippingDueDate != "" || a.state.Projects[0].ShippingDueManaged {
		t.Fatal("removed schedule order must clear its date and source presence")
	}
	revision := a.state.Projects[0].Revision
	write("T-001", "ASAP")
	a.syncSchedule(context.Background(), true)
	if a.state.Projects[0].ShippingDueDate != "" || !a.state.Projects[0].ShippingDueManaged || a.state.Projects[0].Revision <= revision {
		t.Fatal("explicit unknown was not applied")
	}
	write("T-001", "2026-12-23")
	a.syncSchedule(context.Background(), true)
	if a.state.Projects[0].ShippingDueDate != "2026-12-23" {
		t.Fatal("known schedule date did not return")
	}
}

func TestCalendarUndatedSourceMetadataAndFixtures(t *testing.T) {
	a := fixture(t)
	now := time.Now()
	for i := 0; i < 29; i++ {
		p := Project{ID: a.state.next(), Number: "EMPTY-" + fmtID(int64(i)), Name: "이전 자료 " + fmtID(int64(i)), Revision: 1, InfoRevision: 1}
		p.rotate()
		if i < 27 {
			p.Setting.Completed, p.Shipping.Completed = &now, &now
		} else {
			p.Name = "납기표 날짜 미정 " + fmtID(int64(i))
			p.ShippingDueManaged = true
		}
		a.state.Projects = append(a.state.Projects, p)
	}
	cookie := adminCookie(a)
	for name, route := range map[string]string{"work.html": "/work/" + a.state.Board + "?entry=skip", "admin.html": "/", "orders-fragment.html": "/orders/panel", "status.json": "/status/" + a.state.Board} {
		w := call(a, "GET", route, nil, cookie)
		if w.Code != 200 {
			t.Fatal(route, w.Code)
		}
		if name != "status.json" && (strings.Count(w.Body.String(), `data-due-managed="true"`) != 2 || strings.Count(w.Body.String(), `data-due-managed="false"`) != 28) {
			t.Fatal("source metadata not available to initial/poll calendar", name)
		}
		if out := os.Getenv("CALENDAR_UNDATED_HTML_OUT"); out != "" {
			if err := os.WriteFile(filepath.Join(out, name), w.Body.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
}
