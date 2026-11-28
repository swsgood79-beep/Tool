package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ERPDBColumn struct {
	Schema     string `json:"schema"`
	Object     string `json:"object"`
	Kind       string `json:"kind"`
	Column     string `json:"column"`
	Type       string `json:"type"`
	Nullable   bool   `json:"nullable"`
	PrimaryKey bool   `json:"primary_key"`
}
type ERPDBForeignKey struct {
	Schema    string `json:"schema"`
	Object    string `json:"object"`
	Column    string `json:"column"`
	RefSchema string `json:"ref_schema"`
	RefObject string `json:"ref_object"`
	RefColumn string `json:"ref_column"`
}
type erpDBReceiptKey struct{}
type erpDBObserveKey struct{}
type ERPDBObservationAccess struct {
	ViewServerState             bool `json:"view_server_state"`
	ViewServerPerformanceState  bool `json:"view_server_performance_state"`
	AlterAnyEventSession        bool `json:"alter_any_event_session"`
	CreateAnyEventSession       bool `json:"create_any_event_session"`
	AlterAnyEventSessionEnabled bool `json:"alter_any_event_session_enable"`
	DropAnyEventSession         bool `json:"drop_any_event_session"`
	ExtendedEventsAvailable     bool `json:"extended_events_available"`
}

