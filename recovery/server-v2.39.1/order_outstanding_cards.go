package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type erpOrderOutstandingBatchKey struct{}
type orderOutstandingEntry struct {
	Scope, State, Error string
	Rows                []OrderOutstandingRow
	At, Expires         time.Time
}
type orderOutstandingView struct {
	Number   string                `json:"number"`
	Snapshot string                `json:"snapshot,omitempty"`
	State    string                `json:"state"`
	Count    int                   `json:"count"`
	At       *time.Time            `json:"at,omitempty"`
	Error    string                `json:"error,omitempty"`
	Rows     []OrderOutstandingRow `json:"rows,omitempty"`
}

func (a *App) outstandingScope() string {
	return digest(jsonRaw([]any{a.state.Board, a.erpDBSyncConfig()}))
}
func outstandingReportRows(report ERPDBReport) ([]OrderOutstandingRow, error) {
	if report.Truncated {
		return nil, fmt.Errorf("조회 결과가 한도를 초과해 일부가 잘렸습니다. 전체 미입고 현황을 확인할 수 없습니다")
	}
	if report.Status != "order_outstanding" || !report.Connected {
		return nil, fmt.Errorf("미입고 조회 실패 · %s", dbProbeMessage(report))
	}
	for _, row := range report.OutstandingRows {
		if row.Quantity <= 0 {
			return nil, fmt.Errorf("미입고 수량을 확인할 수 없습니다")
		}
	}
	rows := resolveOutstandingCustomers(report.OutstandingRows, report.Customers)
	for i := range rows {
		rows[i].Unit = report.OutstandingRows[i].Unit
		rows[i].CustomerCode = report.OutstandingRows[i].CustomerCode
	}
	return rows, nil
}

