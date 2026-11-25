package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixedLabelsOnly(t *testing.T) {
	for _, label := range []string{"Excel", "&Excel", " eXcEl ", "검색", "발주번호", "수량"} {
		if safeLabel(label) == "" {
			t.Fatalf("missing label %q", label)
		}
	}
	for _, data := range []string{"password123", "A261103-003", "2611-00001", "고객사 비밀 프로젝트", "AR090-S/P-0001-354L-V3#BT", "Excel 보고서 개인 파일", "12345", "홍길동"} {
		if safeLabel(data) != "" {
			t.Fatalf("business value retained %q", data)
		}
	}
}
func TestOutputDoesNotOverwriteFiles(t *testing.T) {
	base := t.TempDir()
	original := filepath.Join(base, "projects-v2.dat")
	os.WriteFile(original, []byte("existing production data"), 0600)
	out, e := outputFolder(base)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.HasPrefix(out, base+string(os.PathSeparator)+"ERP-connection-check-") {
		t.Fatal(out)
	}
	got, _ := os.ReadFile(original)
	if string(got) != "existing production data" {
		t.Fatal("existing file changed")
	}
}
