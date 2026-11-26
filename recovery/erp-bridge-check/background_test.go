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
