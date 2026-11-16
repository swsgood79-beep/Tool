package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSharedOrderCardPagesAndApprovalPolling(t *testing.T) {
	a := fixture(t)
	if err := a.transaction(func(s *State) error {
		p := &s.Projects[0]
		p.Name = "수주 카드 디자인 시험용 긴 프로젝트 이름 1축"
		p.DesignerName, p.DesignerPhone = "설계 시험", "01000000000"
		p.SuppliedMaterials, p.SuppliedMaterialsColor = "사급품 발송 예정", "#FFE0D2"
		p.SpecialInstructions = "치수와 케이블을 확인하세요."
		base := *p
		for i := 0; i < 2; i++ {
			q := base
			q.ID = s.next()
			q.Number = "CARD-" + fmtID(q.ID)
			q.Name = "세팅 완료 수주"
			now := time.Now()
			q.Setting.Worker, q.Setting.Completed = "세팅 시험", &now
			if i == 1 {
				q.Name = "검사 완료 수주"
				q.Shipping.Worker, q.Shipping.Completed = "출하 시험", &now
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
			if name == "work.html" || name == "admin.html" {
				for _, asset := range []string{"order-card-v1.4.35.css", "order-card-v1.4.35.js", "shipping-calendar-v1.4.33.js"} {
					if !strings.Contains(w.Body.String(), asset) {
						t.Fatal("shared card asset missing", name, asset)
					}
				}
			}
			if out := os.Getenv("SHARED_CARD_HTML_OUT"); out != "" {
				if err := os.WriteFile(filepath.Join(out, phase+"-"+name), w.Body.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
