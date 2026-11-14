package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Complete a real inspection ahead of its due date, then exercise initial pages,
// polling and restart. Calendar records must survive independently of active work.
func TestCalendarKeepsCompletedOrdersUntilTheirScheduledDate(t *testing.T) {
	a := fixture(t)
	due := day(time.Now().AddDate(0, 0, 7))
	if err := a.transaction(func(s *State) error {
		s.Projects[0].ShippingDueDate = due
		for _, name := range []string{"미리 완료한 수주", "이전 완료 수주", "숨긴 수주"} {
			p := Project{ID: s.next(), Revision: 1, InfoRevision: 1, Number: "CAL-" + fmtID(s.NextID), Name: name, ShippingDueDate: due, DesignerName: "설계 시험", DesignerPhone: "01000000000"}
			p.rotate()
			if name == "이전 완료 수주" {
				old := time.Now().AddDate(0, 0, -3)
				p.Setting.Completed = &old
				p.Shipping.Completed = &old
				p.Setting.Worker = "이전 세팅"
				p.Shipping.Worker = "이전 출하"
			}
			if name == "숨긴 수주" {
				p.Hidden = true
			}
			s.Projects = append(s.Projects, p)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	finish := func(id int64) {
		t.Helper()
		p := *a.state.project(id)
		if err := a.requestInspection(id, "setting", "세팅 시험", p.SettingForm, ""); err != nil {
			t.Fatal(err)
		}
		if err := a.approve(id, "setting", a.state.project(id).Setting.EventID); err != nil {
			t.Fatal(err)
		}
		p = *a.state.project(id)
		if err := a.requestInspection(id, "shipping", "출하 시험", p.ShippingForm, ""); err != nil {
			t.Fatal(err)
		}
		if err := a.approve(id, "shipping", a.state.project(id).Shipping.EventID); err != nil {
			t.Fatal(err)
		}
	}
	finish(a.state.Projects[1].ID)
	cookie := adminCookie(a)
	check := func(phase string, wantActive int) {
		t.Helper()
		paths := map[string]string{"work.html": "/work/" + a.state.Board + "?entry=skip", "admin.html": "/", "orders-fragment.html": "/orders/panel", "status.json": "/status/" + a.state.Board, "card-1.html": "/card/" + a.state.Board + "/1"}
		for name, path := range paths {
			w := call(a, "GET", path, nil, cookie)
			if w.Code != 200 {
				t.Fatalf("%s: %d", path, w.Code)
			}
			s := w.Body.String()
			if name == "work.html" {
				if !strings.Contains(s, "shipping-calendar-v1.4.28.js") || !strings.Contains(s, "calendar-complete-v1.4.27.css") || !strings.Contains(s, `data-name="미리 완료한 수주" data-due="`+due+`"`) || strings.Contains(s, "숨긴 수주") {
					t.Fatal("missing scheduled completed record, calendar assets, or hidden record exposed")
				}
			}
			if name == "status.json" {
				var v struct {
					Active    []int64              `json:"active"`
					Completed []struct{ ID int64 } `json:"completed_due"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil || len(v.Active) != wantActive || len(v.Completed) != 3-wantActive {
					t.Fatalf("poll separated active and completed records incorrectly: %s", s)
				}
			}
			if out := os.Getenv("CALENDAR_COMPLETED_HTML_OUT"); out != "" {
				if err := os.WriteFile(filepath.Join(out, phase+"-"+name), w.Body.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	check("before", 1)
	finish(1)
	check("after", 0)
	restored, err := newApp(a.dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range restored.state.Projects[:3] {
		if !restored.projectView(p).Completed || p.ShippingDueDate != due {
			t.Fatal("restart lost completed schedule")
		}
	}
}
