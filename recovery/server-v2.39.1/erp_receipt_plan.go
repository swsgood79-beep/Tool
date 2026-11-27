package main

import (
	"fmt"
	"math"
	"strings"
)

// Exact purchase keys, never displayed item names, identify receipt targets.
type ERPReceiptKey struct {
	Unit, PurchaseDate, Purchase string
	Sequence                     int
}
type ERPReceiptLine struct {
	Key                 ERPReceiptKey
	Item, CustomerCode  string
	Remaining, Quantity float64
}
type ERPReceiptPlan struct {
	CustomerCode, Unit string
	Lines              []ERPReceiptLine
}

func buildERPReceiptPlan(lines []ERPReceiptLine) (ERPReceiptPlan, error) {
	p := ERPReceiptPlan{}
	if len(lines) == 0 || len(lines) > 100 {
		return p, fmt.Errorf("입고 품목을1~100행 선택해 주세요")
	}
	seen := map[ERPReceiptKey]bool{}
	for _, line := range lines {
		k := line.Key
		if strings.TrimSpace(k.Unit) == "" || strings.TrimSpace(k.PurchaseDate) == "" || strings.TrimSpace(k.Purchase) == "" || k.Sequence <= 0 || strings.TrimSpace(line.Item) == "" || strings.TrimSpace(line.CustomerCode) == "" {
			return p, fmt.Errorf("발주 행 식별 정보를 확인할 수 없습니다")
		}
		if seen[k] {
			return p, fmt.Errorf("같은 발주 행이 중복 선택됐습니다")
		}
		seen[k] = true
		if math.IsNaN(line.Quantity) || math.IsInf(line.Quantity, 0) || math.IsNaN(line.Remaining) || math.IsInf(line.Remaining, 0) || line.Quantity <= 0 || line.Remaining <= 0 || line.Quantity > line.Remaining || line.Quantity != math.Trunc(line.Quantity) {
			return p, fmt.Errorf("입고 수량은 미입고 잔량 이하의 양수 정수여야 합니다")
		}
		if p.Unit == "" {
			p.Unit = k.Unit
			p.CustomerCode = line.CustomerCode
		}
		if p.Unit != k.Unit || p.CustomerCode != line.CustomerCode {
			return p, fmt.Errorf("사업장과 거래처가 같은 품목만 한 전표로 입고할 수 있습니다")
		}
		p.Lines = append(p.Lines, line)
	}
	return p, nil
}
