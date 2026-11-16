package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCalendarInspectionReturnPagesAndApprovalFixture(t *testing.T) {
	a := fixture(t)
	if err := a.transaction(func(s *State) error {
		p := &s.Projects[0]
		p.Name, p.SpecialInstructions = "특이사항 있는 수주", "신청 전 치수를 확인하세요."
		now := time.Now().In(kst)
		p.ShippingDueDate = day(time.Date(now.Year(), now.Month()+1, 5, 0, 0, 0, 0, kst))
		base := *p
		for i := 0; i < 2; i++ {
			q := base
			q.ID = s.next()
			q.Number = "RETURN-" + fmtID(q.ID)
			q.Name, q.SpecialInstructions = "바로 신청 수주", ""
			if i == 1 {
				q.ShippingDueDate, q.ShippingDueManaged = "", true
			}
			q.rotate()
			s.Projects = append(s.Projects, q)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"before", "after"} {
		if phase == "after" {
			p := *a.state.project(1)
			if err := a.requestInspection(1, "combined", "검사 시험", p.SettingForm, p.CombinedForm); err != nil {
				t.Fatal(err)
			}
			if err := a.approve(1, "combined", a.state.project(1).Setting.EventID); err != nil {
				t.Fatal(err)
			}
		}
		for name, path := range map[string]string{"work.html": "/work/" + a.state.Board + "?entry=skip", "status.json": "/status/" + a.state.Board, "card-1.html": "/card/" + a.state.Board + "/1"} {
			w := call(a, "GET", path, nil)
			if w.Code != 200 {
				t.Fatal(path, w.Code)
			}
			if name == "work.html" && (!strings.Contains(w.Body.String(), "shipping-calendar-v1.4.33.js") || !strings.Contains(w.Body.String(), "mobile-design-v1.4.40.js")) {
				t.Fatal("calendar return scripts missing")
			}
			if out := os.Getenv("CALENDAR_RETURN_HTML_OUT"); out != "" {
				if err := os.WriteFile(filepath.Join(out, phase+"-"+name), w.Body.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
