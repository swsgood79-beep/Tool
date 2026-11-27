package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type erpReceiptSelection struct {
	Key      ERPReceiptKey
	Quantity float64
}
type erpReceiptPreview struct {
	Order, Scope, Fingerprint string
	At                        time.Time
	Plan                      ERPReceiptPlan
}

func receiptRowKey(r OrderOutstandingRow) ERPReceiptKey {
	return ERPReceiptKey{r.Unit, r.PurchaseDate, r.Purchase, r.Sequence}
}
func receiptPlanFromReport(report ERPDBReport, order string, selection []erpReceiptSelection) (ERPReceiptPlan, error) {
	if _, e := outstandingReportRows(report); e != nil {
		return ERPReceiptPlan{}, e
	}
	if len(selection) == 0 || len(selection) > 100 {
		return ERPReceiptPlan{}, fmt.Errorf("1~100행을 선택해 주세요")
	}
	rows := map[ERPReceiptKey]OrderOutstandingRow{}
	for _, r := range report.OutstandingRows {
		if strings.TrimSpace(r.Order) != order {
			return ERPReceiptPlan{}, fmt.Errorf("조회 수주가 일치하지 않습니다")
		}
		k := receiptRowKey(r)
		if _, exists := rows[k]; exists {
			return ERPReceiptPlan{}, fmt.Errorf("중복 발주 행으로 입고를 보류합니다")
		}
		rows[k] = r
	}
	lines := []ERPReceiptLine{}
	for _, s := range selection {
		r, ok := rows[s.Key]
		if !ok {
			return ERPReceiptPlan{}, fmt.Errorf("선택한 발주 행이 변경되거나 입고되어 다시 조회해야 합니다")
		}
		lines = append(lines, ERPReceiptLine{Key: s.Key, Item: r.ItemCode, CustomerCode: r.CustomerCode, Remaining: r.Quantity, Quantity: s.Quantity})
	}
	return buildERPReceiptPlan(lines)
}
func (a *App) erpReceiptPreviewHTTP(w http.ResponseWriter, r *http.Request, s Session) bool {
	if r.URL.Path != "/machining-admin/receipt-preview" {
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	if s.AccountID != "" {
		http.Error(w, "대표 관리자만 이용할 수 있습니다", 403)
		return true
	}
	if r.Method != "POST" {
		http.Error(w, "POST 요청이 필요합니다", 405)
		return true
	}
	order := strings.TrimSpace(r.FormValue("order"))
	found := false
	for _, p := range a.state.Projects {
		if p.Number == order {
			found = true
			break
		}
	}
	if !found || order == "" || len(order) > 100 {
		http.Error(w, "수주를 확인할 수 없습니다", 400)
		return true
	}
	var selected []erpReceiptSelection
	raw := r.FormValue("lines")
	if len(raw) > 65536 || json.Unmarshal([]byte(raw), &selected) != nil || len(selected) == 0 || len(selected) > 100 {
		http.Error(w, "선택 품목을 확인해 주세요", 400)
		return true
	}
	if a.testMode && a.erpReceiptProbe == nil {
		http.Error(w, "격리 시험에서 ERP에 연결하지 않습니다", 409)
		return true
	}
	probe := a.erpReceiptProbe
	if probe == nil {
		probe = platformERPDBProbe
	}
	folder := a.erpDBSyncConfig().Folder
	scope := a.outstandingScope()
	ctx, cancel := context.WithTimeout(context.WithValue(r.Context(), erpOrderOutstandingKey{}, order), 35*time.Second)
	defer cancel()
	// Release dispatcher lock during the read-only DB round trip; revalidate scope afterward.
	var report ERPDBReport
	func() { a.mu.Unlock(); defer a.mu.Lock(); report = decodeDBProbe(jsonRaw(probe(ctx, folder))) }()
	if a.outstandingScope() != scope || !validAccountSession(&a.state, s) {
		http.Error(w, "설정 또는 로그인 권한이 변경됐습니다", 409)
		return true
	}
	plan, e := receiptPlanFromReport(report, order, selected)
	if e != nil {
		http.Error(w, e.Error(), 409)
		return true
	}
	if a.erpReceiptPreviews == nil {
		a.erpReceiptPreviews = map[string]erpReceiptPreview{}
	}
	for id, p := range a.erpReceiptPreviews {
		if time.Since(p.At) > 5*time.Minute {
			delete(a.erpReceiptPreviews, id)
		}
	}
	if len(a.erpReceiptPreviews) >= 20 {
		http.Error(w, "미리보기가 많습니다. 잠시 후 다시 시도해 주세요", 429)
		return true
	}
	id := token()
	p := erpReceiptPreview{Order: order, Scope: scope, Fingerprint: digest(jsonRaw(plan)), At: time.Now(), Plan: plan}
	a.erpReceiptPreviews[id] = p
	jsonReply(w, map[string]any{"ok": true, "preview": id, "order": order, "plan": plan, "at": p.At, "execution_enabled": false, "message": "입고 내용 확인 · ERP 저장 연결은 검증 중입니다. 아직 입고되지 않았습니다"})
	return true
}
