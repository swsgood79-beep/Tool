package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFolderOpensAsPathWithKoreanAndSpacesAndReportsShellFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "엑셀 저장 폴더")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	called := 0
	want := errors.New("shell rejected")
	err := openFolderWith(path, func(got string) error {
		called++
		if got != path {
			t.Fatal("path changed", got)
		}
		return want
	})
	if !errors.Is(err, want) || called != 1 {
		t.Fatal("shell failure hidden", err)
	}
	file := filepath.Join(path, "검사기록.xlsx")
	if err := os.WriteFile(file, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{file, filepath.Join(path, "missing")} {
		if err := openFolderWith(bad, func(string) error { t.Fatal("invalid folder launched"); return nil }); err == nil {
			t.Fatal("invalid path accepted")
		}
	}
}
