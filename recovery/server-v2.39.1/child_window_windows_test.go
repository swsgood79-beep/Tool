package main

import (
	"os/exec"
	"syscall"
	"testing"
)

func TestHideChildWindowBeforeStart(t *testing.T) {
	c := exec.Command("unused.exe")
	hideChildWindow(c)
	if c.SysProcAttr == nil || !c.SysProcAttr.HideWindow || c.SysProcAttr.CreationFlags&0x08000000 == 0 {
		t.Fatal("console not suppressed")
	}
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x200}
	hideChildWindow(c)
	if c.SysProcAttr.CreationFlags != 0x08000200 {
		t.Fatal("other flags lost")
	}
}
