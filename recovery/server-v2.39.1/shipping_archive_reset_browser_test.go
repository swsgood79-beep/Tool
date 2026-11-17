package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestShippingArchiveResetBrowser(t *testing.T) {
	if os.Getenv("ARCHIVE_RESET_BROWSER") != "1" {
		t.Skip("optional admin reset and recovery browser")
	}
	a := fixture(t)
	live := []Project{}
	ids := []string{}
	missing := []Project{}
	for i := 0; i < 8; i++ {
		if i > 0 {
			p := Project{ID: a.state.next(), Number: "RESET-" + fmtID(int64(i)), Name: "보관 삭제 복구 " + fmtID(int64(i)), Revision: 1, InfoRevision: 1}
			p.rotate()
			a.state.Projects = []Project{p}
		} else {
			a.state.Projects[0].Name = "보관 삭제 복구 0"
		}
		record := approveShipment(t, a)
		live = append(live, a.state.Projects[0])
		if i < 4 {
			ids = append(ids, record.ID)
			if w := shippingUpload(t, a, record.ID, 1, false); w.Code != 303 {
				t.Fatal(w.Code)
			}
		} else {
			missing = append(missing, a.state.Projects[0])
			a.state.ShippingRecords = a.state.ShippingRecords[:len(a.state.ShippingRecords)-1]
		}
	}
	a.state.Projects = live
	if err := a.transaction(func(*State) error { return nil }); err != nil {
		t.Fatal(err)
	}
	before := jsonRaw(a.state.Projects)
	cookie := adminCookie(a)
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
	metadata := map[string]any{"ids": ids, "missing": missing, "cookie": map[string]string{"name": cookie.Name, "value": cookie.Value}}
	cmd := exec.Command("node", "audit/browser-archive-reset-v1.4.53.cjs", srv.URL, a.state.Board, string(jsonRaw(metadata)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, jsonRaw(a.state.Projects)) {
		t.Fatal("browser reset changed live inspections")
	}
	if len(a.state.ShippingRecords) != 8 {
		t.Fatal("wrong records after repair")
	}
	for i, p := range live {
		r := shippingRecordForProject(&a.state, p)
		if r == nil {
			t.Fatal("missing obligation")
		}
		if i < 4 && (r.ArchivedAt == nil || len(r.Photos) != 1) {
			t.Fatal("reupload failed")
		}
		if i >= 4 && (r.ArchivedAt != nil || len(r.Photos) != 0) {
			t.Fatal("repair changed pending state")
		}
	}
}
