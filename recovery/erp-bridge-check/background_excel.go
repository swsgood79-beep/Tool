package main

// A verified, just-closed tool-owned workbook may have activated Excel. Restore
// the pre-F9 work window only when that owned workbook was actually foreground.
// A previously open Excel work window is allowed as the return target.
func shouldRestoreAfterExport(currentPID, workPID, excelPID, toolPID uint32, closed, ownedWasForeground bool) bool {
	return closed && ownedWasForeground && currentPID != 0 && excelPID != 0 && toolPID != 0 && workPID != 0 && currentPID == excelPID && excelPID != toolPID && workPID != toolPID
}
