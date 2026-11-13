package main

import (
	"bytes"
	"fmt"
	"github.com/xuri/excelize/v2"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func importFixture(t *testing.T, rows [][]any) []byte {
	t.Helper()
	b := excelize.NewFile()
	defer b.Close()
	b.SetSheetName("Sheet1", "수주등록")
	for i, row := range rows {
		axis, _ := excelize.CoordinatesToCellName(1, i+1)
		if e := b.SetSheetRow("수주등록", axis, &row); e != nil {
			t.Fatal(e)
		}
	}
	buf, e := b.WriteToBuffer()
	if e != nil {
		t.Fatal(e)
	}
	return buf.Bytes()
}
func postWorkbook(a *App, path, field string, data []byte, cookie *http.Cookie, fields map[string]string) *httptest.ResponseRecorder {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		mw.WriteField(k, v)
	}
	f, _ := mw.CreateFormFile(field, "test.xlsx")
	f.Write(data)
	mw.Close()
	r := httptest.NewRequest("POST", "http://127.0.0.1:18791"+path, &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func previewImport(t *testing.T, a *App, data []byte, cookie *http.Cookie) *ImportDraft {
	t.Helper()
	w := postWorkbook(a, "/orders/import/preview", "workbook", data, cookie, map[string]string{"csrf": "admin-csrf"})
	if w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	u, _ := url.Parse(w.Header().Get("Location"))
	d := a.imports[u.Query().Get("draft")]
	if d == nil {
		t.Fatal("missing draft")
	}
	return d
}
func commitImport(a *App, d *ImportDraft, cookie *http.Cookie) *httptest.ResponseRecorder {
	f := url.Values{"csrf": {"admin-csrf"}, "draft": {d.Token}, "nonce": {d.Nonce}}
	for _, r := range d.Rows {
		if !r.Unavailable {
			f.Add("rows", fmt.Sprint(r.Row))
			f.Set(fmt.Sprintf("designer_%d", r.Row), r.DesignerID)
			f.Set(fmt.Sprintf("client_%d", r.Row), r.ClientID)
		}
	}
	return call(a, "POST", "/orders/import/commit", f, cookie)
}
func TestImportTitleMatchingAndAmbiguity(t *testing.T) {
	p := []Designer{{ID: "one", Name: "박상준 계장"}, {ID: "two", Name: "김재열 과장"}}
	for _, name := range []string{"박상준", "박 상 준", "박상준 (계장)"} {
		if got, _ := matchDesigner(name, p); got != "one" {
			t.Fatal("name did not match", name, got)
		}
	}
	p = append(p, Designer{ID: "other", Name: "박상준 과장"})
	if got, _ := matchDesigner("박상준", p); got != "-pick" {
		t.Fatal("ambiguous name auto-selected")
	}
	if got, _ := matchDesigner("박상준 계장", p); got != "one" {
		t.Fatal("exact name did not win")
	}
	if got, _ := matchDesigner("", p); got != "" {
		t.Fatal("empty name matched")
	}
}
func TestImportPreviewCommitLinksPhotosMemoAndNoOverwrite(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	id := token()
	var photo bytes.Buffer
	jpeg.Encode(&photo, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil)
	a.createCanonicalPhoto(id, photo.Bytes())
	e := a.transaction(func(s *State) error {
		s.Designers = []Designer{{ID: token(), Name: "박상준 계장", Phone: "01000000000", Revision: 1}}
		s.ClientNotes = []ClientNote{{ID: token(), Name: "알파 업체", Text: "업체 주의", Revision: 1, Photos: []Photo{{ID: id, Name: "test.jpg", MIME: "image/jpeg", Legacy: true, Width: 2, Height: 2, Size: photo.Len()}}}}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	data := importFixture(t, [][]any{{"수주번호", "프로젝트명", "설계담당자", "특이사항 업체", "수주별 추가 메모"}, {"EX-1", "일괄 시험", "박상준", "알파 업체", "<script>memo</script>"}, {"EX-2", "연결 없음", "", "", ""}, {"T-001", "덮어쓰기 금지", "", "", ""}, {"EX-1", "파일 중복", "", "", ""}})
	d := previewImport(t, a, data, cookie)
	if d.ParseError != "" || len(d.Rows) != 4 || d.SelectedCount() != 2 {
		t.Fatal(d.ParseError, d.Rows)
	}
	if len(a.state.Projects) != 1 {
		t.Fatal("preview wrote state")
	}
	if r := call(a, "GET", "/orders/import?draft="+d.Token, nil, cookie); r.Code != 200 || strings.Contains(r.Body.String(), "<script>memo</script>") {
		t.Fatal("preview failed or unescaped", r.Code)
	}
	oldNonce := d.Nonce
	if r := commitImport(a, d, cookie); r.Code != 303 {
		t.Fatal(r.Code, r.Body.String())
	}
	if d.Result == nil || d.Result.Added != 2 || d.Result.Skipped != 2 {
		t.Fatal("bad import counts", d.Result)
	}
	if len(a.state.Projects) != 3 || a.state.Projects[0].Name != "복원 서버 자동시험" {
		t.Fatal("duplicate overwritten")
	}
	p := a.state.Projects[1]
	if p.DesignerID != a.state.Designers[0].ID || p.ClientID != a.state.ClientNotes[0].ID || !strings.Contains(p.AdditionalInstructions, "memo") {
		t.Fatal("links/memo missing")
	}
	if len(a.projectView(p).Photos) != 1 {
		t.Fatal("source photos missing")
	}
	h := readOriginal(t, a.dir)
	if len(h.Projects[1].Photos) != 1 {
		t.Fatal("source photo not materialized")
	}
	r := call(a, "POST", "/orders/import/commit", url.Values{"csrf": {"admin-csrf"}, "draft": {d.Token}, "nonce": {oldNonce}}, cookie)
	if r.Code != 303 || len(a.state.Projects) != 3 {
		t.Fatal("replayed import duplicated state")
	}
	a2, e := newApp(a.dir)
	if e != nil || len(a2.state.Projects) != 3 {
		t.Fatal("import not durable", e)
	}
}
func TestImportInvalidConnectionsRollbackAndOwnerExpiry(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	data := importFixture(t, [][]any{{"수주번호", "프로젝트명", "설계담당자"}, {"NEW-1", "미연결", "없는 담당자"}, {"NEW-2", "빈칸", ""}})
	d := previewImport(t, a, data, cookie)
	if d.Rows[0].DesignerID != "-pick" {
		t.Fatal("unknown name guessed")
	}
	r := commitImport(a, d, cookie)
	if r.Code != 200 || len(a.state.Projects) != 1 || d.Result != nil {
		t.Fatal("partial commit on unresolved choice")
	}
	other := token()
	a.sessions[hashToken(other)] = Session{CSRF: "other-session", Expires: time.Now().Add(time.Hour)}
	if r := call(a, "GET", "/orders/import?draft="+d.Token, nil, &http.Cookie{Name: "aw_recovered_session", Value: other}); r.Code != 404 {
		t.Fatal("another session accessed draft")
	}
	d.Rows[0].DesignerID = ""
	if r := commitImport(a, d, cookie); r.Code != 303 || len(a.state.Projects) != 3 {
		t.Fatal("explicit no connection failed")
	}
	expired := previewImport(t, a, data, cookie)
	expired.Created = time.Now().Add(-31 * time.Minute)
	if r := call(a, "GET", "/orders/import?draft="+expired.Token, nil, cookie); r.Code != 404 {
		t.Fatal("expired preview accepted")
	}
	if r := call(a, "POST", "/orders/import/commit", url.Values{"draft": {d.Token}}, cookie); r.Code != 403 {
		t.Fatal("CSRF bypass")
	}
}
func TestImportFormulaLimitsSheetsAndTemplate(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	book := excelize.NewFile()
	book.NewSheet("실제 수주")
	headers := []any{"수주번호", "프로젝트명"}
	book.SetSheetRow("실제 수주", "A1", &headers)
	row := []any{"FORMULA", "값 대신 수식"}
	book.SetSheetRow("실제 수주", "A2", &row)
	book.SetCellFormula("실제 수주", "B2", `"수식"`)
	buf, _ := book.WriteToBuffer()
	book.Close()
	d := previewImport(t, a, buf.Bytes(), cookie)
	if d.ParseError == "" {
		t.Fatal("bad first sheet should allow choosing another")
	}
	r := call(a, "POST", "/orders/import/sheet", url.Values{"csrf": {"admin-csrf"}, "draft": {d.Token}, "sheet": {"실제 수주"}}, cookie)
	if r.Code != 303 || len(d.Rows) != 1 || !d.Rows[0].Unavailable {
		t.Fatal("sheet selection/formula guard missing")
	}
	many := [][]any{{"수주번호", "프로젝트명"}}
	for i := 0; i < 1001; i++ {
		many = append(many, []any{fmt.Sprint(i), "프로젝트"})
	}
	if _, e := parseImportRows(importFixture(t, many), "수주등록", a.state); e == nil {
		t.Fatal("row limit ignored")
	}
	template := call(a, "GET", "/orders/import/template.xlsx", nil, cookie)
	b, e := excelize.OpenReader(bytes.NewReader(template.Body.Bytes()))
	if e != nil || template.Code != 200 {
		t.Fatal("template invalid", e)
	}
	defer b.Close()
	if v, _ := b.GetCellValue("수주등록", "A1"); v != "수주번호" {
		t.Fatal("wrong template")
	}
}
func TestWorkerClientDirectoryReadOnlyBoardPhotosAndEscaping(t *testing.T) {
	a := fixture(t)
	id := token()
	var img bytes.Buffer
	jpeg.Encode(&img, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil)
	a.createCanonicalPhoto(id, img.Bytes())
	c := ClientNote{ID: token(), Name: "전체 업체", Text: "<script>unsafe</script>", Revision: 1, Photos: []Photo{{ID: id, Name: "photo.jpg", MIME: "image/jpeg", Legacy: true, Width: 2, Height: 2, Size: img.Len()}}}
	if e := a.transaction(func(s *State) error { s.ClientNotes = []ClientNote{c}; return nil }); e != nil {
		t.Fatal(e)
	}
	path := "/client-directory/" + a.state.Board
	r := call(a, "GET", path, nil)
	if r.Code != 200 || strings.Contains(r.Body.String(), "<script>unsafe</script>") || !strings.Contains(r.Body.String(), "&lt;script&gt;") {
		t.Fatal("directory missing or unsafe", r.Code)
	}
	first := r.Body.String()
	if out := os.Getenv("DIRECTORY_HTML_OUT"); out != "" {
		for name, data := range map[string][]byte{"directory.html": r.Body.Bytes(), "photo.jpg": img.Bytes(), "work.html": call(a, "GET", "/work/"+a.state.Board+"?entry=skip", nil).Body.Bytes()} {
			if err := os.WriteFile(filepath.Join(out, name), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	if r := call(a, "GET", "/client-directory/"+token(), nil); r.Code != 404 {
		t.Fatal("invalid board allowed")
	}
	photoPath := "/client-photo/" + a.state.Board + "/" + c.ID + "/" + id
	if r := call(a, "GET", photoPath, nil); r.Code != 200 || !bytes.Equal(r.Body.Bytes(), img.Bytes()) {
		t.Fatal("master photo unavailable")
	}
	if r := call(a, "POST", path, url.Values{"text": {"change"}}); r.Code != 405 {
		t.Fatal("worker directory writable")
	}
	a.transaction(func(s *State) error { s.ClientNotes[0].Text = "수정됨"; s.ClientNotes[0].Revision++; return nil })
	r = call(a, "GET", path, nil)
	if r.Body.String() == first || !strings.Contains(r.Body.String(), "수정됨") {
		t.Fatal("directory not updated")
	}
	if out := os.Getenv("DIRECTORY_HTML_OUT"); out != "" {
		if err := os.WriteFile(filepath.Join(out, "directory-updated.html"), r.Body.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if r := call(a, "GET", "/work/"+a.state.Board, nil); !strings.Contains(r.Body.String(), "worker-client-directory") || !strings.Contains(r.Body.String(), "entry-motion.js") {
		t.Fatal("directory entry or loading effect absent")
	}
}
func TestWorkbookMergePreservesManualDataAndUpdatesOnlyChangedFields(t *testing.T) {
	a := fixture(t)
	p := a.state.Projects[0]
	a.requestInspection(p.ID, "combined", "기존 작업자", p.SettingForm, p.CombinedForm)
	a.approve(p.ID, "combined", a.state.Projects[0].Setting.EventID)
	s, _ := clone(a.state)
	events := completedEvents(s)
	base, e := workbook(s)
	if e != nil {
		t.Fatal(e)
	}
	book, e := excelize.OpenReader(bytes.NewReader(base))
	if e != nil {
		t.Fatal(e)
	}
	sheet := book.GetSheetList()[0]
	book.SetCellStr(sheet, "D2", "엑셀에서 직접 수정한 이름")
	book.SetCellStr(sheet, "G2", "별도 메모 보존")
	book.SetCellFormula(sheet, "H2", "1+2")
	book.NewSheet("별도 메모")
	book.SetCellStr("별도 메모", "A1", "별도 시트 유지")
	style, _ := book.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "FF0000"}})
	book.SetCellStyle(sheet, "G2", "G2", style)
	buf, _ := book.WriteToBuffer()
	book.Close()
	input := append([]byte{}, buf.Bytes()...)
	bindings, e := bindExistingWorkbook(input, s)
	if e != nil || len(bindings) != 2 {
		t.Fatal("binding failed", e)
	}
	s.WorkbookBase = input
	s.WorkbookBindings = bindings
	merged, e := workbook(s)
	if e != nil {
		t.Fatal(e)
	}
	out, _ := excelize.OpenReader(bytes.NewReader(merged))
	if v, _ := out.GetCellValue(sheet, "D2"); v != "엑셀에서 직접 수정한 이름" {
		t.Fatal("manual value overwritten without server change")
	}
	out.Close()
	for i := range s.Events {
		if s.Events[i].ID == events[0].ID {
			s.Events[i].Worker = "프로그램 변경 이름"
		}
	}
	merged, e = workbook(s)
	if e != nil {
		t.Fatal(e)
	}
	out, _ = excelize.OpenReader(bytes.NewReader(merged))
	defer out.Close()
	for cell, want := range map[string]string{"D2": "프로그램 변경 이름", "G2": "별도 메모 보존"} {
		if v, _ := out.GetCellValue(sheet, cell); v != want {
			t.Fatal(cell, v)
		}
	}
	if v, _ := out.GetCellFormula(sheet, "H2"); v != "1+2" {
		t.Fatal("formula lost")
	}
	if v, _ := out.GetCellValue("별도 메모", "A1"); v != "별도 시트 유지" {
		t.Fatal("extra sheet lost")
	}
	if st, _ := out.GetCellStyle(sheet, "G2"); st != style {
		t.Fatal("manual style lost")
	}
	if !bytes.Equal(s.WorkbookBase, input) {
		t.Fatal("original uploaded workbook mutated")
	}
	// Removing an owned record clears its six fields while retaining the user's extra columns.
	remaining := []Event{}
	for _, event := range s.Events {
		if event.ID != events[0].ID {
			remaining = append(remaining, event)
		}
	}
	s.Events = remaining
	merged, e = workbook(s)
	if e != nil {
		t.Fatal(e)
	}
	cancelled, _ := excelize.OpenReader(bytes.NewReader(merged))
	defer cancelled.Close()
	if v, _ := cancelled.GetCellValue(sheet, "A2"); v != "" {
		t.Fatal("cancelled record not cleared")
	}
	if v, _ := cancelled.GetCellValue(sheet, "G2"); v != "별도 메모 보존" {
		t.Fatal("cancel erased manual memo")
	}
}
func TestWorkbookAttachmentDurableAndNewEventsNotDuplicated(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	base := excelize.NewFile()
	headers := []any{"수주번호", "프로젝트명", "검사구분", "작업자", "완료일자", "완료시간"}
	base.SetSheetRow("Sheet1", "A1", &headers)
	base.SetCellStr("Sheet1", "G2", "수동 메모")
	buf, _ := base.WriteToBuffer()
	base.Close()
	r := postWorkbook(a, "/report-settings", "workbook", buf.Bytes(), cookie, map[string]string{"csrf": "admin-csrf", "action": "attach"})
	if r.Code != 303 {
		t.Fatal(r.Code, r.Body.String())
	}
	p := a.state.Projects[0]
	a.requestInspection(p.ID, "combined", "신규", p.SettingForm, p.CombinedForm)
	a.approve(p.ID, "combined", a.state.Projects[0].Setting.EventID)
	for i := 0; i < 2; i++ {
		data, e := workbook(a.state)
		if e != nil {
			t.Fatal(e)
		}
		b, _ := excelize.OpenReader(bytes.NewReader(data))
		rows, _ := b.GetRows(day(time.Now())[:7])
		b.Close()
		if len(rows) != 3 {
			t.Fatal("new events duplicated or missing", len(rows))
		}
	}
	next, e := newApp(a.dir)
	if e != nil || !bytes.Equal(next.state.WorkbookBase, buf.Bytes()) {
		t.Fatal("base not durable", e)
	}
	if _, e = os.Stat(filepath.Join(a.dir, "검사완료기록-복구서버.xlsx")); e != nil {
		t.Fatal(e)
	}
	r = call(a, "GET", "/report-settings", nil, cookie)
	if r.Code != 200 {
		t.Fatal("settings missing")
	}
	bad := postWorkbook(a, "/report-settings", "workbook", []byte("bad"), cookie, map[string]string{"csrf": "admin-csrf", "action": "attach"})
	if bad.Code != 409 || !bytes.Equal(a.state.WorkbookBase, buf.Bytes()) {
		t.Fatal("failed attachment changed state")
	}
	r = call(a, "GET", "/report.xlsx", nil, cookie)
	if _, e = io.ReadAll(r.Result().Body); e != nil || r.Code != 200 {
		t.Fatal("export failed")
	}
}
