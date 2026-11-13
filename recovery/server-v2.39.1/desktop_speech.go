package main

import (
	"context"
	"encoding/binary"
	"math"
	"strings"
	"sync"
	"unicode"
)

func speechField(s, fallback string, limit int) string {
	s = strings.Join(strings.FieldsFunc(s, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }), " ")
	r := []rune(s)
	if len(r) > limit {
		s = string(r[:limit]) + ", 나머지 내용은 화면에서 확인해 주세요"
	}
	if s == "" {
		return fallback
	}
	return s
}

// Existing request/note alerts only. Text is data, never SSML or code.
func inspectionSpeech(e desktopEvent) string {
	if e.SpeechPreview {
		return ttsTestText
	}
	if e.Rejected || (!e.Pending && !e.IsNote) {
		return ""
	}
	worker := speechField(e.Worker, "작업자", 100)
	project := speechField(e.Project, "이름 없음", 200)
	if e.IsNote {
		return worker + "님이 " + project + " 비고 메시지를 남겼습니다. " + speechField(e.Note, "내용을 화면에서 확인해 주세요", 300)
	}
	kind := map[string]string{"setting": "세팅검사", "shipping": "출하검사", "combined": "세팅 및 출하 통합검사"}[e.Inspection]
	if kind == "" {
		return ""
	}
	return worker + "님이 " + project + "의 " + kind + "를 신청했습니다."
}

type speechJob struct {
	text    string
	preview bool
	config  *TTSConfig
}
type inspectionAnnouncer struct {
	mu            sync.Mutex
	ctx           context.Context
	stop          context.CancelFunc
	cancel        context.CancelFunc
	wake          chan struct{}
	done          chan struct{}
	run           func(context.Context, string) error
	runConfigured func(context.Context, string, TTSConfig) error
	enabled       bool
	status        string
	queue         []speechJob
	seen          map[int64]bool
}

func newInspectionAnnouncer(enabled bool, run func(context.Context, string) error) *inspectionAnnouncer {
	ctx, stop := context.WithCancel(context.Background())
	s := &inspectionAnnouncer{ctx: ctx, stop: stop, wake: make(chan struct{}, 1), done: make(chan struct{}), run: run, seen: map[int64]bool{}}
	s.setEnabled(enabled)
	go s.loop()
	return s
}
func (s *inspectionAnnouncer) setEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.enabled == enabled && s.status != "" {
		return
	}
	s.enabled = enabled
	s.status = "신청 시 자동 읽기"
	if !enabled {
		s.status = "음성 알림 꺼짐"
		s.queue = nil
		if s.cancel != nil {
			s.cancel()
		}
	}
}
func (s *inspectionAnnouncer) state() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// Existing unread notices are still displayed on startup, without reading them again.
func (s *inspectionAnnouncer) prime(events []desktopEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range events {
		if e.ID > 0 {
			s.seen[e.ID] = true
		}
	}
}
func (s *inspectionAnnouncer) enqueue(e desktopEvent) {
	t := inspectionSpeech(e)
	if t == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID > 0 {
		if s.seen[e.ID] {
			return
		}
		s.seen[e.ID] = true
	}
	if (!s.enabled && !e.SpeechPreview) || s.ctx.Err() != nil {
		return
	}
	if len(s.queue) >= 256 {
		s.status = "음성 대기 초과 · 신청 목록 확인"
		return
	}
	var cfg *TTSConfig
	if e.SpeechConfig != nil {
		copy := *e.SpeechConfig
		cfg = &copy
	}
	s.queue = append(s.queue, speechJob{t, e.SpeechPreview, cfg})
	select {
	case s.wake <- struct{}{}:
	default:
	}
}
func (s *inspectionAnnouncer) loop() {
	defer close(s.done)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.wake:
		}
		for {
			s.mu.Lock()
			if len(s.queue) == 0 || (!s.enabled && !s.queue[0].preview) || s.ctx.Err() != nil {
				s.mu.Unlock()
				break
			}
			job := s.queue[0]
			s.queue = s.queue[1:]
			ctx, cancel := context.WithCancel(s.ctx)
			s.cancel = cancel
			s.status = "검사 신청 읽는 중"
			configured := s.runConfigured
			s.mu.Unlock()
			var err error
			if job.config != nil && configured != nil {
				err = configured(ctx, job.text, *job.config)
			} else {
				err = s.run(ctx, job.text)
			}
			s.mu.Lock()
			s.cancel = nil
			if (s.enabled || job.preview) && ctx.Err() == nil {
				s.status = "신청 시 자동 읽기"
				if job.preview {
					s.status = "테스트 재생 완료"
					if !s.enabled {
						s.status += " · 알림 꺼짐"
					}
				}
				if err != nil {
					s.status = "음성 실패 · " + err.Error()
				}
			}
			s.mu.Unlock()
			cancel()
		}
	}
}
func (s *inspectionAnnouncer) close() {
	s.stop()
}

// A small, self-contained two-tone doorbell, followed by silence before speech.
func inspectionChime() []byte {
	const rate = 22050
	n := rate * 11 / 10
	wav := make([]byte, 44+n*2)
	copy(wav, "RIFF")
	binary.LittleEndian.PutUint32(wav[4:], uint32(len(wav)-8))
	copy(wav[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(wav[16:], 16)
	binary.LittleEndian.PutUint16(wav[20:], 1)
	binary.LittleEndian.PutUint16(wav[22:], 1)
	binary.LittleEndian.PutUint32(wav[24:], rate)
	binary.LittleEndian.PutUint32(wav[28:], rate*2)
	binary.LittleEndian.PutUint16(wav[32:], 2)
	binary.LittleEndian.PutUint16(wav[34:], 16)
	copy(wav[36:], "data")
	binary.LittleEndian.PutUint32(wav[40:], uint32(n*2))
	for i := 0; i < n; i++ {
		t := float64(i) / rate
		freq, elapsed := 880.0, t
		if t >= .38 {
			freq, elapsed = 659.25, t-.38
		}
		amplitude := .33 * math.Min(1, elapsed/.012) * math.Exp(-elapsed*5)
		sample := int16(32767 * amplitude * (math.Sin(2*math.Pi*freq*t) + .15*math.Sin(4*math.Pi*freq*t)))
		binary.LittleEndian.PutUint16(wav[44+i*2:], uint16(sample))
	}
	return wav
}
