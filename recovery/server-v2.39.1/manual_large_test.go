package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"hash"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
)

type manualHashWriter struct {
	header http.Header
	status int
	hash   hash.Hash
	size   int64
}

func (w *manualHashWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}
func (w *manualHashWriter) WriteHeader(s int) { w.status = s }
func (w *manualHashWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	w.size += int64(len(b))
	return w.hash.Write(b)
}
func manualStageStart(t *testing.T, a *App, cookie *http.Cookie, size int64) string {
	t.Helper()
	w := call(a, "POST", "/manuals-admin/upload-start", url.Values{"csrf": {"admin-csrf"}, "name": {"대용량 매뉴얼.zip"}, "size": {strconv.FormatInt(size, 10)}}, cookie)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var out struct{ ID string }
	json.Unmarshal(w.Body.Bytes(), &out)
	return out.ID
}
func manualStagePart(a *App, cookie *http.Cookie, id string, index int, b []byte) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791/manuals-admin/upload-part?csrf=admin-csrf&id="+id+"&index="+strconv.Itoa(index), bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/octet-stream")
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func TestManualChunkedRangeAndRollback(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	size := int64(shippingAttachmentChunk + 17)
	id := manualStageStart(t, a, cookie, size)
	first := bytes.Repeat([]byte{42}, shippingAttachmentChunk)
	last := bytes.Repeat([]byte{73}, 17)
	for i, b := range [][]byte{first, last} {
		w := manualStagePart(a, cookie, id, i, b)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		w = manualStagePart(a, cookie, id, i, b)
		if w.Code != 200 {
			t.Fatal("retry rejected")
		}
	}
	if len(a.manualTransfers[id].File.Parts) != 2 {
		t.Fatal("retries duplicated chunks")
	}
	key := token()
	values := url.Values{"csrf": {"admin-csrf"}, "upload_key": {key}, "title": {"큰 매뉴얼"}, "file_id": {id}}
	w := call(a, "POST", "/manuals-admin/save", values, cookie)
	if w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	if len(a.manualTransfers) != 0 {
		t.Fatal("stage not finalized")
	}
	if w = call(a, "POST", "/manuals-admin/save", values, cookie); w.Code != 303 || len(a.state.Manuals) != 1 {
		t.Fatal("final retry failed")
	}
	f := a.state.Manuals[0].Files[0]
	r := httptest.NewRequest("GET", "http://127.0.0.1:18791/manuals-admin/file/"+key+"/"+f.ID, nil)
	r.AddCookie(cookie)
	r.Header.Set("Range", "bytes="+strconv.Itoa(shippingAttachmentChunk-5)+"-"+strconv.Itoa(shippingAttachmentChunk+7))
	out := httptest.NewRecorder()
	a.ServeHTTP(out, r)
	want := append(bytes.Repeat([]byte{42}, 5), bytes.Repeat([]byte{73}, 8)...)
	if out.Code != 206 || !bytes.Equal(out.Body.Bytes(), want) {
		t.Fatal("range incorrect", out.Code, out.Body.Len())
	}
	if len(a.backupPins) != 0 {
		t.Fatal("download pins leaked")
	}
	interrupted := manualStageStart(t, a, cookie, int64(shippingAttachmentChunk))
	if w = manualStagePart(a, cookie, interrupted, 0, first); w.Code != 200 {
		t.Fatal(w.Code)
	}
	part := a.manualTransfers[interrupted].File.Parts[0]
	fresh, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	if len(fresh.state.Manuals) != 1 {
		t.Fatal("restart lost completed post")
	}
	if _, e = os.Stat(fresh.dir + "/photos/" + part.ID); !os.IsNotExist(e) {
		t.Fatal("interrupted stage not cleaned")
	}
	for _, ph := range f.Parts {
		if _, e = os.Stat(fresh.dir + "/photos/" + ph.ID); e != nil {
			t.Fatal("committed attachment removed")
		}
	}
}
func TestManualOver200MBUploadDownload(t *testing.T) {
	if os.Getenv("MANUAL_LARGE_TEST") != "1" {
		t.Skip("optional real 201 MiB transfer")
	}
	a := fixture(t)
	cookie := adminCookie(a)
	size := int64(201 << 20)
	id := manualStageStart(t, a, cookie, size)
	expected := sha256.New()
	var sent int64
	for index := 0; sent < size; index++ {
		n := min(int64(shippingAttachmentChunk), size-sent)
		b := bytes.Repeat([]byte{byte(index + 1)}, int(n))
		expected.Write(b)
		if w := manualStagePart(a, cookie, id, index, b); w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		sent += n
	}
	key := token()
	w := call(a, "POST", "/manuals-admin/save", url.Values{"csrf": {"admin-csrf"}, "upload_key": {key}, "title": {"201MiB 매뉴얼 검증"}, "file_id": {id}}, cookie)
	if w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	f := a.state.Manuals[0].Files[0]
	request := httptest.NewRequest("GET", "http://127.0.0.1:18791/manuals-admin/file/"+key+"/"+f.ID, nil)
	request.AddCookie(cookie)
	sink := &manualHashWriter{hash: sha256.New()}
	a.ServeHTTP(sink, request)
	if sink.status != 200 || sink.size != size || !bytes.Equal(expected.Sum(nil), sink.hash.Sum(nil)) {
		t.Fatal("large transfer mismatch", sink.status, sink.size)
	}
	t.Logf("Real upload/download: %d bytes, %d encrypted chunks, SHA256 %s", size, len(f.Parts), hex.EncodeToString(expected.Sum(nil)))
}
