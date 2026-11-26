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

//go:embed save-dialog.ps1
var saveScript string

type point struct{ X, Y int32 }

func escapePS(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func message(s string)         { showResult(s, "") }
func findERP() uintptr {
	var result uintptr
	cb := syscall.NewCallback(func(h, l uintptr) uintptr {
		if api("IsWindowVisible", h) != 0 && strings.Contains(strings.ToLower(textOf(h)), "primeerp") {
			var pid uint32
			api("GetWindowThreadProcessId", h, uintptr(unsafe.Pointer(&pid)))
			if pid != uint32(os.Getpid()) {
				result = h
				return 0
			}
		}
		return 1
	})
	api("EnumWindows", cb, 0)
	return result
}
func saveDialog(pid uint32) uintptr {
	var result uintptr
	cb := syscall.NewCallback(func(h, l uintptr) uintptr {
		if api("IsWindowVisible", h) == 0 || classOf(h) != "#32770" {
			return 1
		}
		var p uint32
		api("GetWindowThreadProcessId", h, uintptr(unsafe.Pointer(&p)))
		title := textOf(h)
		if p == pid && (strings.Contains(title, "다른 이름으로 저장") || strings.EqualFold(title, "Save As")) {
			result = h
			return 0
		}
		return 1
	})
	api("EnumWindows", cb, 0)
	return result
}
func exportTest() error {
	h := findERP()
	if h == 0 {
		return fmt.Errorf("ERP 발주대비미입고현황 화면을 먼저 열어 주세요")
	}
	var pid uint32
	api("GetWindowThreadProcessId", h, uintptr(unsafe.Pointer(&pid)))
	if saveDialog(pid) != 0 {
		return fmt.Errorf("현재 떠 있는 저장창을 취소하고, 이 도구를 다시 실행해 주세요")
	}
	message("ERP 엑셀 저장 테스트\n\n1. ERP에서 발주대비미입고현황을 열고 가공사업부를 조회해 두세요.\n2. 확인을 누른 뒤, ERP 오른쪽 위 Excel 버튼 위에 마우스를 올리고 F8을 누르세요.\n\n선택한 위치를 클릭하고 새 XLSX 파일 저장을 시도합니다. Excel 이외의 버튼을 선택하지 마세요.\n취소는 Esc, 대기 시간은 60초입니다.")
	api("ShowWindowAsync", h, 9)
	api("SetForegroundWindow", h)
	time.Sleep(300 * time.Millisecond)
	api("GetAsyncKeyState", 0x77)
	var cursor point
	selected := false
	until := time.Now().Add(60 * time.Second)
	for time.Now().Before(until) {
		if api("GetAsyncKeyState", 0x1b)&0x8000 != 0 {
			return fmt.Errorf("사용자가 테스트를 취소했습니다")
		}
		if api("GetAsyncKeyState", 0x77)&0x8000 != 0 {
			api("GetCursorPos", uintptr(unsafe.Pointer(&cursor)))
			selected = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !selected {
		return fmt.Errorf("F8 입력을 기다리다 종료했습니다")
	}
	bounds := rectOf(h)
	x, y := cursor.X-bounds.Left, cursor.Y-bounds.Top
	packed := uintptr(uint64(uint32(cursor.X)) | uint64(uint32(cursor.Y))<<32)
	under := api("WindowFromPoint", packed)
	if api("GetAncestor", under, 2) != h || x < (bounds.Right-bounds.Left)*55/100 || y < 45 || y > 150 {
		return fmt.Errorf("ERP 오른쪽 위 Excel 버튼 위치가 아닙니다. 화면을 펼친 상태에서 다시 지정해 주세요")
	}
	exe, _ := os.Executable()
	folder, e := os.MkdirTemp(filepath.Dir(exe), "ERP-export-test-")
	if e != nil {
		folder, e = os.MkdirTemp(os.TempDir(), "ERP-export-test-")
	}
	if e != nil {
		return e
	}
	report := map[string]any{"tool": "ERP export test v0.1", "when": time.Now().Format(time.RFC3339), "success": false, "mode": "one user-calibrated Excel click and Save As; no purchasing/receipt edits", "offset_x": x, "offset_y": y, "window_width": bounds.Right - bounds.Left, "window_height": bounds.Bottom - bounds.Top}
	finish := func(err error) error {
		if err != nil {
			report["error"] = err.Error()
		}
		raw, _ := json.MarshalIndent(report, "", "  ")
		os.WriteFile(filepath.Join(folder, "ERP-export-test.json"), raw, 0600)
		msg := "엑셀 저장 테스트를 마쳤습니다.\n\nERP-export-test.json 파일을 대화에 첨부해 주세요.\n실제로 저장된 엑셀 파일도 같은 폴더에서 확인할 수 있습니다."
		if err != nil {
			msg = "엑셀 저장 테스트가 완료되지 않았습니다.\n\n" + err.Error() + "\n\n폴더의 ERP-export-test.json 파일을 대화에 첨부해 주세요."
		}
		showResult(msg, folder)
		return nil
	}
	if api("GetForegroundWindow") != h {
		return finish(fmt.Errorf("ERP가 앞에 있지 않아 클릭을 중단했습니다"))
	}
	api("SetCursorPos", uintptr(uint32(cursor.X)), uintptr(uint32(cursor.Y)))
	if api("GetAncestor", api("WindowFromPoint", packed), 2) != h {
		return finish(fmt.Errorf("대상 위치가 다른 창에 가려져 중단했습니다"))
	}
	api("mouse_event", 2, 0, 0, 0, 0)
	api("mouse_event", 4, 0, 0, 0, 0)
	var dialog uintptr
	until = time.Now().Add(20 * time.Second)
	for time.Now().Before(until) {
		dialog = saveDialog(pid)
		if dialog != 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if dialog == 0 {
		return finish(fmt.Errorf("ERP의 다른 이름으로 저장 창을 찾지 못했습니다"))
	}
	file := filepath.Join(folder, "ERP_BASE_"+time.Now().Format("20060102_150405")+".xlsx")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	script := fmt.Sprintf("$DialogHandle=[long]%d\n$ExportPath=%s\n", dialog, escapePS(file)) + saveScript + "\n"
	cmd := exec.CommandContext(ctx, filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe"), "-NoLogo", "-NoProfile", "-NonInteractive", "-EncodedCommand", encodePSScript(script))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if e = cmd.Run(); e != nil {
		return finish(fmt.Errorf("저장창의 파일 이름/저장 버튼을 자동 처리하지 못했습니다. 저장창을 확인해 주세요"))
	}
	until = time.Now().Add(30 * time.Second)
	var last int64 = -1
	stable := 0
	for time.Now().Before(until) {
		st, err := os.Stat(file)
		if err == nil {
			if st.Size() == last && st.Size() > 0 {
				stable++
			} else {
				stable = 0
			}
			last = st.Size()
			if stable >= 2 && validateWorkbook(file) == nil {
				report["success"] = true
				report["file"] = filepath.Base(file)
				report["bytes"] = st.Size()
				return finish(nil)
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return finish(fmt.Errorf("유효한 XLSX 파일 저장을 확인하지 못했습니다"))
}
