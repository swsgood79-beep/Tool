//go:build windows

package main

import (
	"bufio"
	_ "embed"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

//go:embed erp-receipt-helper.ps1
var receiptHelperScript string

//go:embed erp-receipt-filter.cs
var receiptFilterSource string

func platformReceiptHelper(task receiptHelperTask) error {
	script := "$TaskBase64='" + base64.StdEncoding.EncodeToString(jsonRaw(task)) + "'\n" + strings.Replace(receiptHelperScript, "# FILTER_SOURCE", receiptFilterSource, 1)
	cmd := exec.Command(filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe"), "-STA", "-NoProfile", "-NonInteractive", "-Command", erpDBBootstrap)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	cmd.Stdin = strings.NewReader(script)
	var stages receiptHelperStageWriter
	cmd.Stderr = &stages
	output, err := cmd.StdoutPipe()
	if err != nil {
		return receiptHelperLaunchError{Stage: "start"}
	}
	if err = cmd.Start(); err != nil {
		return receiptHelperLaunchError{Stage: "start"}
	}
	ready := make(chan uintptr, 1)
	go func() {
		scanner := bufio.NewScanner(output)
		sent := false
		for scanner.Scan() {
			if handle, ok := receiptHelperReadyHandle(scanner.Text()); ok && !sent {
				ready <- handle
				sent = true
			}
		}
		if !sent {
			ready <- 0
		}
	}()
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	select {
	case handle := <-ready:
		if handle != 0 && receiptHelperVisibleAfter(handle, exited, func(h uintptr) bool { return receiptHelperWindowVisible(h, uint32(cmd.Process.Pid)) }, time.Second) {
			return nil
		}
	case <-time.After(20 * time.Second):
		_ = cmd.Process.Kill()
		<-exited
		return receiptHelperLaunchError{Stage: "timeout"}
	}
	_ = cmd.Process.Kill()
	<-exited
	stage := stages.Stage()
	if stage == "" {
		stage = "visibility"
	}
	return receiptHelperLaunchError{Stage: stage}
}

func receiptHelperWindowVisible(handle uintptr, pid uint32) bool {
	user := syscall.NewLazyDLL("user32.dll")
	visible, _, _ := user.NewProc("IsWindowVisible").Call(handle)
	var owner uint32
	user.NewProc("GetWindowThreadProcessId").Call(handle, uintptr(unsafe.Pointer(&owner)))
	monitor, _, _ := user.NewProc("MonitorFromWindow").Call(handle, 0)
	iconic, _, _ := user.NewProc("IsIconic").Call(handle)
	return visible != 0 && owner == pid && monitor != 0 && iconic == 0
}
