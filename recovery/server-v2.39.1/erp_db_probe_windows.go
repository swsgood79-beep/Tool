//go:build windows

package main

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
)

func platformERPDBProbe(ctx context.Context, folder string) ERPDBReport {
	cmd := erpDBPowerShellCommand(ctx, folder)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	job, e := newERPJob()
	if e != nil {
		return ERPDBReport{Status: "failed", Stage: "process_job"}
	}
	defer syscall.CloseHandle(job)
	var output erpDBOutput
	cmd.Stdout = &output
	e = cmd.Start()
	if e != nil {
		code := 0
		var errno syscall.Errno
		if errors.As(e, &errno) {
			code = int(errno)
		}
		return ERPDBReport{Status: "failed", Stage: "process_start", ErrorCode: code}
	}
	if e = attachERPJob(job, cmd.Process.Pid); e != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return ERPDBReport{Status: "failed", Stage: "process_attach"}
	}
	e = cmd.Wait()
	raw := output.Bytes()
	if ctx.Err() != nil {
		return ERPDBReport{Status: "timeout", Stage: "process_wait"}
	}
	if e != nil {
		code := 0
		var exit *exec.ExitError
		if errors.As(e, &exit) {
			code = exit.ExitCode()
		}
		return ERPDBReport{Status: "failed", Stage: "process_exit", ErrorCode: code}
	}
	return decodeDBProbe(raw)
}
