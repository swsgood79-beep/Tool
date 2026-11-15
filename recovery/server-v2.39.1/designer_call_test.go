package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesignerCallLinksAcrossAdminWorkerAndPolls(t *testing.T) {
	a := fixture(t)
	a.state.Designers = []Designer{{ID: "call-person", Name: "설계 <시험>", Phone: "01000000000", Revision: 1}}
	a.state.Projects[0].DesignerID = "call-person"
	a.state.Projects = append(a.state.Projects, Project{ID: a.state.next(), Number: "NO-PHONE", Name: "번호 없는 담당자", DesignerName: "번호 미등록", Revision: 1})
	c := adminCookie(a)
	for _, phase := range []string{"before", "after"} {
		phone := "01000000000"
		if phase == "after" {
			phone = "01011111111"
			a.state.Designers[0].Phone = phone
			a.state.Projects[0].touch()
		}
		for name, path := range map[string]string{"admin.html": "/", "work.html": "/work/" + a.state.Board + "?entry=skip", "orders-fragment.html": "/orders/panel", "card-1.html": "/card/" + a.state.Board + "/1", "status.json": "/status/" + a.state.Board} {
			w := call(a, "GET", path, nil, c)
			if w.Code != 200 {
				t.Fatal(path, w.Code)
			}
			s := w.Body.String()
			if name != "status.json" {
				if !strings.Contains(s, `href="tel:`+phone+`"`) || !strings.Contains(s, `설계 &lt;시험&gt;`) || strings.Contains(s, `href="tel:"`) {
					t.Fatal("missing/unsafe call link", name)
				}
			}
			if name == "admin.html" || name == "orders-fragment.html" {
				if !strings.Contains(s, `<span class="designer-label">☎ 번호 미등록</span>`) || !strings.Contains(s, `<a class="designer-call designer-label"`) {
					t.Fatal("admin should call only when a phone number is present")
				}
			}
			if out := os.Getenv("DESIGNER_CALL_HTML_OUT"); out != "" {
				if err := os.WriteFile(filepath.Join(out, phase+"-"+name), w.Body.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
