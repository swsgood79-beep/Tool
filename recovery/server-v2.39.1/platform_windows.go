package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"unicode/utf16"
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

// Native Windows toast; source-compatible notification data, different native window implementation.
func nativeNotify(title, body, target string) {
	lit := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	script := `[Windows.UI.Notifications.ToastNotificationManager,Windows.UI.Notifications,ContentType=WindowsRuntime]>$null;[Windows.Data.Xml.Dom.XmlDocument,Windows.Data.Xml.Dom.XmlDocument,ContentType=WindowsRuntime]>$null;$x=New-Object Windows.Data.Xml.Dom.XmlDocument;$x.LoadXml('<toast activationType="protocol"><visual><binding template="ToastGeneric"><text/><text/></binding></visual></toast>');$x.DocumentElement.SetAttribute('launch',` + lit(target) + `);$nodes=$x.GetElementsByTagName('text');$nodes.Item(0).InnerText=` + lit(title) + `;$nodes.Item(1).InnerText=` + lit(body) + `;[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('Microsoft.Windows.PowerShell').Show([Windows.UI.Notifications.ToastNotification]::new($x));`
	words := utf16.Encode([]rune(script))
	data := make([]byte, len(words)*2)
	for i, w := range words {
		binary.LittleEndian.PutUint16(data[i*2:], w)
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(data))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if cmd.Start() == nil {
		cmd.Process.Release()
	}
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
