package main

import (
	"archive/zip"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf16"
)

var operation = "diagnostic"

func encodePSScript(s string) string {
	units := utf16.Encode([]rune(s))
	b := make([]byte, len(units)*2)
	for i, u := range units {
		binary.LittleEndian.PutUint16(b[2*i:], u)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func validateWorkbook(path string) error {
	st, e := os.Stat(path)
	if e != nil {
		return e
	}
	if st.Size() <= 0 || st.Size() > 64<<20 {
		return fmt.Errorf("엑셀 파일 크기를 확인해 주세요")
	}
	z, e := zip.OpenReader(path)
	if e != nil {
		return fmt.Errorf("정상적인 XLSX 파일이 아닙니다")
	}
	defer z.Close()
	found := false
	sheet := false
	var total uint64
	for _, f := range z.File {
		total += f.UncompressedSize64
		if total > 64<<20 {
			return fmt.Errorf("엑셀 내용 크기가 너무 큽니다")
		}
		entry, err := f.Open()
		if err != nil {
			return err
		}
		_, err = io.Copy(io.Discard, io.LimitReader(entry, 64<<20+1))
		entry.Close()
		if err != nil {
			return fmt.Errorf("엑셀 파일 내용 검증 실패")
		}

		if f.Name == "xl/workbook.xml" {
			found = true
		}
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			sheet = true
		}
	}
	if !found || !sheet {
		return fmt.Errorf("엑셀 통합문서 구조가 없습니다")
	}
	return nil
}
