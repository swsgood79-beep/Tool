package main

import (
	"bytes"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func pendingAfterArchiveDeletion(t *testing.T) *App {
	t.Helper()
	a := fixture(t)
	r := approveShipment(t, a)
	if w := shippingUpload(t, a, r.ID, 1, false); w.Code != 303 {
		t.Fatal(w.Code)
	}
	if code := archiveDeleteRequest(a, r.ID); code != 303 || len(a.state.ShippingRecords) != 1 || a.state.ShippingRecords[0].ArchivedAt != nil {
		t.Fatal("reset fixture")
	}
	return a
}
func TestInspectionReopenRemovesPhotoPending(t *testing.T) {
	a := pendingAfterArchiveDeletion(t)
	p := a.state.Projects[0]
	cookie := adminCookie(a)
	w := call(a, "POST", "/archive", url.Values{"csrf": {a.sessions[hashToken(cookie.Value)].CSRF}, "id": {fmtID(p.ID)}}, cookie)
	if w.Code != 303 || len(a.state.ShippingRecords) != 0 || a.state.Projects[0].Shipping.Completed != nil || a.state.Projects[0].Setting.Completed != nil {
		t.Fatal("reopen left photo pending", w.Code, w.Body.String())
	}
	fresh, err := newApp(a.dir)
	if err != nil || len(fresh.state.ShippingRecords) != 0 {
		t.Fatal("reopen not durable", err)
	}
}
func TestInspectionRecordCancelRemovesPhotoPending(t *testing.T) {
	for _, kind := range []string{"setting", "shipping"} {
		t.Run(kind, func(t *testing.T) {
			a := pendingAfterArchiveDeletion(t)
			p := a.state.Projects[0]
			id := p.Shipping.EventID
			if kind == "setting" {
				id = p.Setting.EventID
			}
			var revision int64
			for _, e := range a.state.Events {
				if e.ID == id {
					revision = e.Revision
				}
			}
			r := httptest.NewRequest("POST", "http://127.0.0.1/records/"+fmtID(id), strings.NewReader(url.Values{"action": {"cancel"}, "confirm": {"1"}, "revision": {fmtID(revision)}}.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if err := a.editRecord(r, id); err != nil {
				t.Fatal(err)
			}
			if len(a.state.ShippingRecords) != 0 {
				t.Fatal("cancel left photo pending")
			}
			if kind == "shipping" && a.state.Projects[0].Setting.Completed == nil {
				t.Fatal("shipping-only cancel removed setting completion")
			}
		})
	}
}
func TestCancelledPhotoPendingAutoRepairOnStartup(t *testing.T) {
	a := pendingAfterArchiveDeletion(t)
	a.state.Projects[0].Setting = Stage{}
	a.state.Projects[0].Shipping = Stage{}
	// Simulate an older release that persisted cancellation without clearing the task.
	if err := a.persist(&a.state); err != nil {
		t.Fatal(err)
	}
	fresh, err := newApp(a.dir)
	if err != nil || len(fresh.state.ShippingRecords) != 0 {
		t.Fatal("stale pending not repaired", err)
	}
}
func TestCancellationCleanupKeepsMissingSourceAndArchivedHistory(t *testing.T) {
	a := pendingAfterArchiveDeletion(t)
	if err := a.transaction(func(s *State) error { s.Projects = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	if len(a.state.ShippingRecords) != 1 {
		t.Fatal("sheet disappearance removed independent pending task")
	}
	fresh, err := newApp(a.dir)
	if err != nil || len(fresh.state.ShippingRecords) != 1 {
		t.Fatal("missing-source pending lost at restart", err)
	}
	b := fixture(t)
	r := approveShipment(t, b)
	shippingUpload(t, b, r.ID, 1, false)
	before := jsonRaw(b.state.ShippingRecords)
	if err := b.transaction(func(s *State) error { s.Projects[0].Setting = Stage{}; s.Projects[0].Shipping = Stage{}; return nil }); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, jsonRaw(b.state.ShippingRecords)) {
		t.Fatal("inspection cancel silently deleted archived history")
	}
}
func TestCancellationCleanupRollbackAndNewCycle(t *testing.T) {
	a := pendingAfterArchiveDeletion(t)
	before := jsonRaw(a.state)
	a.directHook = func(phase, name string) error {
		if phase == "written" {
			return errors.New("storage failure")
		}
		return nil
	}
	if err := a.transaction(func(s *State) error { s.Projects[0].Shipping = Stage{}; return nil }); err == nil || !bytes.Equal(before, jsonRaw(a.state)) {
		t.Fatal("failed cancel changed pending task")
	}
	a.directHook = nil
	if err := a.transaction(func(s *State) error {
		s.Projects[0].Setting = Stage{}
		s.Projects[0].Shipping = Stage{}
		s.Projects[0].rotate()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	p := a.state.Projects[0]
	if err := a.requestInspection(p.ID, "combined", "새 작업자", p.SettingForm, p.CombinedForm); err != nil {
		t.Fatal(err)
	}
	if len(a.state.ShippingRecords) != 0 {
		t.Fatal("requested cycle already needs photos")
	}
	p = a.state.Projects[0]
	if err := a.approve(p.ID, "combined", p.Setting.EventID); err != nil {
		t.Fatal(err)
	}
	if len(a.state.ShippingRecords) != 1 || a.state.ShippingRecords[0].Project.Shipping.EventID != a.state.Projects[0].Shipping.EventID {
		t.Fatal("new cycle not captured exactly once")
	}
}
