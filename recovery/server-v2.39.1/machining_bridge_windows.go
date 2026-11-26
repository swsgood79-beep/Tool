//go:build windows

package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

//go:embed embedded/erp-bridge.payload
var managedERPHelper []byte

func (a *App) managedERPPath() (string, error) {
	hash := sha256.Sum256(managedERPHelper)
	dir := filepath.Join(a.dir, "erp-helper")
	if e := os.MkdirAll(dir, 0700); e != nil {
		return "", e
	}
	path := filepath.Join(dir, "erp-bridge-"+hex.EncodeToString(hash[:8])+".exe")
	current, e := os.ReadFile(path)
	if e == nil && sha256.Sum256(current) == hash {
		return path, nil
	}
	if e = atomicFile(path, managedERPHelper); e != nil {
		return "", e
	}
	current, e = os.ReadFile(path)
	if e != nil || sha256.Sum256(current) != hash {
		return "", fmt.Errorf("내장 ERP 연결 구성요소 확인 실패")
	}
	return path, nil
}
func (a *App) runManagedERP(ctx context.Context, setup bool) error {
	path, e := a.managedERPPath()
	if e != nil {
		a.mu.Lock()
		a.machiningBridgeMessage = e.Error()
		a.mu.Unlock()
		return e
	}
	job, e := newERPJob()
	if e != nil {
		a.mu.Lock()
		a.machiningBridgeMessage = e.Error()
		a.mu.Unlock()
		return e
	}
	defer syscall.CloseHandle(job)
	mode := "--managed-capture"
	flags := uint32(0x08000000)
	if setup {
		mode = "--managed-setup"
		flags = 0x10
	}
	cmd := exec.CommandContext(ctx, path, mode)
	cmd.Dir = filepath.Dir(path)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: !setup, CreationFlags: flags}
	if e := cmd.Start(); e != nil {
		a.mu.Lock()
		a.machiningBridgeMessage = "ERP 자동 실행 실패: " + e.Error()
		a.mu.Unlock()
		return e
	}
	if e := attachERPJob(job, cmd.Process.Pid); e != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		a.mu.Lock()
		a.machiningBridgeMessage = e.Error()
		a.mu.Unlock()
		return e
	}
	return cmd.Wait()
}
func (a *App) startManagedERP(ctx context.Context) {
	if a.testMode {
		return
	}
	a.machiningBridgeDone = make(chan struct{})
	go func() {
		defer close(a.machiningBridgeDone)
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		a.superviseERP(ctx, tick.C, a.runManagedERP)
	}()
}