// The caller holds App.mu. Cache entries are memory-only and never enter State,
// shipment records, backups or workbook exports.
func (a *App) orderOutstandingCardsHTTP(w http.ResponseWriter, r *http.Request) bool {
	const prefix = "/order-outstanding/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != "GET" {
		http.Error(w, "조회 요청만 가능합니다", 405)
		return true
	}
	if !equal(strings.TrimPrefix(r.URL.Path, prefix), a.state.Board) {
		http.NotFound(w, r)
		return true
	}
	ids := strings.Split(r.URL.Query().Get("projects"), ",")
	if len(ids) > 10 {
		http.Error(w, "한 번에 10개 수주까지 조회할 수 있습니다", 400)
		return true
	}
	projects := map[string]string{}
	for _, id := range ids {
		n, e := strconv.ParseInt(id, 10, 64)
		if e != nil || n <= 0 {
			http.Error(w, "수주를 확인해 주세요", 400)
			return true
		}
		p := a.state.project(n)
		if p == nil || p.Hidden || p.Shipping.Completed != nil || p.LegacyCompleted {
			http.NotFound(w, r)
			return true
		}
		number := strings.TrimSpace(p.Number)
		if !validText(number, 100) {
			http.Error(w, "수주번호를 확인할 수 없습니다", 400)
			return true
		}
		projects[id] = number
	}
	detail := r.URL.Query().Get("detail") == "1"
	if detail && len(projects) != 1 {
		http.Error(w, "상세 조회는 수주 하나를 선택해 주세요", 400)
		return true
	}
	if a.outstandingCards == nil {
		a.outstandingCards = map[string]orderOutstandingEntry{}
	}
	now := time.Now()
	scope := a.outstandingScope()
	for order, entry := range a.outstandingCards {
		if entry.State != "pending" && (entry.Scope != scope || !now.Before(entry.Expires)) {
			delete(a.outstandingCards, order)
		}
	}
	missing := []string{}
	seen := map[string]bool{}
	for _, order := range projects {
		entry, found := a.outstandingCards[order]
		if r.URL.Query().Get("refresh") == "1" && entry.State != "pending" {
			entry.Expires = time.Time{}
			if found {
				a.outstandingCards[order] = entry
			}
			found = false
		}
		if !found || entry.Scope != scope {
			if !seen[order] {
				missing = append(missing, order)
				seen[order] = true
			}
		}
	}
	if len(missing) > 0 && !a.outstandingBusy {
		for _, order := range missing {
			a.outstandingCards[order] = orderOutstandingEntry{Scope: scope, State: "pending"}
		}
		a.outstandingBusy = true
		cfg := a.erpDBSyncConfig()
		probe := a.erpDBProbe
		ctx := a.erpDBContext
		if ctx == nil {
			ctx = context.Background()
		}
		isolated := a.testMode && probe == nil
		if probe == nil {
			probe = platformERPDBProbe
		}
		go a.fetchOutstandingCards(ctx, missing, scope, cfg, probe, isolated)
	}
	views := map[string]orderOutstandingView{}
	for id, order := range projects {
		entry, found := a.outstandingCards[order]
		view := orderOutstandingView{Number: order, State: "pending"}
		if found && entry.Scope == scope && (entry.State == "pending" || now.Before(entry.Expires)) && !(r.URL.Query().Get("refresh") == "1" && entry.State != "pending") {
			view = orderOutstandingView{Number: order, State: entry.State, Count: len(entry.Rows), Error: entry.Error}
			if !entry.At.IsZero() {
				at := entry.At
				view.At = &at
			}
			if detail && entry.State == "ready" {
				view.Rows = append([]OrderOutstandingRow{}, entry.Rows...)
				session, authenticated := a.session(r)
				if authenticated && receiptHelperAllowed(&a.state, session) {
					view.Snapshot = receiptHelperSnapshot(entry)
				} else {
					for i := range view.Rows {
						view.Rows[i].Unit = ""
						view.Rows[i].CustomerCode = ""
						view.Rows[i].Key = ""
						view.Rows[i].Purchase = ""
						view.Rows[i].PurchaseDate = ""
						view.Rows[i].Sequence = 0
						view.Rows[i].Part = ""
						view.Rows[i].ItemCode = ""
						view.Rows[i].PrepareBlock = ""
						view.Rows[i].SequenceKnown = false
					}
				}
			}
		}
		views[id] = view
	}
	jsonReply(w, map[string]any{"ok": true, "projects": views})
	return true
}
func (a *App) fetchOutstandingCards(parent context.Context, orders []string, scope string, cfg ERPDBSyncConfig, probe func(context.Context, string) ERPDBReport, isolated bool) {
	ctx, cancel := context.WithTimeout(parent, 35*time.Second)
	defer cancel()
	ctx = context.WithValue(ctx, erpOrderOutstandingBatchKey{}, orders)
	var rows []OrderOutstandingRow
	var err error
	if isolated {
		err = fmt.Errorf("격리 테스트에서는 실제 DB를 조회할 수 없습니다")
	} else {
		report := decodeDBProbe(jsonRaw(probe(ctx, cfg.Folder)))
		rows, err = outstandingReportRows(report)
	}
	if ctx.Err() != nil {
		err = fmt.Errorf("미입고 조회 시간이 초과됐습니다. 다시 조회해 주세요")
	}
	grouped := map[string][]OrderOutstandingRow{}
	for _, order := range orders {
		grouped[order] = []OrderOutstandingRow{}
	}
	if err == nil {
		for _, row := range rows {
			order := strings.TrimSpace(row.Order)
			if _, ok := grouped[order]; !ok {
				err = fmt.Errorf("조회 결과의 수주 연결을 확인할 수 없습니다")
				break
			}
			row.Order = ""
			row.Key = receiptRowKey(row)
			row.PrepareBlock = receiptRowBlockReason(row)
			grouped[order] = append(grouped[order], row)
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.outstandingBusy = false
	if a.outstandingScope() != scope {
		for _, order := range orders {
			if a.outstandingCards[order].Scope == scope {
				delete(a.outstandingCards, order)
			}
		}
		return
	}
	now := time.Now()
	for _, order := range orders {
		entry := orderOutstandingEntry{Scope: scope, State: "ready", Rows: grouped[order], At: now, Expires: now.Add(time.Minute)}
		if err != nil {
			entry.State = "error"
			entry.Error = err.Error()
			entry.Rows = nil
			entry.Expires = now.Add(15 * time.Second)
		}
		a.outstandingCards[order] = entry
	}
}
func (a *App) orderOutstandingCardsMarkup(page, name string, value any) string {
	if name != "work.html" && name != "admin.html" {
		return page
	}
	assets := `<link rel="stylesheet" href="/order-outstanding-v1.4.116.css"><link rel="stylesheet" href="/order-outstanding-cards-v1.4.119.css"><script defer src="/order-outstanding-cards-v1.4.130.js"></script>`
	helper := ""
	if root, ok := value.(map[string]any); ok {
		if csrf, ok := root["CSRF"].(string); ok {
			for _, session := range a.sessions {
				if equal(session.CSRF, csrf) && validAccountSession(&a.state, session) && receiptHelperAllowed(&a.state, session) {
					helper = ` data-helper-csrf="` + template.HTMLEscapeString(csrf) + `"`
				}
			}
		}
	}
	assets += `<link rel="stylesheet" href="/erp-receipt-helper-v1.4.130.css"><link rel="stylesheet" href="/admin-card-layout-v1.4.131.css">`
	marker := `<div id="order-outstanding-config" hidden data-endpoint="/order-outstanding/` + template.HTMLEscapeString(a.state.Board) + `"` + helper + `></div>`
	page = strings.Replace(page, "</head>", assets+"</head>", 1)
	return strings.Replace(page, "</body>", marker+"</body>", 1)
}
