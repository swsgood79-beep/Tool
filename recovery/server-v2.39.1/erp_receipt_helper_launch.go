package main

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"
)

// This bounded channel retains only allowlisted stage codes, never stderr text.
type receiptHelperLaunchError struct{ Stage string }

func (e receiptHelperLaunchError) Error() string { return "receipt helper launch: " + e.Stage }
func receiptHelperLaunchMessage(err error) string {
	var failure receiptHelperLaunchError
	if errors.As(err, &failure) {
		switch failure.Stage {
		case "desktop":
			return "PC 도우미를 열 수 없습니다. Alpha가 화면 없는 Windows 세션에서 실행 중입니다. 로그인된 Windows 바탕화면에서 Alpha를 실행해 주세요"
		case "assemblies":
			return "PC 도우미 화면 구성 요소를 불러오지 못했습니다 (도우미 단계: 화면 구성)"
		case "task", "content":
			return "PC 도우미의 준비 목록을 표시하지 못했습니다. 미입고 목록을 새로고침하고 다시 준비해 주세요 (도우미 단계: 준비 목록)"
		case "form", "show", "visibility":
			return "PC 도우미 창의 실제 표시를 확인하지 못했습니다. Windows 작업 표시줄에서 Alpha 도우미를 확인해 주세요 (도우미 단계: 창 표시)"
		case "start":
			return "Windows 도우미 프로세스를 실행하지 못했습니다 (도우미 단계: 실행)"
		case "timeout":
			return "PC 도우미 시작 응답이 지연돼 실행을 중단했습니다. 잠시 후 다시 눌러 주세요 (도우미 단계: 응답 대기)"
		}
	}
	return "PC 도우미를 열지 못했습니다 (도우미 단계: 실행 종료). 회사 PC에서 Alpha를 확인해 주세요"
}
func receiptHelperReadyHandle(line string) (uintptr, bool) {
	const prefix = "ALPHA_HELPER_READY:"
	if !strings.HasPrefix(line, prefix) {
		return 0, false
	}
	value, err := strconv.ParseUint(strings.TrimPrefix(line, prefix), 10, 64)
	return uintptr(value), err == nil && value > 0 && uint64(uintptr(value)) == value
}

type receiptHelperStageWriter struct {
	mu      sync.Mutex
	pending string
	stage   string
}

func (w *receiptHelperStageWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, c := range b {
		if c == '\n' {
			line := strings.TrimSpace(w.pending)
			w.pending = ""
			switch line {
			case "ALPHA_HELPER_FAILED:desktop":
				w.stage = "desktop"
			case "ALPHA_HELPER_FAILED:assemblies":
				w.stage = "assemblies"
			case "ALPHA_HELPER_FAILED:task":
				w.stage = "task"
			case "ALPHA_HELPER_FAILED:content":
				w.stage = "content"
			case "ALPHA_HELPER_FAILED:form":
				w.stage = "form"
			case "ALPHA_HELPER_FAILED:show":
				w.stage = "show"
			}
			continue
		}
		if len(w.pending) < 120 {
			w.pending += string(c)
		}
	}
	return len(b), nil
}
func (w *receiptHelperStageWriter) Stage() string { w.mu.Lock(); defer w.mu.Unlock(); return w.stage }

// A Shown event alone can be followed by immediate process exit or a hidden
// native window. Only report success after a live, visible-window check.
func receiptHelperVisibleAfter(handle uintptr, exited <-chan struct{}, visible func(uintptr) bool, stable time.Duration) bool {
	timer := time.NewTimer(stable)
	defer timer.Stop()
	select {
	case <-exited:
		return false
	case <-timer.C:
	}
	select {
	case <-exited:
		return false
	default:
	}
	return visible(handle)
}
