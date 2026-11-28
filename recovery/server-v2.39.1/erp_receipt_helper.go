package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type receiptHelperSelection struct {
	Key      string  `json:"key"`
	Quantity float64 `json:"quantity"`
}
type receiptHelperLine struct {
	ItemCode     string  `json:"item_code,omitempty"`
	Order        string  `json:"order"`
	Unit         string  `json:"unit"`
	PurchaseDate string  `json:"purchase_date"`
	Purchase     string  `json:"purchase"`
	Sequence     int     `json:"sequence"`
	Part         string  `json:"part"`
	Item         string  `json:"item"`
	Quantity     float64 `json:"quantity"`
	Remaining    float64 `json:"remaining"`
}
type receiptHelperGroup struct {
	Customer     string              `json:"customer"`
	CustomerCode string              `json:"customer_code"`
	Unit         string              `json:"unit"`
	Purchase     string              `json:"purchase"`
	Lines        []receiptHelperLine `json:"lines"`
}
type receiptHelperTask struct {
	ID      string               `json:"id"`
	At      time.Time            `json:"at"`
	Expires time.Time            `json:"expires"`
	Groups  []receiptHelperGroup `json:"groups"`
}

func receiptHelperAllowed(s *State, session Session) bool {
	return managerAccount(s, session) && adminMenus(s, session).Machining
}

