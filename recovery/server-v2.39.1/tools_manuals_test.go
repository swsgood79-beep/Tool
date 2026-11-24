package main

import (
	"bytes"
	"filippo.io/age"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func manualUpload(t *testing.T, a *App, cookie *http.Cookie, admin bool, key string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	m := multipart.NewWriter(&b)
	csrf, base := a.state.Board, "/manuals/"+a.state.Board
	if admin {
		csrf = "admin-csrf"
		base = "/manuals-admin"
	}
	for k, v := range map[string]string{"csrf": csrf, "upload_key": key, "title": "사용 매뉴얼", "text": "일반 첨부", "author": "작업자"} {
		m.WriteField(k, v)
	}
	f, _ := m.CreateFormFile("files", "로봇 매뉴얼.zip")
	f.Write(data)
	m.Close()
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791"+base+"/save", &b)
	r.Header.Set("Content-Type", m.FormDataContentType())
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func TestManualUploadDownloadOwnershipBackup(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	base := "/manuals/" + a.state.Board
	get := call(a, "GET", base, nil)
	var worker *http.Cookie
	for _, c := range get.Result().Cookies() {
		if c.Name == "alpha-community-owner-v1" {
			worker = c
		}
	}
	if worker == nil {
		t.Fatal("owner cookie missing")
	}
	data := bytes.Repeat([]byte("MANUAL-FILE-CONTENT"), 300000)
	key := token()
	w := manualUpload(t, a, worker, false, key, data)
	if w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	if len(a.state.Manuals) != 1 || len(a.state.Manuals[0].Files[0].Parts) != 2 {
		t.Fatal("file not chunked")
	}
	post := a.state.Manuals[0]
	file := post.Files[0]
	path := base + "/file/" + post.ID + "/" + file.ID
	w = call(a, "GET", path, nil)
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), data) || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment;") {
		t.Fatal("download content/headers", w.Code)
	}
	if w = call(a, "GET", "/manuals/wrong/file/"+post.ID+"/"+file.ID, nil); w.Code != 404 {
		t.Fatal("wrong board allowed")
	}
	before := len(a.state.Manuals)
	w = manualUpload(t, a, worker, false, key, data)
	if w.Code != 303 || len(a.state.Manuals) != before {
		t.Fatal("retry duplicated post")
	}
	bad := call(a, "GET", base, nil).Result().Cookies()[0]
	w = call(a, "POST", base+"/delete", url.Values{"csrf": {a.state.Board}, "id": {post.ID}}, bad)
	if w.Code != 409 {
		t.Fatal("foreign worker deleted", w.Code)
	}
	fresh, e := newApp(a.dir)
	if e != nil || len(fresh.state.Manuals) != 1 {
		t.Fatal("restart lost manuals", e)
	}
	identity, _ := age.GenerateX25519Identity()
	a.state.Integration.Backup.Key = RecoveryKey{1, repository, token(), identity.String()}
	backup, e := a.portable(a.state)
	if e != nil {
		t.Fatal(e)
	}
	restored, e := newApp(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if e = restored.restorePortable(a.state.Integration.Backup.Key, backup); e != nil {
		t.Fatal(e)
	}
	w = call(restored, "GET", path, nil)
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), data) {
		t.Fatal("backup lost file", w.Code)
	}
	w = call(a, "POST", "/manuals-admin/delete", url.Values{"csrf": {"admin-csrf"}, "id": {post.ID}}, owner)
	if w.Code != 303 || len(a.state.Manuals) != 0 {
		t.Fatal("admin delete failed", w.Code)
	}
	for _, ph := range file.Parts {
		if _, e := os.Stat(filepath.Join(a.dir, "photos", ph.ID)); !os.IsNotExist(e) {
			t.Fatal("unreferenced file retained")
		}
	}
}
func TestToolAndManualAuthAndLinks(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	for _, path := range []string{"/belt-tension/" + a.state.Board, "/manuals/" + a.state.Board} {
		w := call(a, "GET", path, nil)
		if w.Code != 200 {
			t.Fatal(path, w.Code)
		}
	}
	if w := call(a, "GET", "/belt-tension/wrong", nil); w.Code != 404 {
		t.Fatal(w.Code)
	}
	if w := call(a, "GET", "/belt-tension", nil); w.Code != 303 {
		t.Fatal(w.Code)
	}
	if w := call(a, "GET", "/belt-tension", nil, cookie); w.Code != 200 || !strings.Contains(w.Body.String(), "벨트텐션값 확인 프로그램") || strings.Contains(w.Body.String(), "onclick=") {
		t.Fatal("tool page", w.Code)
	}
	for _, path := range []string{"/community/" + a.state.Board, "/community-admin"} {
		w := call(a, "GET", path, nil, cookie)
		if !strings.Contains(w.Body.String(), "매뉴얼 게시판") {
			t.Fatal("manual tab missing", path)
		}
	}
	if w := manualUpload(t, a, nil, false, token(), []byte("data")); w.Code != 403 {
		t.Fatal("no worker identity allowed", w.Code)
	}
	w := call(a, "POST", "/manuals-admin/delete", url.Values{"csrf": {"bad"}, "id": {token()}}, cookie)
	if w.Code != 403 {
		t.Fatal("CSRF bypass", w.Code)
	}
	if len(a.state.Manuals) != 0 {
		t.Fatal("unauthorized upload stored")
	}
}
