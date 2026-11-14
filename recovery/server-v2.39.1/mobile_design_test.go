package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMobileDesignPreviewPagesAndFragments(t *testing.T) {
	a := fixture(t)
	a.state.Designers = []Designer{{ID: "preview-designer", Name: "김재환", Phone: "01012345678"}}
	a.state.ClientNotes = []ClientNote{{ID: "preview-client", Name: "APS이노베이션", Text: "센서 위치와 케이블 방향을 확인해 주세요.", Revision: 1}}
	p := &a.state.Projects[0]
	p.Name = "APS이노베이션 2축"
	p.Number = "A261020-004"
	p.ClientID = "preview-client"
	p.DesignerID = "preview-designer"
	p.SuppliedMaterials = "사급품 수령"
	p.SuppliedMaterialsColor = "#00B050"
	p.AdditionalInstructions = "체결 토크를 확인해 주세요."
	for _, name := range []string{"마리노 1대, 데스크탑", "메저테스트 15축 1종"} {
		q := Project{ID: a.state.next(), Name: name, Number: "A261021-00" + fmtID(int64(len(a.state.Projects))), Revision: 1, InfoRevision: 1, ShippingDueDate: day(time.Now()), DesignerID: "preview-designer"}
		q.rotate()
		a.state.Projects = append(a.state.Projects, q)
	}
	pending := a.state.Projects[1]
	if err := a.requestInspection(pending.ID, "setting", "탁광현", pending.SettingForm, ""); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	a.state.Projects[2].Setting.Completed = &now
	a.state.Projects[2].Setting.Worker = "김동홍"
	c := adminCookie(a)
	paths := map[string]string{"admin.html": "/", "work.html": "/work/" + a.state.Board + "?entry=skip", "queue.html": "/inspection-requests", "review.html": "/projects/" + fmtID(pending.ID) + "/inspection", "directory.html": "/client-directory/" + a.state.Board, "info.html": "/project-info/" + a.state.Board + "/1", "new.html": "/projects/new", "tts.html": "/tts-settings"}
	for name, path := range paths {
		w := call(a, "GET", path, nil, c)
		s := w.Body.String()
		if w.Code != 200 || strings.Count(s, `id="alpha-design"`) != 1 || !strings.Contains(s, "mobile-design-v1.4.21.js") {
			t.Fatal("missing preview entry", name, w.Code)
		}
		if strings.Contains(s, `data-alpha-design="modern"`) {
			t.Fatal("server forced preview")
		}
		if out := os.Getenv("DESIGN_HTML_OUT"); out != "" {
			if err := os.WriteFile(filepath.Join(out, name), w.Body.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, path := range []string{"/card/" + a.state.Board + "/1", "/orders/panel", "/inspection-requests/panel"} {
		s := call(a, "GET", path, nil, c).Body.String()
		if strings.Contains(s, "mobile-design-v1.4.21") || strings.Contains(s, `id="alpha-design"`) {
			t.Fatal("preview chrome injected into poll fragment", path)
		}
	}
	if out := os.Getenv("DESIGN_HTML_OUT"); out != "" {
		for name, path := range map[string]string{"orders-fragment.html": "/orders/panel", "requests-fragment.html": "/inspection-requests/panel", "card-1.html": "/card/" + a.state.Board + "/1", "status.json": "/status/" + a.state.Board} {
			w := call(a, "GET", path, nil, c)
			if err := os.WriteFile(filepath.Join(out, name), w.Body.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	login := call(a, "GET", "/", nil).Body.String()
	if strings.Contains(login, "alpha-design") || strings.Contains(login, a.state.Board) {
		t.Fatal("preview exposed board on login")
	}
}
