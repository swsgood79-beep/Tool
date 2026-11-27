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
type ERPDBReport struct {
	ConfigReason string            `json:"config_reason,omitempty"`
	Stage        string            `json:"stage,omitempty"`
	Version      string            `json:"program_version"`
	At           time.Time         `json:"at"`
	Status       string            `json:"status"`
	Connected    bool              `json:"connected"`
	ErrorCode    int               `json:"error_code,omitempty"`
	Columns      []ERPDBColumn     `json:"columns"`
	ForeignKeys  []ERPDBForeignKey `json:"foreign_keys"`
	Truncated    bool              `json:"truncated"`
}
type ERPDBView struct {
	Busy     bool   `json:"busy"`
	Message  string `json:"message"`
	Download bool   `json:"download"`
}

func dbProbeMessage(r ERPDBReport) string {
	switch r.Status {
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
		return "DB에는 연결됐지만 구조 조회를 완료하지 못했습니다"
	default:
		return "DB 진단에 실패했습니다. 결과 파일의 오류 코드를 확인해 주세요"
	}
}
func decodeDBProbe(raw []byte) ERPDBReport {
	var r ERPDBReport
	if len(raw) > 8<<20 || json.Unmarshal([]byte(strings.TrimPrefix(string(raw), "\ufeff")), &r) != nil {
		return ERPDBReport{Status: "failed"}
	}
	switch r.ConfigReason {
	case "active_key_missing", "active_key_multiple", "setting_format", "unsupported_options", "missing_server_or_database":
	default:
		r.ConfigReason = ""
	}
	switch r.Stage {
	case "config_path", "config_read", "config_parse", "connection_options", "connect", "columns", "relations", "done":
	default:
		r.Stage = ""
	}
	switch r.Status {
	case "ok", "config_missing", "ambiguous", "config_invalid", "timeout", "unsupported", "login_failed", "database_denied", "tls_failed", "metadata_failed", "failed":
	default:
		r.Status = "failed"
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
		if a.erpDBReport == nil || a.erpDBBusy {
			http.Error(w, "진단 완료 후 내려받을 수 있습니다", 409)
			return true
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="ERP-db-check.json"`)
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
		ctx, cancel := context.WithTimeout(parent, 45*time.Second)
		probe := a.erpDBProbe
		if probe == nil {
			probe = platformERPDBProbe
		}
		a.erpDBBusy = true
		go func() {
			defer cancel()
			result := probe(ctx, folder)
			if ctx.Err() != nil {
				result = ERPDBReport{Status: "timeout"}
			}
			result = decodeDBProbe(jsonRaw(result))
			result.Version = releaseVersion
			result.At = time.Now()
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
