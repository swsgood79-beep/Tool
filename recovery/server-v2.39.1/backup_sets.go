package main

import (
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

const backupSetLabel = "recovered-v3"
const backupPartLabel = "recovered-v3-part"
const backupPendingLabel = "recovered-v3-pending"

type backupSetPart struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}
type backupSetIndex struct {
	Version int             `json:"version"`
	Format  string          `json:"format"`
	BoxID   string          `json:"box_id"`
	Set     string          `json:"set"`
	Created string          `json:"created"`
	Bytes   int64           `json:"bytes"`
	SHA256  string          `json:"sha256"`
	Parts   []backupSetPart `json:"parts"`
}

var backupSetID = regexp.MustCompile(`^alpha-set-[0-9]{8}T[0-9]{6}Z-[A-Za-z0-9_-]{12}$`)

func encryptBackupIndex(index backupSetIndex, key RecoveryKey) ([]byte, error) {
	id, e := age.ParseX25519Identity(key.Identity)
	if e != nil {
		return nil, e
	}
	var b bytes.Buffer
	w, e := age.Encrypt(&b, id.Recipient())
	if e != nil {
		return nil, e
	}
	if e = json.NewEncoder(w).Encode(index); e != nil {
		return nil, e
	}
	if e = w.Close(); e != nil {
		return nil, e
	}
	return b.Bytes(), nil
}
func decodeBackupIndex(data []byte, key RecoveryKey) (backupSetIndex, error) {
	var index backupSetIndex
	id, e := age.ParseX25519Identity(key.Identity)
	if e != nil {
		return index, e
	}
	r, e := age.Decrypt(bytes.NewReader(data), id)
	if e != nil {
		return index, e
	}
	b, e := io.ReadAll(io.LimitReader(r, (8<<20)+1))
	if e != nil {
		return index, e
	}
	if len(b) > 8<<20 {
		return index, errors.New("백업 세트 정보 용량 초과")
	}
	if e = json.Unmarshal(b, &index); e != nil {
		return index, e
	}
	if e = index.validate(key); e != nil {
		return index, e
	}
	return index, nil
}
func (index backupSetIndex) validate(key RecoveryKey) error {
	if index.Version != 1 || index.Format != "alpha-segmented-backup-v1" || index.BoxID != key.BoxID || !backupSetID.MatchString(index.Set) || !pcDigest(index.SHA256) || index.Bytes < 1 || index.Bytes > backupSetMaxBytes || len(index.Parts) < 1 || len(index.Parts) > 10000 {
		return errors.New("분할 백업 정보 오류")
	}
	var total int64
	ids := map[int64]bool{}
	for i, p := range index.Parts {
		if p.ID <= 0 || ids[p.ID] || p.Name != fmt.Sprintf("%s.part-%05d.age", index.Set, i+1) || p.Size < 1 || p.Size > backupPartBytes || !pcDigest(p.SHA256) || p.Size > backupSetMaxBytes-total {
			return errors.New("백업 조각 정보/순서 오류")
		}
		ids[p.ID] = true
		total += p.Size
	}
	if total != index.Bytes {
		return errors.New("백업 전체 용량 불일치")
	}
	return nil
}
func (a *App) backupLongClient() *http.Client {
	client := *a.httpClient()
	client.Timeout = 20 * time.Minute
	return &client
}
func (a *App) listBackupAssets(ctx context.Context, rel Release, bearer string) (Release, error) {
	if rel.Body != "wanryotalk-reconstructed-v2" {
		return rel, nil
	} // Legacy releases remain read-only.
	rel.Assets = nil
	seen := map[int64]bool{}
	for page := 1; page <= 1000; page++ {
		raw, status, e := a.github(ctx, "GET", fmt.Sprintf("/repos/%s/releases/%d/assets?per_page=100&page=%d", repository, rel.ID, page), bearer, nil, "")
		if e != nil {
			return rel, e
		}
		if status != 200 {
			return rel, fmt.Errorf("백업 조각 목록 HTTP %d", status)
		}
		var assets []ReleaseAsset
		if e = json.Unmarshal(raw, &assets); e != nil {
			return rel, e
		}
		for _, asset := range assets {
			if asset.ID <= 0 || seen[asset.ID] {
				return rel, errors.New("백업 조각 목록 중복/손상")
			}
			seen[asset.ID] = true
			rel.Assets = append(rel.Assets, asset)
		}
		if len(assets) < 100 {
			return rel, nil
		}
	}
	return rel, errors.New("백업 파일 목록 안전 한도 초과")
}
func (a *App) uploadBackupReader(ctx context.Context, releaseID int64, name, label, bearer string, r io.Reader, size int64) (ReleaseAsset, error) {
	var asset ReleaseAsset
	req, e := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("https://uploads.github.com/repos/%s/releases/%d/assets?name=%s&label=%s", repository, releaseID, url.QueryEscape(name), url.QueryEscape(label)), r)
	if e != nil {
		return asset, e
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, e := a.backupLongClient().Do(req)
	if e != nil {
		return asset, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return asset, fmt.Errorf("백업 업로드 HTTP %d", resp.StatusCode)
	}
	if e = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&asset); e != nil {
		return asset, e
	}
	if asset.ID <= 0 || asset.Name != name || asset.Label != label || (asset.Size != 0 && asset.Size != size) {
		return asset, errors.New("업로드한 백업 정보 불일치")
	}
	return asset, nil
}
func (a *App) fetchBackupTo(ctx context.Context, id int64, bearer string, w io.Writer, limit int64) (int64, string, error) {
	req, e := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://api.github.com/repos/%s/releases/assets/%d", repository, id), nil)
	if e != nil {
		return 0, "", e
	}
	req.Header.Set("Accept", "application/octet-stream")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, e := a.backupLongClient().Do(req)
	if e != nil {
		return 0, "", e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, "", fmt.Errorf("백업 다운로드 HTTP %d", resp.StatusCode)
	}
	h := sha256.New()
	n, e := copyBackup(ctx, io.MultiWriter(w, h), resp.Body, limit+1)
	if e != nil {
		return n, "", e
	}
	if n > limit {
		return n, "", errors.New("백업 조각 용량 초과")
	}
	return n, hex.EncodeToString(h.Sum(nil)), nil
}
func (a *App) fetchBackupIndex(ctx context.Context, asset ReleaseAsset, bearer string, key RecoveryKey) (backupSetIndex, error) {
	var data bytes.Buffer
	_, _, e := a.fetchBackupTo(ctx, asset.ID, bearer, &data, 8<<20)
	if e != nil {
		return backupSetIndex{}, e
	}
	index, e := decodeBackupIndex(data.Bytes(), key)
	if e == nil && asset.Name != index.Set+".set.age" {
		e = errors.New("백업 세트 이름 불일치")
	}
	return index, e
}
func (a *App) uploadBackupSet(ctx context.Context, rel Release, key RecoveryKey, bearer, path, dir string, partSize int64) (backupSetIndex, string, error) {
	var index backupSetIndex
	if partSize < 1 || partSize > backupPartBytes {
		return index, "", errors.New("백업 조각 용량 설정 오류")
	}
	f, e := os.Open(path)
	if e != nil {
		return index, "", e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return index, "", e
	}
	hash, e := pcHashFile(path)
	if e != nil {
		return index, "", e
	}
	index = backupSetIndex{Version: 1, Format: "alpha-segmented-backup-v1", BoxID: key.BoxID, Set: "alpha-set-" + time.Now().UTC().Format("20060102T150405Z") + "-" + token()[:12], Created: time.Now().UTC().Format(time.RFC3339), Bytes: st.Size(), SHA256: hash}
	created := []int64{}
	committed := false
	defer func() {
		if !committed {
			clean, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			for i := len(created) - 1; i >= 0; i-- {
				_, status, err := a.github(clean, "DELETE", fmt.Sprintf("/repos/%s/releases/assets/%d", repository, created[i]), bearer, nil, "")
				if err != nil || (status != 204 && status != 404) {
					break
				}
			}
		}
	}()
	count := (st.Size() + partSize - 1) / partSize
	for offset := int64(0); offset < st.Size(); offset += partSize {
		if e = ctx.Err(); e != nil {
			return index, "", e
		}
		size := min(partSize, st.Size()-offset)
		name := fmt.Sprintf("%s.part-%05d.age", index.Set, len(index.Parts)+1)
		h := sha256.New()
		if _, e = copyBackup(ctx, h, io.NewSectionReader(f, offset, size), size); e != nil {
			return index, "", e
		}
		expected := hex.EncodeToString(h.Sum(nil))
		a.backupProgress(fmt.Sprintf("백업 조각 %d/%d 업로드 · 전체 %.1f MB", len(index.Parts)+1, count, float64(st.Size())/(1<<20)))
		asset, e := a.uploadBackupReader(ctx, rel.ID, name, backupPartLabel, bearer, io.NewSectionReader(f, offset, size), size)
		if asset.ID > 0 {
			created = append(created, asset.ID)
		}
		if e != nil {
			return index, "", e
		}
		n, actual, e := a.fetchBackupTo(ctx, asset.ID, bearer, io.Discard, size)
		if e != nil {
			return index, "", e
		}
		if n != size || actual != expected {
			return index, "", errors.New("업로드한 백업 조각 검증 실패")
		}
		index.Parts = append(index.Parts, backupSetPart{ID: asset.ID, Name: name, Size: size, SHA256: expected})
	}
	if e = index.validate(key); e != nil {
		return index, "", e
	}
	// The encrypted completion index is published only after all remote parts verify.
	raw, e := encryptBackupIndex(index, key)
	if e != nil {
		return index, "", e
	}
	name := index.Set + ".set.age"
	asset, e := a.uploadBackupReader(ctx, rel.ID, name, backupPendingLabel, bearer, bytes.NewReader(raw), int64(len(raw)))
	if asset.ID > 0 {
		created = append(created, asset.ID)
	}
	if e != nil {
		return index, "", e
	}
	var remote bytes.Buffer
	n, h, e := a.fetchBackupTo(ctx, asset.ID, bearer, &remote, int64(len(raw)))
	if e != nil || n != int64(len(raw)) || h != digest(raw) {
		return index, "", errors.New("완료 백업 정보 검증 실패")
	}
	if _, e = decodeBackupIndex(remote.Bytes(), key); e != nil {
		return index, "", e
	}
	patch, _ := json.Marshal(map[string]string{"label": backupSetLabel})
	answer, status, e := a.github(ctx, "PATCH", fmt.Sprintf("/repos/%s/releases/assets/%d", repository, asset.ID), bearer, patch, "application/json")
	if e != nil {
		return index, "", e
	}
	var published ReleaseAsset
	if status != 200 || json.Unmarshal(answer, &published) != nil || published.ID != asset.ID || published.Name != name || published.Label != backupSetLabel {
		return index, "", errors.New("완료 백업 등록 실패")
	}
	committed = true
	return index, name, nil
}
func completedBackups(rel Release) []ReleaseAsset {
	var out []ReleaseAsset
	for _, asset := range rel.Assets {
		if asset.Label == backupSetLabel && strings.HasSuffix(asset.Name, ".set.age") || ((asset.Label == "recovered-v2" || asset.Label == "alpha-verified-v1") && strings.HasSuffix(asset.Name, ".age")) {
			out = append(out, asset)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Created == out[j].Created {
			return out[i].ID > out[j].ID
		}
		return out[i].Created > out[j].Created
	})
	return out
}
func (a *App) downloadBackupSet(ctx context.Context, key RecoveryKey, bearer string, rel Release, asset ReleaseAsset, dir string) (string, error) {
	index, e := a.fetchBackupIndex(ctx, asset, bearer, key)
	if e != nil {
		return "", e
	}
	available := map[int64]ReleaseAsset{}
	for _, v := range rel.Assets {
		available[v.ID] = v
	}
	for _, part := range index.Parts {
		v, ok := available[part.ID]
		if !ok || v.Name != part.Name || v.Label != backupPartLabel || (v.Size != 0 && v.Size != part.Size) {
			return "", errors.New("복원에 필요한 백업 조각이 없습니다")
		}
	}
	path := filepath.Join(dir, "download.age")
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return "", e
	}
	defer f.Close()
	hash := sha256.New()
	var total int64
	for i, part := range index.Parts {
		a.backupProgress(fmt.Sprintf("복원 조각 %d/%d 내려받기·검증", i+1, len(index.Parts)))
		n, h, e := a.fetchBackupTo(ctx, part.ID, bearer, io.MultiWriter(f, hash), part.Size)
		if e != nil {
			return "", e
		}
		if n != part.Size || h != part.SHA256 {
			return "", errors.New("복원 조각 누락·손상")
		}
		total += n
	}
	if total != index.Bytes || hex.EncodeToString(hash.Sum(nil)) != index.SHA256 {
		return "", errors.New("복원 전체 체크섬 불일치")
	}
	if e = f.Sync(); e != nil {
		return "", e
	}
	return path, f.Close()
}
func (a *App) restoreRemoteBackup(ctx context.Context, key RecoveryKey, bearer string, generation int) error {
	a.mu.Lock()
	empty := backupEmpty(a.state)
	if a.integration.BackupBusy {
		a.mu.Unlock()
		return errors.New("백업/복원이 이미 진행 중입니다")
	}
	if !empty {
		a.mu.Unlock()
		return errors.New("자료가 없는 새 복구 서버에서만 복원할 수 있습니다")
	}
	a.integration.BackupBusy = true
	a.mu.Unlock()
	defer func() { a.mu.Lock(); a.integration.BackupBusy = false; a.mu.Unlock() }()
	resume, err := a.holdInternetForRestore(ctx)
	defer resume()
	if err != nil {
		return err
	}
	rel, e := a.backupRelease(ctx, key, bearer, false)
	if e != nil {
		return e
	}
	rel, e = a.listBackupAssets(ctx, rel, bearer)
	if e != nil {
		return e
	}
	sets := completedBackups(rel)
	if generation < 0 || generation >= len(sets) {
		return errors.New("요청한 복원 세대가 없습니다")
	}
	dir, e := os.MkdirTemp(a.dir, ".backup-restore-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(dir)
	asset := sets[generation]
	if asset.Label != backupSetLabel {
		var b bytes.Buffer
		_, _, e = a.fetchBackupTo(ctx, asset.ID, bearer, &b, maxBackupBytes)
		if e != nil {
			return e
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.restorePortable(key, b.Bytes())
	}
	path, e := a.downloadBackupSet(ctx, key, bearer, rel, asset, dir)
	if e != nil {
		return e
	}
	a.backupProgress("전체 조각 복호화·사진/첨부 검증 중")
	v, e := verifyPortableFile(ctx, path, dir, key)
	if e != nil {
		return e
	}
	_ = os.Remove(filepath.Join(dir, "decoded.zip"))
	_ = os.Remove(path)
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.applyVerifiedPortable(v)
}
func (a *App) pruneBackupSets(ctx context.Context, rel Release, key RecoveryKey, bearer, keep string) error {
	if rel.Body != "wanryotalk-reconstructed-v2" {
		return errors.New("원본 백업은 정리하지 않습니다")
	}
	sets := completedBackups(rel)
	retained := map[string]bool{keep: true}
	for _, set := range sets {
		if set.Label == "alpha-verified-v1" {
			continue
		}
		if len(retained) >= 3 {
			break
		}
		retained[set.Name] = true
	}
	available := map[int64]ReleaseAsset{}
	for _, v := range rel.Assets {
		available[v.ID] = v
	}
	for _, set := range sets {
		if retained[set.Name] || set.Label == "alpha-verified-v1" {
			continue
		}
		if set.Label == "recovered-v2" && !strings.HasPrefix(set.Name, "alpha-") {
			continue // Preserve manually named files outside our old naming convention.
		}
		var parts []backupSetPart
		if set.Label == backupSetLabel {
			index, e := a.fetchBackupIndex(ctx, set, bearer, key)
			if e != nil {
				return e
			}
			parts = index.Parts
			for _, p := range parts {
				v, ok := available[p.ID]
				if !ok || v.Name != p.Name || v.Label != backupPartLabel {
					return errors.New("오래된 백업 조각을 확인하지 못해 정리 보류")
				}
			}
		}
		// Remove its completion index first, so interrupted cleanup never advertises an incomplete set.
		ids := []int64{set.ID}
		for _, p := range parts {
			ids = append(ids, p.ID)
		}
		for _, id := range ids {
			_, status, e := a.github(ctx, "DELETE", fmt.Sprintf("/repos/%s/releases/assets/%d", repository, id), bearer, nil, "")
			if e != nil {
				return e
			}
			if status != 204 && status != 404 {
				return fmt.Errorf("이전 백업 정리 HTTP %d", status)
			}
		}
	}
	return nil
}
