package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/xuri/excelize/v2"
	"html"
	"image"
	"io"

	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func exportFixture(t *testing.T) *App {
	t.Helper()
	a := fixture(t)
	r := approveShipment(t, a)
	if w := shippingUpload(t, a, r.ID, 2, false); w.Code != 303 {
		t.Fatal(w.Code)
	}
	attachmentUpload(t, a, r.ID, "MOT_parameter.prm", "file", []byte("PARAMETER_ONLY_DO_NOT_EMBED"))
	if err := a.commitAnnouncement(func(s *State) error {
		r := &s.ShippingRecords[0]
		at := time.Date(2026, 11, 20, 15, 30, 0, 0, time.UTC)
		r.ArchivedAt = &at
		r.ClientName = "MOT"
		r.Project.ShippingDueDate = "2026-11-21"
		r.Project.Name = "MOT 리니어로봇"
		r.Project.DesignerName = "김설계"
		r.Project.Setting.Worker = "박세팅"
		r.Project.Shipping.Worker = "이출하"
		r.Project.SpecialInstructions = "고객사 공통 주의사항"
		r.Project.AdditionalInstructions = "수주 전용 추가 메모"
		r.Receipts = []MaterialReceipt{{"서보모터", "LS", "APM-SB04", "2개"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return a
}
func exportArchive(t *testing.T, path string) map[string][]byte {
	t.Helper()
	z, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	out := map[string][]byte{}
	for _, f := range z.File {
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		b, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			t.Fatal(e)
		}
		out[f.Name] = b
	}
	return out
}
func validateWorkbookXML(t *testing.T, raw []byte) {
	t.Helper()
	z, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if e != nil {
		t.Fatal(e)
	}
	for _, f := range z.File {
		if strings.Contains(f.Name, "embeddings/") {
			t.Fatal("file or video embedded unexpectedly")
		}
		r, _ := f.Open()
		b, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			t.Fatal(e)
		}
		if strings.HasSuffix(f.Name, ".xml") || strings.HasSuffix(f.Name, ".rels") {
			d := xml.NewDecoder(bytes.NewReader(b))
			for {
				_, e = d.Token()
				if e == io.EOF {
					break
				}
				if e != nil {
					t.Fatal(f.Name, e)
				}
			}
		}
	}
}
func TestShippingExportWorkbookPhotosLinksDatesAndFiles(t *testing.T) {
	a := exportFixture(t)
	r := a.state.ShippingRecords[0]
	records := []ShippingRecord{r, r, r}
	old := time.Date(2026, 10, 31, 14, 30, 0, 0, time.UTC)
	records[2].ArchivedAt = &old
	records[0].Project.Number = "=1+1"
	records[1].Project.Name = "O'Brien : [] / very long project name for sheet collision"
	dir := t.TempDir()
	original, _ := os.ReadFile(filepath.Join(a.dir, "photos", r.Photos[0].ID))
	progress := 0
	size, books, warnings, err := buildShippingExport(context.Background(), dir, records, time.Now(), func(p Photo) ([]byte, error) { return shippingExportPhoto(a.dir, p) }, func(n int) { progress = n }, shippingExportLimits{PhotoBytes: 24 << 20, Records: 1})
	if err != nil {
		t.Fatal(err)
	}
	if size < 1 || books != 4 || warnings != 0 || progress != 3 {
		t.Fatal(size, books, warnings, progress)
	}
	archive := exportArchive(t, filepath.Join(dir, "export.zip"))
	if len(archive) != 5 || archive["2026-11_출하상세_01.xlsx"] == nil || archive["2026-11_출하상세_02.xlsx"] == nil || archive["2026-10_출하상세_01.xlsx"] == nil {
		t.Fatal("Korean month/automatic splitting", len(archive))
	}
	for name, raw := range archive {
		if strings.HasSuffix(name, ".xlsx") {
			validateWorkbookXML(t, raw)
		}
	}
	f, e := excelize.OpenReader(bytes.NewReader(archive["출하목록.xlsx"]))
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	if formula, _ := f.GetCellFormula("출하목록", "A6"); formula != "" {
		t.Fatal("formula injection")
	}
	if value, _ := f.GetCellValue("출하목록", "A6"); value != "=1+1" {
		t.Fatal(value)
	}
	if date, _ := f.GetCellValue("출하목록", "D6"); date != "2026.11.21" {
		t.Fatal("Korean date", date)
	}
	cells := exportArchiveBytes(t, archive["출하목록.xlsx"])
	if !strings.Contains(html.UnescapeString(string(cells["xl/worksheets/sheet1.xml"])), `location="'001_`) {
		t.Fatal("external sheet location missing")
	}
	detail, e := excelize.OpenReader(bytes.NewReader(archive["2026-11_출하상세_01.xlsx"]))
	if e != nil {
		t.Fatal(e)
	}
	defer detail.Close()
	sheet := detail.GetSheetName(1)
	pictureCells, e := detail.GetPictureCells(sheet)
	if e != nil || len(pictureCells) != 2 {
		t.Fatal("embedded photos", e, pictureCells)
	}
	rows, _ := detail.GetRows(sheet)
	text := fmt.Sprint(rows)
	for _, s := range []string{"김설계", "박세팅", "이출하", "LS", "APM-SB04", "수주 전용 추가 메모", "MOT_parameter.prm", "원본 별도 전달"} {
		if !strings.Contains(text, s) {
			t.Fatal("missing detail", s)
		}
	}
	for _, cell := range pictureCells {
		pics, _ := detail.GetPictures(sheet, cell)
		for _, p := range pics {
			cfg, _, err := image.DecodeConfig(bytes.NewReader(p.File))
			if err != nil || cfg.Width > 960 || cfg.Height > 960 {
				t.Fatal("photo preview", err, cfg)
			}
		}
	}
	if !bytes.Equal(original, mustRead(t, filepath.Join(a.dir, "photos", r.Photos[0].ID))) {
		t.Fatal("export modified source")
	}
	for name, b := range archive {
		if strings.HasSuffix(name, ".prm") || bytes.Contains(b, []byte("PARAMETER_ONLY_DO_NOT_EMBED")) {
			t.Fatal("attachment original unexpectedly included")
		}
	}
}
func exportArchiveBytes(t *testing.T, raw []byte) map[string][]byte {
	t.Helper()
	p := filepath.Join(t.TempDir(), "book.zip")
	if err := os.WriteFile(p, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return exportArchive(t, p)
}
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestShippingExportSingleMissingPhotoAndPins(t *testing.T) {
	a := exportFixture(t)
	r := a.state.ShippingRecords[0]
	a.mu.Lock()
	job, records, start, err := a.newShippingExportLocked("", token(), r.ID)
	a.mu.Unlock()
	if err != nil || !start {
		t.Fatal(err)
	}
	defer os.RemoveAll(job.Dir)
	if _, err = a.deleteShippingArchive(r.ID); err != nil {
		t.Fatal(err)
	}
	for _, p := range r.Photos {
		if _, err = os.Stat(filepath.Join(a.dir, "photos", p.ID)); err != nil {
			t.Fatal("pinned source removed", err)
		}
	}
	a.runShippingExport(job, records)
	if job.State != "ready" || !strings.HasSuffix(job.FileName, ".xlsx") {
		t.Fatal(job.State, job.Message)
	}
	book := mustRead(t, filepath.Join(job.Dir, "export.xlsx"))
	validateWorkbookXML(t, book)
	f, err := excelize.OpenReader(bytes.NewReader(book))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if len(f.GetSheetList()) != 2 {
		t.Fatal("single export mixed other orders", f.GetSheetList())
	}
	for _, p := range r.Photos {
		if _, err = os.Stat(filepath.Join(a.dir, "photos", p.ID)); !os.IsNotExist(err) {
			t.Fatal("unused pinned original not released", err)
		}
	}
	dir := t.TempDir()
	_, _, warnings, err := buildShippingExport(context.Background(), dir, []ShippingRecord{r}, time.Now(), func(Photo) ([]byte, error) { return nil, os.ErrNotExist }, nil, shippingExportLimits{PhotoBytes: 1 << 20, Records: 40, Single: true})
	if err != nil || warnings != 2 {
		t.Fatal(err, warnings)
	}
	missing, err := excelize.OpenFile(filepath.Join(dir, "export.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	defer missing.Close()
	if _, err = missing.GetSheetIndex("내보내기 확인"); err != nil {
		t.Fatal("missing photo report", err)
	}
}
func TestShippingExportRoutesAccessAndExpiry(t *testing.T) {
	a := exportFixture(t)
	owner := adminCookie(a)
	path := "/shipping-archive-admin/export"
	if w := call(a, "POST", path, url.Values{"export_key": {token()}, "csrf": {"bad"}}, owner); w.Code != 403 {
		t.Fatal("csrf", w.Code)
	}
	if w := call(a, "GET", path, nil); w.Code != 303 {
		t.Fatal("anonymous", w.Code)
	}
	if w := call(a, "POST", path, url.Values{"export_key": {token()}, "csrf": {"admin-csrf"}}, owner); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	} else {
		var data map[string]any
		json.Unmarshal(w.Body.Bytes(), &data)
		id := data["id"].(string)
		deadline := time.Now().Add(5 * time.Second)
		for {
			a.mu.Lock()
			ready := a.shippingExports[id].State != "building"
			a.mu.Unlock()
			if ready {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("export stuck")
			}
			time.Sleep(10 * time.Millisecond)
		}
		w = call(a, "GET", path+"?job="+id+"&download=1", nil, owner)
		if w.Code != 200 || w.Header().Get("Content-Type") != "application/zip" {
			t.Fatal("download", w.Code)
		}
		a.mu.Lock()
		job := a.shippingExports[id]
		old := time.Now().Add(-3 * time.Hour)
		job.FinishedAt = &old
		job.Readers = 1
		a.cleanupShippingExportsLocked(time.Now())
		if a.shippingExports[id] == nil {
			t.Fatal("active download removed")
		}
		job.Readers = 0
		a.cleanupShippingExportsLocked(time.Now())
		a.mu.Unlock()
		if call(a, "GET", path+"?job="+id, nil, owner).Code != 404 {
			t.Fatal("expiry")
		}
	}
}

type slowExportWriter struct {
	*httptest.ResponseRecorder
	start, release chan struct{}
	once           sync.Once
}

func (w *slowExportWriter) Write(p []byte) (int, error) {
	w.once.Do(func() { close(w.start) })
	<-w.release
	return w.ResponseRecorder.Write(p)
}
func TestShippingExportAccountsAndSlowDownload(t *testing.T) {
	a := accountFixture(t)
	r := approveShipment(t, a)
	if w := shippingUpload(t, a, r.ID, 1, false); w.Code != 303 {
		t.Fatal(w.Code)
	}
	owner := adminCookie(a)
	createAccount(t, a, owner, "export02")
	member, _ := accountLogin(t, a, "export02", accountMemberPassword)
	_, buyer := purchaseAccount(t, a, owner, "exportbuyer")
	a.mu.Lock()
	job, records, _, err := a.newShippingExportLocked("", token(), "")
	a.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(job.Dir)
	a.runShippingExport(job, records)
	path := "/shipping-archive-admin/export?job=" + job.ID
	if call(a, "GET", path, nil, member).Code != 404 {
		t.Fatal("another account read the export")
	}
	if call(a, "GET", path, nil, buyer).Code != 403 {
		t.Fatal("purchasing account read export")
	}
	writer := &slowExportWriter{ResponseRecorder: httptest.NewRecorder(), start: make(chan struct{}), release: make(chan struct{})}
	request := httptest.NewRequest("GET", "http://127.0.0.1:18791"+path+"&download=1", nil)
	request.AddCookie(owner)
	done := make(chan struct{})
	go func() { a.ServeHTTP(writer, request); close(done) }()
	defer func() { close(writer.release); <-done }()
	select {
	case <-writer.start:
	case <-time.After(3 * time.Second):
		t.Fatal("download did not start")
	}
	foreground := make(chan int, 1)
	go func() { foreground <- call(a, "GET", "/work/"+a.state.Board, nil).Code }()
	select {
	case code := <-foreground:
		if code != 200 {
			t.Fatal(code)
		}
	case <-time.After(time.Second):
		t.Fatal("slow export download blocks orders")
	}
	a.mu.Lock()
	readers := job.Readers
	a.mu.Unlock()
	if readers != 1 {
		t.Fatal("download reader not pinned")
	}
}
func TestShippingExportSplitsByPhotoBudget(t *testing.T) {
	a := exportFixture(t)
	r := a.state.ShippingRecords[0]
	old := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	previous := r
	previous.ArchivedAt = &old
	records := []ShippingRecord{r, previous, r}
	dir := t.TempDir()
	_, books, _, err := buildShippingExport(context.Background(), dir, records, time.Now(), func(p Photo) ([]byte, error) { return shippingExportPhoto(a.dir, p) }, nil, shippingExportLimits{PhotoBytes: 1, Records: 40})
	if err != nil || books != 4 {
		t.Fatal("photo budget split", err, books)
	}
	files := exportArchive(t, filepath.Join(dir, "export.zip"))
	if files["2026-11_출하상세_02.xlsx"] == nil || files["2026-10_출하상세_01.xlsx"] == nil {
		t.Fatal("unsorted months overwritten")
	}
}
