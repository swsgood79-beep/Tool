package main

import "fmt"

// Saved calibration deliberately contains no HWND or PID. Bind to a freshly
// discovered PrimeXAR toolbar only if size, relative layout and DPI still match.
type bridgeProfile struct {
	Version                          int
	RootWidth, RootHeight            int32
	Toolbar                          Rect
	ExcelX, ExcelY, SearchX, SearchY int32
	DPI                              uint32
}

func (p bridgeProfile) valid() bool {
	return p.Version == 1 && p.RootWidth >= 600 && p.RootHeight >= 400 && p.DPI >= 96 && p.DPI <= 480 &&
		p.Toolbar.Left >= 0 && p.Toolbar.Top >= 45 && p.Toolbar.Bottom <= 150 && p.Toolbar.Right <= p.RootWidth &&
		p.Toolbar.Right-p.Toolbar.Left >= 300 && p.Toolbar.Bottom-p.Toolbar.Top >= 20 && p.Toolbar.Bottom-p.Toolbar.Top <= 60 &&
		p.ExcelX >= 0 && p.ExcelX < p.Toolbar.Right-p.Toolbar.Left && p.ExcelY >= 0 && p.ExcelY < p.Toolbar.Bottom-p.Toolbar.Top &&
		p.SearchX >= 0 && p.ExcelX-p.SearchX >= 35 && p.ExcelX-p.SearchX <= 100 && p.SearchY >= 0 && p.SearchY < p.Toolbar.Bottom-p.Toolbar.Top
}
func (p bridgeProfile) matches(root Rect, toolbar Control, dpi uint32) bool {
	relative := Rect{toolbar.Bounds.Left - root.Left, toolbar.Bounds.Top - root.Top, toolbar.Bounds.Right - root.Left, toolbar.Bounds.Bottom - root.Top}
	return p.valid() && root.Right-root.Left == p.RootWidth && root.Bottom-root.Top == p.RootHeight && dpi == p.DPI && relative == p.Toolbar && backgroundToolbarCandidate(toolbar, root)
}
func validateProfile(p bridgeProfile) error {
	if !p.valid() {
		return fmt.Errorf("ERP 최초 설정을 다시 진행해 주세요")
	}
	return nil
}
