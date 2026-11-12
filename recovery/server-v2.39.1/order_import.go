package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/unicode/norm"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type ImportRow struct {
	Row                                                 int
	Number, Project, DesignerText, ClientText, Memo     string
	DesignerID, ClientID, MatchNote, ClientNote, Status string
	Unavailable, Selected                               bool
}
type ImportResult struct{ Added, Skipped int }
type ImportDraft struct {
	Token, Nonce, Owner, Filename, Sheet, ParseError string
	Created                                          time.Time
	Data                                             []byte
	Sheets                                           []string
	Rows                                             []ImportRow
	People                                           []Designer
	Notes                                            []ClientNote
	Result                                           *ImportResult
	Running                                          bool
	Progress, Total                                  int
}

func (d *ImportDraft) DesignerLabel(id string) string {
	if id == "-pick" {
		return "선택 필요"
	}
	if id == "" {
		return "담당자 없음"
	}
	for _, p := range d.People {
		if p.ID == id {
			return p.Name + " · " + p.Phone
		}
	}
	return "삭제된 담당자 · 다시 선택하세요"
}
func (d *ImportDraft) ClientLabel(id string) string {
	if id == "-pick" {
		return "선택 필요"
	}
	if id == "" {
		return "업체 특이사항 없음"
	}
	for _, p := range d.Notes {
		if p.ID == id {
			return p.Name
		}
	}
	return "삭제된 업체 · 다시 선택하세요"
}
func (d *ImportDraft) ClientFor(id string) *ClientNote {
	for i := range d.Notes {
		if d.Notes[i].ID == id {
			return &d.Notes[i]
		}
	}
	return nil
}
func (d *ImportDraft) SelectedCount() int {
	n := 0
	for _, r := range d.Rows {
		if r.Selected && !r.Unavailable {
			n++
		}
	}
	return n
}
func importNormal(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || strings.ContainsRune("()（）", r) {
			return -1
		}
		return unicode.ToLower(r)
	}, norm.NFKC.String(strings.TrimSpace(s)))
}

var designerTitles = []string{"대표이사", "부회장", "본부장", "매니저", "대표", "회장", "사장", "전무", "상무", "이사", "실장", "팀장", "부장", "차장", "과장", "대리", "계장", "주임", "사원", "책임", "선임", "수석"}