type ERPDBReceiptMetadata struct {
	Kind     string `json:"kind"`
	Schema   string `json:"schema"`
	Object   string `json:"object"`
	Parent   string `json:"parent"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Position int    `json:"position"`
	Flag     bool   `json:"flag"`
	Detail   string `json:"detail"`
}

type erpReceiptContractKey struct{}
type ERPReceiptDefinition struct {
	Name       string `json:"name"`
	Definition string `json:"definition"`
	Visible    bool   `json:"visible"`
	Redacted   bool   `json:"redacted"`
}
type ERPDBReport struct {
	UIReceipt *ERPUIReceiptReport `json:"ui_receipt,omitempty"`

	ReceiptDefinitions []ERPReceiptDefinition `json:"receipt_definitions,omitempty"`

	Certificates []ERPCertificateRow `json:"certificates,omitempty"`

	Observation *ERPObservationReport `json:"observation,omitempty"`

	ObservationAccess *ERPDBObservationAccess `json:"observation_access,omitempty"`

	ReceiptMetadata       []ERPDBReceiptMetadata `json:"receipt_metadata,omitempty"`
	ReceiptRequested      bool                   `json:"receipt_requested,omitempty"`
	ReceiptViewDefinition bool                   `json:"receipt_view_definition,omitempty"`

	Customers       []ERPDBCustomer       `json:"customer_candidates,omitempty"`
	OutstandingRows []OrderOutstandingRow `json:"outstanding_rows,omitempty"`
	PurchaseRows    []ERPDBPurchase       `json:"purchase_rows,omitempty"`
	DateFrom        string                `json:"date_from,omitempty"`
	DateTo          string                `json:"date_to,omitempty"`
	ConfigReason    string                `json:"config_reason,omitempty"`
	Stage           string                `json:"stage,omitempty"`
	Version         string                `json:"program_version"`
	At              time.Time             `json:"at"`
	Status          string                `json:"status"`
	Connected       bool                  `json:"connected"`
	ErrorCode       int                   `json:"error_code,omitempty"`
	Columns         []ERPDBColumn         `json:"columns"`
	ForeignKeys     []ERPDBForeignKey     `json:"foreign_keys"`
	Truncated       bool                  `json:"truncated"`
}
type ERPDBView struct {
	Busy     bool   `json:"busy"`
	Message  string `json:"message"`
	Download bool   `json:"download"`
}

func dbProbeMessage(r ERPDBReport) string {
	switch r.Status {
	case "ui_receipt":
		return uiReceiptMessage(r.UIReceipt)
	case "receipt_contract":
		if r.Truncated || len(r.ReceiptDefinitions) != 3 {
			return "입고 저장 로직을 모두 확인하지 못했습니다. 결과 파일을 확인해 주세요"
		}
		for _, d := range r.ReceiptDefinitions {
			if !d.Visible || d.Redacted {
				return "일부 저장 로직의 권한 제한 또는 민감정보 제외가 있습니다"
			}
		}
		return "입고 저장 로직 3개 조회 완료 · 실행하거나 자료를 변경하지 않았습니다"
	case "observation_access":
		return "입고 관찰 권한 확인 완료 · 아직 DB 호출 기록을 시작하지 않았습니다"
	case "receipt_metadata":
		message := fmt.Sprintf("입고 연동 구조 진단 완료 · 메타데이터 %d개 · 실제 입고는 실행하지 않았습니다", len(r.ReceiptMetadata))
		if !r.ReceiptViewDefinition {
			message += " · 정의 조회 권한 제한: 일부 구조가 보이지 않을 수 있습니다"
		}
		if r.Truncated {
			message += " · 결과 일부 잘림: 전체 분석 미완료"
		}
		return message
	case "purchase_preview":
		return fmt.Sprintf("가공사업부 발주 미리보기 %d행 조회 완료 · 실제 작업 목록에는 반영하지 않았습니다", len(r.PurchaseRows))
	case "ok":
		if len(r.Columns) == 0 {
			return "DB 연결 성공 · 조회 가능한 테이블·열이 없습니다. 구조 조회 권한을 확인해 주세요"
		}
		return fmt.Sprintf("DB 연결·구조 조회 완료 · 열 %d개 / 관계 %d개 (진단 결과이며 자동 연동 완료는 아닙니다)", len(r.Columns), len(r.ForeignKeys))
	case "config_missing":
		return "ERP 설정을 찾지 못했습니다. ERP를 실행하거나 설치 폴더를 지정해 주세요"
	case "ambiguous":
		return "ERP 설치 경로가 여러 개입니다. 사용할 설치 폴더를 지정해 주세요"
	case "config_invalid":
		switch r.ConfigReason {
		case "active_key_missing":
			return "ERP 설정에서 활성 DatabasePath 항목을 찾지 못했습니다"
		case "active_key_multiple":
			return "활성 DB 설정이 여러 개라 중단했습니다. ERP 설정을 확인해 주세요"
		case "setting_format":
			return "ERP DB 연결 문자열을 해석하지 못했습니다. 진단 결과 파일을 확인해 주세요"
		case "unsupported_options":
			return "ERP DB 연결 옵션을 처리하지 못했습니다. 진단 결과 파일을 확인해 주세요"
		case "missing_server_or_database":
			return "ERP DB 설정에서 서버 또는 데이터베이스 항목을 확인하지 못했습니다"
		}
		return "ERP의 활성 DB 설정을 확인할 수 없습니다. 설정 파일을 수정하지 말고 담당자에게 확인해 주세요"
	case "timeout":
		return "DB 진단 시간이 초과됐습니다. 회사 네트워크 연결을 확인해 주세요"
	case "unsupported":
		return "DB 연결 진단은 회사 Windows PC에서 실행합니다"
	case "login_failed":
		return "DB 로그인에 실패했습니다. ERP 자체 접속이 되는지 확인해 주세요"
	case "database_denied":
		return "DB 접근 권한을 확인할 수 없습니다. 관리자 확인이 필요합니다"
	case "tls_failed":
		return "DB 암호화 연결을 완료하지 못했습니다. 서버 연결 설정 확인이 필요합니다"
	case "metadata_failed":
		if r.Stage == "purchase_preview" {
			return "DB에는 연결됐지만 발주 미리보기 조회를 완료하지 못했습니다"
		}
		return "DB에는 연결됐지만 구조 조회를 완료하지 못했습니다"
	case "failed":
		switch r.Stage {
		case "process_start":
			if r.ErrorCode != 0 {
				return fmt.Sprintf("ERP 조회 프로그램 시작 실패 · Windows 오류 %d", r.ErrorCode)
			}
			return "ERP 조회 프로그램을 시작하지 못했습니다"
		case "process_job", "process_attach":
			return "ERP 조회 프로세스 관리 준비를 완료하지 못했습니다"
		case "process_exit":
			return fmt.Sprintf("ERP 조회 프로그램 실행 실패 · 종료 코드 %d", r.ErrorCode)
		case "response_decode":
			return "ERP 조회 결과를 읽지 못했습니다"
		}
	}
	return "DB 진단에 실패했습니다. 결과 파일의 오류 코드를 확인해 주세요"
}
func decodeDBProbe(raw []byte) ERPDBReport {
	var r ERPDBReport
	if len(raw) > 8<<20 || json.Unmarshal([]byte(strings.TrimPrefix(string(raw), "\ufeff")), &r) != nil {
		return ERPDBReport{Status: "failed", Stage: "response_decode"}
	}
	switch r.ConfigReason {
	case "active_key_missing", "active_key_multiple", "setting_format", "unsupported_options", "missing_server_or_database":
	default:
		r.ConfigReason = ""
	}
	switch r.Stage {
	case "process_job", "process_start", "process_attach", "process_wait", "process_exit", "response_decode", "order_outstanding", "purchase_preview", "config_path", "config_read", "config_parse", "connection_options", "connect", "columns", "relations", "receipt_metadata", "observation_access", "observation", "certificates", "receipt_contract", "ui_receipt", "done":
	default:
		r.Stage = ""
	}
	switch r.Status {
	case "ui_receipt", "receipt_contract", "certificates", "observation", "observation_access", "receipt_metadata", "order_outstanding", "purchase_preview", "ok", "config_missing", "ambiguous", "config_invalid", "timeout", "unsupported", "login_failed", "database_denied", "tls_failed", "metadata_failed", "failed":
	default:
		r.Status = "failed"
	}
	if r.Observation != nil && len(r.Observation.Events) > 2000 {
		r.Observation.Events = r.Observation.Events[:2000]
		r.Observation.Truncated = true
	}
	if len(r.Certificates) > 5000 {
		r.Certificates = r.Certificates[:5000]
		r.Truncated = true
	}
	if r.UIReceipt != nil {
		sanitizeUIReceiptReport(r.UIReceipt)
	}
	if len(r.ReceiptDefinitions) > 3 {
		r.ReceiptDefinitions = r.ReceiptDefinitions[:3]
		r.Truncated = true
	}
	for i := range r.ReceiptDefinitions {
		if len(r.ReceiptDefinitions[i].Definition) > 600000 {
			r.ReceiptDefinitions[i].Definition = ""
			r.Truncated = true
		}
	}
	if len(r.ReceiptMetadata) > 5000 {
		r.ReceiptMetadata = r.ReceiptMetadata[:5000]
		r.Truncated = true
	}
	if len(r.Customers) > 20000 {
		r.Customers = r.Customers[:20000]
		r.Truncated = true
	}
	if len(r.OutstandingRows) > 2000 {
		r.OutstandingRows = r.OutstandingRows[:2000]
		r.Truncated = true
	}
	if len(r.PurchaseRows) > 2000 {
		r.PurchaseRows = r.PurchaseRows[:2000]
		r.Truncated = true
	}
	if len(r.Columns) > 20000 {
		r.Columns = r.Columns[:20000]
		r.Truncated = true
	}
	if len(r.ForeignKeys) > 5000 {
		r.ForeignKeys = r.ForeignKeys[:5000]
		r.Truncated = true
	}
	// Only fixed report fields survive decoding. Provider errors, connection
	// strings, paths and unexpected fields are never forwarded to the browser.
	return r
}
func validERPFolder(s string) bool {
	if s == "" {
		return true
	}
	if len(s) > 500 || strings.ContainsAny(s, "\x00\r\n") {
		return false
	}
	return len(s) >= 3 && ((s[0] >= 'A' && s[0] <= 'Z') || (s[0] >= 'a' && s[0] <= 'z')) && s[1] == ':' && (s[2] == '\\' || s[2] == '/')
}
func (a *App) dbProbeView() ERPDBView {
	if a.erpDBBusy {
		if a.erpDBPreview {
			return ERPDBView{Busy: true, Message: "가공사업부 발주 미리보기 조회 중… 작업 목록에는 반영하지 않습니다"}
		}
		return ERPDBView{Busy: true, Message: "DB 연결 및 구조 확인 중… 업무 행은 조회하지 않습니다"}
	}
	if a.erpDBReport == nil {
		return ERPDBView{Message: "아직 DB 진단을 실행하지 않았습니다"}
	}
	return ERPDBView{Message: dbProbeMessage(*a.erpDBReport), Download: true}
}
func (a *App) erpDBHTTP(w http.ResponseWriter, r *http.Request, s Session) bool {
	if !strings.HasPrefix(r.URL.Path, "/machining-admin/db-check") {
		return false
	}
	if s.AccountID != "" {
		http.Error(w, "대표 관리자만 이용할 수 있습니다", 403)
		return true
	}
	w.Header().Set("Cache-Control", "no-store")
	switch {
	case r.URL.Path == "/machining-admin/db-check" && r.Method == "GET":
		jsonReply(w, a.dbProbeView())
	case r.URL.Path == "/machining-admin/db-check/download" && r.Method == "GET":
		if a.erpDBReport == nil || a.erpDBBusy || (a.erpDBReport.UIReceipt != nil && !a.erpDBReport.UIReceipt.Saved) {
			http.Error(w, "진단 완료 후 내려받을 수 있습니다", 409)
			return true
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		filename := "ERP-db-check.json"
		if a.erpDBReport.UIReceipt != nil {
			filename = "ERP-ui-receipt-check.json"
		}
		if a.erpDBReport.Status == "receipt_contract" {
			filename = "ERP-receipt-contract.json"
		}
		if a.erpDBReport.ReceiptRequested {
			filename = "ERP-receipt-check.json"
		}
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
		json.NewEncoder(w).Encode(a.erpDBReport)
	case r.URL.Path == "/machining-admin/db-check" && r.Method == "POST":
		if a.erpDBBusy {
			http.Error(w, "이미 DB 진단이 진행 중입니다", 409)
			return true
		}
		folder := strings.TrimSpace(r.FormValue("folder"))
		if !validERPFolder(folder) {
			http.Error(w, "회사 PC의 ERP 설치 폴더 경로를 입력해 주세요", 400)
			return true
		}
		if a.testMode && a.erpDBProbe == nil {
			http.Error(w, "격리 테스트에서는 실제 DB를 조회하지 않습니다", 409)
			return true
		}
		parent := a.erpDBContext
		if parent == nil {
			parent = context.Background()
		}
		mode := r.FormValue("mode")
		if mode != "" && mode != "preview" && mode != "receipt" && mode != "observe-access" && mode != "receipt-contract" && mode != "ui-receipt" {
			http.Error(w, "지원하지 않는 진단 방식입니다", 400)
			return true
		}
		if mode == "ui-receipt" {
			parent = context.WithValue(parent, erpUIReceiptKey{}, true)
		}
		if mode == "receipt-contract" {
			parent = context.WithValue(parent, erpReceiptContractKey{}, true)
		}
		if mode == "observe-access" {
			parent = context.WithValue(parent, erpDBObserveKey{}, true)
		}
		if mode == "receipt" {
			parent = context.WithValue(parent, erpDBReceiptKey{}, true)
		}
		var preview erpDBPreviewRange
		if mode == "preview" {
			var e error
			preview, e = erpDBPreviewDates(r.FormValue("from"), r.FormValue("to"))
			if e != nil {
				http.Error(w, e.Error(), 400)
				return true
			}
			parent = context.WithValue(parent, erpDBPreviewKey{}, preview)
		}
		ctx, cancel := context.WithTimeout(parent, 45*time.Second)
		probe := a.erpDBProbe
		if probe == nil {
			probe = platformERPDBProbe
		}
		a.erpDBPreview = mode == "preview"
		a.erpDBBusy = true
		go func() {
			defer cancel()
			result := probe(ctx, folder)
			if ctx.Err() != nil {
				result = ERPDBReport{Status: "timeout"}
			}
			result = decodeDBProbe(jsonRaw(result))
			if mode == "preview" {
				result.DateFrom = preview.From
				result.DateTo = preview.To
			}
			result.ReceiptRequested = mode == "receipt"
			if mode == "ui-receipt" && result.UIReceipt == nil {
				reason := "failed"
				if result.Status == "unsupported" {
					reason = "unsupported"
				}
				if result.Status == "timeout" {
					reason = "timeout"
				}
				result.UIReceipt = &ERPUIReceiptReport{Reason: reason}
				result.Status = "ui_receipt"
			}
			result.Version = releaseVersion
			result.At = time.Now()
			if result.UIReceipt != nil {
				saveUIReceiptReport(a.dir, &result)
			}
			a.mu.Lock()
			a.erpDBReport = &result
			a.erpDBBusy = false
			a.mu.Unlock()
		}()
		w.WriteHeader(202)
		jsonReply(w, a.dbProbeView())
	default:
		http.NotFound(w, r)
	}
	return true
}

// Do not embed bytes.Buffer: its promoted ReadFrom would bypass Write's cap.
type erpDBOutput struct{ buf bytes.Buffer }

func (b *erpDBOutput) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) > 8<<20 {
		return 0, fmt.Errorf("metadata output limit")
	}
	return b.buf.Write(p)
}
func (b *erpDBOutput) Bytes() []byte { return b.buf.Bytes() }
