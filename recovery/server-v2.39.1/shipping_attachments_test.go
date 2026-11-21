package main

import (
	"bytes"
	"encoding/json"
	"filippo.io/age"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func attachmentCall(a *App, record, action string, body []byte, cookie *http.Cookie) *httptest.ResponseRecorder {
	path := "/shipping-records/" + a.state.Board + "/" + record + "/attachment/" + action
	csrf := a.state.Board
	if cookie != nil {
		path = "/shipping-archive-admin/" + record + "/attachment/" + action
		csrf = "admin-csrf"
	}
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791"+path, bytes.NewReader(body))
	r.Header.Set("X-CSRF-Token", csrf)
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func attachmentStart(t *testing.T, a *App, record, name, kind, key string, size int64) string {
	t.Helper()
	w := attachmentCall(a, record, "start", jsonRaw(map[string]any{"Key": key, "Name": name, "Kind": kind, "Size": size}), nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var v struct{ ID string }
	json.Unmarshal(w.Body.Bytes(), &v)
	return v.ID
}
func attachmentUpload(t *testing.T, a *App, record, name, kind string, raw []byte) ShippingAttachment {
	t.Helper()
	u := attachmentStart(t, a, record, name, kind, token(), int64(len(raw)))
	for n, offset := 0, 0; offset < len(raw); n, offset = n+1, offset+shippingAttachmentChunk {
		w := attachmentCall(a, record, "chunk?upload="+u+"&part="+strconv.Itoa(n), raw[offset:min(offset+shippingAttachmentChunk, len(raw))], nil)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if w := attachmentCall(a, record, "finish?upload="+u, nil, nil); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	rec := shippingRecord(&a.state, record)
	return rec.Attachments[len(rec.Attachments)-1]
}
func TestShippingAttachmentsChunksRangePersistenceAndPermissions(t *testing.T) {
	a := fixture(t)
	rec := approveShipment(t, a)
	raw := bytes.Repeat([]byte("parameter=123\n"), 400000)
	file := attachmentUpload(t, a, rec.ID, "../리니어 파라미터.prm", "file", raw)
	if file.Name != "리니어 파라미터.prm" || len(file.Parts) != 2 || shippingRecord(&a.state, rec.ID).ArchivedAt != nil {
		t.Fatal("name/parts or upload completed photo obligation")
	}
	path := "/shipping-records/" + a.state.Board + "/" + rec.ID + "/attachment/file/" + file.ID
	w := call(a, "GET", path, nil)
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), raw) || !strings.Contains(w.Header().Get("Content-Disposition"), "attachment") {
		t.Fatal("original parameter roundtrip", w.Code)
	}
	r := httptest.NewRequest("GET", "http://127.0.0.1:18791"+path, nil)
	r.Header.Set("Range", "bytes=4194300-4194320")
	w = httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != 206 || !bytes.Equal(w.Body.Bytes(), raw[4194300:4194321]) {
		t.Fatal("range across encrypted parts", w.Code, w.Body.Len())
	}
	if w = attachmentCall(a, rec.ID, "delete", jsonRaw(map[string]string{"ID": file.ID}), nil); w.Code != 403 {
		t.Fatal("worker delete")
	}
	r = httptest.NewRequest("POST", "http://127.0.0.1:18791/shipping-records/"+a.state.Board+"/"+rec.ID+"/attachment/start", strings.NewReader(`{}`))
	w = httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("csrf")
	}
	oldDate := time.Now()
	a.state.ShippingRecords[0].ArchivedAt = &oldDate
	small := attachmentUpload(t, a, rec.ID, "packing.mp4", "video", append([]byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'}, bytes.Repeat([]byte{0}, 100)...))
	if !a.state.ShippingRecords[0].ArchivedAt.Equal(oldDate) || small.MIME != "video/mp4" {
		t.Fatal("append date/mime")
	}
	reopened, err := newApp(a.dir)
	if err != nil {
		t.Fatal(err)
	}
	if w = call(reopened, "GET", path, nil); w.Code != 200 || !bytes.Equal(w.Body.Bytes(), raw) {
		t.Fatal("restart")
	}
	owner := adminCookie(reopened)
	if w = attachmentCall(reopened, rec.ID, "delete", jsonRaw(map[string]string{"ID": file.ID}), owner); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, part := range file.Parts {
		if _, err := os.Stat(filepath.Join(a.dir, "photos", part.ID)); !os.IsNotExist(err) {
			t.Fatal("deleted part not reclaimed", err)
		}
	}
	if len(reopened.state.ShippingRecords[0].Attachments) != 1 {
		t.Fatal("deleted another attachment")
	}
}
func TestShippingAttachmentRetryLimitCleanupAndBackup(t *testing.T) {
	a := fixture(t)
	rec := approveShipment(t, a)
	key := token()
	u := attachmentStart(t, a, rec.ID, "file.bin", "file", key, 3)
	for i := 0; i < 2; i++ {
		if w := attachmentCall(a, rec.ID, "chunk?upload="+u+"&part=0", []byte("abc"), nil); w.Code != 200 {
			t.Fatal(w.Code)
		}
	}
	if w := attachmentCall(a, rec.ID, "chunk?upload="+u+"&part=0", []byte("bad"), nil); w.Code != 400 {
		t.Fatal("conflicting retry")
	}
	for i := 0; i < 2; i++ {
		if w := attachmentCall(a, rec.ID, "finish?upload="+u, nil, nil); w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if len(a.state.ShippingRecords[0].Attachments) != 1 {
		t.Fatal("finish retry duplicated")
	}
	if _, err := attachmentFileMeta("x.mp4", "video", (300<<20)+1); err == nil {
		t.Fatal("video limit")
	}
	if _, err := attachmentFileMeta("x.dat", "file", (100<<20)+1); err == nil {
		t.Fatal("file limit")
	}
	bad := attachmentStart(t, a, rec.ID, "bad.mp4", "video", token(), 3)
	if w := attachmentCall(a, rec.ID, "chunk?upload="+bad+"&part=0", []byte("abc"), nil); w.Code != 400 {
		t.Fatal("fake video accepted")
	}
	a.attachmentTransfers.Stages[bad].Updated = time.Now().Add(-3 * time.Hour)
	a.attachmentTransfers.Lock()
	a.cleanupAttachmentStagesLocked()
	a.attachmentTransfers.Unlock()
	if a.attachmentTransfers.Stages[bad] != nil {
		t.Fatal("expired upload retained")
	}
	identity, _ := age.GenerateX25519Identity()
	a.state.Integration.Backup.Key = RecoveryKey{1, repository, token(), identity.String()}
	b, err := a.portable(a.state)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := newApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = restored.restorePortable(a.state.Integration.Backup.Key, b); err != nil {
		t.Fatal(err)
	}
	f := restored.state.ShippingRecords[0].Attachments[0]
	w := call(restored, "GET", "/shipping-records/"+restored.state.Board+"/"+rec.ID+"/attachment/file/"+f.ID, nil)
	if w.Code != 200 || w.Body.String() != "abc" {
		t.Fatal("backup omitted attachment", w.Code, w.Body.String())
	}
	// Reopening/cancelling a pending inspection must reclaim attached binary chunks.
	p := &a.state.Projects[0]
	p.Shipping.Completed = nil
	if err = a.transaction(func(s *State) error { s.Projects[0].Shipping.Completed = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	if len(a.state.ShippingRecords) != 0 {
		t.Fatal("cancel did not clear pending record")
	}
	for _, part := range f.Parts {
		if _, err = os.Stat(filepath.Join(a.dir, "photos", part.ID)); !os.IsNotExist(err) {
			t.Fatal("cancel left attachment")
		}
	}
}
func TestShippingAttachmentUploadDoesNotBlockOrders(t *testing.T) {
	a := fixture(t)
	rec := approveShipment(t, a)
	u := attachmentStart(t, a, rec.ID, "x.bin", "file", token(), 3)
	pr, pw := io.Pipe()
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791/shipping-records/"+a.state.Board+"/"+rec.ID+"/attachment/chunk?upload="+u+"&part=0", pr)
	r.Header.Set("X-CSRF-Token", a.state.Board)
	done := make(chan struct{})
	go func() { a.ServeHTTP(httptest.NewRecorder(), r); close(done) }()
	status := make(chan int, 1)
	go func() { status <- call(a, "GET", "/work/"+a.state.Board, nil).Code }()
	select {
	case code := <-status:
		if code != 200 {
			t.Fatal(code)
		}
	case <-time.After(time.Second):
		t.Fatal("slow attachment held App.mu")
	}
	pw.Write([]byte("abc"))
	pw.Close()
	<-done
}

func TestShippingAttachmentArchiveResetAndDelete(t *testing.T) {
	a := fixture(t)
	r := approveShipment(t, a)
	file := attachmentUpload(t, a, r.ID, "axis.prm", "file", []byte("parameters"))
	if w := shippingUpload(t, a, r.ID, 1, false); w.Code != 303 {
		t.Fatal(w.Code)
	}
	reset, err := a.deleteShippingArchive(r.ID)
	if err != nil || !reset {
		t.Fatal(reset, err)
	}
	current := a.state.ShippingRecords[0]
	if current.ID == r.ID || current.ArchivedAt != nil || len(current.Attachments) != 1 {
		t.Fatal("photo reset lost files")
	}
	if w := call(a, "GET", "/shipping-records/"+a.state.Board+"/"+current.ID+"/attachment/file/"+file.ID, nil); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if err = a.commitAnnouncement(func(s *State) error { s.Projects = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err = a.deleteShippingArchive(current.ID); err != nil {
		t.Fatal(err)
	}
	for _, part := range file.Parts {
		if _, err = os.Stat(filepath.Join(a.dir, "photos", part.ID)); !os.IsNotExist(err) {
			t.Fatal("full archive delete retained chunks")
		}
	}
}
