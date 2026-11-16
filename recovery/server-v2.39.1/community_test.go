package main

import (
	"bytes"
	"filippo.io/age"
	"image"
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

func communityOwner(t *testing.T, a *App) *http.Cookie {
	t.Helper()
	w := call(a, "GET", "/community/"+a.state.Board, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == "alpha-community-owner-v1" {
			return c
		}
	}
	t.Fatal("missing owner cookie")
	return nil
}
func communitySave(t *testing.T, a *App, c *http.Cookie, values map[string]string, photo bool) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	for k, v := range values {
		mw.WriteField(k, v)
	}
	if photo {
		f, _ := mw.CreateFormFile("photos", "현장.png")
		im := image.NewRGBA(image.Rect(0, 0, 2000, 1000))
		png.Encode(f, im)
	}
	mw.Close()
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791/community/"+a.state.Board+"/save", &b)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	if c != nil {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func TestCommunityOwnershipPhotosBackupAndDeletion(t *testing.T) {
	a := fixture(t)
	owner := communityOwner(t, a)
	other := communityOwner(t, a)
	admin := adminCookie(a)
	base := "/community/" + a.state.Board
	before := jsonRaw(a.state.Projects)
	events := jsonRaw(a.state.Events)
	book, _ := os.ReadFile(filepath.Join(a.dir, "검사완료기록-복구서버.xlsx"))
	f := map[string]string{"csrf": a.state.Board, "author": "작업자 시험", "category": "daily", "title": "오늘의 현장 이야기", "text": "사진과 함께 남기는 일상입니다."}
	if w := communitySave(t, a, nil, f, true); w.Code != 403 {
		t.Fatal("anonymous write", w.Code)
	}
	if w := communitySave(t, a, owner, f, true); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	post := a.state.Community[0]
	if len(post.Photos) != 1 || post.Photos[0].Width != 1600 || post.Photos[0].Height != 800 || post.Photos[0].MIME != "image/jpeg" {
		t.Fatal("photo not resized", post.Photos)
	}
	ph := post.Photos[0]
	if w := call(a, "GET", base+"/"+post.ID+"/photos/"+ph.ID, nil, other); w.Code != 200 || w.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatal("photo route", w.Code)
	}
	if w := call(a, "GET", "/community/"+token(), nil); w.Code != 404 {
		t.Fatal("board guard")
	}
	if w := call(a, "GET", base+"/"+post.ID+"/edit", nil, other); w.Code != 403 {
		t.Fatal("edit ownership", w.Code)
	}
	f["id"], f["revision"] = post.ID, "1"
	if w := communitySave(t, a, other, f, false); w.Code != 400 {
		t.Fatal("other owner save")
	}
	bad := url.Values{"csrf": {"wrong"}, "id": {post.ID}}
	if w := call(a, "POST", base+"/like", bad, owner); w.Code != 403 {
		t.Fatal("CSRF")
	}
	v := url.Values{"csrf": {a.state.Board}, "id": {post.ID}}
	for i := 0; i < 2; i++ {
		if w := call(a, "POST", base+"/like", v, other); w.Code != 303 {
			t.Fatal("like", w.Code)
		}
	}
	if len(a.state.Community[0].Likes) != 0 {
		t.Fatal("like toggle")
	}
	v.Set("author", "댓글 시험")
	v.Set("text", "<script>alert(1)</script>")
	if w := call(a, "POST", base+"/comment", v, other); w.Code != 303 {
		t.Fatal("comment", w.Code)
	}
	html := call(a, "GET", base+"/"+post.ID, nil, owner).Body.String()
	if !strings.Contains(html, "&lt;script&gt;") || strings.Contains(html, "<script>alert(1)</script>") {
		t.Fatal("unescaped comment")
	}
	comment := a.state.Community[0].Comments[0]
	v.Set("comment", comment.ID)
	if w := call(a, "POST", base+"/delete-comment", v, owner); w.Code != 400 {
		t.Fatal("comment ownership")
	}
	if w := call(a, "POST", base+"/delete-comment", v, other); w.Code != 303 {
		t.Fatal("comment delete")
	}
	f["revision"] = "1"
	if w := communitySave(t, a, owner, f, false); w.Code != 400 {
		t.Fatal("stale edit")
	}
	restart, err := newApp(a.dir)
	if err != nil || len(restart.state.Community) != 1 {
		t.Fatal("restart", err)
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
	if w := call(fresh, "GET", base+"/"+post.ID+"/photos/"+ph.ID, nil); w.Code != 200 {
		t.Fatal("backup photo missing", w.Code)
	}
	if !bytes.Equal(before, jsonRaw(a.state.Projects)) || !bytes.Equal(events, jsonRaw(a.state.Events)) {
		t.Fatal("community changed orders")
	}
	afterBook, _ := os.ReadFile(filepath.Join(a.dir, "검사완료기록-복구서버.xlsx"))
	if !bytes.Equal(book, afterBook) {
		t.Fatal("community rewrote workbook")
	}
	v.Set("csrf", "admin-csrf")
	if w := call(a, "POST", "/community-admin/delete", v, admin); w.Code != 303 {
		t.Fatal("admin moderation", w.Code)
	}
	if len(a.state.Community) != 0 {
		t.Fatal("delete persisted")
	}
	if _, err = os.Stat(filepath.Join(a.dir, "photos", ph.ID)); !os.IsNotExist(err) {
		t.Fatal("deleted post photo not freed", err)
	}
}
func TestCommunityPagesFixture(t *testing.T) {
	a := fixture(t)
	owner := communityOwner(t, a)
	admin := adminCookie(a)
	base := "/community/" + a.state.Board
	for _, cat := range []string{"work", "suggestion", "daily"} {
		f := map[string]string{"csrf": a.state.Board, "author": "김동홍", "category": cat, "title": map[string]string{"work": "보호커버 체결 순서 공유합니다", "suggestion": "공구 보관 위치를 정하면 어떨까요", "daily": "오늘 점심 맛있게 드셨나요"}[cat], "text": "현장 이야기와 사진을 함께 나눕니다.\n동료 여러분의 의견도 들려주세요."}
		if w := communitySave(t, a, owner, f, true); w.Code != 303 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	id := a.state.Community[2].ID
	v := url.Values{"csrf": {a.state.Board}, "id": {id}, "author": {"박광현"}, "text": {"좋은 이야기 감사합니다!"}}
	if w := call(a, "POST", base+"/comment", v, owner); w.Code != 303 {
		t.Fatal(w.Code)
	}
	out := os.Getenv("COMMUNITY_HTML_OUT")
	routes := map[string]string{"feed.html": base, "work-category.html": base + "?category=work", "suggestion-category.html": base + "?category=suggestion", "daily-category.html": base + "?category=daily", "new.html": base + "/new", "edit.html": base + "/" + id + "/edit", "detail.html": base + "/" + id, "admin.html": "/community-admin", "work.html": "/work/" + a.state.Board + "?entry=skip", "home.html": "/"}
	for name, route := range routes {
		w := call(a, "GET", route, nil, owner, admin)
		if w.Code != 200 {
			t.Fatal(route, w.Code, w.Body.String())
		}
		if out != "" {
			os.WriteFile(filepath.Join(out, name), w.Body.Bytes(), 0600)
		}
	}
	if out != "" {
		os.WriteFile(filepath.Join(out, "fixture.json"), jsonRaw(map[string]string{"board": a.state.Board, "id": id}), 0600)
		b, _ := os.ReadFile(filepath.Join(a.dir, "photos", a.state.Community[2].Photos[0].ID))
		b, _ = unprotect(b)
		os.WriteFile(filepath.Join(out, "photo.jpg"), b, 0600)
	}
}

func TestCommunityPhoneOrientation(t *testing.T) {
	// Minimal valid EXIF orientation segment; malformed offsets must be ignored.
	for orientation := 1; orientation <= 8; orientation++ {
		raw := []byte{0xff, 0xd8, 0xff, 0xe1, 0, 34, 'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 1, 3, 0, 1, 0, 0, 0, byte(orientation), 0, 0, 0, 0, 0, 0, 0}
		if got := communityOrientation(raw); got != orientation {
			t.Fatal("orientation", got, orientation)
		}
		w, h := 3, 2
		if orientation >= 5 {
			w, h = h, w
		}
		seen := map[[2]int]bool{}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				sx, sy := communitySourcePoint(x, y, 3, 2, orientation)
				if sx < 0 || sx >= 3 || sy < 0 || sy >= 2 {
					t.Fatal("orientation outside image")
				}
				seen[[2]int{sx, sy}] = true
			}
		}
		if len(seen) != 6 {
			t.Fatal("orientation lost pixels")
		}
	}
	for _, raw := range [][]byte{nil, {0xff, 0xd8}, {0xff, 0xd8, 0xff, 0xe1, 0xff, 0xff}} {
		if communityOrientation(raw) != 1 {
			t.Fatal("bad exif accepted")
		}
	}
}
