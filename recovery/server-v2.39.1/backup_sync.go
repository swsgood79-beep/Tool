package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"

	"encoding/hex"
	"encoding/json"
	"errors"
	"filippo.io/age"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const maxBackupBytes = 256 << 20

type PortableManifest struct {
	Version                int
	Format, BoxID, Created string
	State                  State
	Checksums              map[string]string
}

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func snapshotHash(s State) string {
	s.Integration.Backup.LastSuccess = ""
	s.Integration.Backup.LastSnapshot = ""
	s.Integration.Backup.LastHash = ""
	s.Integration.Backup.LastAsset = ""
	s.Integration.Backup.LastSize = 0
	b, _ := json.Marshal(s)
	return digest(b)
}
func (a *App) portable(s State) ([]byte, error) {
	identity, e := age.ParseX25519Identity(s.Integration.Backup.Key.Identity)
	if e != nil {
		return nil, e
	}
	manifest := PortableManifest{Version: 1, Format: "wanryotalk-reconstructed-v2", BoxID: s.Integration.Backup.Key.BoxID, Created: time.Now().UTC().Format(time.RFC3339), State: s, Checksums: map[string]string{}}
	entries := map[string][]byte{}
	for _, p := range s.Projects {
		for _, ph := range p.Photos {
			entries["photos/"+ph.ID] = nil
		}
	}
	for _, c := range s.ClientNotes {
		for _, ph := range c.Photos {
			entries["photos/"+ph.ID] = nil
		}
	}
	for name := range entries {
		id := strings.TrimPrefix(name, "photos/")
		if !validPhotoID.MatchString(id) {
			return nil, errors.New("백업 사진 식별정보 오류")
		}
		p := filepath.Join(a.dir, "photos", id)
		legacy := false
		for _, o := range s.Projects {
			for _, ph := range o.Photos {
				if ph.ID == id && ph.Legacy {
					legacy = true
				}
			}
		}
		for _, c := range s.ClientNotes {
			for _, ph := range c.Photos {
				if ph.ID == id && ph.Legacy {
					legacy = true
				}
			}
		}
		if legacy {
			p = filepath.Join(a.dir, "project-photos", id+".dat")
		}
		b, err := os.ReadFile(p)
		if err == nil {
			b, err = unprotect(b)
		}
		if err != nil {
			return nil, fmt.Errorf("사진 백업 실패: %s", id)
		}
		entries[name] = b
		manifest.Checksums[name] = digest(b)
	}
	workbookBytes, e := workbook(s)
	if e != nil {
		return nil, e
	}
	entries["records.xlsx"] = workbookBytes
	manifest.Checksums["records.xlsx"] = digest(workbookBytes)
	m, e := json.Marshal(manifest)
	if e != nil {
		return nil, e
	}
	entries["manifest.json"] = m
	var raw bytes.Buffer
	z := zip.NewWriter(&raw)
	total := 0
	names := []string{}
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		b := entries[name]
		total += len(b)
		if total > maxBackupBytes {
			return nil, errors.New("백업 256MB 제한: 자료 분할이 필요합니다")
		}
		f, err := z.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err = f.Write(b); err != nil {
			return nil, err
		}
	}
	if e = z.Close(); e != nil {
		return nil, e
	}
	var encrypted bytes.Buffer
	w, e := age.Encrypt(&encrypted, identity.Recipient())
	if e != nil {
		return nil, e
	}
	if _, e = w.Write(raw.Bytes()); e != nil {
		return nil, e
	}
	if e = w.Close(); e != nil {
		return nil, e
	}
	return encrypted.Bytes(), nil
}
func decodePortable(encrypted []byte, key RecoveryKey) (PortableManifest, map[string][]byte, error) {
	var m PortableManifest
	id, e := age.ParseX25519Identity(key.Identity)
	if e != nil {
		return m, nil, e
	}
	reader, e := age.Decrypt(bytes.NewReader(encrypted), id)
	if e != nil {
		return m, nil, e
	}
	raw, e := io.ReadAll(io.LimitReader(reader, maxBackupBytes+1))
	if e != nil || len(raw) > maxBackupBytes {
		return m, nil, errors.New("백업 복호화·용량 검증 실패")
	}
	z, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if e != nil {
		return m, nil, e
	}
	entries := map[string][]byte{}
	var total uint64
	for _, f := range z.File {
		if f.Name != "manifest.json" && f.Name != "records.xlsx" && !(strings.HasPrefix(f.Name, "photos/") && validPhotoID.MatchString(strings.TrimSuffix(strings.TrimPrefix(f.Name, "photos/"), ".jpg"))) {
			return m, nil, errors.New("백업 경로 검증 실패")
		}
		if _, ok := entries[f.Name]; ok {
			return m, nil, errors.New("백업 중복 경로")
		}
		total += f.UncompressedSize64
		if total > maxBackupBytes {
			return m, nil, errors.New("압축 해제 용량 제한")
		}
		r, e := f.Open()
		if e != nil {
			return m, nil, e
		}
		b, e := io.ReadAll(io.LimitReader(r, maxBackupBytes+1))
		r.Close()
		if e != nil || uint64(len(b)) != f.UncompressedSize64 {
			return m, nil, errors.New("백업 항목 손상")
		}
		entries[f.Name] = b
	}
	if e = json.Unmarshal(entries["manifest.json"], &m); e != nil {
		return m, nil, e
	}
	if m.Format == "" {
		return originalBackupState(entries["manifest.json"], key, entries)
	}
	if m.Version != 1 || m.Format != "wanryotalk-reconstructed-v2" || m.BoxID != key.BoxID || m.State.Version != 1 || !validPhotoID.MatchString(m.State.Board) {
		return m, nil, errors.New("이 복구판에서 지원하지 않는 백업 형식입니다. 원본 백업은 보존되어 있습니다")
	}
	for name := range entries {
		if name != "manifest.json" {
			if _, ok := m.Checksums[name]; !ok {
				return m, nil, errors.New("체크섬 누락 항목")
			}
		}
	}
	for n, hash := range m.Checksums {
		b, ok := entries[n]
		if !ok || digest(b) != hash {
			return m, nil, errors.New("백업 체크섬 불일치")
		}
	}
	return m, entries, nil
}

