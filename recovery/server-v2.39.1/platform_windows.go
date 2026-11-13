package main

import (
	"encoding/binary"
	"fmt"
	"os/exec"
	"runtime"
	"syscall"
	"unsafe"
)

type blob struct {
	Size uint32
	Data *byte
}

func crypt(data []byte, decrypt bool) ([]byte, error) {
	in := blob{Size: uint32(len(data))}
	if len(data) > 0 {
		in.Data = &data[0]
	}
	var out blob
	lib := syscall.NewLazyDLL("crypt32.dll")
	name := "CryptProtectData"
	if decrypt {
		name = "CryptUnprotectData"
	}
	r, _, e := lib.NewProc(name).Call(uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, 1, uintptr(unsafe.Pointer(&out)))
	runtime.KeepAlive(data)
	if r == 0 {
		return nil, fmt.Errorf("Windows DPAPI: %w", e)
	}
	defer syscall.NewLazyDLL("kernel32.dll").NewProc("LocalFree").Call(uintptr(unsafe.Pointer(out.Data)))
	return append([]byte{}, unsafe.Slice(out.Data, int(out.Size))...), nil
}
func protect(b []byte) ([]byte, error)   { return crypt(b, false) }
func unprotect(b []byte) ([]byte, error) { return crypt(b, true) }
func openBrowser(url string) {
	c := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url)
	hideChildWindow(c)
	if c.Start() == nil {
		c.Process.Release()
	}
}
func lockData(path string) (func(), error) {
	p, e := syscall.UTF16PtrFromString(path)
	if e != nil {
		return nil, e
	}
	h, e := syscall.CreateFile(p, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if e != nil {
		return nil, e
	}
	return func() { syscall.CloseHandle(h) }, nil
}

func originalRunning() bool {
	p, _ := syscall.UTF16PtrFromString(`Local\WanryoTalkNotifier-v1`)
	h, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("OpenMutexW").Call(0x100000, 0, uintptr(unsafe.Pointer(p)))
	if h != 0 {
		syscall.CloseHandle(syscall.Handle(h))
		return true
	}
	return false
}

func manageChild(cmd *exec.Cmd) (func(), error) {
	k := syscall.NewLazyDLL("kernel32.dll")
	job, _, e := k.NewProc("CreateJobObjectW").Call(0, 0)
	if job == 0 {
		return nil, e
	}
	limits := make([]byte, 144)
	binary.LittleEndian.PutUint32(limits[16:], 0x2000)
	ok, _, e := k.NewProc("SetInformationJobObject").Call(job, 9, uintptr(unsafe.Pointer(&limits[0])), uintptr(len(limits)))
	if ok == 0 {
		syscall.CloseHandle(syscall.Handle(job))
		return nil, e
	}
	process, e := syscall.OpenProcess(0x0101, false, uint32(cmd.Process.Pid))
	if e != nil {
		syscall.CloseHandle(syscall.Handle(job))
		return nil, e
	}
	defer syscall.CloseHandle(process)
	ok, _, e = k.NewProc("AssignProcessToJobObject").Call(job, uintptr(process))
	if ok == 0 {
		syscall.CloseHandle(syscall.Handle(job))
		return nil, e
	}
	return func() { syscall.CloseHandle(syscall.Handle(job)) }, nil
}

// Apply before Start: HideWindow alone does not prevent console allocation.
func hideChildWindow(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= 0x08000000 // CREATE_NO_WINDOW
}
