package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"time"
)

func completedEvents(s State) []Event {
	events := []Event{}
	for _, e := range s.Events {
		if !e.IsNote && !e.Pending && !e.Rejected && e.Completed != nil {
			events = append(events, e)
		}
	}
	sort.SliceStable(events, func(i, j int) bool {
		a, b := events[i], events[j]
		ra, rb := eventRow(s, a), eventRow(s, b)
		ma, mb := a.Completed.In(kst).Format("2006-01"), b.Completed.In(kst).Format("2006-01")
		if ma != mb {
			return ma < mb
		}
		if ra[0] != rb[0] {
			return ra[0] < rb[0]
		}
		if a.Inspection != b.Inspection {
			return a.Inspection == "setting"
		}
		return a.Completed.Before(*b.Completed)
	})
	return events
}
func eventRow(s State, e Event) []string {
	order, name := e.Order, e.Project
	if p := s.project(e.ProjectID); p != nil {
		order = p.Number
		name = p.Name
	}
	d, t := "", ""
	if e.Completed != nil {
		d = e.Completed.In(kst).Format("2006-01-02")
		t = e.Completed.In(kst).Format("15:04:05")
	}
	return []string{order, name, stageLabel(e.Inspection), e.Worker, d, t}
}
func esc(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		if r == '\t' || r == '\n' || r == '\r' || r >= 32 && r <= 0xd7ff || r >= 0xe000 && r <= 0xfffd || r >= 0x10000 && r <= 0x10ffff {
			xml.EscapeText(&b, []byte(string(r)))
		}
	}
	return b.String()
}
func workbook(s State) ([]byte, error) {
	if len(s.WorkbookBase) > 0 {
		return mergedWorkbook(s)
	}
	months := map[string][]Event{}
	for _, e := range completedEvents(s) {
		m := e.Completed.In(kst).Format("2006-01")
		months[m] = append(months[m], e)
	}
	if len(months) == 0 {
		months[day(time.Now())[:7]] = []Event{}
	}
	names := []string{}
	for m := range months {
		names = append(names, m)
	}
	sort.Strings(names)
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	put := func(name, body string) error {
		f, e := z.Create(name)
		if e != nil {
			return e
		}
		_, e = f.Write([]byte(body))
		return e
	}
	content := `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>`
	wb := `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`
	rels := `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="styles" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`
	for i, m := range names {
		id := i + 1
		wb += fmt.Sprintf(`<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, m, id, id)
		rels += fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, id, id)
		content += fmt.Sprintf(`<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, id)
		var sh strings.Builder
		sh.WriteString(`<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews><cols><col min="1" max="1" width="22" customWidth="1"/><col min="2" max="2" width="40" customWidth="1"/><col min="3" max="6" width="20" customWidth="1"/></cols><sheetData>`)
		row := func(n, style int, values []string) {
			fmt.Fprintf(&sh, `<row r="%d">`, n)
			for j, v := range values {
				fmt.Fprintf(&sh, `<c r="%c%d" s="%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, 'A'+j, n, style, esc(v))
			}
			sh.WriteString(`</row>`)
		}
		row(1, 1, []string{"수주번호", "프로젝트명", "검사구분", "작업자", "완료일자", "완료시간"})
		style := 2
		last := ""
		for j, e := range months[m] {
			values := eventRow(s, e)
			if last != "" && last != values[0] {
				if style == 2 {
					style = 3
				} else {
					style = 2
				}
			}
			last = values[0]
			row(j+2, style, values)
		}
		sh.WriteString(`</sheetData></worksheet>`)
		if e := put(fmt.Sprintf("xl/worksheets/sheet%d.xml", id), sh.String()); e != nil {
			return nil, e
		}
	}
	files := map[string]string{
		"[Content_Types].xml": content + `</Types>`, "xl/workbook.xml": wb + `</sheets></workbook>`, "xl/_rels/workbook.xml.rels": rels + `</Relationships>`,
		"_rels/.rels":   `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/styles.xml": `<?xml version="1.0" encoding="UTF-8"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="2"><font><sz val="11"/><name val="맑은 고딕"/></font><font><b/><sz val="11"/><name val="맑은 고딕"/></font></fonts><fills count="5"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill><fill><patternFill patternType="solid"><fgColor rgb="FFDBEAFE"/><bgColor indexed="64"/></patternFill></fill><fill><patternFill patternType="solid"><fgColor rgb="FFF0FDF4"/><bgColor indexed="64"/></patternFill></fill><fill><patternFill patternType="solid"><fgColor rgb="FFFFF8DD"/><bgColor indexed="64"/></patternFill></fill></fills><borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders><cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs><cellXfs count="4"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="1" fillId="2" borderId="0" xfId="0" applyFill="1"/><xf numFmtId="0" fontId="0" fillId="3" borderId="0" xfId="0" applyFill="1"/><xf numFmtId="0" fontId="0" fillId="4" borderId="0" xfId="0" applyFill="1"/></cellXfs><cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles></styleSheet>`,
	}
	for name, body := range files {
		if e := put(name, body); e != nil {
			return nil, e
		}
	}
	if e := z.Close(); e != nil {
		return nil, e
	}
	return buf.Bytes(), nil
}
