package main

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func archiveDeleteRequest(a *App, id string) int {
	cookie := adminCookie(a)
	return call(a, "POST", "/shipping-archive-admin/"+id+"/delete", url.Values{"csrf": {a.sessions[hashToken(cookie.Value)].CSRF}, "confirm": {"1"}}, cookie).Code
}
func TestArchiveDeletionRequeuesCompletedLiveOrder(t *testing.T) {
	a := fixture(t)
	r := approveShipment(t, a)
	if w := shippingUpload(t, a, r.ID, 2, false); w.Code != 303 {
		t.Fatal(w.Code)
	}
	before := jsonRaw(a.state.Projects)
	oldPhoto := a.state.ShippingRecords[0].Photos[0]
	snapshot := jsonRaw(a.state.ShippingRecords[0].Project)
	if code := archiveDeleteRequest(a, r.ID); code != 303 {
		t.Fatal(code)
	}
	if len(a.state.ShippingRecords) != 1 {
		t.Fatal("lost photo obligation")
	}
	pending := a.state.ShippingRecords[0]
	if pending.ArchivedAt != nil || len(pending.Photos) != 0 || pending.ID == r.ID || !bytes.Equal(before, jsonRaw(a.state.Projects)) || !bytes.Equal(snapshot, jsonRaw(pending.Project)) {
		t.Fatal("reset changed inspection or snapshot")
	}
	if _, err := os.Stat(filepath.Join(a.dir, "photos", oldPhoto.ID)); !os.IsNotExist(err) {
		t.Fatal("old shipment photo remains")
	}
	if w := call(a, "GET", "/shipping-records/"+a.state.Board+"?pending=1", nil); w.Code != 200 || !strings.Contains(w.Body.String(), pending.Project.Name) {
		t.Fatal("pending not visible")
	}
	if w := call(a, "GET", "/shipping-records/"+a.state.Board, nil); strings.Contains(w.Body.String(), `class="shipment-row"`) {
		t.Fatal("old archive still visible")
	}
	if code := archiveDeleteRequest(a, pending.ID); code != 400 || len(a.state.ShippingRecords) != 1 {
		t.Fatal("pending deleted although inspections remain complete")
	}
	if w := shippingUpload(t, a, r.ID, 1, false); w.Code != 404 {
		t.Fatal("stale pre-delete page can rearchive")
	}
	if w := shippingUpload(t, a, pending.ID, 1, false); w.Code != 303 || a.state.ShippingRecords[0].ArchivedAt == nil {
		t.Fatal("re-registration failed")
	}
	fresh, err := newApp(a.dir)
	if err != nil || len(fresh.state.ShippingRecords) != 1 || len(fresh.state.ShippingRecords[0].Photos) != 1 {
		t.Fatal("restart", err)
	}
}
func TestArchiveDeletionRetainsSpecialPhotosAndRollsBack(t *testing.T) {
	a := fixture(t)
	r := approveShipment(t, a)
	shippingUpload(t, a, r.ID, 1, false)
	var raw bytes.Buffer
	png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 8, 8)))
	id := token()
	protected, _ := protect(raw.Bytes())
	if err := atomicFile(filepath.Join(a.dir, "photos", id), protected); err != nil {
		t.Fatal(err)
	}
	if err := a.commitAnnouncement(func(s *State) error {
		s.ShippingRecords[0].Project.Photos = []Photo{{ID: id, MIME: "image/png", Size: raw.Len()}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	old := jsonRaw(a.state.ShippingRecords)
	a.directHook = func(phase, name string) error {
		if phase == "written" {
			return errors.New("simulated failure")
		}
		return nil
	}
	if code := archiveDeleteRequest(a, r.ID); code != 400 || !bytes.Equal(old, jsonRaw(a.state.ShippingRecords)) {
		t.Fatal("failed reset changed records")
	}
	a.directHook = nil
	if code := archiveDeleteRequest(a, r.ID); code != 303 {
		t.Fatal(code)
	}
	if _, err := os.Stat(filepath.Join(a.dir, "photos", id)); err != nil {
		t.Fatal("special photo was deleted", err)
	}
}
func TestArchiveDeletionWithoutCurrentCompletedCycleIsPermanent(t *testing.T) {
	for _, mode := range []string{"missing", "reopened", "old-cycle"} {
		t.Run(mode, func(t *testing.T) {
			a := fixture(t)
			r := approveShipment(t, a)
			shippingUpload(t, a, r.ID, 1, false)
			err := a.transaction(func(s *State) error {
				if mode == "missing" {
					s.Projects = nil
				} else if mode == "reopened" {
					s.Projects[0].Shipping = Stage{}
				} else {
					s.Projects[0].Shipping.EventID++
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if code := archiveDeleteRequest(a, r.ID); code != 303 || len(a.state.ShippingRecords) != 0 {
				t.Fatal("obsolete cycle requeued")
			}
		})
	}
}
func TestArchiveMissingRecordManualRecovery(t *testing.T) {
	a := fixture(t)
	r := approveShipment(t, a)
	if err := a.commitAnnouncement(func(s *State) error { s.ShippingRecords = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	before := jsonRaw(a.state.Projects)
	if !a.projectView(a.state.Projects[0]).ShippingRecordMissing {
		t.Fatal("missing completed cycle not repairable")
	}
	cookie := adminCookie(a)
	values := url.Values{"csrf": {a.sessions[hashToken(cookie.Value)].CSRF}, "project": {fmtID(r.Project.ID)}}
	w := call(a, "POST", "/shipping-archive-admin/recover", values, cookie)
	if w.Code != 303 || len(a.state.ShippingRecords) != 1 || a.state.ShippingRecords[0].ArchivedAt != nil || !bytes.Equal(before, jsonRaw(a.state.Projects)) {
		t.Fatal("repair", w.Code, w.Body.String())
	}
	if w = call(a, "POST", "/shipping-archive-admin/recover", values, cookie); w.Code != 303 || len(a.state.ShippingRecords) != 1 {
		t.Fatal("repair duplicated task")
	}
	if a.projectView(a.state.Projects[0]).ShippingRecordMissing {
		t.Fatal("repair control not removed")
	}
	if w = call(a, "POST", "/shipping-archive-admin/recover", values); w.Code == 303 {
		t.Fatal("worker repaired without admin session")
	}
	fresh, err := newApp(a.dir)
	if err != nil || len(fresh.state.ShippingRecords) != 1 {
		t.Fatal("repair persistence", err)
	}
}