type GitAsset struct {
	ID                              int64
	Name, Label, BrowserDownloadURL string `json:"-"`
}
type Release struct {
	ID     int64  `json:"id"`
	Body   string `json:"body"`
	Assets []struct {
		ID      int64  `json:"id"`
		Name    string `json:"name"`
		Label   string `json:"label"`
		Created string `json:"created_at"`
	} `json:"assets"`
}

func (a *App) github(ctx context.Context, method, path, token string, body []byte, ctype string) ([]byte, int, error) {
	endpoint := "https://api.github.com" + path
	if strings.HasPrefix(path, "UPLOAD:") {
		endpoint = "https://uploads.github.com" + strings.TrimPrefix(path, "UPLOAD:")
	}
	r, e := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if e != nil {
		return nil, 0, e
	}
	r.Header.Set("Accept", "application/vnd.github+json")
	r.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if ctype != "" {
		r.Header.Set("Content-Type", ctype)
	}
	res, e := a.httpClient().Do(r)
	if e != nil {
		return nil, 0, e
	}
	defer res.Body.Close()
	b, e := io.ReadAll(io.LimitReader(res.Body, maxBackupBytes+1))
	if len(b) > maxBackupBytes {
		return nil, res.StatusCode, errors.New("GitHub 응답 용량 제한")
	}
	return b, res.StatusCode, e
}
func backupTag(key RecoveryKey) string { return "alpha-recovered-backup-" + key.BoxID }
func (a *App) backupRelease(ctx context.Context, key RecoveryKey, token string, create bool) (Release, error) {
	var rel Release
	if key.Repository != repository || !validBoxID.MatchString(key.BoxID) {
		return rel, errors.New("복구키 저장소·식별자 오류")
	}
	tag := backupTag(key)
	p := "/repos/" + repository + "/releases"
	b, status, e := a.github(ctx, "GET", p+"/tags/"+url.PathEscape(tag), token, nil, "")
	if e != nil {
		return rel, e
	}
	legacyRelease := false
	if status == 404 && !create {
		b, status, e = a.github(ctx, "GET", p+"/tags/"+url.PathEscape("alpha-backup-"+key.BoxID), token, nil, "")
		legacyRelease = status == 200
	}
	if status == 404 && create {
		body, _ := json.Marshal(map[string]any{"tag_name": tag, "name": "암호화 복구 서버 백업", "body": "wanryotalk-reconstructed-v2", "prerelease": true, "make_latest": "false"})
		b, status, e = a.github(ctx, "POST", p, token, body, "application/json")
	}
	if e != nil {
		return rel, e
	}
	if status != 200 && status != 201 {
		return rel, fmt.Errorf("GitHub 백업 HTTP %d", status)
	}
	if e = json.Unmarshal(b, &rel); e != nil {
		return rel, e
	}
	if (rel.Body != "wanryotalk-reconstructed-v2" && !(legacyRelease && strings.Contains(rel.Body, "alpha-portable-backup-v1"))) || rel.ID <= 0 {
		return rel, errors.New("다른 Release 형식은 변경하지 않습니다")
	}
	return rel, nil
}
func (a *App) runBackup(ctx context.Context) {
	a.mu.Lock()
	if a.integration.BackupBusy {
		a.mu.Unlock()
		return
	}
	s, e := clone(a.state)
	if e != nil {
		a.mu.Unlock()
		return
	}
	if s.Integration.Backup.Key.Identity == "" || s.Integration.Connection.GitHubToken == "" {
		a.integration.BackupStatus = "복구키와 GitHub 연결키를 먼저 설정하세요"
		a.mu.Unlock()
		return
	}
	a.integration.BackupBusy = true
	a.integration.BackupStatus = "전체 자료 암호화 중"
	a.mu.Unlock()
	snapshotTime := time.Now().UTC().Format(time.RFC3339)
	encrypted, e := a.portable(s)
	var assetName string
	if e == nil {
		var rel Release
		rel, e = a.backupRelease(ctx, s.Integration.Backup.Key, s.Integration.Connection.GitHubToken, true)
		if e == nil {
			assetName = "alpha-" + time.Now().UTC().Format("20060102T150405Z") + "-" + token()[:8] + ".age"
			var status int
			_, status, e = a.github(ctx, "POST", fmt.Sprintf("UPLOAD:/repos/%s/releases/%d/assets?name=%s&label=recovered-v2", repository, rel.ID, url.QueryEscape(assetName)), s.Integration.Connection.GitHubToken, encrypted, "application/octet-stream")
			if e == nil && status != 201 {
				e = fmt.Errorf("암호화 업로드 HTTP %d", status)
			}
			if e == nil {
				fresh, err := a.backupRelease(ctx, s.Integration.Backup.Key, s.Integration.Connection.GitHubToken, false)
				if err != nil {
					e = err
				} else {
					found := int64(0)
					for _, asset := range fresh.Assets {
						if asset.Name == assetName {
							found = asset.ID
						}
					}
					if found == 0 {
						e = errors.New("업로드한 백업 확인 실패")
					} else {
						download, err := a.fetchGitAsset(ctx, found, s.Integration.Connection.GitHubToken)
						if err != nil {
							e = err
						} else if digest(download) != digest(encrypted) {
							e = errors.New("전송된 백업 체크섬 불일치")
						}
						if e == nil {
							if _, _, err = decodePortable(download, s.Integration.Backup.Key); err != nil {
								e = err
							}
						}
						if e == nil {
							e = a.pruneBackups(ctx, fresh, s.Integration.Connection.GitHubToken, assetName)
						}

					}
				}
			}
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.integration.BackupBusy = false
	if e != nil {
		a.integration.BackupStatus = "백업 실패: " + e.Error()
		return
	}
	e = a.transaction(func(next *State) error {
		if next.Integration.Backup.Key.BoxID != s.Integration.Backup.Key.BoxID {
			return errors.New("백업 설정 변경됨")
		}
		b := &next.Integration.Backup
		b.LastHash = snapshotHash(s)
		b.LastSuccess = time.Now().UTC().Format(time.RFC3339)
		b.LastSnapshot = snapshotTime
		if b.LastSnapshot == "" {
			b.LastSnapshot = time.Now().UTC().Format(time.RFC3339)
		}
		b.LastSize = int64(len(encrypted))
		b.LastAsset = assetName
		return nil
	})
	if e != nil {
		a.integration.BackupStatus = "업로드 후 상태 저장 실패"
	} else {
		a.integration.BackupStatus = "암호화 백업 업로드 완료 · 원본 백업은 유지됩니다"
	}
}
func (a *App) maybeBackup(ctx context.Context) {
	a.mu.Lock()
	s := a.state
	b := s.Integration.Backup
	changed := snapshotHash(s) != b.LastHash
	last, _ := time.Parse(time.RFC3339, b.LastSuccess)
	a.mu.Unlock()
	if b.Enabled && changed && time.Since(last) >= 5*time.Minute {
		a.runBackup(ctx)
	}
}
func (a *App) restorePortable(key RecoveryKey, encrypted []byte) error {
	if len(a.state.Projects) > 0 || len(a.state.Events) > 0 || len(a.state.Designers) > 0 || len(a.state.ClientNotes) > 0 || a.state.Integration.Phone.Subscription != nil {
		return errors.New("자료가 없는 새 복구 서버에서만 복원할 수 있습니다")
	}
	m, entries, e := decodePortable(encrypted, key)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Join(a.dir, "photos"), 0700); e != nil {
		return e
	}
	for name, data := range entries {
		if !strings.HasPrefix(name, "photos/") {
			continue
		}
		protected, err := protect(data)
		if err != nil {
			return err
		}
		if err = atomicFile(filepath.Join(a.dir, "photos", strings.TrimPrefix(name, "photos/")), protected); err != nil {
			return err
		}
	}
	next := m.State
	next.LegacySourceHash = ""
	next.LegacyHashes = nil
	next.Integration.Connection.PublicURL = ""
	for i := range next.Projects {
		for j := range next.Projects[i].Photos {
			next.Projects[i].Photos[j].Legacy = false
		}
	}
	for i := range next.ClientNotes {
		for j := range next.ClientNotes[i].Photos {
			next.ClientNotes[i].Photos[j].Legacy = false
		}
	}
	if e = a.persist(next); e != nil {
		return e
	}
	a.state = next
	a.sessions = map[string]Session{}
	a.initIntegration()
	return nil
}
func (a *App) downloadBackup(ctx context.Context, key RecoveryKey, token string, generation int) ([]byte, error) {
	rel, e := a.backupRelease(ctx, key, token, false)
	if e != nil {
		return nil, e
	}
	sort.Slice(rel.Assets, func(i, j int) bool { return rel.Assets[i].Created > rel.Assets[j].Created })
	assets := []int64{}
	for _, x := range rel.Assets {
		if (x.Label == "recovered-v2" || x.Label == "alpha-verified-v1") && strings.HasSuffix(x.Name, ".age") {
			assets = append(assets, x.ID)
		}
	}
	if generation < 0 || generation >= len(assets) {
		return nil, errors.New("요청한 복원 세대가 없습니다")
	}
	r, e := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://api.github.com/repos/%s/releases/assets/%d", repository, assets[generation]), nil)
	if e != nil {
		return nil, e
	}
	r.Header.Set("Accept", "application/octet-stream")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	res, e := a.httpClient().Do(r)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("백업 다운로드 HTTP %d", res.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, maxBackupBytes+1))
	if len(b) > maxBackupBytes {
		return nil, errors.New("백업 용량 초과")
	}
	return b, e
}

