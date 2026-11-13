package main

import (
	"context"
	"encoding/binary"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestInspectionSpeechContainsOnlyRequestedFieldsAndSupportedRequests(t *testing.T) {
	for _, kind := range []string{"setting", "shipping", "combined"} {
		e := desktopEvent{Pending: true, Inspection: kind, Project: "프로젝트\n이름", Worker: "홍길동\x00", Order: "READ-NO-ORDER", Note: "READ-NO-NOTE"}
		text := inspectionSpeech(e)
		if !strings.HasPrefix(text, "홍길동님이 프로젝트 이름의 ") || !strings.HasSuffix(text, "신청했습니다.") || strings.Contains(text, "READ-NO") || strings.ContainsAny(text, "\n\x00") {
			t.Fatal("wrong speech fields", text)
		}
		if kind == "combined" && !strings.Contains(text, "세팅 및 출하 통합검사") {
			t.Fatal(text)
		}
		for _, silent := range []desktopEvent{{Inspection: kind}, {Pending: true, Inspection: kind, Rejected: true}, {Pending: true, Inspection: "unknown"}} {
			if inspectionSpeech(silent) != "" {
				t.Fatal("non-application announced", silent)
			}
		}
	}
	note := inspectionSpeech(desktopEvent{IsNote: true, Project: "샘플", Worker: "김작업", Note: "센서를 확인해 주세요."})
	if note != "김작업님이 샘플 비고 메시지를 남겼습니다. 센서를 확인해 주세요." {
		t.Fatal(note)
	}
}
func awaitSpeech[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(3 * time.Second):
		t.Fatal("speech worker did not respond")
		var zero T
		return zero
	}
}
func TestInspectionAnnouncerFIFOAndDuplicateStartupReplaySuppression(t *testing.T) {
	started := make(chan string, 10)
	release := make(chan struct{}, 10)
	s := newInspectionAnnouncer(true, func(ctx context.Context, text string) error {
		started <- text
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	defer func() { s.close(); awaitSpeech(t, s.done) }()
	event := func(id int64, name string) desktopEvent {
		return desktopEvent{ID: id, Pending: true, Inspection: "setting", Project: name, Worker: "신청자"}
	}
	s.prime([]desktopEvent{event(1, "시작 전 알림")})
	s.enqueue(event(1, "시작 전 알림"))
	s.enqueue(event(2, "첫번째"))
	if text := awaitSpeech(t, started); !strings.Contains(text, "첫번째") {
		t.Fatal(text)
	}
	s.enqueue(event(2, "중복 알림"))
	s.enqueue(event(3, "두번째"))
	s.enqueue(event(4, "세번째"))
	select {
	case <-started:
		t.Fatal("voices overlapped")
	default:
	}
	release <- struct{}{}
	if text := awaitSpeech(t, started); !strings.Contains(text, "두번째") {
		t.Fatal(text)
	}
	release <- struct{}{}
	if text := awaitSpeech(t, started); !strings.Contains(text, "세번째") {
		t.Fatal(text)
	}
	s.close()
}
func TestInspectionAnnouncerMuteCancelsAndClearsQueueAndSurvivesFailure(t *testing.T) {
	started := make(chan string, 10)
	cancelled := make(chan struct{}, 1)
	s := newInspectionAnnouncer(true, func(ctx context.Context, text string) error {
		started <- text
		if strings.Contains(text, "취소") {
			<-ctx.Done()
			cancelled <- struct{}{}
			return ctx.Err()
		}
		return errors.New("synthetic missing voice")
	})
	defer func() { s.close(); awaitSpeech(t, s.done) }()
	e := desktopEvent{ID: 1, Pending: true, Inspection: "shipping", Project: "취소"}
	s.enqueue(e)
	awaitSpeech(t, started)
	e.ID, e.Project = 2, "대기 항목"
	s.enqueue(e)
	s.setEnabled(false)
	awaitSpeech(t, cancelled)
	if s.state() != "음성 알림 꺼짐" {
		t.Fatal(s.state())
	}
	e.ID = 3
	s.enqueue(e) // Muted notifications must not be replayed when enabling.
	s.setEnabled(true)
	s.enqueue(e)
	e.ID, e.Project = 4, "새 신청"
	s.enqueue(e)
	if text := awaitSpeech(t, started); !strings.Contains(text, "새 신청") {
		t.Fatal("muted/queued event leaked", text)
	}
	deadline := time.After(3 * time.Second)
	for !strings.Contains(s.state(), "음성 실패") {
		select {
		case <-deadline:
			t.Fatal("no visible error status")
		case <-time.After(time.Millisecond):
		}
	}
	e.ID = 5
	s.enqueue(e)
	awaitSpeech(t, started) // A failed voice must not terminate the notification worker.
}
func TestInspectionChimeIsValidBoundedPCM(t *testing.T) {
	b := inspectionChime()
	if string(b[:4]) != "RIFF" || string(b[8:16]) != "WAVEfmt " || string(b[36:40]) != "data" || int(binary.LittleEndian.Uint32(b[4:])) != len(b)-8 || int(binary.LittleEndian.Uint32(b[40:])) != len(b)-44 || binary.LittleEndian.Uint32(b[24:]) != 22050 {
		t.Fatal("invalid chime WAV")
	}
	peak := 0
	for i := 44; i < len(b); i += 2 {
		n := int(int16(binary.LittleEndian.Uint16(b[i:])))
		if n < 0 {
			n = -n
		}
		peak = max(peak, n)
	}
	if peak < 1000 || peak > 16000 {
		t.Fatal("silent or clipped chime", peak)
	}
}
func TestDesktopSpeechMuteSurvivesOperatingDataRestart(t *testing.T) {
	a := fixture(t)
	if a.state.Integration.DesktopSpeechMuted {
		t.Fatal("new installation should enable request speech")
	}
	if err := a.transaction(func(s *State) error { s.Integration.DesktopSpeechMuted = true; return nil }); err != nil {
		t.Fatal(err)
	}
	b, err := newApp(a.dir)
	if err != nil || !b.state.Integration.DesktopSpeechMuted || b.state.Projects[0].Name != a.state.Projects[0].Name {
		t.Fatal("mute setting or existing project lost", err)
	}
}

func TestNoticeRingsBeforeBlockedSynthesisAndDoesNotReplay(t *testing.T) {
	calls := make(chan string, 8)
	s := newInspectionAnnouncer(true, func(ctx context.Context, text string) error { calls <- "synthesis"; <-ctx.Done(); return ctx.Err() })
	defer func() { s.close(); awaitSpeech(t, s.done) }()
	s.mu.Lock()
	s.notice = func() { calls <- "ding" }
	s.mu.Unlock()
	event := desktopEvent{ID: 42, Pending: true, Inspection: "setting", Project: "알림", Worker: "작업자"}
	s.enqueue(event)
	if got := awaitSpeech(t, calls); got != "ding" {
		t.Fatal("notice waited for generation", got)
	}
	if got := awaitSpeech(t, calls); got != "synthesis" {
		t.Fatal(got)
	}
	s.enqueue(event)
	select {
	case got := <-calls:
		t.Fatal("duplicate rang again", got)
	default:
	}
	s.setEnabled(false)
	event.ID = 43
	s.enqueue(event)
	select {
	case got := <-calls:
		t.Fatal("muted alert rang", got)
	default:
	}
}
