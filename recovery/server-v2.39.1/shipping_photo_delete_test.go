package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func deleteOneRequest(a *App, id, photo, revision string, cookie *http.Cookie) *httptest.ResponseRecorder {
	return call(a, "POST", "/shipping-archive-admin/"+id+"/delete-photo", url.Values{"csrf": {a.sessions[hashToken(cookie.Value)].CSRF}, "photo": {photo}, "photo_revision": {revision}, "confirm": {"1"}}, cookie)
}
func TestShippingPhotoDeletePermissionsPersistenceAndLastPhoto(t *testing.T) {
	a := fixture(t)
	r := approveShipment(t, a)
	shippingUpload(t, a, r.ID, 3, false)
	cookie := adminCookie(a)
	before, _ := clone(a.state)
	record := shippingRecord(&a.state, r.ID)
	id := record.Photos[1].ID
	rev := shippingPhotoRevision(record)
	for _, req := range []struct {
		path   string
		values url.Values
		cookie *http.Cookie
		want   int
	}{
		{"/shipping-archive-admin/" + r.ID + "/delete-photo", url.Values{"csrf": {"wrong"}}, cookie, 403},
		{"/shipping-archive-admin/" + r.ID + "/delete-photo", url.Values{"csrf": {a.sessions[hashToken(cookie.Value)].CSRF}}, cookie, 400},
		{"/shipping-records/" + a.state.Board + "/" + r.ID + "/delete-photo", url.Values{"csrf": {a.state.Board}, "confirm": {"1"}, "photo": {id}, "photo_revision": {rev}}, nil, 404},
	} {
		var cookies []*http.Cookie
		if req.cookie != nil {
			cookies = append(cookies, req.cookie)
		}
		w := call(a, "POST", req.path, req.values, cookies...)
		if w.Code != req.want {
			t.Fatal(w.Code, req.want)
		}
	}
	if w := deleteOneRequest(a, r.ID, token(), rev, cookie); w.Code != 400 {
		t.Fatal("foreign photo")
	}
	a.directHook = func(phase, name string) error {
		if phase == "written" {
			return errors.New("injected write failure")
		}
		return nil
	}
	if w := deleteOneRequest(a, r.ID, id, rev, cookie); w.Code != 400 {
		t.Fatal("commit failure")
	}
	if len(shippingRecord(&a.state, r.ID).Photos) != 3 {
		t.Fatal("rollback lost photo")
	}
	if _, err := os.Stat(filepath.Join(a.dir, "photos", id)); err != nil {
		t.Fatal("rollback removed file", err)
	}
	a.directHook = nil
	if w := deleteOneRequest(a, r.ID, id, rev, cookie); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	record = shippingRecord(&a.state, r.ID)
	if len(record.Photos) != 2 || record.Photos[0].ID != before.ShippingRecords[0].Photos[0].ID || record.Photos[1].ID != before.ShippingRecords[0].Photos[2].ID || !record.ArchivedAt.Equal(*before.ShippingRecords[0].ArchivedAt) || !bytes.Equal(jsonRaw(record.Project), jsonRaw(before.ShippingRecords[0].Project)) || !bytes.Equal(jsonRaw(a.state.Projects), jsonRaw(before.Projects)) {
		t.Fatal("partial delete changed other information")
	}
	if _, err := os.Stat(filepath.Join(a.dir, "photos", id)); !os.IsNotExist(err) {
		t.Fatal("file not cleaned", err)
	}
	if w := deleteOneRequest(a, r.ID, record.Photos[0].ID, rev, cookie); w.Code != 400 {
		t.Fatal("stale revision accepted")
	}
	fresh, err := newApp(a.dir)
	if err != nil || len(shippingRecord(&fresh.state, r.ID).Photos) != 2 {
		t.Fatal("restart persistence", err)
	}
	for len(a.state.ShippingRecords[0].Photos) > 0 {
		current := a.state.ShippingRecords[0]
		if w := deleteOneRequest(a, current.ID, current.Photos[0].ID, shippingPhotoRevision(&current), cookie); w.Code != 303 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	pending := a.state.ShippingRecords[0]
	if pending.ID == r.ID || pending.ArchivedAt != nil || !bytes.Equal(jsonRaw(a.state.Projects), jsonRaw(before.Projects)) {
		t.Fatal("last deletion did not safely requeue")
	}
	if w := shippingUpload(t, a, r.ID, 1, false); w.Code != 404 {
		t.Fatal("old upload revived reset record")
	}
}
func TestShippingPhotoDeleteLastHistoricalAndSnapshotProtection(t *testing.T) {
	a := fixture(t)
	r := approveShipment(t, a)
	shippingUpload(t, a, r.ID, 1, false)
	record := &a.state.ShippingRecords[0]
	ph := record.Photos[0]
	// Shared snapshot references must keep their physical file even after archive-photo deletion.
	record.Project.Photos = []Photo{ph}
	record.Photos = append(record.Photos, ph)
	rev := shippingPhotoRevision(record)
	if _, err := a.deleteShippingPhoto(r.ID, ph.ID, rev); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(a.dir, "photos", ph.ID)); err != nil {
		t.Fatal("shared file pruned", err)
	}
	// Once the source is gone, deleting its last photo follows existing full-archive deletion semantics.
	a.state.Projects = nil
	record = &a.state.ShippingRecords[0]
	if next, err := a.deleteShippingPhoto(r.ID, ph.ID, shippingPhotoRevision(record)); err != nil || next != "" || len(a.state.ShippingRecords) != 0 {
		t.Fatal("historical last-photo deletion", next, err)
	}
}
func TestShippingPhotoDeleteBrowser(t *testing.T) {
	if os.Getenv("SHIPPING_DELETE_BROWSER") != "1" {
		t.Skip("optional actual photo deletion browser")
	}
	a := fixture(t)
	ids := []string{}
	for i := 0; i < 4; i++ {
		if i > 0 {
			p := Project{ID: a.state.next(), Number: "DELETE-" + fmtID(int64(i)), Name: "사진 개별 삭제 시험", Revision: 1, InfoRevision: 1}
			p.rotate()
			a.state.Projects = append([]Project{p}, a.state.Projects...)
		}
		r := approveShipment(t, a)
		shippingUpload(t, a, r.ID, 3, false)
		ids = append(ids, r.ID)
	}
	before := jsonRaw(a.state.Projects)
	cookie := adminCookie(a)
	srv := httptest.NewServer(a)
	defer srv.Close()
	cmd := exec.Command("node", "audit/browser-shipping-delete-v1.4.77.cjs", srv.URL, a.state.Board, string(jsonRaw(map[string]any{"ids": ids, "cookie": map[string]string{"name": cookie.Name, "value": cookie.Value}})))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, jsonRaw(a.state.Projects)) {
		t.Fatal("inspection changed")
	}
	for _, r := range a.state.ShippingRecords {
		if r.ArchivedAt != nil || len(r.Photos) != 0 {
			t.Fatal("not requeued")
		}
	}
}
