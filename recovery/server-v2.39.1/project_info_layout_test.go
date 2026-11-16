package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectInfoLayoutAcrossRolesAndRefresh(t *testing.T) {
	a := fixture(t)
	p := &a.state.Projects[0]
	p.Name = "특이사항 배치 시험 수주"
	p.DesignerName, p.DesignerPhone = "설계 시험", "01000000000"
	p.SuppliedMaterials, p.SuppliedMaterialsColor = "사급품 11/16 발송 예정", "#FFE0D2"
	p.SpecialInstructions = "볼스크류 적용\n블럭 표면처리 확인"
	p.ShippingDueDate = "2026-11-17"
	for _, variant := range []string{"no-due", "no-phone", "no-material"} {
		q := *p
		q.ID = a.state.next()
		q.Number = variant
		switch variant {
		case "no-due":
			q.ShippingDueDate = ""
		case "no-phone":
			q.DesignerPhone = ""
		case "no-material":
			q.SuppliedMaterials = ""
		}
		a.state.Projects = append(a.state.Projects, q)
	}
	c := adminCookie(a)
	for _, phase := range []string{"before", "after"} {
		if phase == "after" {
			a.state.Projects[0].DesignerPhone = "01011111111"
			a.state.Projects[0].SuppliedMaterials = "사급품 수령"
			a.state.Projects[0].InfoRevision++
		}
		for _, role := range []string{"admin", "worker"} {
			for _, p := range a.state.Projects {
				path := "/projects/" + fmtID(p.ID) + "/info"
				if role == "worker" {
					path = "/project-info/" + a.state.Board + "/" + fmtID(p.ID)
				}
				w := call(a, "GET", path, nil, c)
				body := w.Body.String()
				if w.Code != 200 || strings.Count(body, `class="special-info-meta"`) != 1 || !strings.Contains(body, "project-info-layout-v1.4.30.css") {
					t.Fatal("metadata/layout missing", role, p.ID, w.Code)
				}
				if strings.Contains(body, "☎ 설계담당자") || strings.Contains(body, " · "+p.DesignerPhone+"</a>") || strings.Contains(body, "#ZgotmplZ") {
					t.Fatal("old label or invalid call link", role)
				}
				if out := os.Getenv("PROJECT_INFO_LAYOUT_HTML_OUT"); out != "" {
					if err := os.WriteFile(filepath.Join(out, phase+"-"+role+"-"+fmtID(p.ID)+".html"), w.Body.Bytes(), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}
}
