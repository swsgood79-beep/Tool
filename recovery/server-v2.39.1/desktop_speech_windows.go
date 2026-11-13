package main

import (
	"context"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// WinMM reads SND_MEMORY asynchronously, so the WAV must remain a global GC root.
var nativeBellWave = inspectionChime()
var nativeBellMu sync.Mutex
var nativeBellLast, timeNativeBellUntil time.Time

func playImmediateInspectionChime() {
	nativeBellMu.Lock()
	defer nativeBellMu.Unlock()
	now := time.Now()
	if !nativeBellLast.IsZero() && now.Sub(nativeBellLast) < time.Second {
		return
	}
	nativeBellLast = now
	timeNativeBellUntil = now.Add(1100 * time.Millisecond)
	syscall.NewLazyDLL("winmm.dll").NewProc("PlaySoundW").Call(uintptr(unsafe.Pointer(&nativeBellWave[0])), 0, 0x0007) // ASYNC | MEMORY | NODEFAULT
}
func stopImmediateInspectionChime() {
	nativeBellMu.Lock()
	defer nativeBellMu.Unlock()
	nativeBellLast = time.Time{}
	timeNativeBellUntil = time.Time{}
	syscall.NewLazyDLL("winmm.dll").NewProc("PlaySoundW").Call(0, 0, 0)
}
func nativeSpeechNotBefore() float64 {
	nativeBellMu.Lock()
	defer nativeBellMu.Unlock()
	if timeNativeBellUntil.IsZero() {
		return 0
	}
	return float64(timeNativeBellUntil.UnixNano()) / 1e9
}
func playInspectionSpeech(ctx context.Context, text string) error {
	_, err := runGTTSPython(ctx, text, "speak")
	return err
}
