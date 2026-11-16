package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestApprovedOrderCardFixtureAcrossRolesAndPolling(t *testing.T) {
	a := fixture(t)
	if err := a.transaction(func(s *State) error {
		p := &s.Projects[0]
		p.Name, p.SpecialInstructions = "디자인 확인용 긴 프로젝트 이름 1축", "신청 전 치수 확인"
		p.DesignerName, p.DesignerPhone = "설계 시험", "01000000000"
		p.SuppliedMaterials, p.SuppliedMaterialsColor = "사급품 발송 예정", "#FFE0D2"
		base := *p
		for i, color := range []string{"#000000", "#00B050", "#4C97DB", "#FFB900", "", "#FFE0D2"} {
			q := base
			q.ID, q.Number = s.next(), "DESIGN-"+fmtID(int64(i+2))
			q.Name, q.SuppliedMaterialsColor = "수주 카드 시험 "+fmtID(q.ID), color
			if i == 0 || i == 1 {
				now := time.Now()
				q.Setting.Worker, q.Setting.Completed = "세팅 시험", &now
				if i == 1 {
					q.Shipping.Worker, q.Shipping.Completed = "출하 시험", &now
				}
			}
			if i == 2 {
				q.SpecialInstructions = ""
			}
			if i == 4 {
				q.DesignerName, q.DesignerPhone, q.SuppliedMaterials, q.SpecialInstructions = "", "", "", ""
			}
			if i == 5 {
				q.Hidden = true
			}
			q.rotate()
			s.Projects = append(s.Projects, q)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	c := adminCookie(a)
	for _, phase := range []string{"before", "after"} {
		if phase == "after" {
			p := *a.state.project(1)
			if err := a.requestInspection(1, "combined", "완료 시험", p.SettingForm, p.CombinedForm); err != nil {
				t.Fatal(err)
			}
			if err := a.approve(1, "combined", a.state.project(1).Setting.EventID); err != nil {
				t.Fatal(err)
			}
		}
		for name, route := range map[string]string{"work.html": "/work/" + a.state.Board + "?entry=skip", "admin.html": "/", "orders-fragment.html": "/orders/panel", "status.json": "/status/" + a.state.Board, "card-1.html": "/card/" + a.state.Board + "/1"} {
			w := call(a, "GET", route, nil, c)
			if w.Code != 200 {
				t.Fatal(route, w.Code)
			}
			if out := os.Getenv("APPROVED_CARD_HTML_OUT"); out != "" {
				if err := os.WriteFile(filepath.Join(out, phase+"-"+name), w.Body.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
