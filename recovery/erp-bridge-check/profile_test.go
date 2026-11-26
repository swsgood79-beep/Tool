package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSavedProfileRebindsMovedWindowWithoutHWND(t *testing.T) {
	p := bridgeProfile{Version: 1, RootWidth: 1936, RootHeight: 1056, Toolbar: Rect{232, 80, 1926, 110}, ExcelX: 1516, ExcelY: 22, SearchX: 1450, SearchY: 22, DPI: 96}
	if !p.valid() {
		t.Fatal("valid calibration rejected")
	}
	root := Rect{100, 200, 2036, 1256}
	c := Control{Handle: 123456, Class: "WindowsForms10.Window.8.app.new_process", Visible: true, Enabled: true, Bounds: Rect{332, 280, 2026, 310}}
	if !p.matches(root, c, 96) {
		t.Fatal("translated window rejected")
	}
	if p.matches(root, c, 120) {
		t.Fatal("different DPI accepted")
	}
	changed := root
	changed.Right++
	if p.matches(changed, c, 96) {
		t.Fatal("changed root size accepted")
	}
	changedControl := c
	changedControl.Bounds.Top++
	if p.matches(root, changedControl, 96) {
		t.Fatal("changed toolbar accepted")
	}
	changedControl = c
	changedControl.Enabled = false
	if p.matches(root, changedControl, 96) {
		t.Fatal("disabled toolbar accepted")
	}
	b, _ := json.Marshal(p)
	s := strings.ToLower(string(b))
	if strings.Contains(s, "handle") || strings.Contains(s, "pid") || strings.Contains(s, "hwnd") {
		t.Fatal("live handle persisted")
	}
	for _, change := range []func(*bridgeProfile){func(p *bridgeProfile) { p.Version = 2 }, func(p *bridgeProfile) { p.SearchX = p.ExcelX }, func(p *bridgeProfile) { p.ExcelY = 90 }, func(p *bridgeProfile) { p.Toolbar.Right = 3000 }} {
		q := p
		change(&q)
		if q.valid() {
			t.Fatal("invalid calibration accepted", q)
		}
	}
}