func designerBase(s string) string {
	s = importNormal(s)
	for _, title := range designerTitles {
		if strings.HasSuffix(s, title) && utf8.RuneCountInString(strings.TrimSuffix(s, title)) >= 2 {
			return strings.TrimSuffix(s, title)
		}
	}
	return s
}
func matchDesigner(input string, people []Designer) (string, string) {
	if strings.TrimSpace(input) == "" {
		return "", ""
	}
	exact, base := []string{}, []string{}
	for _, p := range people {
		if importNormal(p.Name) == importNormal(input) {
			exact = append(exact, p.ID)
		}
		if designerBase(p.Name) == designerBase(input) {
			base = append(base, p.ID)
		}
	}
	if len(exact) == 1 {
		return exact[0], ""
	}
	if len(exact) == 0 && len(base) == 1 {
		return base[0], ""
	}
	if len(exact) > 1 || len(base) > 1 {
		return "-pick", "같은 이름의 담당자가 여러 명입니다"
	}
	return "-pick", "일치하는 담당자가 없습니다"
}
func matchClient(input string, notes []ClientNote) (string, string) {
	if strings.TrimSpace(input) == "" {
		return "", ""
	}
	matches := []string{}
	for _, p := range notes {
		if importNormal(p.Name) == importNormal(input) {
			matches = append(matches, p.ID)
		}
	}
	if len(matches) == 1 {
		return matches[0], ""
	}
	if len(matches) > 1 {
		return "-pick", "같은 업체명이 여러 개입니다"
	}
	return "-pick", "등록된 업체가 없습니다"
}
func openImportBook(data []byte) (*excelize.File, error) {
	return excelize.OpenReader(bytes.NewReader(data), excelize.Options{UnzipSizeLimit: 64 << 20, UnzipXMLSizeLimit: 8 << 20})
}
func importHeader(s string) string {
	switch importNormal(s) {
	case "수주번호", "수주no", "erp수주no":
		return "number"
	case "프로젝트명", "프로젝트", "수주명":
		return "project"
	case "설계담당자", "설계담당", "담당자", "설계담당자명", "설계자":
		return "designer"
	case "특이사항업체", "업체특이사항", "특이사항업체명", "업체명", "특이사항":
		return "client"
	case "수주별추가메모", "추가메모", "비고", "수주별메모", "수주별특이사항":
		return "memo"
	}
	return ""
}
func parseImportRows(data []byte, sheet string, s State) ([]ImportRow, error) {
	book, e := openImportBook(data)
	if e != nil {
		return nil, errors.New("엑셀을 읽지 못했습니다. 암호가 없는 .xlsx 파일인지 확인하세요")
	}
	defer book.Close()
	rows, e := book.GetRows(sheet)
	if e != nil {
		return nil, e
	}
	if len(rows) > 20000 {
		return nil, errors.New("읽을 시트가 너무 큽니다. 등록할 행만 새 시트에 넣어 주세요")
	}
	head := -1
	cols := map[string]int{}
	for i, row := range rows {
		if i >= 30 {
			break
		}
		found := map[string]int{}
		duplicate := false
		for j, text := range row {
			key := importHeader(text)
			if key != "" {
				if _, ok := found[key]; ok {
					duplicate = true
				}
				found[key] = j
			}
		}
		_, n := found["number"]
		_, p := found["project"]
		if n && p {
			if duplicate {
				return nil, errors.New("동일한 제목 열이 중복되어 있습니다")
			}
			head = i
			cols = found
			break
		}
	}
	if head < 0 {
		return nil, errors.New("제목 행에 수주번호와 프로젝트명이 필요합니다")
	}
	cell := func(row []string, key string) string {
		col, ok := cols[key]
		if !ok || col >= len(row) {
			return ""
		}
		return row[col]
	}
	existing := map[string]bool{}
	for _, p := range s.Projects {
		existing[strings.ToLower(strings.TrimSpace(p.Number))] = true
	}
	seen := map[string]bool{}
	out := []ImportRow{}
	for i, row := range rows[head+1:] {
		n := strings.TrimSpace(cell(row, "number"))
		p := strings.TrimSpace(cell(row, "project"))
		dt, ct, memo := strings.TrimSpace(cell(row, "designer")), strings.TrimSpace(cell(row, "client")), cell(row, "memo")
		if n == "" && p == "" && dt == "" && ct == "" && strings.TrimSpace(memo) == "" {
			continue
		}
		if len(out) >= 1000 {
			return nil, errors.New("한 시트에서 최대 1,000건까지 등록할 수 있습니다")
		}
		r := ImportRow{Row: head + i + 2, Number: n, Project: p, DesignerText: dt, ClientText: ct, Memo: memo, Selected: true, Status: "등록 가능"}
		r.DesignerID, r.MatchNote = matchDesigner(dt, s.Designers)
		r.ClientID, r.ClientNote = matchClient(ct, s.ClientNotes)
		problem := ""
		if !validText(n, 80) || !validText(p, 120) {
			problem = "수주번호/프로젝트명 누락 또는 길이 초과"
		}
		if utf8.RuneCountInString(memo) > 5000 {
			problem = "추가 메모는 5,000자 이내여야 합니다"
		}
		for _, col := range cols {
			axis, _ := excelize.CoordinatesToCellName(col+1, r.Row)
			formula, err := book.GetCellFormula(sheet, axis)
			if err != nil {
				return nil, err
			}
			if formula != "" {
				problem = "수식 대신 계산된 값을 붙여넣어 주세요"
			}
		}
		key := strings.ToLower(n)
		if existing[key] {
			problem = "이미 등록된 수주 · 건너뜀"
		} else if seen[key] {
			problem = "엑셀 내 중복 수주 · 건너뜀"
		}
		if problem != "" {
			r.Unavailable = true
			r.Selected = false
			r.Status = problem
		} else {
			seen[key] = true
			if r.DesignerID == "-pick" || r.ClientID == "-pick" {
				r.Status = "연결 선택 필요"
			}
		}
		out = append(out, r)
	}
	return out, nil
}
func (a *App) pruneImports() {
	if a.imports == nil {
		a.imports = map[string]*ImportDraft{}
	}
	for k, d := range a.imports {
		if time.Since(d.Created) > 30*time.Minute {
			delete(a.imports, k)
		}
	}
}
func (a *App) importDraft(r *http.Request, s Session) (*ImportDraft, error) {
	a.pruneImports()
	id := r.FormValue("draft")
	if id == "" {
		id = r.URL.Query().Get("draft")
	}
	d := a.imports[id]
	if d == nil || !equal(d.Owner, s.CSRF) {
		return nil, errors.New("미리보기가 만료됐거나 현재 로그인에서 만든 자료가 아닙니다")
	}
	return d, nil
}
func (a *App) renderImport(w http.ResponseWriter, s Session, d *ImportDraft, message string) {
	if d != nil {
		d.People = append([]Designer{}, a.state.Designers...)
		d.Notes = append([]ClientNote{}, a.state.ClientNotes...)
	}
	a.render(w, "order-import.html", map[string]any{"CSRF": s.CSRF, "Draft": d, "Error": message, "People": a.state.Designers, "Notes": a.state.ClientNotes})
}
func importRedirect(w http.ResponseWriter, r *http.Request, d *ImportDraft) {
	http.Redirect(w, r, "/orders/import?draft="+d.Token, 303)
}
func (a *App) importGET(w http.ResponseWriter, r *http.Request, s Session) bool {
	switch r.URL.Path {
	case "/orders/import":
		if r.URL.Query().Get("draft") == "" {
			a.renderImport(w, s, nil, "")
			return true
		}
		d, e := a.importDraft(r, s)
		if e != nil {
			http.Error(w, e.Error(), 404)
			return true
		}
		a.renderImport(w, s, d, "")
		return true
	case "/orders/import/template.xlsx":
		book := excelize.NewFile()
		defer book.Close()
		book.SetSheetName("Sheet1", "수주등록")
		headers := []any{"수주번호", "프로젝트명", "설계담당자", "특이사항 업체", "수주별 추가 메모"}
		book.SetSheetRow("수주등록", "A1", &headers)
		book.SetColWidth("수주등록", "A", "E", 24)
		data, e := book.WriteToBuffer()
		if e != nil {
			http.Error(w, "양식 생성 실패", 500)
		} else {
			w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			w.Header().Set("Content-Disposition", `attachment; filename="order-import-template.xlsx"`)
			w.Write(data.Bytes())
		}
		return true
	}
	return false
}
func (a *App) importPOST(w http.ResponseWriter, r *http.Request, s Session) bool {
	if !strings.HasPrefix(r.URL.Path, "/orders/import/") {
		return false
	}
	if r.URL.Path == "/orders/import/preview" {
		a.pruneImports()
		if len(a.imports) >= 8 {
			var oldest *ImportDraft
			for _, candidate := range a.imports {
				if candidate.Result != nil && (oldest == nil || candidate.Created.Before(oldest.Created)) {
					oldest = candidate
				}
			}
			if oldest != nil {
				delete(a.imports, oldest.Token)
			}
		}
		if len(a.imports) >= 8 {
			http.Error(w, "열린 미리보기가 많습니다. 기존 미리보기를 닫아 주세요", 429)
			return true
		}
		file, header, e := r.FormFile("workbook")
		if e != nil {
			a.renderImport(w, s, nil, "등록할 .xlsx 파일을 선택하세요")
			return true
		}
		defer file.Close()
		if strings.ToLower(filepath.Ext(header.Filename)) != ".xlsx" {
			a.renderImport(w, s, nil, ".xlsx 파일을 선택하세요")
			return true
		}
		data, e := io.ReadAll(io.LimitReader(file, (10<<20)+1))
		if e != nil || len(data) > 10<<20 {
			a.renderImport(w, s, nil, "파일은 10MB 이하여야 합니다")
			return true
		}
		book, e := openImportBook(data)
		if e != nil {
			a.renderImport(w, s, nil, "엑셀을 열 수 없습니다. 암호 없는 .xlsx 파일인지 확인하세요")
			return true
		}
		sheets := book.GetSheetList()
		book.Close()
		if len(sheets) == 0 {
			a.renderImport(w, s, nil, "읽을 시트가 없습니다")
			return true
		}
		d := &ImportDraft{Token: token(), Nonce: token(), Owner: s.CSRF, Filename: filepath.Base(header.Filename), Data: data, Sheets: sheets, Sheet: sheets[0], Created: time.Now()}
		d.Rows, e = parseImportRows(data, d.Sheet, a.state)
		if e != nil {
			d.ParseError = e.Error()
		}
		a.imports[d.Token] = d
		importRedirect(w, r, d)
		return true
	}
	d, e := a.importDraft(r, s)
	if e != nil {
		http.Error(w, e.Error(), 409)
		return true
	}
	if d.Result != nil {
		importRedirect(w, r, d)
		return true
	}
	switch r.URL.Path {
	case "/orders/import/discard":
		delete(a.imports, d.Token)
		http.Redirect(w, r, "/orders/import", 303)
	case "/orders/import/sheet", "/orders/import/refresh":
		sheet := d.Sheet
		if r.URL.Path == "/orders/import/sheet" {
			sheet = r.FormValue("sheet")
		}
		valid := false
		for _, name := range d.Sheets {
			if name == sheet {
				valid = true
			}
		}
		if !valid {
			a.renderImport(w, s, d, "시트를 다시 선택하세요")
			return true
		}
		d.Sheet = sheet
		d.Nonce = token()
		d.Rows, e = parseImportRows(d.Data, sheet, a.state)
		d.ParseError = ""
		if e != nil {
			d.ParseError = e.Error()
		}
		importRedirect(w, r, d)
	case "/orders/import/commit":
		if !equal(d.Nonce, r.FormValue("nonce")) {
			http.Error(w, "이전 미리보기입니다. 새로고침 후 다시 확인하세요", 409)
			return true
		}
		chosen := map[int]bool{}
		for _, v := range r.Form["rows"] {
			n, err := strconv.Atoi(v)
			if err != nil {
				http.Error(w, "선택 행이 올바르지 않습니다", 400)
				return true
			}
			chosen[n] = true
		}
		known := map[int]bool{}
		selected := []ImportRow{}
		for i := range d.Rows {
			row := &d.Rows[i]
			known[row.Row] = true
			row.Selected = chosen[row.Row] && !row.Unavailable
			if !row.Selected {
				continue
			}
			row.DesignerID = r.FormValue(fmt.Sprintf("designer_%d", row.Row))
			row.ClientID = r.FormValue(fmt.Sprintf("client_%d", row.Row))
			selected = append(selected, *row)
		}
		for n := range chosen {
			if !known[n] {
				http.Error(w, "미리보기에 없는 행입니다", 400)
				return true
			}
		}
		if len(selected) == 0 {
			a.renderImport(w, s, d, "등록할 행을 선택하세요")
			return true
		}
		result := &ImportResult{}
		for _, row := range d.Rows {
			if row.Unavailable && (strings.Contains(row.Status, "중복") || strings.Contains(row.Status, "이미 등록")) {
				result.Skipped++
			}
		}
		e = a.transaction(func(state *State) error {
			existing := map[string]bool{}
			for _, p := range state.Projects {
				existing[strings.ToLower(p.Number)] = true
			}
			for _, row := range selected {
				if row.DesignerID == "-pick" || row.ClientID == "-pick" {
					return fmt.Errorf("%d행의 담당자/업체 연결을 선택하거나 없음을 선택하세요", row.Row)
				}
				var person *Designer
				var client *ClientNote
				if row.DesignerID != "" {
					for i := range state.Designers {
						if state.Designers[i].ID == row.DesignerID {
							person = &state.Designers[i]
						}
					}
					if person == nil {
						return fmt.Errorf("%d행 담당자가 삭제됐습니다. 명단을 새로고침하세요", row.Row)
					}
				}
				if row.ClientID != "" {
					for i := range state.ClientNotes {
						if state.ClientNotes[i].ID == row.ClientID {
							client = &state.ClientNotes[i]
						}
					}
					if client == nil {
						return fmt.Errorf("%d행 업체가 삭제됐습니다. 명단을 새로고침하세요", row.Row)
					}
				}
				key := strings.ToLower(row.Number)
				if existing[key] {
					result.Skipped++
					continue
				}
				p := Project{ID: state.next(), Number: row.Number, Name: row.Project, Revision: 1, InfoRevision: 1, AdditionalInstructions: row.Memo}
				p.rotate()
				if person != nil {
					p.DesignerID = person.ID
					p.DesignerName = person.Name
					p.DesignerPhone = person.Phone
				}
				if client != nil {
					p.ClientID = client.ID
					p.SpecialInstructions = client.Text
				}
				state.Projects = append(state.Projects, p)
				existing[key] = true
				result.Added++
			}
			return nil
		})
		if e != nil {
			a.renderImport(w, s, d, e.Error())
			return true
		}
		d.Result = result
		d.Data = nil
		d.Nonce = token()
		importRedirect(w, r, d)
	default:
		http.NotFound(w, r)
	}
	return true
}

// Stable order is used by worker directory rendering too.
func sortedClientNotes(notes []ClientNote) []ClientNote {
	out := append([]ClientNote{}, notes...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
