//go:build windows

package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

func sendTimeout(h, msg, w, l uintptr) (uintptr, bool) {
	var value uintptr
	ok := api("SendMessageTimeoutW", h, msg, w, l, 2, 2000, uintptr(unsafe.Pointer(&value)))
	return value, ok != 0
}
func tryNativeSave(dialog uintptr, file string) (bool, map[string]any, error) {
	detail := map[string]any{"method": "Win32", "stage": "locate_controls"}
	bounds := rectOf(dialog)
	var edits, buttons []Control
	cb := syscall.NewCallback(func(h, l uintptr) uintptr {
		c := Control{Handle: uint64(h), ID: int32(api("GetDlgCtrlID", h)), Class: classOf(h), Visible: api("IsWindowVisible", h) != 0, Enabled: api("IsWindowEnabled", h) != 0, Bounds: rectOf(h)}
		if strings.Contains(strings.ToLower(c.Class), "button") {
			name := textOf(h)
			if strings.Contains(name, "저장") || strings.Contains(strings.ToLower(name), "save") {
				c.Label = "Save"
			}
		}
		if filenameCandidate(c, bounds) {
			edits = append(edits, c)
		}
		if saveButtonCandidate(c, bounds) {
			buttons = append(buttons, c)
		}
		return 1
	})
	api("EnumChildWindows", dialog, cb, 0)
	detail["filename_candidates"] = len(edits)
	detail["save_candidates"] = len(buttons)
	if len(edits) != 1 || len(buttons) != 1 {
		return false, detail, nil
	}
	detail["filename_id"] = edits[0].ID
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		return true, detail, fmt.Errorf("새 저장 경로를 확인할 수 없습니다")
	}
	edit := uintptr(edits[0].Handle)
	target, _ := syscall.UTF16PtrFromString(file)
	detail["stage"] = "set_filename"
	result, ok := sendTimeout(edit, 0x000c, 0, uintptr(unsafe.Pointer(target)))
	runtime.KeepAlive(target)
	if !ok || result == 0 {
		return true, detail, fmt.Errorf("파일 이름 입력 단계에서 응답을 받지 못했습니다")
	}
	buf := make([]uint16, 32768)
	detail["stage"] = "verify_filename"
	_, ok = sendTimeout(edit, 0x000d, uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])))
	if !ok || syscall.UTF16ToString(buf) != file {
		return true, detail, fmt.Errorf("입력된 파일 이름을 확인하지 못해 저장을 중단했습니다")
	}
	detail["stage"] = "click_save"
	api("SetForegroundWindow", dialog)
	if api("PostMessageW", uintptr(buttons[0].Handle), 0x00f5, 0, 0) == 0 {
		return true, detail, fmt.Errorf("저장 버튼 실행 요청을 전달하지 못했습니다")
	}
	detail["stage"] = "save_requested"
	return true, detail, nil
}
