package main

import (
	"bytes"
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/xuri/excelize/v2"
)

func splitSchedule(t *testing.T, ds ...string) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	f.SetSheetName("Sheet1", "조립일정")
	head := []any{"수주NO", "발송예정일"}
	f.SetSheetRow("조립일정", "A1", &head)
	for i, d := range ds {
		r := []any{"T-001", d}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		f.SetSheetRow("조립일정", cell, &r)
	}
	b, e := f.WriteToBuffer()
	if e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func applySplitSchedule(t *testing.T, a *App, ds ...string) {
	t.Helper()
	file := filepath.Join(a.dir, "split-test.xlsx")
	if e := os.WriteFile(file, splitSchedule(t, ds...), 0600); e != nil {
		t.Fatal(e)
	}
	a.state.Integration.Schedule = ScheduleConfig{Enabled: true, Mode: "file", Path: file, Revision: a.state.Integration.Schedule.Revision + 1}
	a.syncSchedule(context.Background(), true)
	if e := text(a.integration.ScheduleStatus["error"]); e != "" {
		t.Fatal(e)
	}
}
func splitFixture(t *testing.T) *App {
	a := fixture(t)
	applySplitSchedule(t, a, "2026-11-30", "2026-12-02", "2026-11-30")
	return a
}
func splitSetting(t *testing.T, a *App) {
	t.Helper()
	p := a.state.Projects[0]
	if e := a.requestInspection(p.ID, "setting", "세팅 작업자", p.SettingForm, ""); e != nil {
		t.Fatal(e)
	}
	p = a.state.Projects[0]
	if e := a.approve(p.ID, "setting", p.Setting.EventID, InspectionInspector{Name: "세팅 검사자"}); e != nil {
		t.Fatal(e)
	}
}
func finishSplitLot(t *testing.T, a *App, index int) ShippingRecord {
	t.Helper()
	p := a.state.Projects[0]
	l := p.DeliveryLots[index]
	if e := a.requestDelivery(p.ID, l.ID, "차수 작업자", l.Form); e != nil {
		t.Fatal(e)
	}
	p = a.state.Projects[0]
	l = p.DeliveryLots[index]
	before := time.Now()
	if e := a.approve(p.ID, "shipping", l.Inspection.EventID, InspectionInspector{Name: "차수 검사자", Username: "test-inspector"}); e != nil {
		t.Fatal(e)
	}
	p = a.state.Projects[0]
	l = p.DeliveryLots[index]
	r := deliveryRecord(&a.state, p, l)
	if r == nil || r.InspectedAt.Before(before) || r.Project.Shipping.Inspector.Name != "차수 검사자" {
		t.Fatal("lot snapshot/time missing", r)
	}
	return *r
}
func TestSplitScheduleGroupingMergedDuplicatesAndInvalid(t *testing.T) {
	ds, conf, e := parseScheduleDates(splitSchedule(t, "2026-12-02", "2026-11-30", "2026-11-30"))
	if e != nil || len(conf) > 0 || strings.Join(ds["T-001"], ",") != "2026-11-30,2026-12-02" {
		t.Fatal(ds, conf, e)
	}
	ds, conf, e = parseScheduleDates(splitSchedule(t, "2026-11-30", "invalid"))
	if e != nil || !conf["T-001"] {
		t.Fatal("invalid date silently accepted")
	}
	a := splitFixture(t)
	if len(a.state.Projects) != 1 || len(a.state.Projects[0].DeliveryLots) != 2 || a.state.Projects[0].ShippingDueDate != "2026-11-30" {
		t.Fatal("order duplicated or dates missing")
	}
	ids := []string{a.state.Projects[0].DeliveryLots[0].ID, a.state.Projects[0].DeliveryLots[1].ID}
	applySplitSchedule(t, a, "2026-12-02", "2026-11-30")
	if a.state.Projects[0].DeliveryLots[0].ID != ids[0] || a.state.Projects[0].DeliveryLots[1].ID != ids[1] {
		t.Fatal("refresh regenerated lot IDs")
	}
	cell, e := scheduleCellDates("2026-11-30\n2026-12-02")
	if e != nil || len(cell) != 2 {
		t.Fatal(cell, e)
	}
	if _, e := scheduleCellDates("2026-11-30 ~ 2026-12-02"); e == nil {
		t.Fatal("ambiguous range accepted")
	}
}
func TestSplitFullProcessOneOrderUntilBothPhotosAndExport(t *testing.T) {
	a := splitFixture(t)
	p := a.state.Projects[0]
	if e := a.requestDelivery(p.ID, p.DeliveryLots[0].ID, "홍길동", p.DeliveryLots[0].Form); e == nil {
		t.Fatal("shipping before setting")
	}
	if e := a.requestInspection(p.ID, "combined", "홍길동", p.SettingForm, p.CombinedForm); e == nil {
		t.Fatal("combined request bypassed split")
	}
	splitSetting(t, a)
	first := finishSplitLot(t, a, 0)
	if len(a.state.ShippingRecords) != 1 || a.state.Projects[0].Shipping.Completed != nil {
		t.Fatal("first approval completed entire order")
	}
	if w := shippingUpload(t, a, first.ID, 1, false); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	if a.state.Projects[0].Shipping.Completed != nil || splitArchiveDone(&a.state, first.DeliveryGroup) {
		t.Fatal("first photo archived entire order")
	}
	second := finishSplitLot(t, a, 1)
	if len(a.state.ShippingRecords) != 2 || a.state.Projects[0].Shipping.Completed != nil {
		t.Fatal("second approval completed before photos")
	}
	if w := shippingUpload(t, a, second.ID, 1, false); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	p = a.state.Projects[0]
	if p.Shipping.Completed == nil || !splitArchiveDone(&a.state, p.DeliveryGroup) || len(splitListRecords(&a.state, false)) != 1 {
		t.Fatal("final group not archived once")
	}
	if len(completedEvents(a.state)) != 3 {
		t.Fatal("setting must only be inspected once")
	}
	for _, r := range a.state.ShippingRecords {
		if r.Project.ShippingDueDate != p.DeliveryLots[r.DeliveryNumber-1].Date || len(r.Photos) != 1 {
			t.Fatal("lot evidence mixed")
		}
	}
	out := t.TempDir()
	records := append([]ShippingRecord{}, a.state.ShippingRecords...)
	_, books, _, e := buildShippingExport(context.Background(), out, records, time.Now(), func(ph Photo) ([]byte, error) { return shippingExportPhoto(a.dir, ph) }, nil, shippingExportLimits{PhotoBytes: 24 << 20, Records: 40, Single: true})
	if e != nil || books != 1 {
		t.Fatal(e)
	}
	f, e := excelize.OpenFile(filepath.Join(out, "export.xlsx"))
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	all := ""
	for _, sheet := range f.GetSheetList() {
		rows, _ := f.GetRows(sheet)
		all += string(jsonRaw(rows))
	}
	if !strings.Contains(all, "1차 출고") || !strings.Contains(all, "2차 출고") || !strings.Contains(all, "2026.12.02") {
		t.Fatal("export lacks lot dates/evidence", all[:min(1000, len(all))])
	}
	restored, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	if len(restored.state.Projects) != 1 || len(restored.state.Projects[0].DeliveryLots) != 2 || restored.state.Projects[0].Shipping.Completed == nil || len(restored.state.ShippingRecords) != 2 {
		t.Fatal("restart lost lots")
	}
}
func TestSplitLateSecondDatePreservesSingleInspectionAndPhotos(t *testing.T) {
	a := fixture(t)
	a.state.Projects[0].ShippingDueDate = "2026-11-30"
	first := approveShipment(t, a)
	if w := shippingUpload(t, a, first.ID, 1, false); w.Code != 303 {
		t.Fatal(w.Code)
	}
	oldEvent := a.state.Projects[0].Shipping.EventID
	oldSetting := a.state.Projects[0].Setting.EventID
	applySplitSchedule(t, a, "2026-11-30", "2026-12-02")
	p := a.state.Projects[0]
	if len(p.DeliveryLots) != 2 || p.DeliveryLots[0].Inspection.EventID != oldEvent || p.Setting.EventID != oldSetting || p.Shipping.Completed != nil {
		t.Fatal("conversion reset or duplicated history")
	}
	rec := deliveryRecord(&a.state, p, p.DeliveryLots[0])
	if rec == nil || rec.ID != first.ID || len(rec.Photos) != 1 || rec.ArchivedAt == nil {
		t.Fatal("conversion lost first photo")
	}
	applySplitSchedule(t, a, "2026-11-30")
	p = a.state.Projects[0]
	if len(p.DeliveryLots) != 2 || !p.DeliveryLots[1].Missing || p.Shipping.Completed != nil {
		t.Fatal("sheet removal auto-completed the order")
	}
}
func TestSplitRejectStaleRequestAndPermissions(t *testing.T) {
	a := splitFixture(t)
	splitSetting(t, a)
	p := a.state.Projects[0]
	l := p.DeliveryLots[0]
	path := "/delivery-request/" + a.state.Board + "/" + fmtID(p.ID) + "/" + l.ID
	vals := url.Values{"csrf": {a.state.Board}, "worker": {"작업자"}, "submission": {l.Form}}
	bad := cloneValues(vals)
	bad.Set("csrf", "forged")
	if w := call(a, "POST", path, bad); w.Code != 403 {
		t.Fatal("csrf")
	}
	if w := call(a, "POST", path, vals); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := call(a, "POST", path, vals); w.Code != 409 {
		t.Fatal("duplicate request")
	}
	request := a.state.Projects[0].DeliveryLots[0].Inspection.EventID
	if e := a.rejectInspection(p.ID, "shipping", request); e != nil {
		t.Fatal(e)
	}
	if e := a.approve(p.ID, "shipping", request); e == nil {
		t.Fatal("stale approval after rejection")
	}
	l = a.state.Projects[0].DeliveryLots[0]
	if l.Inspection.EventID != 0 || equal(l.Form, vals.Get("submission")) || a.state.Projects[0].Setting.Completed == nil {
		t.Fatal("rejection damaged common setting")
	}
	if e := a.requestDelivery(p.ID, l.ID, "재신청", l.Form); e != nil {
		t.Fatal(e)
	}
	owner := adminCookie(a)
	w := call(a, "GET", "/projects/"+fmtID(p.ID)+"/inspection", nil, owner)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "1차 검사 완료") || !strings.Contains(w.Body.String(), "2차 출고") {
		t.Fatal("review lacks lots", w.Code, w.Body.String())
	}
}
func TestSplitBackupPhotoResetAndPersistFailure(t *testing.T) {
	a := splitFixture(t)
	splitSetting(t, a)
	first := finishSplitLot(t, a, 0)
	shippingUpload(t, a, first.ID, 1, false)
	second := finishSplitLot(t, a, 1)
	shippingUpload(t, a, second.ID, 1, false)
	reset, e := a.deleteShippingArchive(first.ID)
	if e != nil || !reset {
		t.Fatal(e)
	}
	p := a.state.Projects[0]
	r := deliveryRecord(&a.state, p, p.DeliveryLots[0])
	if p.Shipping.Completed != nil || r == nil || r.ArchivedAt != nil || r.ID == first.ID || p.DeliveryLots[1].Inspection.Completed == nil {
		t.Fatal("reset affected wrong lot")
	}
	identity, _ := age.GenerateX25519Identity()
	a.state.Integration.Backup.Key = RecoveryKey{1, repository, token(), identity.String()}
	data, e := a.portable(a.state)
	if e != nil {
		t.Fatal(e)
	}
	fresh, e := newApp(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if e = fresh.restorePortable(a.state.Integration.Backup.Key, data); e != nil {
		t.Fatal(e)
	}
	if len(fresh.state.Projects[0].DeliveryLots) != 2 || len(fresh.state.ShippingRecords) != 2 {
		t.Fatal("backup lost slots")
	}
	l := a.state.Projects[0].DeliveryLots[0]
	before := jsonRaw(a.state.Projects)
	a.state.LegacySourceHash = "stale"
	if e := a.requestDelivery(p.ID, l.ID, "잘못된 요청", l.Form); e == nil {
		t.Fatal("completed lot accepted")
	}
	if !bytes.Equal(before, jsonRaw(a.state.Projects)) {
		t.Fatal("failed request changed state")
	}
}

func TestSplitRefreshDuringPendingAndParallelRequests(t *testing.T) {
	for _, combined := range []bool{false, true} {
		a := fixture(t)
		a.state.Projects[0].ShippingDueDate = "2026-11-30"
		if !combined {
			p := a.state.Projects[0]
			if e := a.requestInspection(p.ID, "setting", "세팅", p.SettingForm, ""); e != nil {
				t.Fatal(e)
			}
			if e := a.approve(p.ID, "setting", a.state.Projects[0].Setting.EventID); e != nil {
				t.Fatal(e)
			}
		}
		p := a.state.Projects[0]
		kind, form := "shipping", p.ShippingForm
		if combined {
			kind, form = "combined", p.SettingForm
		}
		if e := a.requestInspection(p.ID, kind, "진행 중 작업자", form, p.CombinedForm); e != nil {
			t.Fatal(e)
		}
		old := a.state.Projects[0].Shipping.EventID
		applySplitSchedule(t, a, "2026-11-30", "2026-12-02")
		p = a.state.Projects[0]
		if p.DeliveryLots[0].Inspection.EventID != old || !p.DeliveryLots[0].Inspection.Pending || p.CombinedPending {
			t.Fatal("pending conversion changed request")
		}
		if combined {
			if e := a.approve(p.ID, "setting", p.Setting.EventID); e != nil {
				t.Fatal(e)
			}
		}
		p = a.state.Projects[0]
		l := p.DeliveryLots[1]
		if e := a.requestDelivery(p.ID, l.ID, "다른 차수", l.Form); e != nil {
			t.Fatal(e)
		}
		if e := a.approve(p.ID, "shipping", old); e != nil {
			t.Fatal(e)
		}
		p = a.state.Projects[0]
		if !p.DeliveryLots[1].Inspection.Pending || p.Shipping.Completed != nil {
			t.Fatal("first approval consumed second request")
		}
		if e := a.approve(p.ID, "shipping", old); e == nil {
			t.Fatal("duplicate approval accepted")
		}
	}
}
func TestSplitMergedScheduleAndThirdDate(t *testing.T) {
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "조립일정")
	head := []any{"수주NO", "발송예정일"}
	f.SetSheetRow("조립일정", "A1", &head)
	for _, row := range []int{2, 4} {
		r := []any{"T-001", "2026-11-30"}
		if row == 4 {
			r[1] = "2026-12-02"
		}
		cell, _ := excelize.CoordinatesToCellName(1, row)
		f.SetSheetRow("조립일정", cell, &r)
		from, _ := excelize.CoordinatesToCellName(1, row)
		to, _ := excelize.CoordinatesToCellName(1, row+1)
		f.MergeCell("조립일정", from, to)
		from, _ = excelize.CoordinatesToCellName(2, row)
		to, _ = excelize.CoordinatesToCellName(2, row+1)
		f.MergeCell("조립일정", from, to)
	}
	b, _ := f.WriteToBuffer()
	f.Close()
	ds, conf, e := parseScheduleDates(b.Bytes())
	if e != nil || len(conf) != 0 || len(ds["T-001"]) != 2 {
		t.Fatal(ds, conf, e)
	}
	a := splitFixture(t)
	splitSetting(t, a)
	first := finishSplitLot(t, a, 0)
	shippingUpload(t, a, first.ID, 1, false)
	original := a.state.Projects[0].DeliveryLots[0]
	applySplitSchedule(t, a, "2026-11-30", "2026-12-02", "2026-12-04", "2026-12-04")
	p := a.state.Projects[0]
	if len(p.DeliveryLots) != 3 || p.DeliveryLots[0].ID != original.ID || p.DeliveryLots[0].Inspection.EventID != original.Inspection.EventID || p.DeliveryLots[2].Number != 3 {
		t.Fatal("third date duplicated or reset work")
	}
}

func TestSplitRescheduleInvalidatesOnlyChangedForm(t *testing.T) {
	a := splitFixture(t)
	splitSetting(t, a)
	before := a.state.Projects[0].DeliveryLots
	applySplitSchedule(t, a, "2026-11-30", "2026-12-03")
	p := a.state.Projects[0]
	if p.DeliveryLots[0].ID != before[0].ID || p.DeliveryLots[0].Form != before[0].Form {
		t.Fatal("unchanged lot form was invalidated")
	}
	if p.DeliveryLots[1].ID != before[1].ID || p.DeliveryLots[1].Form == before[1].Form {
		t.Fatal("reschedule did not keep ID and rotate form")
	}
	if e := a.requestDelivery(p.ID, before[1].ID, "이전 화면", before[1].Form); e == nil {
		t.Fatal("stale dated request accepted")
	}
}
