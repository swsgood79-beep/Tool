//go:build windows

package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

//go:embed inspect-uia.ps1
var uiaScript string
var user32 = syscall.NewLazyDLL("user32.dll")
var kernel32 = syscall.NewLazyDLL("kernel32.dll")

func api(name string, args ...uintptr) uintptr {
	r, _, _ := user32.NewProc(name).Call(args...)
	return r
}
func textOf(h uintptr) string {
	buf := make([]uint16, 512)
	api("GetWindowTextW", h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}
func classOf(h uintptr) string {
	buf := make([]uint16, 256)
	api("GetClassNameW", h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}
func rectOf(h uintptr) Rect {
	var r Rect
	api("GetWindowRect", h, uintptr(unsafe.Pointer(&r)))
	return r
}
func prepareConsole() { kernel32.NewProc("SetConsoleOutputCP").Call(65001) }
func processFile(pid uint32) string {
	h, _, _ := kernel32.NewProc("OpenProcess").Call(0x1000, 0, uintptr(pid))
	if h == 0 {
		return ""
	}
	defer syscall.CloseHandle(syscall.Handle(h))
	buf := make([]uint16, 32768)
	size := uint32(len(buf))
	ok, _, _ := kernel32.NewProc("QueryFullProcessImageNameW").Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ok == 0 {
		return ""
	}
	return filepath.Base(syscall.UTF16ToString(buf[:size]))
}
func collectWindows(folder string) ([]Window, error) {
	out := []Window{}
	callback := syscall.NewCallback(func(h, l uintptr) uintptr {
		if len(out) >= 3 {
			return 0
		}
		if api("IsWindowVisible", h) == 0 || !strings.Contains(strings.ToLower(textOf(h)), "primeerp") {
			return 1
		}
		var pid uint32
		api("GetWindowThreadProcessId", h, uintptr(unsafe.Pointer(&pid)))
		if pid == uint32(os.Getpid()) {
			return 1
		}
		w := Window{Handle: uint64(h), ProcessID: pid, ProcessFile: processFile(pid), Class: classOf(h), Bounds: rectOf(h), Controls: []Control{}}
		children := syscall.NewCallback(func(child, l uintptr) uintptr {
			if len(w.Controls) >= 500 {
				return 0
			}
			class := classOf(child)
			label := ""
			if strings.Contains(strings.ToLower(class), "button") || strings.EqualFold(class, "Static") {
				label = safeLabel(textOf(child))
			}
			w.Controls = append(w.Controls, Control{Handle: uint64(child), Parent: uint64(api("GetParent", child)), ID: int32(api("GetDlgCtrlID", child)), Class: class, Label: label, Visible: api("IsWindowVisible", child) != 0, Enabled: api("IsWindowEnabled", child) != 0, Bounds: rectOf(child)})
			return 1
		})
		api("EnumChildWindows", h, children, 0)
		out = append(out, w)
		return 1
	})
	api("EnumWindows", callback, 0)
	for i := range out {
		fmt.Printf("확인한 ERP 창 %d: 화면 구조를 읽는 중…\n", i+1)
		collectUIA(&out[i], folder, i)
	}
	return out, nil
}
func collectUIA(w *Window, folder string, index int) {
	powershell := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	destination := filepath.Join(folder, fmt.Sprintf(".uia-%d.json", index))
	defer os.Remove(destination)
	quoted := "'" + strings.ReplaceAll(destination, "'", "''") + "'"
	script := fmt.Sprintf("$TargetHandle=[long]%d\n$Destination=%s\n", w.Handle, quoted) + uiaScript + "\n"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, powershell, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", "-")
	cmd.Stdin = strings.NewReader(script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			w.UIAStatus = "Timed out; Win32 metadata retained"
		} else {
			w.UIAStatus = "Unavailable; Win32 metadata retained"
		}
		return
	}
	raw, err := os.ReadFile(destination)
	if err != nil || len(raw) > 512<<10 {
		w.UIAStatus = "No UI Automation metadata; Win32 metadata retained"
		return
	}
	if !jsonValid(raw) {
		w.UIAStatus = "Invalid UI Automation output; Win32 metadata retained"
		return
	}
	w.UIA = raw
	w.UIAStatus = "Read-only metadata collected"
}
func showResult(message, folder string) {
	msg, _ := syscall.UTF16PtrFromString(message)
	title, _ := syscall.UTF16PtrFromString("알파 ERP 연결 진단 v" + version)
	api("MessageBoxW", 0, uintptr(unsafe.Pointer(msg)), uintptr(unsafe.Pointer(title)), 0x40)
	runtime.KeepAlive(msg)
	runtime.KeepAlive(title)
	if folder != "" {
		verb, _ := syscall.UTF16PtrFromString("open")
		p, _ := syscall.UTF16PtrFromString(folder)
		syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW").Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(p)), 0, 0, 1)
		runtime.KeepAlive(verb)
		runtime.KeepAlive(p)
	}
}
