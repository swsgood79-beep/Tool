package main

import (
	"os"
	"os/exec"
	"syscall"
)

// Linux is the test host. Production Windows builds use DPAPI above.
func protect(b []byte) ([]byte, error)   { return b, nil }
func unprotect(b []byte) ([]byte, error) { return b, nil }
func openBrowser(url string)             {}
func lockData(path string) (func(), error) {
	f, e := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		f.Close()
		return nil, e
	}
	return func() { f.Close() }, nil
}
func nativeNotify(title, body, target string) {}

func originalRunning() bool                     { return false }
func manageChild(cmd *exec.Cmd) (func(), error) { return func() {}, nil }

func lockOriginalApp() (func(), error)                       { return func() {}, nil }
func nativeError(message string)                             {}
func startDesktop(a *App, stop func()) (func(), bool, error) { return func() {}, false, nil }

func configureDesktop(test bool) {}

func nativeFirstStart(message string) int { return 2 }

func hideChildWindow(cmd *exec.Cmd) {}

func openDataFolder(path string) error {
	return openFolderWith(path, func(absolute string) error { return exec.Command("xdg-open", absolute).Run() })
}
