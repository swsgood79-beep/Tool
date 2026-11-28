package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type erpOrderOutstandingKey struct{}
type OrderOutstandingRow struct {
	PurchaseDate string  `json:"purchase_date,omitempty"`
	Purchase     string  `json:"purchase,omitempty"`
	Sequence     int     `json:"sequence,omitempty"`
	Part         string  `json:"part,omitempty"`
	Key          string  `json:"key,omitempty"`
	Order        string  `json:"order,omitempty"`
	CustomerCode string  `json:"customer_code,omitempty"`
	Unit         string  `json:"unit,omitempty"`
	Robot        string  `json:"robot"`
	Item         string  `json:"item"`
	Quantity     float64 `json:"quantity"`
	Customer     string  `json:"customer"`
	Due          string  `json:"due"`
}

func (a *App) orderOutstandingHTTP(w http.ResponseWriter, r *http.Request, s Session) bool {
	if r.URL.Path != "/machining/outstanding" {
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != "GET" {
		http.Error(w, "GET 요청이 필요합니다", 405)
		return true
	}
	if !machiningSession(&a.state, s) && !(managerAccount(&a.state, s) && adminMenus(&a.state, s).Machining) {
		http.Error(w, "가공 메뉴 권한이 필요합니다", 403)
		return true
	}
	order := strings.TrimSpace(r.URL.Query().Get("order"))
	found := false
	for _, j := range a.state.MachiningJobs {
		if j.Order == order {
			found = true
			break
		}
	}
	if order == "" || len(order) > 100 || !found {
		http.Error(w, "수주를 확인할 수 없습니다", 400)
		return true
	}
	if a.testMode && a.erpDBProbe == nil {
		http.Error(w, "격리 테스트에서 실제 DB를 조회할 수 없습니다", 409)
		return true
	}
	probe := a.erpDBProbe
	if probe == nil {
		probe = platformERPDBProbe
	}
	ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
	defer cancel()
	ctx = context.WithValue(ctx, erpOrderOutstandingKey{}, order)
	report := probe(ctx, a.erpDBSyncConfig().Folder)
	at := time.Now()
	if report.Status != "order_outstanding" || !report.Connected || report.Truncated {
		message := "미입고 조회를 완료하지 못했습니다. 회사 PC의 ERP 연결을 확인하고 새로고침해 주세요"
		if report.Truncated {
			message = "조회 결과가 한도를 초과해 일부가 잘렸습니다. 전체 미입고 현황을 확인할 수 없습니다"
		}
		w.WriteHeader(502)
		jsonReply(w, map[string]any{"ok": false, "error": message, "at": at})
		return true
	}
	for _, row := range report.OutstandingRows {
		if row.Quantity <= 0 {
			w.WriteHeader(502)
			jsonReply(w, map[string]any{"ok": false, "error": fmt.Sprint("미입고 수량을 확인할 수 없습니다"), "at": at})
			return true
		}
	}
	rows := resolveOutstandingCustomers(report.OutstandingRows, report.Customers)
	if rows == nil {
		rows = []OrderOutstandingRow{}
	}
	jsonReply(w, map[string]any{"ok": true, "order": order, "at": at, "rows": rows})
	return true
}
