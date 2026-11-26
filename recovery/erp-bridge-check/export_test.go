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
