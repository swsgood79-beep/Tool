package main

import (
	"bytes"
	"errors"
	"github.com/xuri/excelize/v2"
	"net/http"
	"strings"
)

type MaterialReceipt struct {
	Kind         string `json:"kind"`
	Manufacturer string `json:"manufacturer"`
	PartNumber   string `json:"part_number"`
	Quantity     string `json:"quantity"`
}

func receiptHeader(s string) string { return strings.Join(strings.Fields(s), "") }

// Only explicit Excel merges extend order numbers; blank unmerged rows never inherit an order.
func parseMaterialReceipts(data []byte) (map[string][]MaterialReceipt, bool, error) {
	b, e := excelize.OpenReader(bytes.NewReader(data))
	if e != nil {
		return nil, false, e
	}
	defer b.Close()
	sheet := ""
	for _, n := range b.GetSheetList() {
		if receiptHeader(n) == "사급품수령내역" {
			sheet = n
			break
		}
	}
	if sheet == "" {
		return nil, false, nil
	}
	rows, e := b.GetRows(sheet)
	if e != nil {
		return nil, true, e
	}
	if len(rows) > 20000 {
		return nil, true, errors.New("사급품 수령 내역 20,000행 제한")
	}
	head := -1
	cols := map[string]int{}
	keys := []string{"수주번호", "종류", "제조사", "품번", "수량"}
	for i, row := range rows {
		if i > 30 {
			break
		}
		c := map[string]int{}
		for j, v := range row {
			c[receiptHeader(v)] = j
		}
		ok := true
		for _, k := range keys {
			if _, found := c[k]; !found {
				ok = false
			}
		}
		if ok {
			head = i
			cols = c
			break
		}
	}
	if head < 0 {
		return nil, true, errors.New("수주번호·종류·제조사·품번·수량 열을 찾을 수 없습니다")
	}
	merges, e := b.GetMergeCells(sheet)
	if e != nil {
		return nil, true, e
	}
	merged := map[int]string{}
	for _, m := range merges {
		sc, sr, e1 := excelize.CellNameToCoordinates(m.GetStartAxis())
		ec, er, e2 := excelize.CellNameToCoordinates(m.GetEndAxis())
		if e1 != nil || e2 != nil || sr <= head+1 || er > 20000 || cols["수주번호"]+1 < sc || cols["수주번호"]+1 > ec {
			continue
		}
		value, e := b.GetCellValue(sheet, m.GetStartAxis())
		if e != nil {
			return nil, true, e
		}
		for row := sr; row <= er; row++ {
			merged[row] = value
		}
	}
	out := map[string][]MaterialReceipt{}
	for i := head + 1; i < len(rows); i++ {
		row := rows[i]
		get := func(key string) string {
			c := cols[key]
			if c >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[c])
		}
		number := get("수주번호")
		if number == "" {
			number = strings.TrimSpace(merged[i+1])
		}
		if number == "" {
			continue
		}
		item := MaterialReceipt{get("종류"), get("제조사"), get("품번"), get("수량")}
		if item.Kind == "" && item.Manufacturer == "" && item.PartNumber == "" && item.Quantity == "" {
			continue
		}
		out[number] = append(out[number], item)
	}
	return out, true, nil
}

// Uses the existing worker-board secret; does not make the workbook or historical orders public.
func (a *App) materialReceiptsHTTP(w http.ResponseWriter, r *http.Request) bool {
	const prefix = "/material-receipts/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		return false
	}
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}
	if !equal(strings.TrimPrefix(r.URL.Path, prefix), a.state.Board) {
		http.NotFound(w, r)
		return true
	}
	number := strings.TrimSpace(r.URL.Query().Get("number"))
	var project *Project
	for i := range a.state.Projects {
		p := &a.state.Projects[i]
		if strings.TrimSpace(p.Number) == number && !p.Hidden {
			project = p
			break
		}
	}
	if project == nil {
		http.NotFound(w, r)
		return true
	}
	rows := []MaterialReceipt{}
	received := strings.TrimSpace(project.SuppliedMaterials) == "사급품 수령"
	if received {
		rows = append(rows, a.state.MaterialReceipts[number]...)
	}
	jsonReply(w, map[string]any{"number": project.Number, "name": project.Name, "status": project.SuppliedMaterials, "received": received, "items": rows, "updated": a.state.MaterialReceiptsUpdated, "error": a.state.MaterialReceiptsError})
	return true
}
