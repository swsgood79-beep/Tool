package main

import (
	"archive/zip"
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"
)

func TestExportWorkbookValidation(t *testing.T) {
	for _, valid := range []bool{true, false} {
		p := filepath.Join(t.TempDir(), "report.xlsx")
		f, _ := os.Create(p)
		z := zip.NewWriter(f)
		a, _ := z.Create("xl/workbook.xml")
		a.Write([]byte("<workbook/>"))
		if valid {
			a, _ = z.Create("xl/worksheets/sheet1.xml")
			a.Write([]byte("<worksheet/>"))
		}
		z.Close()
		f.Close()
		err := validateWorkbook(p)
		if (err == nil) != valid {
			t.Fatal(valid, err)
		}
		os.WriteFile(p, []byte("unfinished file"), 0600)
		if validateWorkbook(p) == nil {
			t.Fatal("incomplete accepted")
		}
	}
}
func TestPowerShellUnicodeTransport(t *testing.T) {
	s := "$ExportPath='C:\\사용자\\입고예정.xlsx'; # 저장"
	b, e := base64.StdEncoding.DecodeString(encodePSScript(s))
	if e != nil {
		t.Fatal(e)
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[2*i:])
	}
	if string(utf16.Decode(u)) != s {
		t.Fatal("Korean transport changed")
	}
}

func TestSaveControlSelection(t *testing.T) {
	d := Rect{Left: 0, Top: 0, Right: 960, Bottom: 560}
	c := Control{ID: 1152, Class: "Edit", Visible: true, Enabled: true, Bounds: Rect{Left: 130, Top: 450, Right: 940, Bottom: 475}}
	if !filenameCandidate(c, d) {
		t.Fatal("classic filename not recognized")
	}
	c.ID = 1001
	if !filenameCandidate(c, d) {
		t.Fatal("modern filename not recognized")
	}
	c.Bounds.Top = 60
	if filenameCandidate(c, d) {
		t.Fatal("search bar accepted")
	}
	c.Bounds.Top = 450
	c.ID = 999
	if filenameCandidate(c, d) {
		t.Fatal("unidentified input accepted")
	}
	b := Control{ID: 1, Class: "Button", Label: "Save", Visible: true, Enabled: true, Bounds: Rect{Left: 750, Top: 520, Right: 830, Bottom: 545}}
	if !saveButtonCandidate(b, d) {
		t.Fatal("save rejected")
	}
	b.Label = "Cancel"
	if saveButtonCandidate(b, d) {
		t.Fatal("cancel accepted")
	}
}
