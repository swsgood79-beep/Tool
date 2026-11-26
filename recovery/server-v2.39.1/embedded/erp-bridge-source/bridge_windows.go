//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

type bridgeCalibration struct {
	root, target          uintptr
	c                     Control
	bounds                Rect
	screen, local, search point
}

var bridgeCal *bridgeCalibration
var bridgeStarted bool

func bridgeFolder() (string, error) {
	h, e := os.UserConfigDir()
	if e != nil {
		return "", e
	}
	p := filepath.Join(h, "AlphaERPBridge")
	e = os.MkdirAll(p, 0700)
	return p, e
}
func bridgeWrite(name string, b []byte) error {
	dir, e := bridgeFolder()
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(dir, ".bridge-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), filepath.Join(dir, name))
}
func bridgePublish(file string) error {
	if e := validateWorkbook(file); e != nil {
		return e
	}
	b, e := os.ReadFile(file)
	if e != nil {
		return e
	}
	if len(b) > 16<<20 {
		return fmt.Errorf("ERP 엑셀은 16MB 이하여야 합니다")
	}
	return bridgeWrite("current.xlsx", b)
}
func bridgeStatus(message string) {
	b, _ := json.Marshal(map[string]any{"message": message, "at": time.Now()})
	_ = bridgeWrite("status.json", b)
	fmt.Println(time.Now().Format("15:04:05"), message)
}
func runBridge() {
	// A per-session Windows mutex prevents simultaneous toolbar/save requests.
	name, _ := syscall.UTF16PtrFromString("Local\\AlphaERPAutomaticBridge")
	h, _, e := kernel32.NewProc("CreateMutexW").Call(0, 0, uintptr(unsafe.Pointer(name)))
	if h == 0 || e == syscall.Errno(183) {
		message("ERP 연결 도구가 이미 실행 중이거나 실행 잠금을 확보하지 못했습니다")
		return
	}
	defer kernel32.NewProc("CloseHandle").Call(h)
	for {
		bridgeStatus("ERP 수집 중 · 창을 최소화하지 마세요")
		err := backgroundTest()
		if err != nil {
			bridgeStatus(err.Error())
			if bridgeCal == nil || err.Error() != "처음 위치 지정 시 ERP 최소화를 풀어 주세요" {
				message("ERP 자동 연결을 중단했습니다.\n" + err.Error() + "\n보고서와 조회 조건을 확인한 뒤 연결 도구를 다시 실행해 주세요.")
				return
			}
		} else {
			bridgeStarted = true
			bridgeStatus("ERP 저장 및 전달 완료 · 다음 수집은 5분 후")
		}
		// No GUI thread is blocked: the console remains available to close the tool.
		next := time.Now().Add(5 * time.Minute)
		for time.Now().Before(next) {
			time.Sleep(time.Second)
		}
	}
}
