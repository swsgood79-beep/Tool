package main

import (
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
