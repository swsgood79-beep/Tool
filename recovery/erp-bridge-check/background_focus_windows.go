//go:build windows

package main

import (
	"os"
	"unsafe"
)

func backgroundRestoreWorkWindow(previous uintptr, erpPID uint32) (attempted, restored bool) {
	if previous == 0 || api("IsWindow", previous) == 0 || api("IsIconic", previous) != 0 || api("GetAncestor", previous, 2) != previous {
		return false, false
	}
	current := api("GetForegroundWindow")
	var currentPID, workPID uint32
	api("GetWindowThreadProcessId", current, uintptr(unsafe.Pointer(&currentPID)))
	api("GetWindowThreadProcessId", previous, uintptr(unsafe.Pointer(&workPID)))
	if !shouldRestoreBackground(currentPID, workPID, erpPID, uint32(os.Getpid())) {
		return false, false
	}
	if api("GetForegroundWindow") != current {
		return false, false
	}
	// This target is the work window recorded before F9, never the ERP/dialog.
	// Windows may deny the request; do not force focus with keyboard/thread tricks.
	api("SetForegroundWindow", previous)
	return true, api("GetForegroundWindow") == previous
}
