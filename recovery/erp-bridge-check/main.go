// Read-only ERP window diagnostics. Never clicks, submits or reads database credentials.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"
)

const version = "0.2"

type Rect struct{ Left, Top, Right, Bottom int32 }
type Control struct {
	Handle           uint64 `json:",omitempty"`
	Parent           uint64 `json:",omitempty"`
	ID               int32  `json:",omitempty"`
	Class            string
	Label            string `json:",omitempty"`
	Visible, Enabled bool
	Bounds           Rect
}
type Window struct {
	Handle             uint64
	ProcessID          uint32
	ProcessFile, Class string
	Bounds             Rect
	Controls           []Control
	UIA                json.RawMessage `json:",omitempty"`
	UIAStatus          string
}
type Report struct {
	ToolVersion string
	CreatedAt   string
	Mode        string
	Windows     []Window
	Notes       []string
}

// Only fixed ERP labels are written; edit values, rows and document/customer names are omitted.
func safeLabel(s string) string {
	s = strings.Join(strings.Fields(strings.ReplaceAll(s, "&", "")), " ")
	labels := []string{"Excel", "엑셀", "검색", "조회", "창닫기", "창 닫기", "발주현황", "발주일", "발주일자", "발주번호", "거래처", "수주번호", "PROJECT", "품번", "품명", "규격", "OPTION", "수량", "납기예정일", "입고수량", "품번구분"}
	for _, label := range labels {
		if strings.EqualFold(s, label) {
			return label
		}
	}
	return ""
}
func normalizeJSON(raw []byte) ([]byte, error) {
	raw = bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})
	if len(raw) >= 2 && (bytes.Equal(raw[:2], []byte{0xff, 0xfe}) || bytes.Equal(raw[:2], []byte{0xfe, 0xff})) {
		var order binary.ByteOrder = binary.LittleEndian
		if raw[0] == 0xfe {
			order = binary.BigEndian
		}
		raw = raw[2:]
		if len(raw)%2 != 0 {
			return nil, fmt.Errorf("invalid UTF16 length")
		}
		units := make([]uint16, len(raw)/2)
		for i := range units {
			units[i] = order.Uint16(raw[i*2:])
		}
		raw = []byte(string(utf16.Decode(units)))
	}
	raw = bytes.TrimSpace(raw)
	if !json.Valid(raw) {
		return nil, fmt.Errorf("invalid JSON")
	}
	return raw, nil
}
func outputFolder(base string) (string, error) {
	for _, candidate := range []string{base, filepath.Join(os.TempDir(), "AlphaERPCheck")} {
		if err := os.MkdirAll(candidate, 0700); err != nil {
			continue
		}
		folder, err := os.MkdirTemp(candidate, "ERP-connection-check-"+time.Now().Format("20060102-150405")+"-")
		if err == nil {
			return folder, nil
		}
	}
	return "", fmt.Errorf("진단 결과 폴더를 만들 수 없습니다")
}
func run() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	folder, err := outputFolder(filepath.Dir(exe))
	if err != nil {
		return "", err
	}
	fmt.Println("ERP 화면 구조를 확인합니다. ERP의 발주현황 화면을 열어 두세요.")
	windows, err := collectWindows(folder)
	report := Report{ToolVersion: version, CreatedAt: time.Now().Format(time.RFC3339), Mode: "Read-only window metadata; no ERP clicks, edits, export or database access", Windows: windows, Notes: []string{"입력값·표의 품목 내용·비밀번호·DB 연결 설정은 수집하지 않습니다.", "결과는 이 PC에만 저장되며 자동 전송하지 않습니다.", "이 도구는 연결 가능성 확인용이며 자동 수집 기능은 아직 포함하지 않습니다."}}
	if err != nil {
		report.Notes = append(report.Notes, err.Error())
	}
	if len(windows) == 0 {
		report.Notes = append(report.Notes, "PrimeERP 창을 찾지 못했습니다. ERP 발주현황 화면을 연 뒤 다시 실행하세요.")
	}
	raw, e := json.MarshalIndent(report, "", "  ")
	if e != nil {
		return "", e
	}
	filename := filepath.Join(folder, "ERP-connection-check.json")
	if e = os.WriteFile(filename, append(raw, '\n'), 0600); e != nil {
		return "", e
	}
	summary := "ERP 연결 진단을 마쳤습니다.\r\n\r\n" + fmt.Sprintf("확인한 ERP 창: %d개\r\n", len(windows)) + "결과 파일: ERP-connection-check.json\r\n\r\n이 파일을 대화에 첨부해 주세요. 결과 폴더를 엽니다.\r\n\r\nERP 자료 변경이나 파일 자동 전송은 하지 않았습니다."
	if len(windows) == 0 {
		summary = "PrimeERP 창을 찾지 못했습니다.\r\nERP 발주현황 화면을 열고 다시 실행해 주세요.\r\n\r\n결과: " + filename
	}
	showResult(summary, folder)
	return filename, nil
}
func main() {
	prepareConsole()
	if operation == "background" {
		if err := backgroundTest(); err != nil {
			showResult(err.Error(), "")
			os.Exit(1)
		}
		return
	}
	if operation == "export" {
		if err := exportTest(); err != nil {
			showResult(err.Error(), "")
			os.Exit(1)
		}
		return
	}
	path, err := run()
	if err != nil {
		showResult("ERP 진단 실패: "+err.Error(), "")
		os.Exit(1)
	}
	fmt.Println("결과:", path)
}
