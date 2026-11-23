package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

func (a *App) runBackup(ctx context.Context) { a.runSegmentedBackup(ctx, backupPartBytes) }
func (a *App) runSegmentedBackup(ctx context.Context, partBytes int64) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Hour)
	defer cancel()
	a.mu.Lock()
	if a.integration.BackupBusy {
		a.mu.Unlock()
		return
	}
	if a.state.Integration.Backup.Key.Identity == "" || a.state.Integration.Connection.GitHubToken == "" {
		a.integration.BackupStatus = "복구키와 GitHub 연결키를 먼저 설정하세요"
		a.mu.Unlock()
		return
	}
	if e := a.transaction(func(s *State) error {
		s.Integration.Backup.LastAttempt = time.Now().UTC().Format(time.RFC3339)
		return nil
	}); e != nil {
		a.integration.BackupStatus = "백업 시작 상태 저장 실패"
		a.mu.Unlock()
		return
	}
	s, e := clone(a.state)
	if e != nil {
		a.integration.BackupStatus = "백업 스냅샷 생성 실패"
		a.mu.Unlock()
		return
	}
	epoch := a.epoch
	refs := backupAssets(s)
	if a.backupPins == nil {
		a.backupPins = map[string]int{}
	}
	for _, p := range refs {
		a.backupPins[p.ID]++
	}
	a.integration.BackupBusy = true
	a.integration.BackupStatus = "전체 자료를 디스크에서 암호화 중 · 200MB 단위 분할"
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.integration.BackupBusy = false
		for _, p := range refs {
			a.backupPins[p.ID]--
			if a.backupPins[p.ID] <= 0 {
				delete(a.backupPins, p.ID)
			}
		}
		a.pruneCommunityPhotos(refs)
	}()
	snapshotTime := time.Now().UTC().Format(time.RFC3339)
	var index backupSetIndex
	var name string
	var rel Release
	dir, e := os.MkdirTemp(a.dir, ".backup-build-")
	if e == nil {
		defer os.RemoveAll(dir)
		var path string
		path, e = buildPortableFile(ctx, a.dir, dir, s)
		if e == nil {
			a.backupProgress("암호화 자료 전체 검증 중")
			verifyDir, err := os.MkdirTemp(dir, "verify-")
			if err != nil {
				e = err
			} else {
				_, e = verifyPortableFile(ctx, path, verifyDir, s.Integration.Backup.Key)
				_ = os.RemoveAll(verifyDir)
			}
		}

		if e == nil {
			rel, e = a.backupRelease(ctx, s.Integration.Backup.Key, s.Integration.Connection.GitHubToken, true)
		}
		if e == nil {
			index, name, e = a.uploadBackupSet(ctx, rel, s.Integration.Backup.Key, s.Integration.Connection.GitHubToken, path, dir, partBytes)
		}
	}
	a.mu.Lock()
	if e != nil {
		a.integration.BackupStatus = "백업 실패 · 직전 정상 백업 유지: " + e.Error()
		_ = a.transaction(func(next *State) error { next.Integration.Backup.LastError = e.Error(); return nil })
		a.mu.Unlock()
		return
	}
	e = a.transaction(func(next *State) error {
		if a.epoch != epoch || next.Integration.Backup.Key.BoxID != s.Integration.Backup.Key.BoxID {
			return errors.New("백업 중 설정이 변경되어 이전 백업은 정리하지 않습니다")
		}
		b := &next.Integration.Backup
		b.LastHash = snapshotHash(s)
		b.LastSuccess = time.Now().UTC().Format(time.RFC3339)
		b.LastSnapshot = snapshotTime
		b.LastAsset = name
		b.LastSize = index.Bytes
		b.LastParts = len(index.Parts)
		b.LastError = ""
		return nil
	})
	if e != nil {
		a.integration.BackupStatus = "백업 업로드는 검증됐지만 상태 저장 실패 · 이전 백업 유지"
		a.mu.Unlock()
		return
	}
	a.integration.BackupStatus = fmt.Sprintf("백업 완료 · %d개 조각 · %s · 이전 정상 백업 유지", len(index.Parts), backupSizeLabel(index.Bytes))
	a.mu.Unlock()
	// Prune only after both remote verification and local success have committed.
	fresh, cleanErr := a.backupRelease(ctx, s.Integration.Backup.Key, s.Integration.Connection.GitHubToken, false)
	if cleanErr == nil {
		fresh, cleanErr = a.listBackupAssets(ctx, fresh, s.Integration.Connection.GitHubToken)
	}
	if cleanErr == nil {
		cleanErr = a.pruneBackupSets(ctx, fresh, s.Integration.Backup.Key, s.Integration.Connection.GitHubToken, name)
	}
	if cleanErr != nil {
		a.backupProgress("백업 완료 · 이전 백업 정리 보류: " + cleanErr.Error())
	}
}