// Display PART_NO is optional and is not part of the purchase-line identity.
func receiptRowBlockReason(row OrderOutstandingRow) string {
	missing := []string{}
	if strings.TrimSpace(row.Unit) == "" {
		missing = append(missing, "사업장")
	}
	if strings.TrimSpace(row.PurchaseDate) == "" {
		missing = append(missing, "발주일")
	}
	if strings.TrimSpace(row.Purchase) == "" {
		missing = append(missing, "발주번호")
	}
	if row.Sequence < 0 || row.Sequence == 0 && !row.SequenceKnown {
		missing = append(missing, "발주 행 번호")
	}
	if len(missing) > 0 {
		return "ERP에서 " + strings.Join(missing, "·") + "를 확인할 수 없어 선택할 수 없습니다"
	}
	return ""
}
func receiptRowKey(row OrderOutstandingRow) string {
	if receiptRowBlockReason(row) != "" {
		return ""
	}
	return digest(jsonRaw([]any{row.Unit, row.PurchaseDate, row.Purchase, row.Sequence}))
}
func receiptHelperSnapshot(entry orderOutstandingEntry) string {
	return digest(jsonRaw([]any{entry.Scope, entry.At, entry.Rows}))
}
func buildReceiptHelper(order string, rows []OrderOutstandingRow, selections []receiptHelperSelection) (receiptHelperTask, error) {
	task := receiptHelperTask{ID: token(), At: time.Now(), Expires: time.Now().Add(20 * time.Minute)}
	if len(selections) == 0 || len(selections) > 50 {
		return task, fmt.Errorf("품목을 1~50개 선택해 주세요")
	}
	index := map[string]OrderOutstandingRow{}
	duplicate := map[string]bool{}
	for _, row := range rows {
		key := receiptRowKey(row)
		if _, ok := index[key]; ok {
			duplicate[key] = true
		}
		index[key] = row
	}
	groups := map[string]*receiptHelperGroup{}
	seen := map[string]bool{}
	for _, choice := range selections {
		row, ok := index[choice.Key]
		if !ok || choice.Key == "" || duplicate[choice.Key] || seen[choice.Key] {
			return task, fmt.Errorf("발주 행을 고유하게 확인할 수 없습니다. 새로고침해 주세요")
		}
		seen[choice.Key] = true
		if row.CustomerCode == "" || row.Customer == "" {
			return task, fmt.Errorf("거래처를 확인할 수 없는 품목은 준비할 수 없습니다")
		}
		if math.IsNaN(choice.Quantity) || math.IsInf(choice.Quantity, 0) || choice.Quantity <= 0 || choice.Quantity > row.Quantity || choice.Quantity > 1e8 {
			return task, fmt.Errorf("입고할 수량은 미입고 잔량 이내로 입력해 주세요")
		}
		key := row.Unit + "\x00" + row.CustomerCode + "\x00" + row.Purchase
		group := groups[key]
		if group == nil {
			group = &receiptHelperGroup{Customer: row.Customer, CustomerCode: row.CustomerCode, Unit: row.Unit, Purchase: row.Purchase}
			groups[key] = group
		}
		group.Lines = append(group.Lines, receiptHelperLine{ItemCode: row.ItemCode, Order: order, Unit: row.Unit, PurchaseDate: row.PurchaseDate, Purchase: row.Purchase, Sequence: row.Sequence, Part: row.Part, Item: row.Item, Quantity: choice.Quantity, Remaining: row.Quantity})
	}
	keys := []string{}
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		g := groups[k]
		sort.Slice(g.Lines, func(i, j int) bool {
			if g.Lines[i].PurchaseDate != g.Lines[j].PurchaseDate {
				return g.Lines[i].PurchaseDate < g.Lines[j].PurchaseDate
			}
			return g.Lines[i].Sequence < g.Lines[j].Sequence
		})
		task.Groups = append(task.Groups, *g)
	}
	return task, nil
}
func (a *App) receiptHelperHTTP(w http.ResponseWriter, r *http.Request, s Session) bool {
	if r.URL.Path != "/erp-receipt-helper/prepare" {
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	if !receiptHelperAllowed(&a.state, s) {
		http.Error(w, "가공 메뉴 권한이 있는 관리자만 이용할 수 있습니다", 403)
		return true
	}
	if r.Method != "POST" {
		http.Error(w, "POST 요청이 필요합니다", 405)
		return true
	}
	if r.ContentLength > 20000 || len(r.FormValue("selections")) > 16000 {
		http.Error(w, "선택한 품목이 너무 많습니다", 400)
		return true
	}
	id, e := strconv.ParseInt(r.FormValue("project"), 10, 64)
	p := a.state.project(id)
	if e != nil || p == nil || p.Hidden || p.Shipping.Completed != nil || p.LegacyCompleted {
		http.Error(w, "진행 중인 수주를 확인해 주세요", 409)
		return true
	}
	entry := a.outstandingCards[strings.TrimSpace(p.Number)]
	if entry.State != "ready" || entry.Scope != a.outstandingScope() || !time.Now().Before(entry.Expires) || !equal(r.FormValue("snapshot"), receiptHelperSnapshot(entry)) {
		http.Error(w, "미입고 자료가 변경됐거나 만료됐습니다. 새로고침 후 다시 선택해 주세요", 409)
		return true
	}
	var choices []receiptHelperSelection
	if json.Unmarshal([]byte(r.FormValue("selections")), &choices) != nil {
		http.Error(w, "선택한 품목을 확인해 주세요", 400)
		return true
	}
	task, err := buildReceiptHelper(p.Number, entry.Rows, choices)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return true
	}
	if r.FormValue("launch") == "1" {
		if time.Since(a.receiptHelperLast) < 10*time.Second {
			http.Error(w, "이미 도우미를 실행했습니다. 회사 PC를 확인해 주세요", 429)
			return true
		}
		launch := a.receiptHelperLaunch
		if launch == nil {
			if a.testMode {
				http.Error(w, "격리 테스트에서는 PC를 조작할 수 없습니다", 409)
				return true
			}
			launch = platformReceiptHelper
		}
		if err = launch(task); err != nil {
			http.Error(w, "PC 도우미를 열지 못했습니다. 회사 PC에서 Alpha가 실행 중인지 확인해 주세요", 503)
			return true
		}
		a.receiptHelperLast = time.Now()
	}
	jsonReply(w, map[string]any{"ok": true, "task": task, "launched": r.FormValue("launch") == "1"})
	return true
}
