//go:build windows

package main

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
)

//go:embed erp-db-probe.ps1
var erpDBScript string

func platformERPDBProbe(ctx context.Context, folder string) ERPDBReport {
	prefix := "$SyncOnly=$false\n$PurchasePreview=$false\n"
	if dates, ok := erpDBPreviewOptions(ctx); ok {
		prefix = "$PurchasePreview=$true\n$DateFrom='" + strings.ReplaceAll(dates.From, "-", "") + "'\n$DateTo='" + strings.ReplaceAll(dates.To, "-", "") + "'\n"
	}
	if syncOnly, _ := ctx.Value(erpDBSyncKey{}).(bool); syncOnly {
		prefix += "$SyncOnly=$true\n"
	}
	if order, ok := ctx.Value(erpOrderOutstandingKey{}).(string); ok {
		prefix += "$OutstandingOrder='" + strings.ReplaceAll(order, "'", "''") + "'\n"
	}
	script := prefix + "$ERPFolder='" + strings.ReplaceAll(folder, "'", "''") + "'\n" + erpDBScript
	u := utf16.Encode([]rune(script))
	b := make([]byte, len(u)*2)
	for i, v := range u {
		binary.LittleEndian.PutUint16(b[i*2:], v)
	}
	cmd := exec.CommandContext(ctx, filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe"), "-NoLogo", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(b))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	job, e := newERPJob()
	if e != nil {
		return ERPDBReport{Status: "failed"}
	}
	defer syscall.CloseHandle(job)
	var output erpDBOutput
	cmd.Stdout = &output
	e = cmd.Start()
	if e != nil {
		return ERPDBReport{Status: "failed"}
	}
	if e = attachERPJob(job, cmd.Process.Pid); e != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return ERPDBReport{Status: "failed"}
	}
	e = cmd.Wait()
	raw := output.Bytes()
	if ctx.Err() != nil {
		return ERPDBReport{Status: "timeout"}
	}
	if e != nil {
		return ERPDBReport{Status: "failed"}
	}
	return decodeDBProbe(raw)
}