var validBoxID = regexp.MustCompile(`^[A-Za-z0-9_-]{16,64}$`)

func (a *App) fetchGitAsset(ctx context.Context, id int64, bearer string) ([]byte, error) {
	r, e := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://api.github.com/repos/%s/releases/assets/%d", repository, id), nil)
	if e != nil {
		return nil, e
	}
	r.Header.Set("Accept", "application/octet-stream")
	if bearer != "" {
		r.Header.Set("Authorization", "Bearer "+bearer)
	}
	res, e := a.httpClient().Do(r)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("암호화 백업 다운로드 HTTP %d", res.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, maxBackupBytes+1))
	if len(b) > maxBackupBytes {
		return nil, errors.New("백업 용량 제한")
	}
	return b, e
}

func (a *App) pruneBackups(ctx context.Context, rel Release, bearer, keepName string) error {
	if rel.Body != "wanryotalk-reconstructed-v2" {
		return errors.New("원본 백업 Release는 정리하지 않습니다")
	}
	sort.Slice(rel.Assets, func(i, j int) bool {
		if rel.Assets[i].Created == rel.Assets[j].Created {
			return rel.Assets[i].ID > rel.Assets[j].ID
		}
		return rel.Assets[i].Created > rel.Assets[j].Created
	})
	kept := 1
	for _, asset := range rel.Assets {
		if asset.Label != "recovered-v2" || !strings.HasPrefix(asset.Name, "alpha-") || !strings.HasSuffix(asset.Name, ".age") || asset.Name == keepName {
			continue
		}
		if kept < 3 {
			kept++
			continue
		}
		_, status, e := a.github(ctx, "DELETE", fmt.Sprintf("/repos/%s/releases/assets/%d", repository, asset.ID), bearer, nil, "")
		if e != nil {
			return e
		}
		if status != 204 && status != 404 {
			return fmt.Errorf("새 백업은 검증됐으나 이전 복구판 백업 정리 HTTP %d", status)
		}
	}
	return nil
}
