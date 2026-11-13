package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"github.com/xuri/excelize/v2"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"
)

var materialHex = regexp.MustCompile(`^[0-9A-F]{6}$`)

func materialColorHex(s string) string {
	s = strings.ToUpper(strings.TrimPrefix(s, "#"))
	if len(s) == 8 {
		s = s[2:]
	}
	if materialHex.MatchString(s) {
		return "#" + s
	}
	return ""
}
func materialFillColor(style *excelize.Style) string {
	if style != nil && style.Fill.Type == "pattern" && style.Fill.Pattern == 1 && len(style.Fill.Color) > 0 {
		return materialColorHex(style.Fill.Color[0])
	}
	return ""
}

type materialColorRule struct {
	region, kind, operator, formula string
	format, priority                int
	stop                            bool
}
type materialColorReader struct {
	book              *excelize.File
	rules             []materialColorRule
	base, conditional map[int]string
}

func materialXMLPart(z *zip.Reader, name string) ([]byte, error) {
	for _, f := range z.File {
		if f.Name == name {
			r, e := f.Open()
			if e != nil {
				return nil, e
			}
			defer r.Close()
			b, e := io.ReadAll(io.LimitReader(r, (32<<20)+1))
			if len(b) > 32<<20 {
				return nil, errors.New("일정표 XML 용량 초과")
			}
			return b, e
		}
	}
	return nil, errors.New("일정표 내부 파일을 찾을 수 없습니다")
}
func newMaterialColorReader(book *excelize.File, data []byte) (*materialColorReader, error) {
	z, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		return nil, e
	}
	wb, e := materialXMLPart(z, "xl/workbook.xml")
	if e != nil {
		return nil, e
	}
	var names struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
			ID   string `xml:"id,attr"`
		} `xml:"sheets>sheet"`
	}
	if e = xml.Unmarshal(wb, &names); e != nil {
		return nil, e
	}
	id := ""
	for _, s := range names.Sheets {
		if s.Name == "조립일정" {
			id = s.ID
		}
	}
	rels, e := materialXMLPart(z, "xl/_rels/workbook.xml.rels")
	if e != nil {
		return nil, e
	}
	var relationships struct {
		Items []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	if e = xml.Unmarshal(rels, &relationships); e != nil {
		return nil, e
	}
	target := ""
	for _, r := range relationships.Items {
		if r.ID == id {
			target = r.Target
		}
	}
	if strings.HasPrefix(target, "/") {
		target = strings.TrimPrefix(target, "/")
	} else {
		target = path.Join("xl", target)
	}
	target = path.Clean(target)
	if !strings.HasPrefix(target, "xl/") {
		return nil, errors.New("일정표 시트 경로를 확인하세요")
	}
	sheet, e := materialXMLPart(z, target)
	if e != nil {
		return nil, e
	}
	var ws struct {
		Formats []struct {
			Range string `xml:"sqref,attr"`
			Rules []struct {
				Kind     string   `xml:"type,attr"`
				Operator string   `xml:"operator,attr"`
				Format   *int     `xml:"dxfId,attr"`
				Priority int      `xml:"priority,attr"`
				Stop     bool     `xml:"stopIfTrue,attr"`
				Formula  []string `xml:"formula"`
			} `xml:"cfRule"`
		} `xml:"conditionalFormatting"`
	}
	if e = xml.Unmarshal(sheet, &ws); e != nil {
		return nil, e
	}
	reader := &materialColorReader{book: book, base: map[int]string{}, conditional: map[int]string{}}
	for _, f := range ws.Formats {
		for _, r := range f.Rules {
			if r.Format != nil && len(r.Formula) == 1 {
				reader.rules = append(reader.rules, materialColorRule{f.Range, r.Kind, r.Operator, r.Formula[0], *r.Format, r.Priority, r.Stop})
			}
		}
	}
	sort.SliceStable(reader.rules, func(i, j int) bool { return reader.rules[i].priority < reader.rules[j].priority })
	return reader, nil
}
func materialRangeContains(region, cell string) bool {
	col, row, e := excelize.CellNameToCoordinates(strings.ReplaceAll(cell, "$", ""))
	if e != nil {
		return false
	}
	for _, r := range strings.Fields(region) {
		parts := strings.Split(r, ":")
		start, end := parts[0], parts[0]
		if len(parts) == 2 {
			end = parts[1]
		}
		sc, sr, e1 := excelize.CellNameToCoordinates(strings.ReplaceAll(start, "$", ""))
		ec, er, e2 := excelize.CellNameToCoordinates(strings.ReplaceAll(end, "$", ""))
		if e1 == nil && e2 == nil && col >= sc && col <= ec && row >= sr && row <= er {
			return true
		}
	}
	return false
}
func (r *materialColorReader) color(cell, value string) (string, error) {
	id, e := r.book.GetCellStyle("조립일정", cell)
	if e != nil {
		return "", e
	}
	base, ok := r.base[id]
	if !ok {
		style, e := r.book.GetStyle(id)
		if e != nil {
			return "", e
		}
		base = materialFillColor(style)
		if base == "" {
			base = "#FFFFFF"
		}
		r.base[id] = base
	}
	for _, rule := range r.rules {
		if !materialRangeContains(rule.region, cell) || rule.kind != "cellIs" || rule.operator != "equal" {
			continue
		}
		formula := strings.TrimSpace(rule.formula)
		if len(formula) < 2 || formula[0] != '"' || formula[len(formula)-1] != '"' {
			continue
		}
		literal := strings.ReplaceAll(formula[1:len(formula)-1], `""`, `"`)
		if !strings.EqualFold(value, literal) {
			continue
		}
		color, ok := r.conditional[rule.format]
		if !ok {
			style, e := r.book.GetConditionalStyle(rule.format)
			if e != nil {
				return "", e
			}
			color = materialFillColor(style)
			r.conditional[rule.format] = color
		}
		if color != "" {
			return color, nil
		}
		if rule.stop {
			break
		}
	}
	return base, nil
}
