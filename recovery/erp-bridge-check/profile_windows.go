//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var bridgeManagedCapture bool

func bridgeDPI(root uintptr) uint32 {
	p := user32.NewProc("GetDpiForWindow")
	if p.Find() != nil {
		return 96
	}
	r, _, _ := p.Call(root)
	return uint32(r)
}
func saveBridgeProfile() error {
	if bridgeCal == nil {
		return fmt.Errorf("ERP 최초 설정이 없습니다")
	}
	c := bridgeCal
	root := c.bounds
	t := c.c.Bounds
	p := bridgeProfile{Version: 1, RootWidth: root.Right - root.Left, RootHeight: root.Bottom - root.Top, Toolbar: Rect{t.Left - root.Left, t.Top - root.Top, t.Right - root.Left, t.Bottom - root.Top}, ExcelX: c.local.X, ExcelY: c.local.Y, SearchX: c.search.X, SearchY: c.search.Y, DPI: bridgeDPI(c.root)}
	if e := validateProfile(p); e != nil {
		return e
	}
	b, e := json.Marshal(p)
	if e != nil {
		return e
	}
	return bridgeWrite("profile.json", b)
}
func loadBridgeProfile() error {
	dir, e := bridgeFolder()
	if e != nil {
		return e
	}
	b, e := os.ReadFile(filepath.Join(dir, "profile.json"))
	if e != nil {
		return fmt.Errorf("관리자 화면에서 ERP 최초 설정을 진행해 주세요")
	}
	var p bridgeProfile
	if len(b) > 8192 || json.Unmarshal(b, &p) != nil {
		return fmt.Errorf("ERP 설정 파일을 확인할 수 없습니다")
	}
	if e = validateProfile(p); e != nil {
		return e
	}
	root := backgroundERP()
	if root == 0 {
		return fmt.Errorf("ERP 실행 대기 · 발주대비미입고현황을 열어 주세요")
	}
	if api("IsIconic", root) != 0 {
		return fmt.Errorf("ERP 최소화를 풀어 주세요 · 기존 가공 기록은 유지됩니다")
	}
	bounds := rectOf(root)
	dpi := bridgeDPI(root)
	type target struct {
		h uintptr
		c Control
	}
	var found []target
	cb := syscall.NewCallback(func(h, l uintptr) uintptr {
		c := Control{Handle: uint64(h), Class: classOf(h), Visible: api("IsWindowVisible", h) != 0, Enabled: api("IsWindowEnabled", h) != 0, Bounds: rectOf(h)}
		if p.matches(bounds, c, dpi) && api("GetAncestor", h, 2) == root {
			found = append(found, target{h, c})
		}
		return 1
	})
	api("EnumChildWindows", root, cb, 0)
	if len(found) != 1 {
		return fmt.Errorf("ERP 화면 크기나 배치가 달라졌습니다. ERP 최초 설정을 다시 진행해 주세요")
	}
	t := found[0]
	screen := point{X: t.c.Bounds.Left + p.ExcelX, Y: t.c.Bounds.Top + p.ExcelY}
	// Verify the actual live HWND belongs to the newly discovered ERP process.
	var rootPID, toolbarPID uint32
	api("GetWindowThreadProcessId", root, uintptr(unsafe.Pointer(&rootPID)))
	api("GetWindowThreadProcessId", t.h, uintptr(unsafe.Pointer(&toolbarPID)))
	if rootPID == 0 || rootPID != toolbarPID {
		return fmt.Errorf("ERP 도구 모음 소속 확인 실패")
	}
	bridgeCal = &bridgeCalibration{root: root, target: t.h, c: t.c, bounds: bounds, screen: screen, local: point{X: p.ExcelX, Y: p.ExcelY}, search: point{X: p.SearchX, Y: p.SearchY}}
	bridgeStarted = true
	return nil
}
func runManagedBridge(setup bool) bool {
	name, _ := syscall.UTF16PtrFromString("Local\\AlphaERPAutomaticBridge")
	h, _, e := kernel32.NewProc("CreateMutexW").Call(0, 0, uintptr(unsafe.Pointer(name)))
	if h == 0 || e == syscall.Errno(183) {
		bridgeStatus("기존 ERP 연결 도구를 종료해 주세요. 본 프로그램이 연결을 관리합니다")
		if h != 0 {
			kernel32.NewProc("CloseHandle").Call(h)
		}
		return false
	}
	defer kernel32.NewProc("CloseHandle").Call(h)
	operation = "bridge"
	bridgeManagedCapture = !setup
	if !setup {
		if e := loadBridgeProfile(); e != nil {
			bridgeStatus(e.Error())
			return false
		}
	}
	bridgeStatus("ERP 수집 중 · 최초 설정 후 자동으로 연결됩니다")
	if e := backgroundTest(); e != nil {
		bridgeStatus(e.Error())
		if setup {
			message("ERP 최초 설정을 완료하지 못했습니다.\n" + e.Error())
		}
		return false
	}
	if setup {
		if e := saveBridgeProfile(); e != nil {
			bridgeStatus(e.Error())
			message(e.Error())
			return false
		}
		message("ERP 최초 설정을 저장했습니다.\n이 창을 닫아도 본 프로그램이 이후 수집을 자동으로 관리합니다.")
	}
	bridgeStatus("ERP 저장 및 전달 완료 · 본 프로그램에서 다음 수집을 자동 실행합니다")
	return true
}
