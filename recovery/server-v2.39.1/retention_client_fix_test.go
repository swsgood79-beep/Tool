package main

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestClientDeletePhotoIgnoresOldSourceCopies(t *testing.T) {
	for _, count := range []int{2, 3} {
		t.Run(fmtID(int64(count)), func(t *testing.T) {
			a := fixture(t)
			cookie := adminCookie(a)
			c := ClientNote{ID: token(), Name: "사진 계산 시험", Text: "원본 안내", Revision: 1}
			copies := []Photo{}
			for _, red := range []uint8{30, 130, 230} {
				c.Photos = append(c.Photos, duplicateTestPhoto(t, a, red, false))
				copies = append(copies, duplicateTestPhoto(t, a, red, true))
			}
			own := []Photo{}
			for i := 0; i < count; i++ {
				own = append(own, duplicateTestPhoto(t, a, uint8(60+i*40), false))
			}
			a.state.ClientNotes = []ClientNote{c}
			a.state.Projects[0].ClientID = c.ID
			a.state.Projects[0].Photos = append(copies, own...)
			w := call(a, "POST", "/client-notes", url.Values{"csrf": {"admin-csrf"}, "action": {"save"}, "id": {c.ID}, "revision": {"1"}, "client_name": {c.Name}, "client_text": {"삭제 후 안내"}, "remove_photo": {c.Photos[0].ID}}, cookie)
			if w.Code != 303 {
				t.Fatal("fewer actual photos incorrectly blocked", w.Code, w.Body.String())
			}
			if len(a.state.ClientNotes[0].Photos) != 2 || len(a.state.Projects[0].Photos) != count || len(a.projectView(a.state.Projects[0]).Photos) != count+2 {
				t.Fatal("own/source counts incorrect")
			}
			for _, ph := range copies {
				if _, e := os.Stat(filepath.Join(a.dir, "project-photos", ph.ID+".dat")); e != nil {
					t.Fatal("original file deleted", e)
				}
			}
			fresh, e := newApp(a.dir)
			if e != nil {
				t.Fatal(e)
			}
			if len(fresh.projectView(fresh.state.Projects[0]).Photos) != count+2 {
				t.Fatal("photo count changed after restart")
			}
		})
	}
}
func TestClientPhotoLimitErrorRendersEditForm(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	c := ClientNote{ID: token(), Name: "한도 시험", Text: "안내", Revision: 1}
	for _, red := range []uint8{10, 240} {
		c.Photos = append(c.Photos, duplicateTestPhoto(t, a, red, false))
	}
	a.state.ClientNotes = []ClientNote{c}
	a.state.Projects[0].ClientID = c.ID
	for _, red := range []uint8{50, 90, 130, 170} {
		a.state.Projects[0].Photos = append(a.state.Projects[0].Photos, duplicateTestPhoto(t, a, red, false))
	}
	before := string(jsonRaw(a.state))
	w := call(a, "POST", "/client-notes", url.Values{"csrf": {"admin-csrf"}, "action": {"save"}, "id": {c.ID}, "revision": {"1"}, "client_name": {c.Name}, "client_text": {"입력 유지"}}, cookie)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "업체 특이사항 수정") || !strings.Contains(w.Body.String(), "입력 유지") || !strings.Contains(w.Body.String(), "5장") {
		t.Fatal("form/error lost", w.Code)
	}
	if before != string(jsonRaw(a.state)) {
		t.Fatal("invalid update partially saved")
	}
}
func TestMissingSheetKeepsUnfinishedOrderAndRecovers(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	p := a.state.Projects[0]
	if e := a.requestInspection(p.ID, "setting", "진행 작업자", p.SettingForm, p.CombinedForm); e != nil {
		t.Fatal(e)
	}
	a.state.Projects[0].ShippingDueDate = "2026-11-24"
	a.state.Projects[0].ShippingDueManaged = true
	a.state.MaterialReceipts = map[string][]MaterialReceipt{p.Number: {{Kind: "모터", Quantity: "2"}}}
	events := string(jsonRaw(a.state.Events))
	stage := string(jsonRaw(a.state.Projects[0].Setting))
	file := filepath.Join(t.TempDir(), "schedule.xlsx")
	a.state.Integration.Schedule = ScheduleConfig{Enabled: true, Mode: "file", Path: file, Revision: 1}
	write := func(number, date string) {
		b := excelize.NewFile()
		defer b.Close()
		b.SetSheetName("Sheet1", "조립일정")
		b.SetCellValue("조립일정", "A1", "수주번호")
		b.SetCellValue("조립일정", "B1", "발송예정일")
		b.SetCellValue("조립일정", "A2", number)
		b.SetCellValue("조립일정", "B2", date)
		if e := b.SaveAs(file); e != nil {
			t.Fatal(e)
		}
	}
	write("OTHER", "2026-11-25")
	a.syncSchedule(context.Background(), true)
	got := a.state.Projects[0]
	if got.ShippingDueDate != "2026-11-24" || !got.ScheduleMissing || !got.ShippingDueManaged || got.Hidden || got.Shipping.Completed != nil || len(a.state.MaterialReceipts[p.Number]) != 1 {
		t.Fatal("unfinished order lost date/state/materials", got)
	}
	if events != string(jsonRaw(a.state.Events)) || stage != string(jsonRaw(got.Setting)) {
		t.Fatal("missing sheet changed inspection")
	}
	for _, path := range []string{"/", "/work/" + a.state.Board + "?entry=skip"} {
		w := call(a, "GET", path, nil, cookie)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "엑셀에서 제외됨") || !strings.Contains(w.Body.String(), "2026-11-24") {
			t.Fatal("retained order/warning missing", path, w.Code)
		}
	}
	fresh, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	if !fresh.state.Projects[0].ScheduleMissing || fresh.state.Projects[0].ShippingDueDate != "2026-11-24" {
		t.Fatal("restart lost retention")
	}
	write(p.Number, "2026-11-26")
	a.syncSchedule(context.Background(), true)
	got = a.state.Projects[0]
	if got.ScheduleMissing || got.ShippingDueDate != "2026-11-26" || !got.ShippingDueManaged {
		t.Fatal("returned order did not reconnect", got)
	}
}
