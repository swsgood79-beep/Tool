package main

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func archivePhotoUpload(t *testing.T, a *App, id, key, revision string, count int) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	m := multipart.NewWriter(&b)
	m.WriteField("csrf", a.state.Board)
	m.WriteField("upload_key", key)
	m.WriteField("photo_revision", revision)
	var raw bytes.Buffer
	png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 10, 10)))
	for i := 0; i < count; i++ {
		f, _ := m.CreateFormFile("photos", "packing.png")
		f.Write(raw.Bytes())
	}
	m.Close()
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791/shipping-records/"+a.state.Board+"/"+id+"/add-photos", &b)
	r.Header.Set("Content-Type", m.FormDataContentType())
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func TestArchivePhotoAppendPreservesOriginalAndDate(t *testing.T) {
	a := fixture(t)
	r := approveShipment(t, a)
	if w := shippingUpload(t, a, r.ID, 1, false); w.Code != 303 {
		t.Fatal(w.Code)
	}
	before, _ := clone(a.state)
	record := &a.state.ShippingRecords[0]
	key, revision := token(), shippingPhotoRevision(record)
	if w := archivePhotoUpload(t, a, r.ID, key, revision, 2); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	record = &a.state.ShippingRecords[0]
	if len(record.Photos) != 3 || record.Photos[0].ID != before.ShippingRecords[0].Photos[0].ID || !record.ArchivedAt.Equal(*before.ShippingRecords[0].ArchivedAt) || record.Memo != before.ShippingRecords[0].Memo || !bytes.Equal(jsonRaw(record.Project), jsonRaw(before.ShippingRecords[0].Project)) {
		t.Fatal("append changed original data")
	}
	if w := archivePhotoUpload(t, a, r.ID, key, revision, 2); w.Code != 303 || len(a.state.ShippingRecords[0].Photos) != 3 {
		t.Fatal("repeat POST duplicated photos")
	}
	if w := archivePhotoUpload(t, a, r.ID, token(), revision, 1); w.Code != 400 {
		t.Fatal("stale second tab accepted")
	}
	record = &a.state.ShippingRecords[0]
	if w := archivePhotoUpload(t, a, r.ID, token(), shippingPhotoRevision(record), 18); w.Code != 400 {
		t.Fatal("total greater than 20 accepted")
	}
	if w := archivePhotoUpload(t, a, r.ID, token(), shippingPhotoRevision(record), 0); w.Code != 400 {
		t.Fatal("empty addition accepted")
	}
	files, _ := os.ReadDir(filepath.Join(a.dir, "photos"))
	if len(files) != 3 {
		t.Fatal("rejected additions leaked files", len(files))
	}
	fresh, err := newApp(a.dir)
	if err != nil || len(fresh.state.ShippingRecords[0].Photos) != 3 {
		t.Fatal("append restart", err)
	}
}
func TestArchivePhotoAppendRollbackAndPermission(t *testing.T) {
	a := fixture(t)
	r := approveShipment(t, a)
	if w := archivePhotoUpload(t, a, r.ID, token(), shippingPhotoRevision(&r), 1); w.Code != 400 {
		t.Fatal("pending record can append")
	}
	shippingUpload(t, a, r.ID, 1, false)
	record := &a.state.ShippingRecords[0]
	if w := call(a, "POST", "/shipping-records/"+a.state.Board+"/"+r.ID+"/add-photos", url.Values{"csrf": {"wrong"}}); w.Code != 403 {
		t.Fatal("csrf")
	}
	a.directHook = func(phase, name string) error {
		if phase == "written" {
			return errors.New("test failure")
		}
		return nil
	}
	w := archivePhotoUpload(t, a, r.ID, token(), shippingPhotoRevision(record), 1)
	if w.Code != 400 || len(a.state.ShippingRecords[0].Photos) != 1 {
		t.Fatal("failed commit mutated archive")
	}
	files, _ := os.ReadDir(filepath.Join(a.dir, "photos"))
	if len(files) != 1 {
		t.Fatal("rollback leaked photo")
	}
}
func TestArchiveListDatesPaginationAndSort(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, kst)
	state := State{}
	for i := 0; i < 45; i++ {
		month := time.November
		d := i + 1
		if i >= 25 {
			month = time.October
			d = i - 24
		}
		date := time.Date(2026, month, d, 23, 59, 59, 0, kst)
		state.ShippingRecords = append(state.ShippingRecords, ShippingRecord{ID: fmtID(int64(i)), InspectedAt: now.AddDate(-1, 0, 0), ArchivedAt: &date, ClientName: "고객사", Project: Project{Number: fmtID(int64(100 - i)), Name: "프로젝트", ShippingDueDate: "2026-12-" + fmtID(int64(30-i%20))}})
	}
	state.ShippingRecords = append(state.ShippingRecords, ShippingRecord{ID: "pending", InspectedAt: now})
	check := func(q url.Values) map[string]any {
		t.Helper()
		v, e := shippingListView(&state, "/archive", q, now)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	v := check(url.Values{})
	if v["Total"] != 45 || v["Pages"] != 3 || len(v["Records"].([]ShippingRecord)) != 20 {
		t.Fatal("page size/count", v)
	}
	v = check(url.Values{"page": {"2"}, "q": {"고객사"}})
	groups := v["Months"].([]ShippingMonth)
	if len(groups) != 2 || groups[0].Count != 25 || groups[1].Count != 20 {
		t.Fatal("month totals across pages")
	}
	v = check(url.Values{"period": {"month"}})
	if v["Total"] != 25 {
		t.Fatal("period must use archive date")
	}
	v = check(url.Values{"period": {"last"}})
	if v["Total"] != 20 {
		t.Fatal("last month")
	}
	v = check(url.Values{"period": {"custom"}, "start": {"2026-11-01"}, "end": {"2026-11-02"}})
	if v["Total"] != 2 {
		t.Fatal("date endpoints not inclusive")
	}
	for _, sortBy := range []string{"due", "number"} {
		v = check(url.Values{"period": {"month"}, "sort": {sortBy}})
		records := v["Records"].([]ShippingRecord)
		for i := 1; i < len(records); i++ {
			a, b := records[i-1].Project, records[i].Project
			if sortBy == "due" && a.ShippingDueDate > b.ShippingDueDate || sortBy == "number" && a.Number > b.Number {
				t.Fatal("sorting", sortBy)
			}
		}
	}
	v = check(url.Values{"q": {"missing"}, "page": {"999"}})
	if v["Page"] != 1 || v["Pages"] != 1 || len(v["Records"].([]ShippingRecord)) != 0 {
		t.Fatal("empty page")
	}
	if _, err := shippingListView(&state, "/archive", url.Values{"period": {"custom"}, "start": {"2026-11-02"}, "end": {"2026-11-01"}}, now); err == nil {
		t.Fatal("reversed dates accepted")
	}
}
