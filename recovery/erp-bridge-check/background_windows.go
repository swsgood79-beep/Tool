//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

func waitBackgroundKey(vk, id uintptr, instruction string) (point, error) {
	if api("RegisterHotKey", 0, id, 0x4000, vk) == 0 {
		return point{}, fmt.Errorf("시험 키를 확보하지 못했습니다. 다른 프로그램의 단축키 설정을 확인해 주세요")
	}
	defer api("UnregisterHotKey", 0, id)
	if api("RegisterHotKey", 0, id+1, 0x4000, 0x1b) == 0 {
		return point{}, fmt.Errorf("취소 키를 확보하지 못했습니다")
	}
	defer api("UnregisterHotKey", 0, id+1)
	message(instruction)
	api("GetAsyncKeyState", vk)
	until := time.Now().Add(90 * time.Second)
	for time.Now().Before(until) {
		if api("GetAsyncKeyState", 0x1b)&0x8000 != 0 {
			return point{}, fmt.Errorf("사용자가 시험을 취소했습니다")
		}
		if api("GetAsyncKeyState", vk)&0x8000 != 0 {
			var p point
			api("GetCursorPos", uintptr(unsafe.Pointer(&p)))
			release := time.Now().Add(2 * time.Second)
			for api("GetAsyncKeyState", vk)&0x8000 != 0 && time.Now().Before(release) {
				time.Sleep(20 * time.Millisecond)
			}
			if api("GetAsyncKeyState", vk)&0x8000 != 0 {
				return point{}, fmt.Errorf("시험 키를 놓고 다시 실행해 주세요")
			}
			return p, nil
		}
		time.Sleep(30 * time.Millisecond)
	}
	return point{}, fmt.Errorf("시험 키 입력을 기다리다 종료했습니다")
}
func backgroundERP() uintptr {
	var found []uintptr
	cb := syscall.NewCallback(func(h, l uintptr) uintptr {
		if api("IsWindowVisible", h) == 0 || !strings.Contains(strings.ToLower(textOf(h)), "primeerp") {
			return 1
		}
		var pid uint32
		api("GetWindowThreadProcessId", h, uintptr(unsafe.Pointer(&pid)))
		if strings.EqualFold(processFile(pid), "PrimeXAR.exe") {
			found = append(found, h)
		}
		return 1
	})
	api("EnumWindows", cb, 0)
	if len(found) == 1 {
		return found[0]
	}
	return 0
}
func backgroundTest() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	root := backgroundERP()
	if root == 0 {
		return fmt.Errorf("PrimeXAR.exe의 ERP 창 한 개를 열어 주세요")
	}
	if api("IsIconic", root) != 0 {
		return fmt.Errorf("처음 위치 지정 시 ERP 최소화를 풀어 주세요")
	}
	var pid uint32
	api("GetWindowThreadProcessId", root, uintptr(unsafe.Pointer(&pid)))
	if saveDialog(pid) != 0 {
		return fmt.Errorf("ERP에 열려 있는 저장창을 취소하고 다시 실행해 주세요")
	}
	p, e := waitBackgroundKey(0x77, 0x4201, "ERP 가려진 창 내보내기 시험 v0.3\n\n1. 발주대비미입고현황 → 가공사업부 → 검색한 목록을 열어 두세요.\n2. 이 안내에서 확인을 누른 뒤 ERP Excel 버튼에 마우스를 올리고 F8을 누르세요.\n\n이번에는 실제 마우스 클릭 대신 창 내부 메시지를 보냅니다. Excel 이외의 위치를 지정하지 마세요.\n취소: Esc")
	if e != nil {
		return e
	}
	packed := uintptr(uint64(uint32(p.X)) | uint64(uint32(p.Y))<<32)
	target := api("WindowFromPoint", packed)
	c := Control{Handle: uint64(target), Class: classOf(target), Visible: api("IsWindowVisible", target) != 0, Enabled: api("IsWindowEnabled", target) != 0, Bounds: rectOf(target)}
	rootBounds := rectOf(root)
	if api("GetAncestor", target, 2) != root || !backgroundTargetValid(c, rootBounds, p.X, p.Y) {
		return fmt.Errorf("ERP 상단 Excel 도구 모음 위치를 확인하지 못했습니다. 화면 크기를 유지하고 다시 지정해 주세요")
	}
	local := p
	if api("ScreenToClient", target, uintptr(unsafe.Pointer(&local))) == 0 {
		return fmt.Errorf("도구 모음 내부 좌표를 확인하지 못했습니다")
	}
	exe, _ := os.Executable()
	folder, e := os.MkdirTemp(filepath.Dir(exe), "ERP-background-test-")
	if e != nil {
		folder, e = os.MkdirTemp(os.TempDir(), "ERP-background-test-")
	}
	if e != nil {
		return e
	}
	report := map[string]any{"tool": "ERP background test v0.3", "when": time.Now().Format(time.RFC3339), "success": false, "mode": "calibrated toolbar messages; targeted filename input/Save; guarded work-window foreground restore; no global mouse/keyboard input", "toolbar_class": c.Class, "toolbar_width": c.Bounds.Right - c.Bounds.Left, "toolbar_height": c.Bounds.Bottom - c.Bounds.Top, "local_x": local.X, "local_y": local.Y}
	finish := func(err error) error {
		if err != nil {
			report["error"] = err.Error()
		}
		raw, _ := json.MarshalIndent(report, "", "  ")
		os.WriteFile(filepath.Join(folder, "ERP-background-test.json"), raw, 0600)
		msg := "가려진 창 내보내기 시험을 마쳤습니다.\nERP-background-test.json을 대화에 첨부해 주세요."
		if err != nil {
			msg = "가려진 창 내보내기 시험이 완료되지 않았습니다.\n\n" + err.Error() + "\n\nERP-background-test.json을 대화에 첨부해 주세요.\nERP 저장창이 남아 있으면 직접 취소해 주세요."
		}
		showResult(msg, folder)
		return nil
	}
	_, e = waitBackgroundKey(0x78, 0x4211, "Excel 위치를 지정했습니다. 아직 내보내기를 실행하지 않았습니다.\n\n1. 확인을 누릅니다.\n2. Chrome 등 다른 창으로 ERP의 Excel 버튼을 완전히 가려 주세요.\n3. ERP 크기·위치·탭은 바꾸지 말고, 다른 창에서 F9를 누르세요.\n\nF9 뒤에는 최대 약 110초간 기다려 주세요. 자동 저장창이나 엑셀이 생길 수 있습니다. 자동으로 열린 시험 엑셀은 수정하지 마세요.\n최소화·화면 잠금은 이번 시험 대상이 아닙니다. 취소: Esc")
	if e != nil {
		return finish(e)
	}
	if api("IsWindow", target) == 0 || api("IsIconic", root) != 0 || !backgroundGeometrySame(c.Bounds, rectOf(target)) || !backgroundGeometrySame(rootBounds, rectOf(root)) || api("GetAncestor", target, 2) != root || api("IsWindowEnabled", target) == 0 || api("IsWindowVisible", target) == 0 {
		return finish(fmt.Errorf("ERP 창이나 도구 모음이 바뀌어 실행을 중단했습니다"))
	}
	before := api("GetForegroundWindow")
	covered := api("GetAncestor", api("WindowFromPoint", packed), 2) != root
	report["covered_at_start"] = covered
	report["erp_foreground_at_start"] = api("GetAncestor", before, 2) == root
	if !covered || api("GetAncestor", before, 2) == root {
		return finish(fmt.Errorf("Excel 버튼이 다른 창에 가려진 상태가 아닙니다. 다른 창으로 덮고 다시 시험해 주세요"))
	}
	if saveDialog(pid) != 0 {
		return finish(fmt.Errorf("ERP 저장창이 이미 열려 있어 중단했습니다"))
	}
	// Observe foreground changes without recording unrelated window titles/processes.
	var mu sync.Mutex
	changes, erpSamples, saveSamples := 0, 0, 0
	restoreAttempts, restoreSuccess := 0, 0
	restoredWindows := map[uintptr]bool{}
	previous := before
	stop, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				fg := api("GetForegroundWindow")
				var p uint32
				api("GetWindowThreadProcessId", fg, uintptr(unsafe.Pointer(&p)))
				mu.Lock()
				if fg != previous {
					changes++
					previous = fg
				}
				if fg == root {
					erpSamples++
				}
				if p == pid && fg != root {
					saveSamples++
				}
				if p == pid && !restoredWindows[fg] {
					attempted, restored := backgroundRestoreWorkWindow(before, pid)
					if attempted {
						restoredWindows[fg] = true
						restoreAttempts++
						if restored {
							restoreSuccess++
						}
					}
				}
				mu.Unlock()
			}
		}
	}()
	observed := false
	observe := func() {
		if observed {
			return
		}
		close(stop)
		<-stopped
		mu.Lock()
		report["foreground_restore_attempts"] = restoreAttempts
		report["foreground_restore_successes"] = restoreSuccess
		report["foreground_change_samples"] = changes
		report["erp_foreground_samples"] = erpSamples
		report["erp_child_foreground_samples"] = saveSamples
		report["foreground_preserved_at_end"] = api("GetForegroundWindow") == before
		mu.Unlock()
		observed = true
	}
	defer observe()
	requestPoint := uintptr(uint32(uint16(local.X)) | uint32(uint16(local.Y))<<16)
	// One calibrated Excel request only. No foreground/global-input fallback.
	for _, msg := range []struct{ id, w uintptr }{{0x0200, 0}, {0x0201, 1}, {0x0202, 0}} {
		if api("PostMessageW", target, msg.id, msg.w, requestPoint) == 0 {
			observe()
			return finish(fmt.Errorf("ERP 도구 모음 실행 메시지를 전달하지 못했습니다"))
		}
	}
	report["export_message_requested"] = true
	var dialog uintptr
	until := time.Now().Add(20 * time.Second)
	for time.Now().Before(until) {
		dialog = saveDialog(pid)
		if dialog != 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if dialog == 0 {
		observe()
		return finish(fmt.Errorf("백그라운드 요청 후 ERP 저장창이 열리지 않았습니다. 이 도구 모음은 메시지 실행을 지원하지 않을 수 있습니다"))
	}
	report["save_dialog_found"] = true
	file := filepath.Join(folder, "ERP_BASE_"+time.Now().Format("20060102_150405")+".xlsx")
	handled, nativeDetail, nativeErr := backgroundNativeSave(dialog, file)
	report["native_save"] = nativeDetail
	if handled && nativeErr != nil {
		observe()
		return finish(nativeErr)
	}
	if !handled {
		details := filepath.Join(folder, ".uia-save.json")
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		script := fmt.Sprintf("$DialogHandle=[long]%d\n$ExportPath=%s\n$ResultPath=%s\n", dialog, escapePS(file), escapePS(details)) + saveScript + "\n"
		cmd := exec.CommandContext(ctx, filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe"), "-NoLogo", "-NoProfile", "-NonInteractive", "-EncodedCommand", encodePSScript(script))
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		err := cmd.Run()
		if raw, e := os.ReadFile(details); e == nil {
			if raw, e = normalizeJSON(raw); e == nil {
				report["uia_save"] = json.RawMessage(raw)
			}
		}
		os.Remove(details)
		if err != nil {
			observe()
			return finish(fmt.Errorf("ERP 저장창의 UI Automation 처리를 완료하지 못했습니다. 실제 클릭으로 전환하지 않고 중단했습니다"))
		}
	}
	until = time.Now().Add(30 * time.Second)
	var last int64 = -1
	stable := 0
	for time.Now().Before(until) {
		if st, e := os.Stat(file); e == nil {
			if st.Size() > 0 && st.Size() == last {
				stable++
			} else {
				stable = 0
			}
			last = st.Size()
			if stable >= 2 && validateWorkbook(file) == nil {
				report["success"] = true
				report["file"] = filepath.Base(file)
				report["bytes"] = st.Size()
				cleanup, cleanupErr := cleanupExportExcel(file, folder)
				report["export_excel_cleanup"] = map[string]any{"stage": cleanup.Stage, "matched": cleanup.Matched, "closed": cleanup.Closed, "remaining": cleanup.Remaining, "skipped_unsaved": cleanup.SkippedUnsaved, "errors": cleanup.Errors, "close_verified": cleanup.CloseVerified, "owned_workbook_was_foreground": cleanup.OwnedForegroundPID != 0}
				if cleanupErr != nil {
					report["export_excel_cleanup_error"] = cleanupErr.Error()
				}
				attempted, restored := backgroundRestoreAfterExport(before, cleanup)
				report["export_excel_work_restore_attempted"] = attempted
				report["export_excel_work_restore_confirmed"] = restored
				if e := validateWorkbook(file); e != nil {
					report["success"] = false
					observe()
					return finish(fmt.Errorf("엑셀 닫기 처리 후 저장 파일 검증 실패: %v", e))
				}
				observe()
				return finish(nil)
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	report["save_dialog_still_open"] = saveDialog(pid) != 0
	if st, e := os.Stat(file); e == nil {
		report["file_exists"] = true
		report["bytes"] = st.Size()
		if e := validateWorkbook(file); e != nil {
			report["validation_error"] = e.Error()
		}
	} else {
		report["file_exists"] = false
	}
	observe()
	return finish(fmt.Errorf("지정 폴더에서 정상 XLSX 파일을 확인하지 못했습니다"))
}
