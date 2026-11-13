package main

import (
	"bytes"
	"errors"
	"github.com/xuri/excelize/v2"
	"strings"
)

// Column N is only used when its header identifies supplied materials; a changed layout never clears prior values.
func parseScheduleMaterials(data []byte) (map[string]string, bool, error) {
	b, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, false, err
	}
	defer b.Close()
	rows, err := b.GetRows("조립일정", excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, false, err
	}
	if len(rows) > 20000 {
		return nil, false, errors.New("일정표 20,000행 제한")
	}
	head, numberCol := -1, -1
	for i, row := range rows {
		if i > 30 {
			break
		}
		nc, dc := -1, -1
		for j, value := range row {
			v := strings.ToLower(strings.NewReplacer(" ", "", "\n", "", "\r", "").Replace(value))
			if v == "수주no" || v == "erp수주no" || v == "수주번호" {
				nc = j
			}
			if v == "발송예정일" {
				dc = j
			}
		}
		if nc >= 0 && dc >= 0 {
			head, numberCol = i, nc
			break
		}
	}
	if head < 0 {
		return nil, false, errors.New("수주NO·발송예정일 열을 찾을 수 없습니다")
	}
	if len(rows[head]) <= 13 || !strings.Contains(rows[head][13], "사급") {
		return nil, false, nil
	}
	mergedNumbers, mergedMaterials := map[int]string{}, map[int]string{}
	merges, err := b.GetMergeCells("조립일정")
	if err != nil {
		return nil, false, err
	}
	for _, m := range merges {
		sc, sr, e1 := excelize.CellNameToCoordinates(m.GetStartAxis())
		ec, er, e2 := excelize.CellNameToCoordinates(m.GetEndAxis())
		if e1 != nil || e2 != nil || sr <= head+1 || er > 20000 {
			continue
		}
		for _, target := range []struct {
			col    int
			values map[int]string
			raw    bool
		}{{numberCol + 1, mergedNumbers, true}, {14, mergedMaterials, false}} {
			if target.col < sc || target.col > ec {
				continue
			}
			value, e := b.GetCellValue("조립일정", m.GetStartAxis(), excelize.Options{RawCellValue: target.raw})
			if e != nil {
				return nil, false, e
			}
			for r := sr; r <= er; r++ {
				target.values[r] = value
			}
		}
	}
	values := map[string][]string{}
	for index, row := range rows[head+1:] {
		r := head + index + 2
		n := ""
		if len(row) > numberCol {
			n = strings.TrimSpace(row[numberCol])
		}
		if n == "" {
			n = strings.TrimSpace(mergedNumbers[r])
		}
		if n == "" {
			continue
		}
		cell, _ := excelize.CoordinatesToCellName(14, r)
		value, e := b.GetCellValue("조립일정", cell)
		if e != nil {
			return nil, false, e
		}
		if value == "" {
			value = mergedMaterials[r]
		}
		value = strings.TrimSpace(strings.ReplaceAll(value, "\r\n", "\n"))
		if _, ok := values[n]; !ok {
			values[n] = nil
		}
		if value != "" {
			duplicate := false
			for _, old := range values[n] {
				if old == value {
					duplicate = true
					break
				}
			}
			if !duplicate {
				values[n] = append(values[n], value)
			}
		}
	}
	out := map[string]string{}
	for n, parts := range values {
		out[n] = strings.Join(parts, "\n")
	}
	return out, true, nil
}
