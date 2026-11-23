package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

func handlePCUpdateHelper(args []string) (bool, error) {
	if len(args) == 0 || args[0] != "--pc-update-helper" {
		return false, nil
	}
	if len(args) != 2 {
		return true, errors.New("업데이트 도우미 설정이 없습니다")
	}
	raw, e := os.ReadFile(args[1])
	if e != nil {
		return true, e
	}
	if len(raw) > 32768 {
		return true, errors.New("업데이트 정보 크기 오류")
	}
	var p pcUpdatePlan
	if e = json.Unmarshal(raw, &p); e != nil {
		return true, e
	}
	if e = p.validate(args[1]); e != nil {
		return true, e
	}
	process, e := syscall.OpenProcess(0x00100000|0x1000, false, uint32(p.ParentPID))
	if e != nil {
		return true, e
	}
	defer syscall.CloseHandle(process)
	// Verify the exact process being waited on is the original executable.
	buf := make([]uint16, 32768)
	size := uint32(len(buf))
	ok, _, qe := kernel32.NewProc("QueryFullProcessImageNameW").Call(uintptr(process), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ok == 0 || !sameWindowsPath(syscall.UTF16ToString(buf[:size]), p.Target) {
		return true, fmt.Errorf("업데이트 대상 프로세스 확인 실패: %v", qe)
	}
	if e = os.WriteFile(filepath.Join(p.Stage, "helper-ready"), []byte("ready"), 0600); e != nil {
		return true, e
	}
	// Do not touch the live binary until graceful shutdown and all deferred unlocks finish.
	result, e := syscall.WaitForSingleObject(process, 180000)
	if e != nil || result != 0 {
		return true, errors.New("기존 프로그램이 종료되지 않아 교체를 중단했습니다. 기존 파일은 유지됩니다")
	}
	if e = p.validate(args[1]); e != nil {
		if h, err := pcHashFile(p.Target); err == nil && h == p.OldSHA256 {
			_ = pcLaunchUpdated(p.Target, p.Args, "")
		}
		return true, e
	}
	e = pcReplaceAndRestart(p, pcLaunchUpdated)
	if e != nil {
		_ = os.WriteFile(filepath.Join(p.Stage, "error.txt"), []byte(e.Error()), 0600)
	}
	return true, e
}
func sameWindowsPath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
func pcLaunchUpdated(target string, args []string, ready string) error {
	full := append([]string{}, args...)
	if ready != "" {
		full = append(full, "-pc-update-ready", ready)
	}
	c := exec.Command(target, full...)
	c.Dir = filepath.Dir(target)
	// The updated GUI must open normally; HideWindow would hide its first window.
	if e := c.Start(); e != nil {
		return e
	}
	if ready == "" {
		return c.Process.Release()
	}
	exited := make(chan error, 1)
	go func() { exited <- c.Wait() }()
	timer := time.NewTimer(2 * time.Minute)
	defer timer.Stop()
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case e := <-exited:
			if e == nil {
				e = errors.New("시작 확인 전에 종료되었습니다")
			}
			return e
		case <-tick.C:
			if b, e := os.ReadFile(ready); e == nil && string(b) == "ready" {
				return nil
			}
		case <-timer.C:
			return errPCUpdateStillStarting
		}
	}
}
func preparePCUpdateHelper(stage, target, data, listen string) error {
	data, e := filepath.Abs(data)
	if e != nil {
		return e
	}
	old, e := pcHashFile(target)
	if e != nil {
		return e
	}
	next, e := pcHashFile(filepath.Join(stage, "new.exe"))
	if e != nil {
		return e
	}
	p := pcUpdatePlan{Target: target, Stage: stage, OldSHA256: old, NewSHA256: next, ParentPID: os.Getpid(), Args: pcRestartArgs(data, listen)}
	plan := filepath.Join(stage, "plan.json")
	if e = p.validate(plan); e != nil {
		return e
	}
	if e = pcWriteJSON(plan, p); e != nil {
		return e
	}
	// The helper is a separate copy of the currently running, trusted executable.
	src, e := os.ReadFile(target)
	if e != nil {
		return e
	}
	helper := filepath.Join(stage, "helper.exe")
	if e = os.WriteFile(helper, src, 0700); e != nil {
		return e
	}
	c := exec.Command(helper, "--pc-update-helper", plan)
	c.Dir = stage
	hideChildWindow(c)
	if e = c.Start(); e != nil {
		return e
	}
	defer c.Process.Release()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = c.Process.Kill()
			return errors.New("도우미를 실행하지 못했습니다. 기존 프로그램은 계속 실행합니다")
		case <-tick.C:
			if b, e := os.ReadFile(filepath.Join(stage, "helper-ready")); e == nil && string(b) == "ready" {
				return nil
			}
		}
	}
}

func (d *winDesktop) pcUpdateCommand() {
	if d.Updater == nil || d.Closing {
		return
	}
	view := d.Updater.snapshot()
	if view.Busy {
		return
	}
	if !view.Available {
		go d.Updater.check(d.UpdateContext)
		return
	}
	message := "PC v" + view.Version + "로 업데이트하시겠습니까?\n\n다운로드와 검증 후 프로그램을 재시작합니다. 재시작하는 동안 휴대폰 접속·알림이 잠시 중단됩니다. 기존 수주·사진·저장 위치는 유지합니다."
	if win("MessageBoxW", d.Window, uintptr(unsafe.Pointer(wide(message))), uintptr(unsafe.Pointer(wide("알파 PC 업데이트"))), 0x24) != 6 {
		return
	}
	go func() {
		target, e := os.Executable()
		if e != nil {
			d.Updater.failed(e)
			return
		}
		stage, e := d.Updater.download(d.UpdateContext, target)
		if e != nil {
			return
		}
		if d.UpdateContext.Err() != nil {
			os.RemoveAll(stage)
			return
		}
		parsed, e := url.Parse(d.App.localURL)
		if e != nil {
			d.Updater.failed(e)
			return
		}
		if e = preparePCUpdateHelper(stage, target, d.App.dir, parsed.Host); e != nil {
			d.Updater.failed(e)
			return
		}
		win("PostMessageW", d.Window, wmPCUpdate, 0, 0)
	}()
}
