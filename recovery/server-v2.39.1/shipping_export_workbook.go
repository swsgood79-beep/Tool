package main

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

type shippingExportLimits struct {
	PhotoBytes, Records int
	Single              bool
}
type exportPhoto struct {
	Data                []byte
	Width, Height       int
	Name, Kind, Problem string
	Number              int
}
type exportEntry struct {
	Record      ShippingRecord
	Book, Sheet string
	Warnings    int
}
type exportWarning struct{ Order, Project, Photo, Message string }
type exportStyles struct{ Title, CompactTitle, Section, Header, Body, Label, Link, Date, Stamp, Muted, Center, SmallCenter int }
type exportBook struct {
	File      *excelize.File
	Style     exportStyles
	Err       error
	Locations map[string]map[string]string
}

func exportPtr[T any](v T) *T { return &v }
func (b *exportBook) check(err error) {
	if b.Err == nil && err != nil {
		b.Err = err
	}
}
func (b *exportBook) str(sheet, cell, text string) { b.check(b.File.SetCellStr(sheet, cell, text)) }
func (b *exportBook) style(sheet, start, end string, id int) {
	b.check(b.File.SetCellStyle(sheet, start, end, id))
}
func (b *exportBook) merge(sheet, start, end, text string, id int) {
	if start != end {
		b.check(b.File.MergeCell(sheet, start, end))
	}
	b.str(sheet, start, text)
	b.style(sheet, start, end, id)
}
func (b *exportBook) height(sheet string, row int, value float64) {
	b.check(b.File.SetRowHeight(sheet, row, value))
}
func (b *exportBook) link(sheet, cell, target, location, label string) {
	b.str(sheet, cell, label)
	kind := "External"
	if target == "" {
		kind = "Location"
		target = location
	}
	b.check(b.File.SetCellHyperLink(sheet, cell, target, kind))
	b.style(sheet, cell, cell, b.Style.Link)
	if kind == "External" && location != "" {
		if b.Locations[sheet] == nil {
			b.Locations[sheet] = map[string]string{}
		}
		b.Locations[sheet][cell] = location
	}
}
func exportLocation(sheet string) string { return "'" + strings.ReplaceAll(sheet, "'", "''") + "'!A1" }
func newExportBook(first string) *exportBook {
	f := excelize.NewFile()
	b := &exportBook{File: f, Locations: map[string]map[string]string{}}
	b.check(f.SetDocProps(&excelize.DocProperties{Creator: "알파작업완료확인", LastModifiedBy: "알파작업완료확인", Title: "출하 기록", Language: "ko-KR"}))
	b.check(f.SetSheetName("Sheet1", first))
	style := func(s *excelize.Style) int { id, err := f.NewStyle(s); b.check(err); return id }
	font := func(bold bool, c string, size float64) *excelize.Font {
		return &excelize.Font{Family: "Malgun Gothic", Size: size, Color: c, Bold: bold}
	}
	border := []excelize.Border{}
	for _, edge := range []string{"left", "right", "top", "bottom"} {
		border = append(border, excelize.Border{Type: edge, Color: "D5DEEB", Style: 1})
	}
	b.Style.Title = style(&excelize.Style{Font: font(true, "FFFFFF", 20), Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"17365D"}}, Alignment: &excelize.Alignment{Vertical: "center", Indent: 1, WrapText: true}})
	b.Style.CompactTitle = style(&excelize.Style{Font: font(true, "FFFFFF", 16), Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"17365D"}}, Alignment: &excelize.Alignment{Vertical: "center", WrapText: true, Indent: 1}})
	b.Style.Center = style(&excelize.Style{Font: font(false, "172B46", 10), Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}, Border: border})
	b.Style.SmallCenter = style(&excelize.Style{Font: font(false, "172B46", 9), Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}, Border: border})
	b.Style.Section = style(&excelize.Style{Font: font(true, "FFFFFF", 11), Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"24466E"}}, Alignment: &excelize.Alignment{Vertical: "center", Indent: 1}})
	b.Style.Header = style(&excelize.Style{Font: font(true, "FFFFFF", 10), Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"17365D"}}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}, Border: border})
	b.Style.Body = style(&excelize.Style{Font: font(false, "172B46", 10), Alignment: &excelize.Alignment{Vertical: "center", WrapText: true}, Border: border})
	b.Style.Label = style(&excelize.Style{Font: font(true, "24466E", 10), Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E8EFF8"}}, Alignment: &excelize.Alignment{Vertical: "center", WrapText: true}, Border: border})
	b.Style.Link = style(&excelize.Style{Font: &excelize.Font{Family: "Malgun Gothic", Size: 10, Color: "155DCC", Underline: "single"}, Alignment: &excelize.Alignment{Vertical: "center"}, Border: border})
	date := "yyyy.mm.dd"
	stamp := "yyyy.mm.dd hh:mm"
	b.Style.Date = style(&excelize.Style{Font: font(false, "172B46", 10), CustomNumFmt: &date, Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"}, Border: border})
	b.Style.Stamp = style(&excelize.Style{Font: font(false, "172B46", 10), CustomNumFmt: &stamp, Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"}, Border: border})
	b.Style.Muted = style(&excelize.Style{Font: font(false, "64748B", 10), Alignment: &excelize.Alignment{Vertical: "center", WrapText: true}})
	return b
}
func (b *exportBook) sheet(name string, detail bool) {
	if name != b.File.GetSheetName(0) {
		_, err := b.File.NewSheet(name)
		b.check(err)
	}
	widths := []float64{19, 32, 18, 17, 24, 10, 12, 18}
	if detail {
		widths = []float64{13, 13, 13, 13, 13, 13, 13, 13}
	}
	for i, w := range widths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		b.check(b.File.SetColWidth(name, col, col, w))
	}
	b.check(b.File.SetSheetView(name, 0, &excelize.ViewOptions{ShowGridLines: exportPtr(false), ZoomScale: exportPtr(85.0)}))
	orientation := "portrait"
	if !detail {
		orientation = "landscape"
	}
	b.check(b.File.SetSheetProps(name, &excelize.SheetPropsOptions{FitToPage: exportPtr(true)}))
	b.check(b.File.SetPageMargins(name, &excelize.PageLayoutMarginsOptions{Left: exportPtr(.25), Right: exportPtr(.25), Top: exportPtr(.4), Bottom: exportPtr(.4), Header: exportPtr(.15), Footer: exportPtr(.15)}))
	b.check(b.File.SetPageLayout(name, &excelize.PageLayoutOptions{Size: exportPtr(9), Orientation: &orientation, FitToWidth: exportPtr(1), FitToHeight: exportPtr(0)}))
}
func (b *exportBook) title(sheet, title, note string) {
	b.merge(sheet, "A1", "H2", title, b.Style.Title)
	height := float64(max(43, ((utf8.RuneCountInString(title)+34)/35)*24+8))
	b.height(sheet, 1, height/2)
	b.height(sheet, 2, height/2)
	b.merge(sheet, "A3", "H3", note, b.Style.Muted)
	b.height(sheet, 3, 32)
}
func excelWallTime(t time.Time) time.Time {
	v := t.In(kst)
	return time.Date(v.Year(), v.Month(), v.Day(), v.Hour(), v.Minute(), v.Second(), 0, time.UTC)
}
func (b *exportBook) date(sheet, cell string, t *time.Time, stamp bool) {
	if t == nil {
		b.str(sheet, cell, "—")
		return
	}
	b.check(b.File.SetCellValue(sheet, cell, excelWallTime(*t)))
	id := b.Style.Date
	if stamp {
		id = b.Style.Stamp
	}
	b.style(sheet, cell, cell, id)
}
func exportPhotoPreview(raw []byte) ([]byte, int, int, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > 40000000 {
		return nil, 0, 0, errors.New("사진 크기 오류")
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, 0, err
	}
	scale := math.Min(1, 960/float64(max(cfg.Width, cfg.Height)))
	w, h := max(1, int(float64(cfg.Width)*scale)), max(1, int(float64(cfg.Height)*scale))
	for {
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
		for _, quality := range []int{82, 68, 52} {
			var data bytes.Buffer
			if err = jpeg.Encode(&data, dst, &jpeg.Options{Quality: quality}); err != nil {
				return nil, 0, 0, err
			}
			if data.Len() <= 220<<10 || max(w, h) <= 320 {
				return data.Bytes(), w, h, nil
			}
		}
		w, h = max(1, int(float64(w)*.8)), max(1, int(float64(h)*.8))
	}
}
func prepareExportPhotos(ctx context.Context, r ShippingRecord, read func(Photo) ([]byte, error)) ([]exportPhoto, []exportWarning) {
	list := []exportPhoto{}
	warnings := []exportWarning{}
	for group, photos := range [][]Photo{r.Photos, r.Project.Photos} {
		kind := "출하사진"
		if group == 1 {
			kind = "특이사항 사진"
		}
		for i, p := range photos {
			ph := exportPhoto{Name: p.Name, Kind: kind, Number: i + 1}
			data, err := read(p)
			if ctx.Err() != nil {
				err = ctx.Err()
			}
			if err == nil {
				ph.Data, ph.Width, ph.Height, err = exportPhotoPreview(data)
			}
			if err != nil {
				ph.Problem = "사진 원본을 읽을 수 없거나 형식이 손상되었습니다."
				warnings = append(warnings, exportWarning{r.Project.Number, r.Project.Name, p.Name, ph.Problem})
			}
			list = append(list, ph)
		}
	}
	return list, warnings
}
func exportSheetName(n int, r ShippingRecord) string {
	name := fmt.Sprintf("%03d_%s_%s", n, r.Project.Name, r.Project.Number)
	name = strings.Map(func(c rune) rune {
		if strings.ContainsRune(":\\/?*[]", c) || c < ' ' {
			return '_'
		}
		return c
	}, name)
	runes := []rune(name)
	if len(runes) > 31 {
		name = string(runes[:31])
	}
	return strings.Trim(name, "'")
}
func (b *exportBook) index(sheet, title string, entries []exportEntry, external bool, now time.Time) {
	b.sheet(sheet, false)
	b.title(sheet, title, "보관 날짜 기준 · 작성 "+now.In(kst).Format("2006.01.02 15:04")+" · 파일·영상 원본은 별도 전달")
	headers := []string{"수주번호", "프로젝트명", "고객사", "보관일", "세팅 / 출하", "사진", "첨부", "상세"}
	for i, text := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 5)
		b.str(sheet, cell, text)
	}
	b.style(sheet, "A5", "H5", b.Style.Header)
	b.height(sheet, 5, 27)
	for i, e := range entries {
		row := i + 6
		r := e.Record
		values := []string{r.Project.Number, r.Project.Name, exportPresent(r.ClientName), "", r.Project.Setting.Worker + " / " + r.Project.Shipping.Worker, fmt.Sprintf("%d장", len(r.Photos)), fmt.Sprintf("%d개", len(r.Attachments)), "상세 보기"}
		for c, text := range values {
			cell, _ := excelize.CoordinatesToCellName(c+1, row)
			b.str(sheet, cell, text)
		}
		b.style(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("H%d", row), b.Style.Body)
		lines := max(utf8.RuneCountInString(r.Project.Name)/16, utf8.RuneCountInString(values[4])/12)
		b.height(sheet, row, float64(max(32, min(132, (lines+1)*16))))
		b.date(sheet, fmt.Sprintf("D%d", row), r.ArchivedAt, false)
		target := ""
		if external {
			target = e.Book
		}
		b.link(sheet, fmt.Sprintf("H%d", row), target, exportLocation(e.Sheet), "상세 보기")
	}
	last := max(6, len(entries)+5)
	if len(entries) > 0 {
		b.check(b.File.AddTable(sheet, &excelize.Table{Range: fmt.Sprintf("A5:H%d", last), Name: "ShipmentIndex", StyleName: "TableStyleMedium2", ShowRowStripes: exportPtr(true)}))
	}
	b.check(b.File.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 5, TopLeftCell: "A6", ActivePane: "bottomLeft"}))
}
func (b *exportBook) section(sheet string, row int, title string) {
	b.merge(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("H%d", row), title, b.Style.Section)
	b.height(sheet, row, 20)
}
func (b *exportBook) text(sheet string, row int, label, text string) int {
	b.section(sheet, row, label)
	row++
	if strings.TrimSpace(text) == "" {
		text = "—"
	}
	for len(text) > 0 {
		runes := []rune(text)
		n := min(400, len(runes))
		part := string(runes[:n])
		text = string(runes[n:])
		b.merge(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("H%d", row), part, b.Style.Body)
		lines := 0
		for _, line := range strings.Split(part, "\n") {
			lines += max(1, (utf8.RuneCountInString(line)+57)/58)
		}
		b.height(sheet, row, float64(min(400, lines*17+12)))
		row++
	}
	return row + 1
}
func (b *exportBook) detail(entry exportEntry, photos []exportPhoto, standalone bool) {
	r, sheet := entry.Record, entry.Sheet
	b.sheet(sheet, true)
	b.check(b.File.SetColWidth(sheet, "A", "L", 9))
	title := r.Project.Name + " · 출하 상세"
	b.merge(sheet, "A1", "I2", title, b.Style.CompactTitle)
	titleHeight := float64(max(34, ((utf8.RuneCountInString(title)+29)/30)*20+8))
	b.height(sheet, 1, titleHeight/2)
	b.height(sheet, 2, titleHeight/2)
	b.merge(sheet, "J1", "L2", "", b.Style.Link)
	if standalone {
		b.link(sheet, "J1", "", exportLocation("목차"), "← 목차로")
	} else {
		b.link(sheet, "J1", "출하목록.xlsx", exportLocation("출하목록"), "← 전체 목록")
	}
	b.height(sheet, 3, 5)
	for row := 4; row <= 8; row++ {
		b.merge(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("F%d", row), "", b.Style.Body)
		b.merge(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), "", b.Style.Label)
		b.height(sheet, row, 19)
	}
	for cell, text := range map[string]string{"A4": "수주번호", "C4": r.Project.Number, "A5": "출고 예정", "A6": "검사 완료", "A7": "보관 완료", "A8": "고객사", "C8": exportPresent(r.ClientName)} {
		b.str(sheet, cell, text)
	}
	if due, err := time.ParseInLocation("2006-01-02", r.Project.ShippingDueDate, kst); err == nil {
		b.date(sheet, "C5", &due, false)
	} else {
		b.str(sheet, "C5", exportPresent(r.Project.ShippingDueDate))
	}
	b.date(sheet, "C6", &r.InspectedAt, true)
	b.date(sheet, "C7", r.ArchivedAt, true)
	for _, x := range []struct {
		From, To, Label, Name string
		Date                  *time.Time
	}{{"G", "H", "설계담당자", r.Project.DesignerName, nil}, {"I", "J", "세팅 작업자", r.Project.Setting.Worker, r.Project.Setting.Completed}, {"K", "L", "출하 작업자", r.Project.Shipping.Worker, r.Project.Shipping.Completed}} {
		b.merge(sheet, x.From+"4", x.To+"4", x.Label, b.Style.Header)
		b.merge(sheet, x.From+"5", x.To+"6", exportDash(x.Name), b.Style.Center)
		text := "—"
		if x.Date != nil {
			text = x.Date.In(kst).Format("2006.01.02\n15:04")
		}
		b.merge(sheet, x.From+"7", x.To+"8", text, b.Style.SmallCenter)
	}
	maxName := max(utf8.RuneCountInString(r.Project.Setting.Worker), utf8.RuneCountInString(r.Project.Shipping.Worker))
	if maxName > 7 {
		b.height(sheet, 5, float64(min(70, ((maxName+6)/7)*12)))
		b.height(sheet, 6, 19)
	}
	b.height(sheet, 9, 5)
	row := 10
	material := strings.TrimSpace(r.Project.SuppliedMaterials)
	heading := "사급품 현황 / 수령내역"
	if material != "" {
		heading += " · " + material
	}
	b.detailSection(sheet, row, heading)
	row++
	if len(r.Receipts) > 0 {
		for _, x := range []struct{ From, To, Text string }{{"A", "C", "종류"}, {"D", "F", "제조사"}, {"G", "J", "품번"}, {"K", "L", "수량"}} {
			b.merge(sheet, fmt.Sprintf("%s%d", x.From, row), fmt.Sprintf("%s%d", x.To, row), x.Text, b.Style.Header)
		}
		b.height(sheet, row, 19)
		row++
		for _, item := range r.Receipts {
			for _, x := range []struct{ From, To, Text string }{{"A", "C", item.Kind}, {"D", "F", item.Manufacturer}, {"G", "J", item.PartNumber}, {"K", "L", item.Quantity}} {
				b.merge(sheet, fmt.Sprintf("%s%d", x.From, row), fmt.Sprintf("%s%d", x.To, row), x.Text, b.Style.Body)
			}
			b.height(sheet, row, 21)
			row++
		}
	} else {
		if material == "" {
			material = "등록된 사급품 현황 없음"
		}
		b.merge(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("L%d", row), material, b.Style.Body)
		b.height(sheet, row, 20)
		row++
	}
	notes := []string{}
	for _, x := range []struct{ Label, Text string }{{"업체 특이사항", r.Project.SpecialInstructions}, {"수주별 추가 메모", r.Project.AdditionalInstructions}, {"출하 메모", r.Memo}} {
		if strings.TrimSpace(x.Text) != "" {
			notes = append(notes, x.Label+" · "+x.Text)
		}
	}
	if len(notes) > 0 {
		row = b.detailText(sheet, row, "특이사항 / 출하 메모", strings.Join(notes, "\n"))
	}
	currentKind := ""
	for i, photo := range photos {
		if photo.Kind != currentKind {
			b.detailSection(sheet, row, photo.Kind)
			row++
			currentKind = photo.Kind
		}
		left := photo.Number%2 == 1
		if left && photo.Number > 1 && (photo.Number-1)%6 == 0 {
			b.check(b.File.InsertPageBreak(sheet, fmt.Sprintf("A%d", row)))
		}
		col, end := "A", "F"
		if !left {
			col, end = "G", "L"
		}
		b.merge(sheet, fmt.Sprintf("%s%d", col, row), fmt.Sprintf("%s%d", end, row), fmt.Sprintf("%s %d · %s", photo.Kind, photo.Number, photo.Name), b.Style.Label)
		b.height(sheet, row, float64(max(19, ((utf8.RuneCountInString(photo.Name)+9+29)/30)*12+5)))
		for y := row + 1; y <= row+8; y++ {
			b.height(sheet, y, 18)
		}
		b.height(sheet, row+9, 4)
		if photo.Problem != "" {
			b.merge(sheet, fmt.Sprintf("%s%d", col, row+1), fmt.Sprintf("%s%d", end, row+8), "사진 확인 필요\n"+photo.Problem, b.Style.Body)
		} else {
			scale := math.Min(350/float64(photo.Width), 180/float64(photo.Height))
			offset := 5 + int((350-float64(photo.Width)*scale)/2)
			b.check(b.File.AddPictureFromBytes(sheet, fmt.Sprintf("%s%d", col, row+1), &excelize.Picture{Extension: ".jpg", File: photo.Data, Format: &excelize.GraphicOptions{AltText: fmt.Sprintf("%s %d", photo.Kind, photo.Number), Name: fmt.Sprintf("Photo%d", i+1), LockAspectRatio: true, OffsetX: offset, OffsetY: 4, ScaleX: scale, ScaleY: scale, Positioning: "oneCell"}}))
		}
		last := i == len(photos)-1 || photos[i+1].Kind != photo.Kind
		if !left || last {
			row += 10
		}
	}
	if len(r.Attachments) > 0 {
		b.detailSection(sheet, row, "첨부파일 / 동영상 · 원본 별도 전달")
		row++
		for _, file := range r.Attachments {
			kind := "파일"
			if file.Kind == "video" {
				kind = "동영상"
			}
			b.merge(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), kind, b.Style.Label)
			b.merge(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("I%d", row), file.Name, b.Style.Body)
			b.merge(sheet, fmt.Sprintf("J%d", row), fmt.Sprintf("L%d", row), exportSize(file.Size), b.Style.Body)
			b.height(sheet, row, 23)
			row++
		}
	}
	note := r.ExportSourceNote
	if note != "" {
		b.merge(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("L%d", row), note, b.Style.Muted)
		b.height(sheet, row, 30)
		row++
	}
	b.merge(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("L%d", row), "사진은 엑셀용 미리보기입니다. 파일·동영상 원본은 별도로 전달합니다.", b.Style.Muted)
	b.height(sheet, row, 20)
	b.check(b.File.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 2, TopLeftCell: "A3", ActivePane: "bottomLeft"}))
}
func (b *exportBook) detailSection(sheet string, row int, title string) {
	b.merge(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("L%d", row), title, b.Style.Section)
	b.height(sheet, row, 20)
}
func (b *exportBook) detailText(sheet string, row int, title, text string) int {
	b.detailSection(sheet, row, title)
	row++
	for len(text) > 0 {
		runes := []rune(text)
		n := min(400, len(runes))
		part := string(runes[:n])
		text = string(runes[n:])
		b.merge(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("L%d", row), part, b.Style.Body)
		lines := 0
		for _, line := range strings.Split(part, "\n") {
			lines += max(1, (utf8.RuneCountInString(line)+53)/54)
		}
		b.height(sheet, row, float64(min(400, lines*15+8)))
		row++
	}
	return row
}
func exportPresent(s string) string {
	if strings.TrimSpace(s) == "" {
		return "미등록"
	}
	return s
}
func exportDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}
func exportSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1<<20 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
}

