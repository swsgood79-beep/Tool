package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) *App {
	t.Helper()
	a, e := newApp(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	e = a.transaction(func(s *State) error {
		p := Project{ID: s.next(), Revision: 1, InfoRevision: 1, Number: "T-001", Name: "복원 서버 자동시험", ShippingDueDate: day(time.Now())}
		p.rotate()
		s.Projects = append(s.Projects, p)
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	return a
}
func call(a *App, method, path string, form url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	r := httptest.NewRequest(method, "http://127.0.0.1:18791"+path, body)
	if form != nil {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for _, c := range cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func adminCookie(a *App) *http.Cookie {
	value := token()
	a.sessions[hashToken(value)] = Session{"admin-csrf", time.Now().Add(time.Hour)}
	return &http.Cookie{Name: "aw_recovered_session", Value: value}
}
func TestInspectionApprovalAndRestart(t *testing.T) {
	a := fixture(t)
	p := a.state.Projects[0]
	if e := a.requestInspection(p.ID, "combined", "작업자", p.SettingForm, p.CombinedForm); e != nil {
		t.Fatal(e)
	}
	p = a.state.Projects[0]
	if !p.Setting.Pending || !p.Shipping.Pending || p.Setting.Completed != nil || p.Shipping.Completed != nil {
		t.Fatal("request completed inspection prematurely")
	}
	if e := a.requestInspection(p.ID, "combined", "중복", p.SettingForm, p.CombinedForm); e == nil {
		t.Fatal("duplicate accepted")
	}
	if e := a.approve(p.ID, "combined", p.Setting.EventID+999); e == nil {
		t.Fatal("stale approval accepted")
	}
	if e := a.approve(p.ID, "setting", p.Setting.EventID); e != nil {
		t.Fatal(e)
	}
	p = a.state.Projects[0]
	if p.Setting.Completed == nil || !p.Shipping.Pending || p.CombinedPending {
		t.Fatal("partial approval lost shipping request")
	}
	if e := a.approve(p.ID, "shipping", p.Shipping.EventID); e != nil {
		t.Fatal(e)
	}
	if r := call(a, "GET", "/work/"+a.state.Board, nil); r.Code != 200 || !strings.Contains(r.Body.String(), "세팅 작업자") {
		t.Fatal("completed worker template", r.Code, r.Body.String())
	}
	restored, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	if restored.state.Projects[0].Shipping.Completed == nil || len(completedEvents(restored.state)) != 2 {
		t.Fatal("restart lost completed records")
	}
	r := httptest.NewRequest("POST", "http://127.0.0.1", strings.NewReader(url.Values{"revision": {fmtID(restored.state.Events[0].Revision)}, "action": {"cancel"}, "confirm": {"1"}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ParseForm()
	if e = restored.editRecord(r, restored.state.Events[0].ID); e != nil {
		t.Fatal(e)
	}
	p = restored.state.Projects[0]
	if p.Setting.Completed != nil || p.Shipping.Completed != nil || len(completedEvents(restored.state)) != 0 || p.SettingForm == a.state.Projects[0].SettingForm {
		t.Fatal("cancel did not reset dependent stage and rotate forms")
	}
}
func TestRoleCSRFAndWorkerJSON(t *testing.T) {
	a := fixture(t)
	c := adminCookie(a)
	p := a.state.Projects[0]
	if r := call(a, "GET", "/", nil); r.Code != 303 {
		t.Fatal("admin allowed without login", r.Code)
	}
	form := url.Values{"project": {fmtID(p.ID)}, "inspection": {"shipping"}, "worker": {"작업자"}, "submission": {p.ShippingForm}}
	if r := call(a, "POST", "/complete/"+a.state.Board, form); r.Code != 409 {
		t.Fatal("shipping accepted before setting", r.Code)
	}
	form.Set("inspection", "setting")
	form.Set("submission", p.SettingForm)
	if r := call(a, "POST", "/complete/"+a.state.Board, form); r.Code != 303 {
		t.Fatal(r.Code, r.Body.String())
	}
	if a.state.Projects[0].Setting.Completed != nil {
		t.Fatal("request auto completed")
	}
	approval := url.Values{"csrf": {a.state.Board}, "inspection": {"setting"}, "request": {fmtID(a.state.Projects[0].Setting.EventID)}}
	if r := call(a, "POST", "/projects/1/inspection", approval); r.Code != 401 {
		t.Fatal("worker approved without login", r.Code)
	}
	if r := call(a, "POST", "/projects/1/inspection", approval, c); r.Code != 403 {
		t.Fatal("worker token bypassed admin CSRF", r.Code)
	}
	approval.Set("csrf", "admin-csrf")
	if r := call(a, "POST", "/projects/1/inspection", approval, c); r.Code != 303 {
		t.Fatal(r.Code, r.Body.String())
	}
	if r := call(a, "GET", "/status/"+a.state.Board, nil); r.Code != 200 || !strings.Contains(r.Body.String(), `"active":[1]`) {
		t.Fatal(r.Code, r.Body.String())
	}
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791/projects", strings.NewReader(""))
	r.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross origin allowed")
	}
	r = httptest.NewRequest("GET", "http://evil.example/", nil)
	w = httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("foreign Host allowed")
	}
}
func TestLoginLogout(t *testing.T) {
	a, e := newApp(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	get := call(a, "GET", "/login", nil)
	if icon := call(a, "GET", "/favicon.ico", nil); icon.Code != 204 || len(icon.Result().Cookies()) != 0 || icon.Header().Get("Location") != "" {
		t.Fatal("favicon redirected to login and invalidated nonce")
	}
	if get.Header().Get("Referrer-Policy") != "same-origin" {
		t.Fatal("form navigation Origin would be stripped")
	}
	csrf := regexp.MustCompile(`name="csrf" value="([^"]+)"`).FindStringSubmatch(get.Body.String())
	if len(csrf) != 2 {
		t.Fatal("missing login nonce")
	}
	cookies := get.Result().Cookies()
	form := url.Values{"csrf": {csrf[1]}, "password": {"test-password-12345"}, "confirm": {"test-password-12345"}}
	post := call(a, "POST", "/login", form, cookies...)
	if post.Code != 303 {
		t.Fatal(post.Code, post.Body.String())
	}
	var session *http.Cookie
	for _, c := range post.Result().Cookies() {
		if c.Name == "aw_recovered_session" {
			session = c
		}
	}
	if session == nil || a.state.PasswordHash == "" || strings.Contains(a.state.PasswordHash, "test-password") {
		t.Fatal("password not hashed")
	}
	if r := call(a, "GET", "/", nil, session); r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	s := a.sessions[hashToken(session.Value)]
	if r := call(a, "POST", "/logout", url.Values{"csrf": {s.CSRF}}, session); r.Code != 303 {
		t.Fatal(r.Code)
	}
	if r := call(a, "GET", "/", nil, session); r.Code != 303 {
		t.Fatal("logout did not invalidate session")
	}
	if r := call(a, "POST", "/login", form, cookies...); r.Code != 403 {
		t.Fatal("login nonce replay allowed")
	}
}
func TestRecoveredTemplatesAndExplicitMissingRoutes(t *testing.T) {
	a := fixture(t)
	c := adminCookie(a)
	p := a.state.Projects[0]
	if e := a.requestInspection(p.ID, "combined", "시험 작업자", p.SettingForm, p.CombinedForm); e != nil {
		t.Fatal(e)
	}
	for _, path := range []string{"/", "/projects/new", "/projects/1/edit", "/projects/1/inspection", "/projects/1/info", "/projects/1/delete", "/designers", "/designers.json", "/client-notes", "/client-notes/list.json", "/unread", "/orders/panel", "/inspection-requests", "/inspection-requests/panel", "/records/2/edit", "/work/" + a.state.Board, "/project-info/" + a.state.Board + "/1", "/card/" + a.state.Board + "/1"} {
		r := call(a, "GET", path, nil, c)
		if r.Code != 200 {
			t.Errorf("%s: %d %s", path, r.Code, r.Body.String())
		}
	}
	for _, path := range []string{"/backup", "/mobile-admin/passkey", "/market-settings", "/schedule", "/orders/import"} {
		r := call(a, "GET", path, nil, c)
		if r.Code != 501 {
			t.Errorf("missing route %s was not explicit: %d", path, r.Code)
		}
	}
	if r := call(a, "GET", "/projects/1/edit", nil, c); !strings.Contains(r.Body.String(), `name="shipping_due_date"`) {
		t.Fatal("missing manual due input")
	}
}
func TestFailedStateWriteRollsBack(t *testing.T) {
	a := fixture(t)
	a.dir = filepath.Join(a.dir, "nonexistent", "nested")
	before := a.state.Projects[0].Name
	if e := a.transaction(func(s *State) error { s.Projects[0].Name = "must not commit"; return nil }); e == nil {
		t.Fatal("write unexpectedly succeeded")
	}
	if a.state.Projects[0].Name != before {
		t.Fatal("failed disk write changed in-memory state")
	}
}
func TestWorkbookOnlyApprovedEventsAndXML(t *testing.T) {
	a := fixture(t)
	p := a.state.Projects[0]
	if e := a.requestInspection(p.ID, "combined", "=not a formula", p.SettingForm, p.CombinedForm); e != nil {
		t.Fatal(e)
	}
	if len(completedEvents(a.state)) != 0 {
		t.Fatal("pending events exported")
	}
	if e := a.approve(p.ID, "combined", a.state.Projects[0].Setting.EventID); e != nil {
		t.Fatal(e)
	}
	b, e := workbook(a.state)
	if e != nil {
		t.Fatal(e)
	}
	z, e := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if e != nil {
		t.Fatal(e)
	}
	rows := ""
	for _, f := range z.File {
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		data, _ := io.ReadAll(r)
		r.Close()
		d := xml.NewDecoder(bytes.NewReader(data))
		for {
			_, e = d.Token()
			if e == io.EOF {
				break
			}
			if e != nil {
				t.Fatalf("%s invalid XML: %v", f.Name, e)
			}
		}
		if strings.Contains(f.Name, "sheet1.xml") {
			rows = string(data)
		}
	}
	if !strings.Contains(rows, "완료시간") || strings.Count(rows, "<row ") != 3 || strings.Contains(rows, "<f>") {
		t.Fatal("wrong workbook rows or formula")
	}
	if _, e = os.Stat(filepath.Join(a.dir, "검사완료기록-복구서버.xlsx")); e != nil {
		t.Fatal(e)
	}
}

func TestDirectoriesLinkedPhotosAndEscaping(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	post := func(path string, f url.Values) *httptest.ResponseRecorder {
		f.Set("csrf", "admin-csrf")
		return call(a, "POST", path, f, cookie)
	}
	if r := post("/designers", url.Values{"action": {"create"}, "name": {"설계 담당"}, "phone": {"010-1234-5678"}}); r.Code != 303 {
		t.Fatal(r.Code, r.Body.String())
	}
	// A real multipart upload exercises image verification, encrypted file storage and data metadata.
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("csrf", "admin-csrf")
	mw.WriteField("action", "create")
	mw.WriteField("client_name", "시험 업체")
	mw.WriteField("client_text", "<script>alert(1)</script> 검사 주의")
	part, e := mw.CreateFormFile("photos", "test.png")
	if e != nil {
		t.Fatal(e)
	}
	imageBytes := new(bytes.Buffer)
	if e = png.Encode(imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 2))); e != nil {
		t.Fatal(e)
	}
	part.Write(imageBytes.Bytes())
	mw.Close()
	req := httptest.NewRequest("POST", "http://127.0.0.1:18791/client-notes", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	if rec.Code != 303 {
		t.Fatal(rec.Code, rec.Body.String())
	}
	d, c := a.state.Designers[0], a.state.ClientNotes[0]
	p := a.state.Projects[0]
	edit := url.Values{"revision": {fmtID(p.InfoRevision)}, "number": {p.Number}, "name": {p.Name}, "shipping_due_date": {p.ShippingDueDate}, "designer_id": {d.ID}, "client_template_id": {c.ID}, "client_template_revision": {fmtID(c.Revision)}, "sync_client": {"1"}}
	if r := post("/projects/1/edit", edit); r.Code != 303 {
		t.Fatal(r.Code, r.Body.String())
	}
	if r := post("/designers", url.Values{"action": {"save"}, "id": {d.ID}, "revision": {fmtID(d.Revision)}, "name": {"새 직급 이름"}, "phone": {"01099998888"}}); r.Code != 303 {
		t.Fatal(r.Code, r.Body.String())
	}
	if r := post("/client-notes", url.Values{"action": {"save"}, "id": {c.ID}, "revision": {fmtID(c.Revision)}, "client_name": {c.Name}, "client_text": {"<script>alert(2)</script> 수정 내용"}}); r.Code != 303 {
		t.Fatal(r.Code, r.Body.String())
	}
	v := a.projectView(a.state.Projects[0])
	if v.DesignerName != "새 직급 이름" || !strings.Contains(v.SpecialInstructions, "수정 내용") || len(v.Photos) != 1 {
		t.Fatal("linked source changes not reflected", v)
	}
	path := "/project-info/" + a.state.Board + "/1"
	r := call(a, "GET", path, nil)
	if r.Code != 200 || strings.Contains(r.Body.String(), "<script>alert(2)</script>") || !strings.Contains(r.Body.String(), "&lt;script&gt;") {
		t.Fatal("unescaped notes", r.Code)
	}
	photoPath := "/project-photo/" + a.state.Board + "/1/" + v.Photos[0].ID
	r = call(a, "GET", photoPath, nil)
	if r.Code != 200 || !bytes.Equal(r.Body.Bytes(), imageBytes.Bytes()) {
		t.Fatal("photo round trip failed")
	}
	c = a.state.ClientNotes[0]
	if r := post("/client-notes", url.Values{"action": {"delete"}, "id": {c.ID}, "revision": {fmtID(c.Revision)}}); r.Code != 303 {
		t.Fatal(r.Code, r.Body.String())
	}
	v = a.projectView(a.state.Projects[0])
	if v.ClientID != "" || len(v.Photos) != 1 || !strings.Contains(v.SpecialInstructions, "수정 내용") {
		t.Fatal("deleting master lost snapshot")
	}
}
