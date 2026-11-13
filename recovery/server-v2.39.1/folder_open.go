package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// Filesystem paths are opened separately from browser URLs.
func openFolderWith(path string, launch func(string) error) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	st, err := os.Stat(absolute)
	if err != nil {
		return fmt.Errorf("저장 폴더 확인 실패: %w", err)
	}
	if !st.IsDir() {
		return fmt.Errorf("저장 위치가 폴더가 아닙니다: %s", absolute)
	}
	return launch(absolute)
}