// Excel external workbook links need a relationship to the file plus a worksheet
// location attribute. Keep locations explicit instead of depending on URL fragments.
func (b *exportBook) save(path string) error {
	if b.Err != nil {
		return b.Err
	}
	if err := b.File.SaveAs(path); err != nil {
		return err
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	replacements := map[string]map[string]string{}
	for sheet, cells := range b.Locations {
		index, err := b.File.GetSheetIndex(sheet)
		if err != nil || index < 0 {
			archive.Close()
			return errors.New("export worksheet link")
		}
		replacements[fmt.Sprintf("xl/worksheets/sheet%d.xml", index+1)] = cells
	}
	temp := path + ".rewrite"
	f, err := os.Create(temp)
	if err != nil {
		archive.Close()
		return err
	}
	z := zip.NewWriter(f)
	fail := func(err error) error { z.Close(); f.Close(); archive.Close(); os.Remove(temp); return err }
	for _, entry := range archive.File {
		r, err := entry.Open()
		if err != nil {
			return fail(err)
		}
		dst, err := z.CreateHeader(&entry.FileHeader)
		if err != nil {
			r.Close()
			return fail(err)
		}
		if cells := replacements[entry.Name]; cells != nil {
			raw, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				return fail(err)
			}
			text := string(raw)
			tags := regexp.MustCompile(`<hyperlink\b[^>]*>`)
			refs := regexp.MustCompile(`\bref="([^"]+)"`)
			found := map[string]int{}
			text = tags.ReplaceAllStringFunc(text, func(tag string) string {
				match := refs.FindStringSubmatch(tag)
				if len(match) != 2 {
					return tag
				}
				location, ok := cells[match[1]]
				if !ok {
					return tag
				}
				found[match[1]]++
				suffix := ">"
				if strings.HasSuffix(tag, "/>") {
					suffix = "/>"
				}
				return strings.TrimSuffix(tag, suffix) + ` location="` + esc(location) + `"` + suffix
			})
			for cell := range cells {
				if found[cell] != 1 {
					return fail(errors.New("export hyperlink location not found"))
				}
			}

			if _, err = io.WriteString(dst, text); err != nil {
				return fail(err)
			}
		} else {
			_, err = io.Copy(dst, r)
			r.Close()
			if err != nil {
				return fail(err)
			}
		}
	}
	if err = z.Close(); err != nil {
		return fail(err)
	}
	if err = f.Close(); err != nil {
		archive.Close()
		os.Remove(temp)
		return err
	}
	if err = archive.Close(); err != nil {
		os.Remove(temp)
		return err
	}
	// Windows cannot replace an existing file with os.Rename, and this file is still
	// private staging data. Remove only after the replacement has been fully closed.
	if err = os.Remove(path); err != nil {
		os.Remove(temp)
		return err
	}
	return os.Rename(temp, path)
}
func (b *exportBook) guide(sheet string) {
	b.sheet(sheet, true)
	b.title(sheet, "출하 기록 · 사용 안내", "이 폴더의 엑셀 파일을 같은 위치에 두고 사용하세요.")
	b.text(sheet, 5, "전체 목록과 상세 보기", "출하목록.xlsx에서 수주번호·프로젝트명·고객사로 필터하고 상세 보기를 누릅니다. 월별 상세 엑셀은 보관 날짜 기준이며 자료가 많으면 같은 달도 여러 파일로 나눕니다. 상세 시트의 전체 목록으로 링크로 돌아올 수 있습니다.")
	b.text(sheet, 12, "사진과 별도 첨부", "사진은 용량을 줄인 미리보기로 엑셀 안에 포함합니다. 사진 원본은 프로그램 자료에서 유지됩니다. 파라미터 파일·동영상은 엑셀에 넣지 않으며 상세 시트에 이름·종류·용량을 표시합니다. 고객사에 필요할 때 원본을 따로 전달해 주세요.")
	b.text(sheet, 20, "확인할 점", "다운로드한 ZIP은 먼저 전체 압축을 풉니다. 파일 이름과 폴더 위치를 유지해야 엑셀 간 상세 링크가 이어집니다. 엑셀에 표시되는 외부 파일 열기 확인은 사용하는 Excel의 설정에 따릅니다. 사진을 읽지 못한 항목은 상세 시트와 내보내기 확인 시트에 표시합니다.")
}
func (b *exportBook) warnings(warnings []exportWarning) {
	if len(warnings) == 0 {
		return
	}
	sheet := "내보내기 확인"
	b.sheet(sheet, false)
	b.title(sheet, "내보내기 확인", fmt.Sprintf("사진 확인이 필요한 항목 %d개 · 해당 원본은 프로그램에서 확인해 주세요.", len(warnings)))
	for i, text := range []string{"수주번호", "프로젝트명", "사진 파일명", "내용"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 5)
		b.str(sheet, cell, text)
	}
	b.style(sheet, "A5", "D5", b.Style.Header)
	b.check(b.File.SetColWidth(sheet, "C", "C", 30))
	b.check(b.File.SetColWidth(sheet, "D", "D", 54))
	for i, w := range warnings {
		row := i + 6
		for col, text := range []string{w.Order, w.Project, w.Photo, w.Message} {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			b.str(sheet, cell, text)
		}
		b.style(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row), b.Style.Body)
		b.height(sheet, row, 36)
	}
}
func buildShippingExport(ctx context.Context, dir string, records []ShippingRecord, now time.Time, read func(Photo) ([]byte, error), progress func(int), limits shippingExportLimits) (int64, int, int, error) {
	if limits.Records < 1 || limits.PhotoBytes < 1 {
		return 0, 0, 0, errors.New("invalid export limits")
	}
	for _, record := range records {
		if record.ArchivedAt == nil {
			return 0, 0, 0, errors.New("unarchived export record")
		}
	}
	records = append([]ShippingRecord{}, records...)
	sort.SliceStable(records, func(i, j int) bool { return records[i].ArchivedAt.After(*records[j].ArchivedAt) })
	if limits.Single {
		if len(records) != 1 || records[0].ArchivedAt == nil {
			return 0, 0, 0, errors.New("invalid single export")
		}
		if err := ctx.Err(); err != nil {
			return 0, 0, 0, err
		}
		r := records[0]
		photos, warnings := prepareExportPhotos(ctx, r, read)
		if err := ctx.Err(); err != nil {
			return 0, 0, 0, err
		}
		b := newExportBook("목차")
		defer b.File.Close()
		entry := exportEntry{Record: r, Sheet: exportSheetName(1, r), Warnings: len(warnings)}
		b.detail(entry, photos, true)
		b.File.SetActiveSheet(1)
		b.index("목차", "출하 기록 · "+r.Project.Name, []exportEntry{entry}, false, now)
		b.warnings(warnings)
		if err := b.save(filepath.Join(dir, "export.xlsx")); err != nil {
			return 0, 0, 0, err
		}
		st, err := os.Stat(filepath.Join(dir, "export.xlsx"))
		if err != nil {
			return 0, 0, 0, err
		}
		if progress != nil {
			progress(1)
		}
		return st.Size(), 1, len(warnings), nil
	}
	entries := []exportEntry{}
	warnings := []exportWarning{}
	files := []string{}
	var book *exportBook
	bookEntries := []exportEntry{}
	photoBytes, workbooks := 0, 0
	month, currentMonth := "", ""
	part := 0
	filename := ""
	defer func() {
		if book != nil {
			book.File.Close()
		}
	}()
	flush := func() error {
		if book == nil {
			return nil
		}
		book.index("목차", currentMonth+" 출하 상세", bookEntries, false, now)
		if err := book.save(filepath.Join(dir, filename)); err != nil {
			return err
		}
		if err := book.File.Close(); err != nil {
			return err
		}
		book = nil
		files = append(files, filename)
		workbooks++
		bookEntries = nil
		photoBytes = 0
		return nil
	}
	for i, r := range records {
		if err := ctx.Err(); err != nil {
			return 0, 0, 0, err
		}
		if r.ArchivedAt == nil {
			return 0, 0, 0, errors.New("unarchived export record")
		}
		photos, issues := prepareExportPhotos(ctx, r, read)
		if err := ctx.Err(); err != nil {
			return 0, 0, 0, err
		}
		bytes := 0
		for _, p := range photos {
			bytes += len(p.Data)
		}
		month = r.ArchivedAt.In(kst).Format("2006-01")
		if book != nil && (month != currentMonth || len(bookEntries) >= limits.Records || photoBytes+bytes > limits.PhotoBytes) {
			if err := flush(); err != nil {
				return 0, 0, 0, err
			}
		}
		if book == nil {
			if month != currentMonth {
				part = 0
			}
			currentMonth = month
			part++
			filename = fmt.Sprintf("%s_출하상세_%02d.xlsx", month, part)
			book = newExportBook("목차")
		}
		entry := exportEntry{Record: r, Book: filename, Sheet: exportSheetName(len(bookEntries)+1, r), Warnings: len(issues)}
		book.detail(entry, photos, false)
		if book.Err != nil {
			return 0, 0, 0, book.Err
		}
		entries = append(entries, entry)
		bookEntries = append(bookEntries, entry)
		photoBytes += bytes
		warnings = append(warnings, issues...)
		if progress != nil {
			progress(i + 1)
		}
	}
	if err := flush(); err != nil {
		return 0, 0, 0, err
	}
	main := newExportBook("출하목록")
	defer main.File.Close()
	main.index("출하목록", "출하 기록 전체 목록", entries, true, now)
	main.guide("사용 안내")
	main.warnings(warnings)
	if len(warnings) > 0 {
		main.str("출하목록", "A4", fmt.Sprintf("사진 확인 필요 %d개 · 내보내기 확인 시트를 확인하세요.", len(warnings)))
	}
	if err := main.save(filepath.Join(dir, "출하목록.xlsx")); err != nil {
		return 0, 0, 0, err
	}
	files = append([]string{"출하목록.xlsx"}, files...)
	guide := []byte("출하 기록 엑셀 사용 안내\r\n\r\n1. ZIP 전체를 같은 폴더에 압축 해제합니다.\r\n2. 출하목록.xlsx의 상세 보기로 수주 상세를 엽니다.\r\n3. 사진은 엑셀 안에 포함되어 프로그램 없이 확인할 수 있습니다.\r\n4. 파일·동영상은 이름과 용량만 표시합니다. 필요한 원본은 별도로 전달하세요.\r\n5. 월 구분은 보관 날짜 기준이며, 같은 달도 자료량에 따라 여러 파일로 나눕니다.\r\n")
	if err := os.WriteFile(filepath.Join(dir, "먼저읽기.txt"), guide, 0600); err != nil {
		return 0, 0, 0, err
	}
	files = append(files, "먼저읽기.txt")
	output := filepath.Join(dir, "export.partial")
	f, err := os.Create(output)
	if err != nil {
		return 0, 0, 0, err
	}
	z := zip.NewWriter(f)
	fail := func(err error) (int64, int, int, error) { z.Close(); f.Close(); os.Remove(output); return 0, 0, 0, err }
	sort.Strings(files)
	for _, name := range files {
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		src, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			return fail(err)
		}
		entry, err := z.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store, Modified: now})
		if err != nil {
			src.Close()
			return fail(err)
		}
		_, err = io.Copy(entry, src)
		src.Close()
		if err != nil {
			return fail(err)
		}
	}
	if err = z.Close(); err != nil {
		return fail(err)
	}
	if err = f.Close(); err != nil {
		return 0, 0, 0, err
	}
	if err = os.Rename(output, filepath.Join(dir, "export.zip")); err != nil {
		return 0, 0, 0, err
	}
	st, err := os.Stat(filepath.Join(dir, "export.zip"))
	if err != nil {
		return 0, 0, 0, err
	}
	for _, name := range files {
		_ = os.Remove(filepath.Join(dir, name))
	}
	return st.Size(), workbooks + 1, len(warnings), nil
}
