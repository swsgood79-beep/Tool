package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

func parseMachiningWorkbook(data []byte) ([]MachiningSource, error) {
	if len(data) == 0 || len(data) > 16<<20 {
		return nil, errors.New("ERP 엑셀은 16MB 이하 XLSX 파일로 선택해 주세요")
	}
	z, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		return nil, errors.New("정상 XLSX 파일이 아닙니다")
	}
	var expanded uint64
	for _, f := range z.File {
		if f.UncompressedSize64 > (64<<20)-expanded {
			return nil, errors.New("엑셀 압축 해제 크기가 너무 큽니다")
		}
		expanded += f.UncompressedSize64
	}
	f, e := excelize.OpenReader(bytes.NewReader(data), excelize.Options{UnzipSizeLimit: 64 << 20, UnzipXMLSizeLimit: 8 << 20})
	if e != nil {
		return nil, errors.New("ERP 엑셀 파일을 열 수 없습니다")
	}
	defer f.Close()
	required := []string{"발주번호", "거래처", "수주번호", "프로젝트", "로봇품번", "품번", "품명", "수량", "입고수", "입고예정일"}
	var result []MachiningSource
	found := false
	for _, sheet := range f.GetSheetList() {
		iter, err := f.Rows(sheet)
		if err != nil {
			return nil, err
		}
		headers := map[string]int{}
		headerRow := 0
		rowNo := 0
		for iter.Next() {
			rowNo++
			if rowNo > 20000 {
				iter.Close()
				return nil, errors.New("엑셀은 시트당 20,000행 이내로 가져와 주세요")
			}
			cols, err := iter.Columns()
			if err != nil {
				iter.Close()
				return nil, err
			}
			if headerRow == 0 {
				if rowNo > 30 {
					break
				}
				candidate := map[string]int{}
				duplicate := false
				for i, v := range cols {
					v = strings.TrimSpace(v)
					if _, ok := candidate[v]; ok && v != "" {
						duplicate = true
					}
					candidate[v] = i
				}
				ok := !duplicate
				for _, key := range required {
					if _, yes := candidate[key]; !yes {
						ok = false
					}
				}
				if ok {
					if found {
						iter.Close()
						return nil, errors.New("ERP 목록 시트가 여러 개입니다. 원본 보고서 한 개를 선택해 주세요")
					}
					headers = candidate
					headerRow = rowNo
					found = true
				}
				continue
			}
			get := func(key string) string {
				i, ok := headers[key]
				if !ok || i >= len(cols) {
					return ""
				}
				return strings.TrimSpace(cols[i])
			}
			if !strings.EqualFold(get("품명"), "BASE") {
				continue
			}
			if get("거래처") != "가공사업부" {
				iter.Close()
				return nil, fmt.Errorf("%d행: 거래처가 가공사업부가 아닙니다. 조회 조건을 확인해 주세요", rowNo)
			}
			quantity, err := machiningQuantity(get("수량"))
			if err != nil || quantity < 1 {
				iter.Close()
				return nil, fmt.Errorf("%d행: 발주 수량을 확인해 주세요", rowNo)
			}
			received, err := machiningQuantity(get("입고수"))
			if err != nil || received < 0 || received > quantity {
				iter.Close()
				return nil, fmt.Errorf("%d행: 입고 수량을 확인해 주세요", rowNo)
			}
			if received == quantity {
				continue
			}
			for _, key := range []string{"발주번호", "수주번호", "프로젝트", "로봇품번", "품번"} {
				if !validText(get(key), 200) {
					iter.Close()
					return nil, fmt.Errorf("%d행: %s 값을 확인해 주세요", rowNo, key)
				}
			}
			result = append(result, MachiningSource{Row: rowNo, Purchase: get("발주번호"), Order: get("수주번호"), Project: get("프로젝트"), Robot: get("로봇품번"), Part: get("품번"), Due: get("입고예정일"), Quantity: quantity, Received: received})
		}
		err = iter.Error()
		iter.Close()
		if err != nil && err != io.EOF {
			return nil, err
		}
	}
	if !found {
		return nil, errors.New("발주대비미입고현황의 필수 열을 찾지 못했습니다. ERP에서 내보낸 원본 XLSX를 선택해 주세요")
	}
	if len(result) == 0 {
		return nil, errors.New("가공사업부의 BASE 미입고 항목이 없습니다")
	}
	return result, nil
}
func machiningQuantity(s string) (int, error) {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" {
		return 0, errors.New("빈 수량")
	}
	f, e := strconv.ParseFloat(s, 64)
	if e != nil || f < 0 || f > 1000000 || f != float64(int(f)) {
		return 0, errors.New("정수 수량 필요")
	}
	return int(f), nil
}
