package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

// Static script; project/worker names arrive through UTF-8 JSON on stdin, not interpolation.
const inspectionSpeechScript = `$ErrorActionPreference = 'Stop'
[Console]::InputEncoding = New-Object System.Text.UTF8Encoding($false)
[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false)
$s = $null
try {
  Add-Type -AssemblyName System.Speech
  $s = New-Object System.Speech.Synthesis.SpeechSynthesizer
  $voices = @($s.GetInstalledVoices() | Where-Object { $_.Enabled -and $_.VoiceInfo.Culture.Name -eq 'ko-KR' })
  if ($voices.Count -eq 0) { throw 'Korean voice is not installed' }
  $voice = $voices | Where-Object { $_.VoiceInfo.Gender -eq 'Female' } | Select-Object -First 1
  if ($null -eq $voice) { $voice = $voices[0] }
  $s.SelectVoice($voice.VoiceInfo.Name)
  $s.Rate = 0
  $s.Volume = 90
  $message = [Console]::In.ReadToEnd() | ConvertFrom-Json
  $s.Speak([string]$message.text)
  [Console]::Write('OK')
} catch { exit 1 } finally { if ($null -ne $s) { $s.Dispose() } }
`

func speechEncodedCommand() string {
	units := utf16.Encode([]rune(inspectionSpeechScript))
	b := make([]byte, len(units)*2)
	for i, u := range units {
		binary.LittleEndian.PutUint16(b[i*2:], u)
	}
	return base64.StdEncoding.EncodeToString(b)
}
func playInspectionSpeech(ctx context.Context, text string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	wav := inspectionChime()
	syscall.NewLazyDLL("winmm.dll").NewProc("PlaySoundW").Call(uintptr(unsafe.Pointer(&wav[0])), 0, 0x0006) // MEMORY | NODEFAULT, synchronous on worker
	runtime.KeepAlive(wav)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	// Absolute system path avoids resolving executables in a data/download directory.
	path := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	cmd := exec.CommandContext(ctx, path, "-NoLogo", "-NoProfile", "-NonInteractive", "-STA", "-EncodedCommand", speechEncodedCommand())
	hideChildWindow(cmd)
	payload, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return err
	}
	cmd.Stdin = bytes.NewReader(payload)
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "OK" {
		return errors.New("Windows 한국어 음성 재생 실패")
	}
	return nil
}
