package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestShippingArchiveAddAndListBrowser(t *testing.T) {
	if os.Getenv("ARCHIVE_BROWSER") != "1" {
		t.Skip("optional actual archive list and append browser suite")
	}
	a := fixture(t)
	ids := []string{}
	now := time.Now().In(kst)
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, kst)
	for i := 0; i < 4; i++ {
		if i > 0 {
			p := Project{ID: a.state.next(), Number: "PHOTO-" + fmtID(int64(i)), Name: "사진 추가 시험 " + fmtID(int64(i)), Revision: 1, InfoRevision: 1}
			p.rotate()
			a.state.Projects = []Project{p}
		} else {
			a.state.Projects[0].Name = "사진 추가 시험 0"
		}
		r := approveShipment(t, a)
		if w := shippingUpload(t, a, r.ID, 1, false); w.Code != 303 {
			t.Fatal(w.Code)
		}
		date := first.AddDate(0, 0, 7+i)
		a.state.ShippingRecords[i].ArchivedAt = &date
		ids = append(ids, r.ID)
	}
	ph := a.state.ShippingRecords[0].Photos[0]
	for i := 0; i < 45; i++ {
		date := first.AddDate(0, 0, i%25)
		if i >= 25 {
			date = first.AddDate(0, -1, i-25)
		}
		a.state.ShippingRecords = append(a.state.ShippingRecords, ShippingRecord{ID: token(), Project: Project{Number: "ORDER-" + fmtID(int64(100-i)), Name: "월별 정리 시험 " + fmtID(int64(i)), ShippingDueDate: day(first.AddDate(0, 0, 27-i%25))}, ClientName: "고객사A", InspectedAt: now.AddDate(-1, 0, 0), ArchivedAt: &date, Photos: []Photo{ph}})
	}
	if err := a.transaction(func(s *State) error { s.Projects = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	a.state.Integration.Connection.Provider = "pages"
	a.state.Integration.Connection.PublicURL = "https://old-worker.trycloudflare.com"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()
		a.ServeHTTP(rec, r)
		for k, vs := range rec.Header() {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		body := rec.Body.Bytes()
		if strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
			body = bytes.ReplaceAll(body, []byte(`data-origin="https://old-worker.trycloudflare.com"`), []byte(`data-origin="http://`+r.Host+`"`))
		}
		w.WriteHeader(rec.Code)
		w.Write(body)
	}))
	defer srv.Close()
	meta := map[string]any{"ids": ids, "start": day(first), "end": day(first.AddDate(0, 0, 1)), "dates": []string{a.state.ShippingRecords[0].ArchivedAt.Format(time.RFC3339), a.state.ShippingRecords[1].ArchivedAt.Format(time.RFC3339), a.state.ShippingRecords[2].ArchivedAt.Format(time.RFC3339), a.state.ShippingRecords[3].ArchivedAt.Format(time.RFC3339)}}
	cmd := exec.Command("node", "audit/browser-archive-v1.4.51.cjs", srv.URL, a.state.Board, string(jsonRaw(meta)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	for i, id := range ids {
		record := shippingRecord(&a.state, id)
		if len(record.Photos) != 3 || !record.ArchivedAt.Equal(first.AddDate(0, 0, 7+i)) {
			t.Fatal("append changed archive date or photos", i, len(record.Photos))
		}
	}
}
