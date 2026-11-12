package main

import (
	"testing"
	"unsafe"
)

func TestWin32X64ABIFromOriginalRTTI(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("x64 target ABI")
	}
	checks := []struct {
		name      string
		got, want uintptr
	}{
		{"WNDCLASSEXW.size", unsafe.Sizeof(winClass{}), 80}, {"WNDCLASSEXW.wndProc", unsafe.Offsetof(winClass{}.Proc), 8}, {"WNDCLASSEXW.class", unsafe.Offsetof(winClass{}.Class), 64},
		{"MSG.size", unsafe.Sizeof(winMessage{}), 48}, {"MSG.wParam", unsafe.Offsetof(winMessage{}.WParam), 16}, {"MSG.point", unsafe.Offsetof(winMessage{}.Pt), 36},
		{"PAINTSTRUCT.size", unsafe.Sizeof(winPaint{}), 72}, {"PAINTSTRUCT.rect", unsafe.Offsetof(winPaint{}.Paint), 12},
		{"NOTIFYICONDATAW.size", unsafe.Sizeof(winNotifyIcon{}), 976}, {"NOTIFYICONDATAW.window", unsafe.Offsetof(winNotifyIcon{}.Window), 8}, {"NOTIFYICONDATAW.tip", unsafe.Offsetof(winNotifyIcon{}.Tip), 40}, {"NOTIFYICONDATAW.state", unsafe.Offsetof(winNotifyIcon{}.State), 296}, {"NOTIFYICONDATAW.info", unsafe.Offsetof(winNotifyIcon{}.Info), 304}, {"NOTIFYICONDATAW.version", unsafe.Offsetof(winNotifyIcon{}.Version), 816}, {"NOTIFYICONDATAW.title", unsafe.Offsetof(winNotifyIcon{}.Title), 820}, {"NOTIFYICONDATAW.infoFlags", unsafe.Offsetof(winNotifyIcon{}.InfoFlags), 948}, {"NOTIFYICONDATAW.guid", unsafe.Offsetof(winNotifyIcon{}.GUID), 952}, {"NOTIFYICONDATAW.balloon", unsafe.Offsetof(winNotifyIcon{}.Balloon), 968},
		{"DRAWITEMSTRUCT.size", unsafe.Sizeof(winDrawItem{}), 64}, {"DRAWITEMSTRUCT.window", unsafe.Offsetof(winDrawItem{}.Window), 24}, {"DRAWITEMSTRUCT.dc", unsafe.Offsetof(winDrawItem{}.DC), 32}, {"DRAWITEMSTRUCT.rect", unsafe.Offsetof(winDrawItem{}.Rect), 40},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %d want %d", c.name, c.got, c.want)
		}
	}
}
