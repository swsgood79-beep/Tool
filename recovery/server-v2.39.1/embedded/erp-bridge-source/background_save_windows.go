//go:build windows

package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

// Unlike desktop keyboard input, these messages are addressed only to the
// verified filename Edit of this ERP Save As dialog. No activation is required.
func backgroundNativeSave(dialog uintptr, file string) (bool, map[string]any, error) {
	detail := map[string]any{"method": "Win32 targeted WM_CHAR", "stage": "locate_controls"}
	var edits, buttons []Control
	until := time.Now().Add(5 * time.Second)
	for time.Now().Before(until) {
		edits = nil
		buttons = nil
		bounds := rectOf(dialog)
		cb := syscall.NewCallback(func(h, l uintptr) uintptr {
			c := Control{Handle: uint64(h), ID: int32(api("GetDlgCtrlID", h)), Class: classOf(h), Visible: api("IsWindowVisible", h) != 0, Enabled: api("IsWindowEnabled", h) != 0, Bounds: rectOf(h)}
			if strings.Contains(strings.ToLower(c.Class), "button") {
				c.Label = safeSaveLabel(textOf(h))
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
		if len(edits) == 1 && len(buttons) == 1 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	detail["filename_candidates"], detail["save_candidates"] = len(edits), len(buttons)
	if len(edits) != 1 || len(buttons) != 1 {
		return false, detail, nil
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		return true, detail, fmt.Errorf("새 시험 파일 경로를 확인하지 못했습니다")
	}
	edit, button := uintptr(edits[0].Handle), uintptr(buttons[0].Handle)
	detail["filename_id"] = edits[0].ID
	detail["filename_class"] = edits[0].Class
	detail["stage"] = "select_filename"
	if _, ok := sendTimeout(edit, 0x00b1, 0, ^uintptr(0)); !ok {
		return true, detail, fmt.Errorf("파일 이름 선택 메시지에 응답이 없습니다")
	}
	detail["stage"] = "send_filename_chars"
	end := time.Now().Add(10 * time.Second)
	for i, u := range utf16.Encode([]rune(file)) {
		if time.Now().After(end) || api("IsWindow", edit) == 0 {
			return true, detail, fmt.Errorf("파일 이름 입력이 지연되거나 저장창이 닫혔습니다")
		}
		if _, ok := sendTimeout(edit, 0x0102, uintptr(u), 1); !ok {
			detail["character_index"] = i
			return true, detail, fmt.Errorf("파일 이름 문자 전달에 실패했습니다")
		}
	}
	// Read the live filename back, including Korean and surrogate pairs, before
	// sending the single Save command. Never log other edit contents.
	detail["stage"] = "verify_filename"
	buf := make([]uint16, 32768)
	_, ok := sendTimeout(edit, 0x000d, uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])))
	if !ok || syscall.UTF16ToString(buf) != file {
		return true, detail, fmt.Errorf("파일 이름이 새 경로와 일치하지 않아 저장을 중단했습니다")
	}
	detail["filename_verified"] = true
	if api("GetAncestor", edit, 2) != dialog || api("GetAncestor", button, 2) != dialog || api("IsWindowEnabled", button) == 0 {
		return true, detail, fmt.Errorf("저장창의 입력칸·저장 버튼 상태가 변경되었습니다")
	}
	// BN_CLICKED for the uniquely identified IDOK Save button. Address the dialog
	// directly; BM_CLICK can require an active dialog on some Windows versions.
	detail["stage"] = "request_save"
	if api("PostMessageW", dialog, 0x0111, 1, button) == 0 {
		return true, detail, fmt.Errorf("저장창에 저장 명령을 전달하지 못했습니다")
	}
	detail["stage"] = "save_requested"
	return true, detail, nil
}
