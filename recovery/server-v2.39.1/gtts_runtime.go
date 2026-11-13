package main

import (
	"archive/zip"
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Full, licensed Windows runtime is bundled so company PCs need no Python/pip installation.
//
//go:embed tts_assets/python-gtts-win-x64.zip tts_assets/runtime-manifest.json tts_assets/tts_helper.py
var ttsAssets embed.FS
var ttsRuntimeMu sync.Mutex
var ttsRuntimePath string
var gttsSlots = make(chan struct{}, 2)

func prepareTTSRuntime() (string, error) {
	ttsRuntimeMu.Lock()
	defer ttsRuntimeMu.Unlock()
	if ttsRuntimePath != "" {
		return ttsRuntimePath, nil
	}
	data, _ := ttsAssets.ReadFile("tts_assets/python-gtts-win-x64.zip")
	helper, _ := ttsAssets.ReadFile("tts_assets/tts_helper.py")
	manifest, _ := ttsAssets.ReadFile("tts_assets/runtime-manifest.json")
	var m struct{ Files map[string]string }
	if err := json.Unmarshal(manifest, &m); err != nil {
		return "", err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	base := filepath.Join(cache, "AlphaWorkCompleteRecovered", "gtts-runtime")
	if err = os.MkdirAll(base, 0700); err != nil {
		return "", err
	}
	key := digest(append(append(append([]byte{}, data...), helper...), manifest...))
	target := filepath.Join(base, key[:24])
	verify := func(dir string) bool {
		marker, err := os.ReadFile(filepath.Join(dir, ".runtime-sha256"))
		if err != nil || string(marker) != key {
			return false
		}
		for n, h := range m.Files {
			b, e := os.ReadFile(filepath.Join(dir, filepath.FromSlash(n)))
			if e != nil || digest(b) != h {
				return false
			}
		}
		b, e := os.ReadFile(filepath.Join(dir, "tts_helper.py"))
		return e == nil && bytes.Equal(b, helper)
	}
	if _, err = os.Stat(target); err == nil {
		if !verify(target) {
			return "", errors.New("음성 런타임 손상: 캐시 폴더를 확인하세요")
		}
		ttsRuntimePath = target
		return target, nil
	}
	stage, err := os.MkdirTemp(base, ".prepare-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	for _, f := range z.File {
		n := f.Name
		if strings.Contains(n, "\\") || strings.HasPrefix(n, "/") || strings.Contains(n, ":") || path.Clean(n) != n || strings.HasPrefix(n, "..") || f.UncompressedSize64 > 128<<20 || m.Files[n] == "" {
			return "", errors.New("잘못된 음성 런타임 파일")
		}
		in, e := f.Open()
		if e != nil {
			return "", e
		}
		b, e := io.ReadAll(io.LimitReader(in, 128<<20))
		in.Close()
		if e != nil || digest(b) != m.Files[n] {
			return "", errors.New("음성 런타임 체크섬 오류")
		}
		p := filepath.Join(stage, filepath.FromSlash(n))
		if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			return "", e
		}
		if e = os.WriteFile(p, b, 0600); e != nil {
			return "", e
		}
	}
	if err = os.WriteFile(filepath.Join(stage, "tts_helper.py"), helper, 0600); err != nil {
		return "", err
	}
	if err = os.WriteFile(filepath.Join(stage, ".runtime-sha256"), []byte(key), 0600); err != nil {
		return "", err
	}
	if !verify(stage) {
		return "", errors.New("음성 런타임 설치 검증 실패")
	}
	if err = os.Rename(stage, target); err != nil {
		return "", err
	}
	ttsRuntimePath = target
	return target, nil
}
func runGTTSPython(ctx context.Context, text, mode string) ([]byte, error) {
	return runTTSPython(ctx, text, mode, defaultTTSConfig())
}
func runTTSPython(ctx context.Context, text, mode string, cfg TTSConfig) ([]byte, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if runtime.GOOS != "windows" {
		return nil, errors.New("내장 음성 런타임은 Windows용입니다")
	}
	select {
	case gttsSlots <- struct{}{}:
		defer func() { <-gttsSlots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	root, err := prepareTTSRuntime()
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(filepath.Dir(root), ".job-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(90*cfg.Repeat)*time.Second)
	defer cancel()
	job := map[string]any{"text": text, "voice": cfg.Voice, "speed": cfg.Speed, "repeat": cfg.Repeat}
	if mode == "speak" {
		job["not_before"] = nativeSpeechNotBefore()
	}
	payload, _ := json.Marshal(job)
	cmd := exec.CommandContext(ctx, filepath.Join(root, "python.exe"), "-I", "-B", "-X", "utf8", filepath.Join(root, "tts_helper.py"), mode, dir)
	hideChildWindow(cmd)
	cmd.Stdin = bytes.NewReader(append(payload, '\n'))
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = io.Discard
	if err = cmd.Start(); err != nil {
		return nil, errors.New("TTS Python 런타임 시작 실패")
	}
	release, childErr := manageChild(cmd)
	if childErr != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, errors.New("TTS 프로세스 관리 실패")
	}
	defer release()
	err = cmd.Wait()
	out := output.Bytes()
	if err != nil {
		return nil, fmt.Errorf("%s / Python worker: %v", cfg.label(), err)
	}
	var result struct {
		OK     bool   `json:"ok"`
		MP3    string `json:"mp3"`
		Error  string `json:"error"`
		Detail string `json:"detail"`
	}
	if len(out) > 12<<20 || json.Unmarshal(bytes.TrimSpace(out), &result) != nil {
		return nil, fmt.Errorf("%s / Python worker 응답 형식 오류", cfg.label())
	}
	if !result.OK {
		return nil, fmt.Errorf("%s / %s: %s", cfg.label(), result.Error, result.Detail)
	}
	if mode == "speak" {
		return nil, nil
	}
	audio, err := base64.StdEncoding.DecodeString(result.MP3)
	if err != nil || len(audio) < 4 || len(audio) > 8<<20 {
		return nil, errors.New("음성 데이터 형식 오류")
	}
	return audio, nil
}
