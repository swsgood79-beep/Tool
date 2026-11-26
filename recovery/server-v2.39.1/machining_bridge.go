package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// The bridge and server run under the same Windows user. No public upload or
// unauthenticated HTTP import endpoint is involved.
func machiningBridgeDir() (string, error) {
	h, e := os.UserConfigDir()
	return filepath.Join(h, "AlphaERPBridge"), e
}
func (a *App) machiningBridgeEnabled() bool {
	_, e := os.Stat(filepath.Join(a.dir, "erp-bridge-enabled"))
	return e == nil
}
func (a *App) machiningBridgeStatus() string {
	if !a.machiningBridgeEnabled() {
		return "자동 반영 꺼짐"
	}
	if a.machiningBridgeSetting || a.machiningBridgeSetup {
		return "ERP 최초 설정 진행 중 · 열린 안내 창을 확인해 주세요"
	}
	dir, e := machiningBridgeDir()
	if e == nil {
		if b, e := os.ReadFile(filepath.Join(dir, "status.json")); e == nil && len(b) < 8192 {
			var status struct {
				Message string    `json:"message"`
				At      time.Time `json:"at"`
			}
			if json.Unmarshal(b, &status) == nil && status.Message != "" {
				if time.Since(status.At) > 7*time.Minute {
					return "ERP 자동 연결 상태를 확인 중입니다"
				}
				return status.Message + " · " + a.machiningBridgeMessage
			}
		}
	}
	if a.machiningBridgeMessage != "" {
		return a.machiningBridgeMessage
	}
	return "ERP 자동 연결 시작 대기"
}
func (a *App) importBridgeFile(path string) error {
	st, e := os.Lstat(path)
	if e != nil {
		return e
	}
	if !st.Mode().IsRegular() || st.Size() > 16<<20 {
		return fmt.Errorf("ERP 파일 형식 또는 크기를 확인해 주세요")
	}
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, (16<<20)+1))
	if e != nil {
		return e
	}
	rows, e := parseMachiningWorkbook(b)
	if e != nil {
		return e
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.machiningBridgeEnabled() {
		return nil
	}
	e = a.importMachining(MachiningDraft{Created: time.Now(), Filename: "ERP 자동 연결.xlsx", Hash: digest(b), Rows: rows}, MachiningActor{ID: "erp-bridge", Name: "ERP 자동 연결"})
	if e == nil {
		a.machiningBridgeMessage = "자동 반영 확인 · " + time.Now().In(kst).Format("01.02 15:04:05")
	}
	return e
}
func (a *App) startMachiningBridge(ctx context.Context) {
	if a.testMode {
		return
	}
	a.startManagedERP(ctx)
	go func() {
		tick := time.NewTicker(10 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				a.mu.Lock()
				enabled := a.machiningBridgeEnabled()
				a.mu.Unlock()
				if !enabled {
					continue
				}
				dir, e := machiningBridgeDir()
				if e == nil {
					e = a.importBridgeFile(filepath.Join(dir, "current.xlsx"))
				}
				if e != nil {
					a.mu.Lock()
					if os.IsNotExist(e) {
						a.machiningBridgeMessage = "ERP 자동 연결 대기 · 최초 설정이 필요하면 아래 설정 버튼을 눌러 주세요"
					} else {
						a.machiningBridgeMessage = "자동 반영 보류: " + e.Error()
					}
					a.mu.Unlock()
				}
			}
		}
	}()
}

func (a *App) waitManagedERP() {
	if a.machiningBridgeDone != nil {
		<-a.machiningBridgeDone
	}
}
