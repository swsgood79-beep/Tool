package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExplicitInspectionEntryPendingFixture(t *testing.T) {
	a := fixture(t)
	if err := a.transaction(func(s *State) error {
		p := &s.Projects[0]
		now := time.Now()
		p.Setting.Completed = &now
		p.Setting.Worker = "세팅 시험"
		p.Name = "출하검사 신청 대기 시험"
		p.DesignerName, p.DesignerPhone = "설계 시험", "01000000000"
		p.SpecialInstructions = "검사 전 특이사항 확인"
		q := *p
		q.ID = s.next()
		q.Number = "ENTRY-2"
		q.Name = "특이사항 없는 수주"
		q.SpecialInstructions = ""
		q.rotate()
		s.Projects = append(s.Projects, q)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	p := *a.state.project(1)
	if err := a.requestInspection(1, "shipping", "출하 시험", p.ShippingForm, ""); err != nil {
		t.Fatal(err)
	}
	c := adminCookie(a)
	for _, phase := range []string{"before", "changed"} {
		if phase == "changed" {
			if err := a.transaction(func(s *State) error {
				s.project(2).Name = "자동 갱신된 수주"
				s.project(2).Revision++
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		}
		routes := map[string]string{"work.html": "/work/" + a.state.Board + "?entry=skip", "status.json": "/status/" + a.state.Board, "card-1.html": "/card/" + a.state.Board + "/1", "card-2.html": "/card/" + a.state.Board + "/2", "admin.html": "/"}
		for name, path := range routes {
			w := call(a, "GET", path, nil, c)
			if w.Code != 200 {
				t.Fatal(path, w.Code)
			}
			if name == "work.html" {
				body := w.Body.String()
				if !strings.Contains(body, "02 출하검사 확인 대기") || !strings.Contains(body, "worker-dialog-v1.4.41.js") || !strings.Contains(body, "order-card-v1.4.105.js") {
					t.Fatal("pending state or entry assets missing")
				}
			}
			if out := os.Getenv("INSPECTION_ENTRY_HTML_OUT"); out != "" {
				os.WriteFile(filepath.Join(out, phase+"-"+name), w.Body.Bytes(), 0600)
			}
		}
	}
}
