package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type erpDBSyncKey struct{}
type ERPDBSyncConfig struct {
	Enabled bool   `json:"enabled"`
	Folder  string `json:"folder"`
	From    string `json:"from"`
	To      string `json:"to"`
}

func (a *App) erpDBSyncConfig() ERPDBSyncConfig {
	var c ERPDBSyncConfig
	b, e := os.ReadFile(filepath.Join(a.dir, "erp-db-sync.json"))
	if e != nil || len(b) > 4096 || json.Unmarshal(b, &c) != nil {
		return ERPDBSyncConfig{}
	}
	if _, e = erpDBPreviewDates(c.From, c.To); e != nil || !validERPFolder(c.Folder) {
		return ERPDBSyncConfig{}
	}
	return c
}
func (a *App) erpDBSyncView() map[string]any {
	c := a.erpDBSyncConfig()
	message := a.erpDBSyncMessage
	if message == "" {
		if c.Enabled {
			message = "DB 자동 연결 시작 대기"
		} else {
			message = "DB 자동 연결 꺼짐"
		}
	}
	return map[string]any{"enabled": c.Enabled, "from": c.From, "to": c.To, "folder": c.Folder, "message": message, "busy": a.erpDBBusy}
}
func (a *App) erpDBSyncHTTP(w http.ResponseWriter, r *http.Request, s Session) bool {
	if r.URL.Path != "/machining-admin/db-sync" {
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	if s.AccountID != "" {
		http.Error(w, "대표 관리자만 설정할 수 있습니다", 403)
		return true
	}
	if r.Method == "GET" {
		jsonReply(w, a.erpDBSyncView())
		return true
	}
	if r.Method != "POST" {
		http.Error(w, "지원하지 않는 요청입니다", 405)
		return true
	}
	c := a.erpDBSyncConfig()
	switch r.FormValue("action") {
	case "disable":
		c.Enabled = false
	case "enable":
		if a.testMode && a.erpDBProbe == nil {
			http.Error(w, "격리 테스트에서는 실제 DB를 조회하지 않습니다", 409)
			return true
		}
		d, e := erpDBPreviewDates(r.FormValue("from"), r.FormValue("to"))
		if e != nil {
			http.Error(w, e.Error(), 400)
			return true
		}
		folder := strings.TrimSpace(r.FormValue("folder"))
		if !validERPFolder(folder) {
			http.Error(w, "ERP 설치 폴더를 확인해 주세요", 400)
			return true
		}
		c = ERPDBSyncConfig{true, folder, d.From, d.To}
	default:
		http.Error(w, "설정 요청을 확인해 주세요", 400)
		return true
	}
	if c.Enabled {
		if e := os.Remove(filepath.Join(a.dir, "erp-bridge-enabled")); e != nil && !os.IsNotExist(e) {
			http.Error(w, "기존 화면 수집 설정 해제 실패", 500)
			return true
		}
	}
	if e := atomicFile(filepath.Join(a.dir, "erp-db-sync.json"), jsonRaw(c)); e != nil {
		http.Error(w, "DB 연결 설정 저장 실패", 500)
		return true
	}
	a.erpDBSyncGeneration++
	a.erpDBSyncNow = c.Enabled
	if c.Enabled {
		a.erpDBSyncMessage = "DB 조회 대기 · ERP 화면을 켜두지 않아도 됩니다"
	} else {
		a.erpDBSyncMessage = "DB 자동 연결 꺼짐 · 기존 작업 기록 유지"
	}
	jsonReply(w, a.erpDBSyncView())
	return true
}

// All lines must validate before a single work record is changed. A missing
// robot is never guessed from another row with the same base part number.
func machiningDBRows(r ERPDBReport) ([]MachiningSource, error) {
	if !r.Connected || r.Status != "purchase_preview" || r.Truncated {
		return nil, fmt.Errorf("DB 전체 조회가 완료되지 않아 기존 목록을 유지합니다")
	}
	rows := []MachiningSource{}
	seen := map[string]bool{}
	for _, v := range r.PurchaseRows {
		if !strings.EqualFold(strings.TrimSpace(v.ItemName), "BASE") {
			continue
		}
		if v.Quantity < 0 || v.Quantity > 1000000 || v.Received < 0 || v.Received > v.Quantity {
			return nil, fmt.Errorf("발주·입고 수량을 확인할 수 없어 반영을 보류했습니다")
		}
		if v.Quantity == v.Received {
			continue
		}
		if v.WarehouseStatus != "0" {
			return nil, fmt.Errorf("미입고 상태를 확인할 수 없어 반영을 보류했습니다")
		}
		for _, s := range []string{v.Unit, v.PurchaseDate, v.Purchase, v.Order, v.Project, v.Item} {
			if !validText(s, 200) {
				return nil, fmt.Errorf("발주 필수 정보가 없어 반영을 보류했습니다")
			}
		}
		if v.Sequence < 1 {
			return nil, fmt.Errorf("발주 행 번호가 없어 반영을 보류했습니다")
		}
		key := string(jsonRaw([]any{v.Unit, v.PurchaseDate, v.Purchase, v.Sequence}))
		if seen[key] {
			return nil, fmt.Errorf("발주 행이 중복 조회되어 반영을 보류했습니다")
		}
		seen[key] = true
		if v.RobotMatches != 1 || !validText(v.Robot, 200) {
			return nil, fmt.Errorf("로봇품번을 확정하지 못한 항목이 있어 반영을 보류했습니다. DB 발주 미리보기 결과를 확인해 주세요")
		}
		due, e := time.Parse("2006-01-02", strings.TrimSpace(v.Due))
		if e != nil {
			return nil, fmt.Errorf("입고예정일 형식을 확인할 수 없어 반영을 보류했습니다")
		}
		rows = append(rows, MachiningSource{Row: v.Sequence, Purchase: v.Purchase, Order: v.Order, Project: v.Project, Robot: v.Robot, Part: v.Item, Due: due.Format("2006-01-02"), Quantity: v.Quantity, Received: v.Received})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("BASE 미입고 항목이 없어 기존 목록을 유지합니다")
	}
	return rows, nil
}
func (a *App) syncERPDB(ctx context.Context) {
	a.mu.Lock()
	c := a.erpDBSyncConfig()
	if !c.Enabled || a.erpDBBusy {
		a.mu.Unlock()
		return
	}
	gen := a.erpDBSyncGeneration
	a.erpDBSyncNow = false
	a.erpDBBusy = true
	a.erpDBPreview = true
	a.erpDBSyncMessage = "DB 조회 중 · 기존 작업 기록 유지"
	probe := a.erpDBProbe
	if probe == nil {
		probe = platformERPDBProbe
	}
	a.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	ctx = context.WithValue(ctx, erpDBPreviewKey{}, erpDBPreviewRange{c.From, c.To})
	ctx = context.WithValue(ctx, erpDBSyncKey{}, true)
	report := decodeDBProbe(jsonRaw(probe(ctx, c.Folder)))
	report.DateFrom = c.From
	report.DateTo = c.To
	report.At = time.Now()
	report.Version = releaseVersion
	rows, err := machiningDBRows(report)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.erpDBBusy = false
	a.erpDBReport = &report
	// A disabled/reconfigured sync or canceled app must never apply its old result.
	if a.erpDBSyncGeneration != gen || a.erpDBSyncConfig() != c {
		return
	}
	if ctx.Err() != nil {
		a.erpDBSyncMessage = "DB 조회 시간 초과 · 기존 작업 기록 유지 · 다음 주기에 재시도합니다"
		return
	}
	if err == nil {
		data := jsonRaw(struct {
			From, To string
			Rows     []MachiningSource
		}{c.From, c.To, rows})
		err = a.importMachining(MachiningDraft{Created: time.Now(), Filename: "ERP DB 자동 연결", Hash: digest(data), Rows: rows}, MachiningActor{ID: "erp-db", Name: "ERP DB 자동 연결"})
	}
	if err != nil {
		a.erpDBSyncMessage = "자동 반영 보류 · " + err.Error()
		return
	}
	units := 0
	for _, v := range rows {
		units += v.Quantity - v.Received
	}
	a.erpDBSyncMessage = fmt.Sprintf("DB 자동 연결 정상 · %s · %d품목 / %d개 · 5분마다 갱신", time.Now().In(kst).Format("01.02 15:04:05"), len(machiningGroups(rows)), units)
}
func (a *App) startDBSync(ctx context.Context) {
	a.erpDBSyncDone = make(chan struct{})
	go func() {
		defer close(a.erpDBSyncDone)
		tick := time.NewTicker(2 * time.Second)
		defer tick.Stop()
		var next time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-tick.C:
				a.mu.Lock()
				ready := a.erpDBSyncConfig().Enabled && !a.erpDBBusy && (a.erpDBSyncNow || !now.Before(next))
				a.mu.Unlock()
				if ready {
					a.syncERPDB(ctx)
					next = now.Add(5 * time.Minute)
				}
			}
		}
	}()
}
