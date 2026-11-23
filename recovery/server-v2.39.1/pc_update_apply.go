package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Retain the last two successful binary backups. Only known updater-created
// files inside completed stages for this exact executable are eligible.
func pcCleanCompletedUpdates(target string) {
	entries, err := os.ReadDir(filepath.Dir(target))
	if err != nil {
		return
	}
	type completed struct {
		path string
		at   time.Time
	}
	var stages []completed
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), ".alpha-pc-update-") {
			continue
		}
		stage := filepath.Join(filepath.Dir(target), entry.Name())
		info, err := os.Stat(filepath.Join(stage, "complete"))
		if err != nil {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(stage, "plan.json"))
		if err != nil || len(raw) > 32768 {
			continue
		}
		var p pcUpdatePlan
		if json.Unmarshal(raw, &p) != nil || p.Target != target || p.Stage != stage {
			continue
		}
		stages = append(stages, completed{stage, info.ModTime()})
	}
	sort.Slice(stages, func(i, j int) bool { return stages[i].at.After(stages[j].at) })
	for i, stage := range stages {
		// The active helper may still hold its copy; a later launch retries.
		_ = os.Remove(filepath.Join(stage.path, "helper.exe"))
		if i < 2 {
			continue
		}
		for _, name := range []string{"previous.exe", "failed.exe", "new.exe", "package.zip", "plan.json", "helper-ready", "started", "complete", "error.txt"} {
			_ = os.Remove(filepath.Join(stage.path, name))
		}
		_ = os.Remove(stage.path) // Unknown files are never removed recursively.
	}
}

type pcUpdatePlan struct {
	Target    string   `json:"target"`
	Stage     string   `json:"stage"`
	OldSHA256 string   `json:"old_sha256"`
	NewSHA256 string   `json:"new_sha256"`
	ParentPID int      `json:"parent_pid"`
	Args      []string `json:"args"`
}

func (p pcUpdatePlan) validate(planPath string) error {
	if !filepath.IsAbs(p.Target) || !pcSafeStage(p.Stage) || filepath.Clean(filepath.Dir(p.Target)) != filepath.Clean(filepath.Dir(p.Stage)) || filepath.Clean(planPath) != filepath.Join(p.Stage, "plan.json") || !pcDigest(p.OldSHA256) || !pcDigest(p.NewSHA256) || p.ParentPID <= 0 {
		return errors.New("업데이트 준비 경로/정보가 올바르지 않습니다")
	}
	if len(p.Args) != 5 || p.Args[0] != "-data" || !filepath.IsAbs(p.Args[1]) || p.Args[2] != "-listen" || !strings.HasPrefix(p.Args[3], "127.0.0.1:") || p.Args[4] != "-no-browser" {
		return errors.New("업데이트 재시작 설정이 올바르지 않습니다")
	}
	if h, e := pcHashFile(p.Target); e != nil || h != p.OldSHA256 {
		return errors.New("기존 프로그램이 변경되었습니다")
	}
	if h, e := pcHashFile(filepath.Join(p.Stage, "new.exe")); e != nil || h != p.NewSHA256 {
		return errors.New("새 실행 파일 검증에 실패했습니다")
	}
	return nil
}
func pcWriteJSON(path string, v any) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	return os.WriteFile(path, b, 0600)
}
func pcUpdateReadyFile(path string) error {
	if path == "" {
		return nil
	}
	stage := filepath.Dir(path)
	if !pcSafeStage(stage) || filepath.Base(path) != "started" {
		return errors.New("올바르지 않은 업데이트 시작 확인 경로")
	}
	return os.WriteFile(path, []byte("ready"), 0600)
}

// Parent has fully exited and released the data/mutex locks before this is called.
// Launch reports nil only when the replacement reaches its ready marker. A launch
// failure is rolled back; a still-running startup is left alone to avoid killing
// a process that may be saving business data.
var errPCUpdateStillStarting = errors.New("새 프로그램이 시작 중입니다. 중복 실행하지 말고 창을 확인해 주세요. 이전 실행 파일은 업데이트 폴더에 보관했습니다")

func pcReplaceAndRestart(p pcUpdatePlan, launch func(string, []string, string) error) error {
	keepOld := func(cause error) error {
		if err := launch(p.Target, p.Args, ""); err != nil {
			return fmt.Errorf("이전 실행 파일은 유지되지만 다시 실행하지 못했습니다: %v (%w)", err, cause)
		}
		return cause
	}
	backup := filepath.Join(p.Stage, "previous.exe")
	next := filepath.Join(p.Stage, "new.exe")
	if _, e := os.Stat(backup); !os.IsNotExist(e) {
		return keepOld(errors.New("이전 버전 보관 경로를 사용할 수 없습니다"))
	}
	if e := os.Rename(p.Target, backup); e != nil {
		return keepOld(fmt.Errorf("기존 파일을 교체할 수 없습니다: %w", e))
	}
	restore := func(cause error) error {
		if e := os.Rename(backup, p.Target); e != nil {
			return fmt.Errorf("이전 버전 복원 실패: %v; 보관 위치: %s", e, backup)
		}
		if e := launch(p.Target, p.Args, ""); e != nil {
			return fmt.Errorf("이전 버전을 복원했으나 실행하지 못했습니다: %w", e)
		}
		return fmt.Errorf("새 버전을 실행하지 못해 이전 버전으로 돌아왔습니다: %w", cause)
	}
	if e := os.Rename(next, p.Target); e != nil {
		return restore(e)
	}
	ready := filepath.Join(p.Stage, "started")
	e := launch(p.Target, p.Args, ready)
	if e == nil {
		return os.WriteFile(filepath.Join(p.Stage, "complete"), []byte(time.Now().Format(time.RFC3339)), 0600)
	}
	if errors.Is(e, errPCUpdateStillStarting) {
		return e
	}
	// The failed replacement process has exited. Retain it for diagnostics.
	if move := os.Rename(p.Target, filepath.Join(p.Stage, "failed.exe")); move != nil {
		return fmt.Errorf("새 버전 실행 실패; 이전 버전 위치 %s: %w", backup, move)
	}
	return restore(e)
}
