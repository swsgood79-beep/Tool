package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Exercise actual initial pages and polled cards, including orders completed
// before their shipping date, legacy completion, hidden and removed schedules.
func TestWorkerTodayDueFixturesIncludeScheduledCompletions(t *testing.T) {
	a := fixture(t)
	now := time.Now()
	old := now.AddDate(0, 0, -3)
	due := day(now)
	next := day(now.AddDate(0, 0, 1))
	if err := a.transaction(func(s *State) error {
		p := &s.Projects[0]
		p.ShippingDueDate = due
		p.DesignerName, p.DesignerPhone = "설계 시험", "01000000000"
		p.SuppliedMaterials, p.SuppliedMaterialsColor = "사급품 수령", "#00B050"
		base := *p
		for i := 2; i <= 10; i++ {
			q := base
			q.ID = s.next()
			q.Number = "DUE-" + fmtID(q.ID)
			q.Name = "납기 시험 " + fmtID(q.ID)
			if i == 2 || i == 3 || i == 4 || i == 5 || i == 8 || i == 9 {
				q.Setting.Completed = &now
				q.Shipping.Completed = &now
				q.Setting.Worker = "세팅 시험"
				q.Shipping.Worker = "출하 시험"
			}
			if i == 3 {
				q.Setting.Completed = &old
				q.Shipping.Completed = &old
			}
			if i == 5 || i == 6 {
				q.ShippingDueDate = next
			}
			if i == 7 {
				q.Hidden = true
			}
			if i == 8 || i == 9 {
				q.ShippingDueDate = ""
				q.ShippingDueManaged = i == 8
			}
			q.rotate()
			s.Projects = append(s.Projects, q)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// The old completion timestamp must exist in the preserved original schema.
	if err := a.transaction(func(s *State) error {
		p := s.project(4)
		p.LegacyCompleted = true
		p.Setting.Completed, p.Shipping.Completed = nil, nil
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	c := adminCookie(a)
	for _, phase := range []string{"before", "approved", "removed"} {
		if phase == "approved" {
			p := *a.state.project(1)
			if err := a.requestInspection(1, "combined", "신청 시험", p.SettingForm, p.CombinedForm); err != nil {
				t.Fatal(err)
			}
			if err := a.approve(1, "combined", a.state.project(1).Setting.EventID); err != nil {
				t.Fatal(err)
			}
		}
		if phase == "removed" {
			if err := a.transaction(func(s *State) error {
				p := s.project(2)
				p.ShippingDueDate = ""
				p.ShippingDueManaged = false
				p.Revision++
				p = s.project(3)
				p.ShippingDueDate = next
				p.Revision++
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		}
		paths := map[string]string{"work.html": "/work/" + a.state.Board + "?entry=skip", "status.json": "/status/" + a.state.Board}
		for i := 1; i <= 10; i++ {
			if i != 7 {
				paths["card-"+fmtID(int64(i))+".html"] = "/card/" + a.state.Board + "/" + fmtID(int64(i))
			}
		}
		for name, route := range paths {
			w := call(a, "GET", route, nil, c)
			if w.Code != 200 {
				t.Fatal(route, w.Code)
			}
			if name == "work.html" {
				s := w.Body.String()
				if !strings.Contains(s, "worker-v1.4.35.js") || !strings.Contains(s, "worker-due-view-v1.4.35.js") || strings.Contains(s, `data-project="7"`) {
					t.Fatal("wrong worker assets or hidden order exposed")
				}
			}
			if name == "status.json" {
				var status struct {
					Active    []int64              `json:"active"`
					Completed []struct{ ID int64 } `json:"completed_due"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
					t.Fatal(err)
				}
				want := 3
				if phase != "before" {
					want = 2
				}
				if len(status.Active) != want {
					t.Fatal("active work filter changed", len(status.Active), want)
				}
			}
			if out := os.Getenv("WORKER_TODAY_DUE_HTML_OUT"); out != "" {
				if err := os.WriteFile(filepath.Join(out, phase+"-"+name), w.Body.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
