//go:build windows

package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

//go:embed close-export-excel.ps1
var closeExportScript string

type exportExcelResult struct {
	Stage              string `json:"stage"`
	Matched            int    `json:"matched"`
	Closed             int    `json:"closed"`
	Remaining          int    `json:"remaining"`
	SkippedUnsaved     int    `json:"skipped_unsaved"`
	Errors             int    `json:"errors"`
	OwnedForegroundPID uint32 `json:"owned_foreground_pid"`
	CloseVerified      bool   `json:"close_verified"`
}

func cleanupExportExcel(file, folder string) (exportExcelResult, error) {
	var result exportExcelResult
	path := filepath.Join(folder, ".export-excel.json")
	defer os.Remove(path)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	script := fmt.Sprintf("$ExportPath=%s\n$ResultPath=%s\n", escapePS(file), escapePS(path)) + closeExportScript + "\n"
	cmd := exec.CommandContext(ctx, filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe"), "-NoLogo", "-NoProfile", "-NonInteractive", "-STA", "-EncodedCommand", encodePSScript(script))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Run()
	if raw, e := os.ReadFile(path); e == nil && len(raw) < 32768 {
		if data, e := normalizeJSON(raw); e == nil {
			if e := json.Unmarshal(data, &result); e != nil {
				return result, fmt.Errorf("엑셀 닫기 결과 형식을 확인하지 못했습니다")
			}
		}
	}
	if ctx.Err() != nil {
		return result, fmt.Errorf("내보낸 엑셀 확인 시간이 초과됐습니다. 다른 문서는 닫지 않았습니다")
	}
	if err != nil {
		return result, fmt.Errorf("내보낸 엑셀의 자동 닫기 결과를 확인하지 못했습니다")
	}
	return result, nil
}
func backgroundRestoreAfterExport(previous uintptr, result exportExcelResult) (attempted, restored bool) {
	if previous == 0 || api("IsWindow", previous) == 0 || api("IsIconic", previous) != 0 || api("GetAncestor", previous, 2) != previous {
		return false, false
	}
	if !strings.EqualFold(processFile(result.OwnedForegroundPID), "EXCEL.EXE") {
		return false, false
	}
	current := api("GetForegroundWindow")
	if current == previous {
		return false, true
	}
	var currentPID, workPID uint32
	api("GetWindowThreadProcessId", current, uintptr(unsafe.Pointer(&currentPID)))
	api("GetWindowThreadProcessId", previous, uintptr(unsafe.Pointer(&workPID)))
	if !shouldRestoreAfterExport(currentPID, workPID, result.OwnedForegroundPID, uint32(os.Getpid()), result.CloseVerified, result.OwnedForegroundPID != 0) {
		return false, false
	}
	if api("GetForegroundWindow") != current {
		return false, false
	}
	api("SetForegroundWindow", previous)
	return true, api("GetForegroundWindow") == previous
}
