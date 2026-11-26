package main

import "strings"

// A calibrated toolbar-local coordinate is valid only for this live window,
// unchanged geometry and report. Never reuse a saved numeric HWND next launch.
func backgroundToolbarCandidate(c Control, root Rect) bool {
	width, height := c.Bounds.Right-c.Bounds.Left, c.Bounds.Bottom-c.Bounds.Top
	return c.Visible && c.Enabled && strings.HasPrefix(strings.ToLower(c.Class), "windowsforms10.window.") && width >= 300 && height >= 20 && height <= 60 && c.Bounds.Top >= root.Top+45 && c.Bounds.Bottom <= root.Top+150
}
func backgroundTargetValid(c Control, root Rect, x, y int32) bool {
	return backgroundToolbarCandidate(c, root) && x >= root.Left+(root.Right-root.Left)*55/100 && x >= c.Bounds.Left && x < c.Bounds.Right && y >= c.Bounds.Top && y < c.Bounds.Bottom
}
func backgroundGeometrySame(before, after Rect) bool { return before == after }
