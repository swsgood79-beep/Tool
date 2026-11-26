package main

import (
	"os"
	"strings"
	"testing"
)

func TestBackgroundCalibrationRejectsNonToolbarTargets(t *testing.T) {
	root := Rect{Left: -8, Top: -8, Right: 1928, Bottom: 1048}
	c := Control{Class: "WindowsForms10.Window.8.app.test", Visible: true, Enabled: true, Bounds: Rect{Left: 224, Top: 72, Right: 1918, Bottom: 102}}
	if !backgroundTargetValid(c, root, 1754, 89) {
		t.Fatal("calibrated toolbar rejected")
	}
	for _, change := range []func(*Control){
		func(c *Control) { c.Class = "Edit" }, func(c *Control) { c.Visible = false }, func(c *Control) { c.Enabled = false }, func(c *Control) { c.Bounds.Bottom = 1016 }, func(c *Control) { c.Bounds.Top = 180; c.Bounds.Bottom = 210 },
	} {
		candidate := c
		change(&candidate)
		if backgroundTargetValid(candidate, root, 1754, 89) {
			t.Fatal("non toolbar accepted", candidate)
		}
	}
	if backgroundTargetValid(c, root, 400, 89) || backgroundTargetValid(c, root, 1754, 140) {
		t.Fatal("non Excel toolbar location accepted")
	}
	moved := c.Bounds
	moved.Left++
	if backgroundGeometrySame(c.Bounds, moved) {
		t.Fatal("changed geometry accepted")
	}
}
func TestBackgroundRunnerNeverUsesForegroundOrGlobalInputFallback(t *testing.T) {
	// The background entrypoint must remain isolated from the foreground v0.4 flow.
	b, e := os.ReadFile("background_windows.go")
	if e != nil {
		t.Fatal(e)
	}
	s := string(b)
	for _, forbidden := range []string{`api("SetForegroundWindow"`, `api("ShowWindow`, `api("SetCursorPos"`, `api("mouse_event"`, `api("SendInput"`, `api("keybd_event"`, `tryNativeSave(`, `exportTest(`, `typeFilename(`} {
		if strings.Contains(s, forbidden) {
			t.Fatal("foreground/global-input fallback added", forbidden)
		}
	}
	if !strings.Contains(s, `covered_at_start`) || !strings.Contains(s, `foreground_preserved_at_end`) {
		t.Fatal("background execution evidence missing")
	}
}

func TestBackgroundFocusRestoreOnlyForERPInterruption(t *testing.T) {
	cases := []struct {
		current, work, erp, tool uint32
		want                     bool
	}{
		{10, 20, 10, 30, true},  // ERP appeared over the previously selected work app.
		{40, 20, 10, 30, false}, // User switched to another application: do not interfere.
		{10, 10, 10, 30, false}, // Never activate ERP as the return window.
		{10, 30, 10, 30, false}, // Never activate this test tool as the return window.
		{0, 20, 10, 30, false}, {10, 0, 10, 30, false}, {10, 20, 0, 30, false},
	}
	for _, c := range cases {
		if got := shouldRestoreBackground(c.current, c.work, c.erp, c.tool); got != c.want {
			t.Fatal(c, got)
		}
	}
}
func TestBackgroundSaveLabelsAndInputIsolation(t *testing.T) {
	for _, label := range []string{"저장(&S)", "저장", "&Save", "Save"} {
		if safeSaveLabel(label) != "Save" {
			t.Fatal("save label rejected", label)
		}
	}
	for _, label := range []string{"취소", "Cancel", "저장 확인", "Save As", "another file"} {
		if safeSaveLabel(label) != "" {
			t.Fatal("unapproved control accepted", label)
		}
	}
	for _, file := range []string{"background_windows.go", "background_save_windows.go", "background_focus_windows.go"} {
		b, e := os.ReadFile(file)
		if e != nil {
			t.Fatal(e)
		}
		s := string(b)
		for _, bad := range []string{`api("SetCursorPos"`, `api("mouse_event"`, `api("SendInput"`, `api("keybd_event"`, `tryNativeSave(`, `typeFilename(`, `clickDialogControl(`} {
			if strings.Contains(s, bad) {
				t.Fatal(file, bad)
			}
		}
		if file != "background_focus_windows.go" && strings.Contains(s, `api("SetForegroundWindow"`) {
			t.Fatal("unexpected foreground request", file)
		}
		if file == "background_focus_windows.go" {
			guard, activate := strings.Index(s, "if !shouldRestoreBackground("), strings.Index(s, `api("SetForegroundWindow", previous)`)
			if guard < 0 || activate < guard {
				t.Fatal("foreground restore bypasses process guard")
			}
		}
	}
}
