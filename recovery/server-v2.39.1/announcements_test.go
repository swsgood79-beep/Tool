package main

import (
	"bytes"
	"filippo.io/age"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func noticePNG(t *testing.T) []byte {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, 240, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 240; x++ {
			im.Set(x, y, color.RGBA{30, 90, 130, 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func noticePOST(t *testing.T, a *App, c *http.Cookie, fields map[string]string, photos int) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < photos; i++ {
		f, err := mw.CreateFormFile("photos", "보호커버.png")
		if err != nil {
			t.Fatal(err)
		}
		f.Write(noticePNG(t))
	}
	mw.Close()
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791/announcements/save", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	if c != nil {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func TestAnnouncementsCRUDPermissionsPersistenceAndBackup(t *testing.T) {
	a := fixture(t)
	c := adminCookie(a)
	book := filepath.Join(a.dir, "검사완료기록-복구서버.xlsx")
	beforeBook, _ := os.ReadFile(book)
	beforeProject := jsonRaw(a.state.Projects)
	beforeEvents := jsonRaw(a.state.Events)
	fields := map[string]string{"csrf": "admin-csrf", "title": "출하 전 안전 점검 안내", "text": "보호커버와 체결 상태를 확인해 주세요.\n이상이 있으면 관리자에게 알려 주세요.", "urgent": "1", "pinned": "1", "published": "1"}
	if w := noticePOST(t, a, nil, fields, 1); w.Code != 401 {
		t.Fatal("unauthenticated writer", w.Code)
	}
	bad := map[string]string{}
	for k, v := range fields {
		bad[k] = v
	}
	bad["csrf"] = "wrong"
	if w := noticePOST(t, a, c, bad, 1); w.Code != 403 {
		t.Fatal("CSRF", w.Code)
	}
	if w := noticePOST(t, a, c, fields, 2); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	n := a.state.Announcements[0]
	if len(n.Photos) != 2 || !n.Urgent || !n.Pinned || !n.Published {
		t.Fatal("new notice fields", n)
	}
	photo := "/notice-photo/" + a.state.Board + "/" + n.ID + "/" + n.Photos[0].ID
	if w := call(a, "GET", photo, nil); w.Code != 200 || !bytes.Equal(w.Body.Bytes(), noticePNG(t)) {
		t.Fatal("photo", w.Code)
	}
	for _, p := range []string{"/notices/" + token(), "/notice-panel/" + token(), "/notice-photo/" + token() + "/" + n.ID + "/" + n.Photos[0].ID} {
		if w := call(a, "GET", p, nil); w.Code != 404 {
			t.Fatal("wrong board", p, w.Code)
		}
	}
	if w := call(a, "POST", "/notices/"+a.state.Board, url.Values{"title": {"bad"}}); w.Code != 405 {
		t.Fatal("worker write allowed", w.Code)
	}
	if w := call(a, "GET", "/announcements", nil); w.Code != 303 {
		t.Fatal("admin read allowed", w.Code)
	}
	work := call(a, "GET", "/work/"+a.state.Board+"?entry=skip", nil).Body.String()
	if i, j := strings.Index(work, `id="worker-announcements"`), strings.Index(work, `class="worker-tools"`); i < 0 || i > j {
		t.Fatal("wrong placement")
	}
	if !bytes.Equal(beforeProject, jsonRaw(a.state.Projects)) || !bytes.Equal(beforeEvents, jsonRaw(a.state.Events)) {
		t.Fatal("notice changed inspection records")
	}
	afterBook, _ := os.ReadFile(book)
	if !bytes.Equal(beforeBook, afterBook) {
		t.Fatal("notice rewrote workbook")
	}
	// Editing requires the latest revision, limits total photos and permits removal.
	fields["id"], fields["revision"] = n.ID, "0"
	if w := noticePOST(t, a, c, fields, 0); w.Code != 400 {
		t.Fatal("stale edit", w.Code)
	}
	fields["revision"] = "1"
	if w := noticePOST(t, a, c, fields, 4); w.Code != 400 {
		t.Fatal("combined photo limit", w.Code)
	}
	fields["title"] = "<script>alert(1)</script>"
	fields["text"] = "<img src=x onerror=alert(1)>"
	fields["remove_photo"] = n.Photos[0].ID
	if w := noticePOST(t, a, c, fields, 0); w.Code != 303 {
		t.Fatal("edit", w.Code, w.Body.String())
	}
	n = a.state.Announcements[0]
	if len(n.Photos) != 1 || n.Revision != 2 {
		t.Fatal("edit fields")
	}
	detail := call(a, "GET", "/notices/"+a.state.Board+"/"+n.ID, nil).Body.String()
	if strings.Contains(detail, "<script>alert(1)</script>") || !strings.Contains(detail, "&lt;script&gt;") {
		t.Fatal("unsafe notice html")
	}
	restarted, err := newApp(a.dir)
	if err != nil || len(restarted.state.Announcements) != 1 || restarted.state.Announcements[0].Title != n.Title {
		t.Fatal("restart lost notice", err)
	}
	identity, _ := age.GenerateX25519Identity()
	key := RecoveryKey{1, repository, token(), identity.String()}
	a.state.Integration.Backup.Key = key
	encrypted, err := a.portable(a.state)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := newApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = fresh.restorePortable(key, encrypted); err != nil {
		t.Fatal(err)
	}
	restoredPhoto := "/notice-photo/" + fresh.state.Board + "/" + n.ID + "/" + n.Photos[0].ID
	if w := call(fresh, "GET", restoredPhoto, nil); w.Code != 200 || !bytes.Equal(w.Body.Bytes(), noticePNG(t)) {
		t.Fatal("backup lost notice photo", w.Code)
	}
	// A notice-only live database also cannot be overwritten by a restore.
	fresh.state.Projects = nil
	fresh.state.Events = nil
	fresh.state.Designers = nil
	fresh.state.ClientNotes = nil
	if err = fresh.restorePortable(key, encrypted); err == nil {
		t.Fatal("restore overwrote notices")
	}
	if w := call(a, "POST", "/announcements/end", url.Values{"csrf": {"admin-csrf"}, "id": {n.ID}, "revision": {"2"}}, c); w.Code != 303 {
		t.Fatal("end", w.Code)
	}
	for _, p := range []string{photo, restoredPhoto, "/notices/" + a.state.Board + "/" + n.ID} {
		if w := call(a, "GET", p, nil); w.Code != 404 {
			t.Fatal("ended notice exposed", p, w.Code)
		}
	}
	panel := call(a, "GET", "/notice-panel/"+a.state.Board, nil).Body.String()
	if !strings.Contains(panel, `hidden`) || strings.Contains(panel, "<script>") {
		t.Fatal("empty notices not hidden")
	}
	if w := call(a, "GET", "/announcements/"+n.ID+"/edit", nil, c); w.Code != 200 {
		t.Fatal("ended notice no longer editable")
	}
}

func TestAnnouncementScreensAndPollingFixture(t *testing.T) {
	a := fixture(t)
	c := adminCookie(a)
	fields := map[string]string{"csrf": "admin-csrf", "title": "출하 전 안전 점검 안내", "text": "오늘 출하 예정 제품은\n보호커버와 체결 상태를 확인해 주세요.\n이상이 있으면 관리자에게 알려 주세요.", "urgent": "1", "pinned": "1", "published": "1"}
	if w := noticePOST(t, a, c, fields, 2); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	id := a.state.Announcements[0].ID
	out := os.Getenv("ANNOUNCEMENT_HTML_OUT")
	for _, phase := range []string{"before", "updated", "ended"} {
		if phase == "updated" {
			fields["id"], fields["revision"] = id, "1"
			fields["title"] = "공지 내용이 변경되었습니다"
			if w := noticePOST(t, a, c, fields, 0); w.Code != 303 {
				t.Fatal(w.Code)
			}
		}
		if phase == "ended" {
			if w := call(a, "POST", "/announcements/end", url.Values{"csrf": {"admin-csrf"}, "id": {id}, "revision": {"2"}}, c); w.Code != 303 {
				t.Fatal(w.Code)
			}
		}
		paths := map[string]string{"work.html": "/work/" + a.state.Board + "?entry=skip", "admin.html": "/", "manage.html": "/announcements", "new.html": "/announcements/new", "edit.html": "/announcements/" + id + "/edit", "admin-detail.html": "/announcements/" + id, "list.html": "/notices/" + a.state.Board, "panel.html": "/notice-panel/" + a.state.Board}
		if phase != "ended" {
			paths["detail.html"] = "/notices/" + a.state.Board + "/" + id
		}
		for name, route := range paths {
			w := call(a, "GET", route, nil, c)
			if w.Code != 200 {
				t.Fatal(name, w.Code, w.Body.String())
			}
			if out != "" {
				if err := os.WriteFile(filepath.Join(out, phase+"-"+name), w.Body.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	if out != "" {
		meta := map[string]any{"board": a.state.Board, "id": id, "photos": a.state.Announcements[0].Photos}
		os.WriteFile(filepath.Join(out, "fixture.json"), jsonRaw(meta), 0600)
		os.WriteFile(filepath.Join(out, "photo.png"), noticePNG(t), 0600)
	}
}
