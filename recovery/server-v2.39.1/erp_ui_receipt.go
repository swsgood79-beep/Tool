package main

import (
	"fmt"
	"path/filepath"
	"time"
)

type erpUIReceiptKey struct{}
type ERPUIReceiptElement struct {
	LegacyRole    uint32 `json:"legacy_role"`
	LegacyState   uint32 `json:"legacy_state"`
	LegacyChildID int    `json:"legacy_child_id"`

	Index        int      `json:"index"`
	Parent       int      `json:"parent"`
	Depth        int      `json:"depth"`
	Type         string   `json:"type"`
	Class        string   `json:"class"`
	AutomationID string   `json:"automation_id"`
	Name         string   `json:"name"`
	Value        string   `json:"value,omitempty"`
	Patterns     []string `json:"patterns"`
	Enabled      bool     `json:"enabled"`
	Offscreen    bool     `json:"offscreen"`
	Row          int      `json:"row"`
	Column       int      `json:"column"`
	GridRows     int      `json:"grid_rows"`
	GridColumns  int      `json:"grid_columns"`
	Toggle       string   `json:"toggle,omitempty"`
	Selected     bool     `json:"selected"`
}
type ERPUIReceiptReport struct {
	Reason           string                `json:"reason"`
	Window           string                `json:"window"`
	Elements         []ERPUIReceiptElement `json:"elements"`
	Truncated        bool                  `json:"truncated"`
	ReadErrors       int                   `json:"read_errors"`
	PasswordExcluded int                   `json:"password_excluded"`
	Saved            bool                  `json:"saved"`
}

func sanitizeUIReceiptReport(report *ERPUIReceiptReport) {
	if len(report.Elements) > 1200 {
		report.Elements = report.Elements[:1200]
		report.Truncated = true
	}
	for i := range report.Elements {
		element := &report.Elements[i]
		for _, field := range []*string{&element.Name, &element.Value, &element.Class, &element.AutomationID} {
			if len(*field) > 2000 {
				*field = (*field)[:2000]
				report.Truncated = true
			}
		}
		if len(element.Patterns) > 24 {
			element.Patterns = element.Patterns[:24]
			report.Truncated = true
		}
	}
}

func uiReceiptMessage(report *ERPUIReceiptReport) string {
	if report == nil {
		return "ERP 화면 인식 결과를 읽지 못했습니다"
	}
	if !report.Saved {
		return "ERP 화면 인식 결과의 보호 저장에 실패했습니다"
	}
	switch report.Reason {
	case "captured":
		message := fmt.Sprintf("ERP 화면 인식 확인 완료 · %d개 요소 · 자동 체크 가능 범위는 결과 분석 후 확인합니다", len(report.Elements))
		if report.Truncated || report.ReadErrors > 0 {
			message += " · 일부 요소 잘림/읽기 제한 있음"
		}
		return message
	case "erp_missing":
		return "회사 PC에서 ERP를 실행하고 입고등록의 발주 검색 목록을 열어 주세요"
	case "erp_ambiguous":
		return "ERP가 여러 개 실행 중입니다. 작업할 ERP 한 개를 남겨 주세요"
	case "window_missing":
		return "ERP 입고등록 → Data 불러오기 창(PG03F2120)을 열고 다시 확인해 주세요"
	case "window_ambiguous":
		return "입고 발주 검색 창이 여러 개입니다. 한 개를 남겨 주세요"
	case "timeout":
		return "ERP 화면 인식 시간이 초과됐습니다. 창 상태를 확인한 뒤 다시 실행해 주세요"
	case "unsupported":
		return "ERP 화면 인식 확인은 회사 Windows PC에서 실행합니다"
	default:
		return "ERP 화면 요소를 읽지 못했습니다. ERP와 Alpha의 실행 권한 및 창 상태를 확인해 주세요"
	}
}

func saveUIReceiptReport(directory string, report *ERPDBReport) {
	report.UIReceipt.Saved = true
	raw := jsonRaw(report)
	sealed, err := protect(raw)
	if err == nil {
		var verified []byte
		verified, err = unprotect(sealed)
		if err == nil && string(verified) != string(raw) {
			err = fmt.Errorf("UI receipt protection verification failed")
		}
	}
	if err == nil {
		filename := fmt.Sprintf("ERP-ui-receipt-check-%d.dat", time.Now().UnixNano())
		err = atomicFile(filepath.Join(directory, filename), sealed)
	}
	if err != nil {
		report.UIReceipt.Saved = false
	}
}
