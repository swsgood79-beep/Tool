package main

import (
	"os"
	"strings"
	"testing"
)

func TestExportRestoreRequiresClosedOwnedWorkbook(t *testing.T) {
	cases := []struct {
		current, work, excel, tool uint32
		closed, owned, want        bool
	}{
		{10, 20, 10, 30, true, true, true}, {10, 10, 10, 30, true, true, true},
		{10, 20, 10, 30, false, true, false}, {10, 20, 10, 30, true, false, false},
		{40, 20, 10, 30, true, true, false}, {10, 30, 10, 30, true, true, false},
		{30, 20, 30, 30, true, true, false}, {0, 20, 10, 30, true, true, false},
		{10, 0, 10, 30, true, true, false}, {10, 20, 10, 0, true, true, false},
	}
	for _, c := range cases {
		if got := shouldRestoreAfterExport(c.current, c.work, c.excel, c.tool, c.closed, c.owned); got != c.want {
			t.Fatal(c, got)
		}
	}
}
func TestExportCleanupOnlyClosesExactSavedWorkbook(t *testing.T) {
	b, e := os.ReadFile("close-export-excel.ps1")
	if e != nil {
		t.Fatal(e)
	}
	s := string(b)
	for _, bad := range []string{".Quit(", "Stop-Process", "CreateObject", "Workbooks.Open", "DisplayAlerts", "EnableEvents", ".Kill(", ".Visible="} {
		if strings.Contains(s, bad) {
			t.Fatal(bad)
		}
	}
	for _, required := range []string{"$book.FullName", "GetFullPath", "OrdinalIgnoreCase", "if(!(Is-Our-Workbook $book)){continue}", "if(!$book.Saved)", "$book.Close($false)", "$stillOpen"} {
		if !strings.Contains(s, required) {
			t.Fatal(required)
		}
	}
	if strings.Count(s, ".Close(") != 1 || strings.Index(s, "if(!$book.Saved)") > strings.Index(s, "$book.Close($false)") {
		t.Fatal("close guard missing")
	}
	b, e = os.ReadFile("background_excel_windows.go")
	if e != nil {
		t.Fatal(e)
	}
	s = string(b)
	for _, bad := range []string{"SendInput", "keybd_event", "mouse_event", "SetCursorPos", "AttachThreadInput"} {
		if strings.Contains(s, bad) {
			t.Fatal(bad)
		}
	}
	if strings.Index(s, "if !shouldRestoreAfterExport(") > strings.Index(s, `api("SetForegroundWindow"`) {
		t.Fatal("restore guard missing")
	}
}
