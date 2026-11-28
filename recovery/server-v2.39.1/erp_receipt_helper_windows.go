//go:build windows

package main

import (
	"bufio"
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

//go:embed erp-receipt-helper.ps1
var receiptHelperScript string

//go:embed erp-receipt-filter.cs
var receiptFilterSource string

func platformReceiptHelper(task receiptHelperTask) error {
	script := "$TaskBase64='" + base64.StdEncoding.EncodeToString(jsonRaw(task)) + "'\n" + strings.Replace(receiptHelperScript, "# FILTER_SOURCE", receiptFilterSource, 1)
	cmd := exec.Command(filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe"), "-STA", "-NoProfile", "-NonInteractive", "-Command", erpDBBootstrap)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	cmd.Stdin = strings.NewReader(script)
	output, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	ready := make(chan bool, 1)
	go func() {
		scanner := bufio.NewScanner(output)
		sent := false
		for scanner.Scan() {
			if scanner.Text() == "ALPHA_HELPER_READY" && !sent {
				ready <- true
				sent = true
			}
		}
		if !sent {
			ready <- false
		}
	}()
	go func() { _ = cmd.Wait() }()
	select {
	case ok := <-ready:
		if ok {
			return nil
		}
	case <-time.After(12 * time.Second):
	}
	_ = cmd.Process.Kill()
	return fmt.Errorf("receipt helper startup failed")
}
