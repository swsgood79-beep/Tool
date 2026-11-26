package main

import "strings"

func safeSaveLabel(label string) string {
	label = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(label), "&", ""))
	if label == "save" || label == "저장" || label == "저장(s)" || label == "save(s)" {
		return "Save"
	}
	return ""
}

// Restore only a previously captured non-ERP work window, and only while the
// ERP currently owns foreground. Never override another user-selected app.
func shouldRestoreBackground(currentPID, workPID, erpPID, toolPID uint32) bool {
	return erpPID != 0 && toolPID != 0 && workPID != 0 && currentPID == erpPID && workPID != erpPID && workPID != toolPID
}
