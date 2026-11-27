package main

import (
	"context"
	"fmt"
	"github.com/xuri/excelize/v2"
	"net/http"
	"strings"
	"time"
	"unicode"
)

type erpCertificateKey struct{}
type ERPCertificateRow struct {
	Unit           string `json:"unit"`
	Order          string `json:"order"`
	Customer       string `json:"customer"`
	Client         string `json:"client"`
	Project        string `json:"project"`
	Item           string `json:"item"`
	Serial         string `json:"serial"`
	Remark         string `json:"remark"`
	CommandDate    string `json:"command_date"`
	CommandNo      string `json:"command_no"`
	Sequence       int    `json:"sequence"`
	InspectionDate string `json:"inspection_date"`
	Judgment       string `json:"judgment"`
	Registered     bool   `json:"registered"`
	Certificate    string `json:"certificate"`
}
type erpCertificateJob struct {
	ID, Order, Message string
	Busy, OK           bool
	At                 time.Time
	Rows               []ERPCertificateRow
}

func validCertificateOrder(s string) bool {
	if len(s) < 1 || len(s) > 100 {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func (a *App) certificateView() map[string]any {
	j := a.erpCertificates
	if j == nil {
		return map[string]any{"message": "수주번호를 입력해 조회하세요"}
	}
	return map[string]any{"id": j.ID, "order": j.Order, "busy": j.Busy, "message": j.Message, "ok": j.OK, "at": j.At, "rows": j.Rows}
}
func (a *App) erpCertificateHTTP(w http.ResponseWriter, r *http.Request, s Session) bool {
	if !strings.HasPrefix(r.URL.Path, "/machining-admin/certificates") {
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	if s.AccountID != "" {
		http.Error(w, "대표 관리자만 조회할 수 있습니다", 403)
		return true
	}
	switch {
	case r.Method == "GET" && r.URL.Path == "/machining-admin/certificates":
		a.render(w, "erp-certificates.html", map[string]any{"CSRF": s.CSRF})
	case r.Method == "GET" && r.URL.Path == "/machining-admin/certificates/status":
		jsonReply(w, a.certificateView())
	case r.Method == "GET" && r.URL.Path == "/machining-admin/certificates/export":
		j := a.erpCertificates
		if j == nil || j.Busy || !j.OK || r.URL.Query().Get("id") != j.ID || time.Since(j.At) > 10*time.Minute {
			http.Error(w, "정상 조회 후10분 이내에 엑셀을 받으세요", 409)
			break
		}
		b, e := certificateWorkbook(j)
		if e != nil {
			http.Error(w, "엑셀 생성 실패", 500)
			break
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="ERP-inspection-certificates.xlsx"`)
		w.Write(b)
	case r.Method == "POST" && r.URL.Path == "/machining-admin/certificates":
		if a.erpCertificates != nil && a.erpCertificates.Busy {
			http.Error(w, "조회 중입니다", 409)
			break
		}
		order := strings.TrimSpace(r.FormValue("order"))
		if !validCertificateOrder(order) {
			http.Error(w, "수주번호를 확인해 주세요", 400)
			break
		}
		if a.testMode && a.erpCertificateProbe == nil {
			http.Error(w, "격리 시험에서 ERP에 접속하지 않습니다", 409)
			break
		}
		folder := a.erpDBSyncConfig().Folder
		if f := strings.TrimSpace(r.FormValue("folder")); f != "" {
			folder = f
		}
		if !validERPFolder(folder) {
			http.Error(w, "설치 폴더를 확인해 주세요", 400)
			break
		}
		j := &erpCertificateJob{ID: token(), Order: order, Busy: true, Message: "ERP 조회 중…"}
		a.erpCertificates = j
		probe := a.erpCertificateProbe
		if probe == nil {
			probe = platformERPDBProbe
		}
		parent := a.erpDBContext
		if parent == nil {
			parent = context.Background()
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.WithValue(parent, erpCertificateKey{}, order), 45*time.Second)
			defer cancel()
			report := decodeDBProbe(jsonRaw(probe(ctx, folder)))
			a.mu.Lock()
			defer a.mu.Unlock()
			j.Busy = false
			j.At = time.Now()
			j.Message = "조회 실패 · 기존 결과를 재사용하지 않습니다"
			if ctx.Err() != nil {
				j.Message = "조회 시간 초과"
				return
			}
			if report.Status != "certificates" || !report.Connected {
				j.Message = fmt.Sprintf("ERP 조회 실패 · 단계 %s · 오류코드 %d", report.Stage, report.ErrorCode)
				return
			}
			if report.Truncated {
				j.Message = "결과가 잘렸습니다. 전체 엑셀로 저장할 수 없습니다"
				return
			}
			keys := map[string]bool{}
			for _, row := range report.Certificates {
				key := fmt.Sprintf("%q/%q/%q/%d", row.Unit, row.CommandDate, row.CommandNo, row.Sequence)
				if strings.TrimSpace(row.Order) != order || keys[key] {
					j.Message = "수주 불일치 또는 중복 행이 있어 결과를 보류합니다"
					return
				}
				keys[key] = true
			}
			j.Rows = report.Certificates
			j.OK = true
			j.Message = fmt.Sprintf("조회 완료 · %d행 · ERP 화면과 최초 대조가 필요합니다", len(j.Rows))
			if len(j.Rows) == 0 {
				j.Message = "조회 완료 · 해당 수주의 생산지시/성적서 목록이 없습니다"
			}
		}()
		w.WriteHeader(202)
		jsonReply(w, a.certificateView())
	default:
		http.NotFound(w, r)
	}
	return true
}
func certificateWorkbook(j *erpCertificateJob) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "검사성적서 목록"
	if e := f.SetSheetName("Sheet1", sheet); e != nil {
		return nil, e
	}
	heads := []string{"사업장", "수주번호", "거래처", "최종고객", "PROJECT", "품번", "SERIAL_NO", "비고", "생산지시일", "생산지시번호", "SEQ", "검사일", "판정(ERP 코드)", "성적서 등록", "성적서 양식번호"}
	for n, h := range heads {
		cell, _ := excelize.CoordinatesToCellName(n+1, 1)
		if e := f.SetCellStr(sheet, cell, h); e != nil {
			return nil, e
		}
	}
	for n, r := range j.Rows {
		registered := "미등록"
		if r.Registered {
			registered = "등록"
		}
		values := []string{r.Unit, r.Order, r.Customer, r.Client, r.Project, r.Item, r.Serial, r.Remark, r.CommandDate, r.CommandNo, fmt.Sprint(r.Sequence), r.InspectionDate, r.Judgment, registered, r.Certificate}
		for c, v := range values {
			cell, _ := excelize.CoordinatesToCellName(c+1, n+2)
			if e := f.SetCellStr(sheet, cell, v); e != nil {
				return nil, e
			}
		}
	}
	f.SetColWidth(sheet, "A", "O", 20)
	f.SetColWidth(sheet, "F", "H", 38)
	f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	f.AutoFilter(sheet, fmt.Sprintf("A1:O%d", len(j.Rows)+1), nil)
	f.NewSheet("조회 정보")
	for n, v := range []string{"수주번호: " + j.Order, "조회 시각(KST): " + j.At.In(time.FixedZone("KST", 9*3600)).Format("2006-01-02 15:04:05"), "ERP 읽기 전용 조회. 화면과 최초 대조 필요. 검사 측정값/성적서 원본 양식은 포함하지 않습니다.", "검사일 필터 없음 · 사업장 전체 · 생산지시 행별 목록 · 미등록 구분 포함"} {
		f.SetCellStr("조회 정보", fmt.Sprintf("A%d", n+1), v)
	}
	f.SetColWidth("조회 정보", "A", "A", 110)
	b, e := f.WriteToBuffer()
	if e != nil {
		return nil, e
	}
	return b.Bytes(), nil
}
