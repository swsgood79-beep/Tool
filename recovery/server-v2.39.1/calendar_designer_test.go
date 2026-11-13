package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCalendarDesignerMetadataAcrossRolesAndPolling(t *testing.T) {
	a := fixture(t)
	a.state.Designers = []Designer{{ID: "calendar-person", Name: `시험 담당자 <이름>`, Phone: "01000000000"}}
	a.state.Projects[0].DesignerID = "calendar-person"
	a.state.Projects[0].SpecialInstructions = "시험 특이사항"
	complete := a.state.Projects[0]
	complete.ID = 2
	complete.Number = "T-002"
	now := time.Now()
	complete.Shipping.Completed = &now
	complete.Setting.Worker = "세팅 시험"
	complete.Shipping.Worker = "출하 시험"
	a.state.Projects = append(a.state.Projects, complete, Project{ID: 3, Number: "T-003", Name: "담당자 미지정", ShippingDueDate: day(now), Revision: 1})
	check := func(name string, v any, filename string) {
		t.Helper()
		w := httptest.NewRecorder()
		a.render(w, name, v)
		s := w.Body.String()
		if w.Code != 200 || !strings.Contains(s, `data-designer-name="시험 담당자 &lt;이름&gt;"`) || !strings.Contains(s, `data-designer-call="tel:01000000000"`) {
			t.Fatalf("missing escaped metadata in %s", name)
		}
		if filename != "" {
			if d := os.Getenv("CALENDAR_HTML_OUT"); d != "" {
				if e := os.WriteFile(filepath.Join(d, filename), w.Body.Bytes(), 0600); e != nil {
					t.Fatal(e)
				}
			}
		}
	}
	v := a.view(a.state.Board)
	check("admin.html", v, "admin.html")
	check("work.html", v, "work.html")
	check("admin-order-items", v, "")
	check("project-card", a.projectView(complete), "")
	a.state.Designers[0].Name = "변경 담당자"
	w := httptest.NewRecorder()
	a.render(w, "project-card", a.projectView(complete))
	if !strings.Contains(w.Body.String(), `data-designer-name="변경 담당자"`) {
		t.Fatal("directory edit not reflected")
	}
}
