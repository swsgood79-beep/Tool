//go:build windows

package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
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
	detail["stage"] = "type_filename"
	// WM_SETTEXT can change visible text without updating the shell dialog's
	// cached filename. Real keyboard input delivers the normal edit notifications.
	if err := typeFilename(dialog, edit, file); err != nil {
		return true, detail, err
	}
	time.Sleep(300 * time.Millisecond)
	buf := make([]uint16, 32768)
	detail["stage"] = "verify_filename"
	_, ok := sendTimeout(edit, 0x000d, uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])))
	if !ok || syscall.UTF16ToString(buf) != file {
		return true, detail, fmt.Errorf("입력된 파일 이름을 확인하지 못해 저장을 중단했습니다")
	}
	detail["stage"] = "click_save"
	if err := clickDialogControl(dialog, uintptr(buttons[0].Handle)); err != nil {
		return true, detail, err
	}
	detail["stage"] = "save_requested"
	return true, detail, nil
}

// This test tool is distributed as Windows amd64. INPUT is 40 bytes on x64.
type keyboardInput struct {
	Kind      uint32
	Padding   uint32
	VK        uint16
	Scan      uint16
	Flags     uint32
	Timestamp uint32
	Extra     uintptr
	Tail      [8]byte
}

var _ [40 - unsafe.Sizeof(keyboardInput{})]byte
var _ [unsafe.Sizeof(keyboardInput{}) - 40]byte

func clickDialogControl(dialog, control uintptr) error {
	api("SetForegroundWindow", dialog)
	if api("GetForegroundWindow") != dialog {
		return fmt.Errorf("저장창이 앞에 있지 않아 입력을 중단했습니다")
	}
	r := rectOf(control)
	x, y := (r.Left+r.Right)/2, (r.Top+r.Bottom)/2
	api("SetCursorPos", uintptr(uint32(x)), uintptr(uint32(y)))
	packed := uintptr(uint64(uint32(x)) | uint64(uint32(y))<<32)
	if api("WindowFromPoint", packed) != control {
		return fmt.Errorf("입력 대상이 가려져 중단했습니다")
	}
	api("mouse_event", 2, 0, 0, 0, 0)
	api("mouse_event", 4, 0, 0, 0, 0)
	return nil
}
func typeFilename(dialog, edit uintptr, file string) error {
	if err := clickDialogControl(dialog, edit); err != nil {
		return err
	}
	time.Sleep(150 * time.Millisecond)
	// Confirm keyboard focus belongs to the identified filename Edit.
	info := struct {
		Size                                               uint32
		Flags                                              uint32
		Active, Focus, Capture, MenuOwner, MoveSize, Caret uintptr
		CaretRect                                          Rect
	}{}
	info.Size = uint32(unsafe.Sizeof(info))
	thread := api("GetWindowThreadProcessId", dialog, 0)
	if api("GetGUIThreadInfo", thread, uintptr(unsafe.Pointer(&info))) == 0 || info.Focus != edit || api("GetForegroundWindow") != dialog {
		return fmt.Errorf("파일 이름 칸의 키보드 포커스를 확인하지 못했습니다")
	}
	inputs := []keyboardInput{{Kind: 1, VK: 0x11}, {Kind: 1, VK: 0x41}, {Kind: 1, VK: 0x41, Flags: 2}, {Kind: 1, VK: 0x11, Flags: 2}}
	for _, u := range utf16.Encode([]rune(file)) {
		inputs = append(inputs, keyboardInput{Kind: 1, Scan: u, Flags: 4}, keyboardInput{Kind: 1, Scan: u, Flags: 6})
	}
	sent := api("SendInput", uintptr(len(inputs)), uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(inputs[0]))
	runtime.KeepAlive(inputs)
	if sent != uintptr(len(inputs)) {
		// Release possible Ctrl state after a partial SendInput failure.
		api("keybd_event", 0x11, 0, 2, 0)
		return fmt.Errorf("파일 이름 키보드 입력이 완료되지 않았습니다")
	}
	return nil
}
