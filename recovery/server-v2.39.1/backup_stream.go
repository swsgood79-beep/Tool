package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"filippo.io/age"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const backupPartBytes int64 = 200 << 20
const backupSetMaxBytes int64 = 1 << 40 // Resource guard; not the old 256 MiB dataset ceiling.
const backupManifestMaxBytes int64 = 128 << 20
const backupMaxEntries = 1000000

func backupSizeLabel(n int64) string {
	if n >= 1<<30 {
		return fmt.Sprintf("%.2f GB", float64(n)/(1<<30))
	}
	return exportSize(n)
}

func backupAssets(s State) []Photo {
	refs := map[string]Photo{}
	add := func(photos []Photo) {
		for _, p := range photos {
			if old, ok := refs[p.ID]; !ok || p.Legacy || !old.Legacy {
				refs[p.ID] = p
			}
		}
	}
	add(manualParts(s))
	for _, p := range s.Projects {
		add(p.Photos)
	}
	for _, c := range s.ClientNotes {
		add(c.Photos)
	}
	for _, n := range s.Announcements {
		add(n.Photos)
	}
	for _, p := range s.Community {
		add(p.Photos)
	}
	for _, r := range s.ShippingRecords {
		add(shippingStoredAssets(r))
	}
	ids := make([]string, 0, len(refs))
	for id := range refs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]Photo, 0, len(ids))
	for _, id := range ids {
		out = append(out, refs[id])
	}
	return out
}
func (a *App) backupProgress(message string) {
	a.mu.Lock()
	a.integration.BackupStatus = message
	a.mu.Unlock()
}
func buildPortableFile(ctx context.Context, source, dir string, s State) (path string, err error) {
	identity, e := age.ParseX25519Identity(s.Integration.Backup.Key.Identity)
	if e != nil {
		return "", e
	}
	path = filepath.Join(dir, "payload.age")
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return "", e
	}
	defer f.Close()
	encrypted, e := age.Encrypt(f, identity.Recipient())
	if e != nil {
		return "", e
	}
	z := zip.NewWriter(encrypted)
	m := PortableManifest{Version: 1, Format: "wanryotalk-reconstructed-v3", BoxID: s.Integration.Backup.Key.BoxID, Created: time.Now().UTC().Format(time.RFC3339), State: s, Checksums: map[string]string{}}
	var total int64
	write := func(name string, data []byte) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if int64(len(data)) > backupSetMaxBytes-total {
			return errors.New("백업 안전 용량 상한 1TiB 초과")
		}
		total += int64(len(data))
		w, e := z.Create(name)
		if e != nil {
			return e
		}
		if _, e = w.Write(data); e != nil {
			return e
		}
		m.Checksums[name] = digest(data)
		return nil
	}
	refs := backupAssets(s)
	if len(refs) > backupMaxEntries {
		return "", errors.New("백업 파일 수 안전 한도 초과")
	}
	for _, p := range refs {
		b, e := shippingExportPhoto(source, p)
		if e != nil {
			return "", fmt.Errorf("백업 원본 파일 %s 읽기 실패: %w", p.ID, e)
		}
		if e = write("photos/"+p.ID, b); e != nil {
			return "", e
		}
	}
	book, e := workbook(s)
	if e != nil {
		return "", e
	}
	if e = write("records.xlsx", book); e != nil {
		return "", e
	}
	raw, e := json.Marshal(m)
	if e != nil {
		return "", e
	}
	if int64(len(raw)) > backupManifestMaxBytes {
		return "", errors.New("백업 수주 정보 안전 용량 초과")
	}
	w, e := z.Create("manifest.json")
	if e != nil {
		return "", e
	}
	if _, e = w.Write(raw); e != nil {
		return "", e
	}
	if e = z.Close(); e != nil {
		return "", e
	}
	if e = encrypted.Close(); e != nil {
		return "", e
	}
	if e = f.Sync(); e != nil {
		return "", e
	}
	if e = f.Close(); e != nil {
		return "", e
	}
	st, e := os.Stat(path)
	if e != nil || st.Size() > backupSetMaxBytes {
		return "", errors.New("암호화 백업 안전 용량 상한 초과")
	}
	return path, nil
}

type verifiedPortable struct {
	Manifest PortableManifest
	Entries  map[string]string
}

// Decrypt to private temporary disk, then validate every entry before any live write.
func verifyPortableFile(ctx context.Context, path, dir string, key RecoveryKey) (verifiedPortable, error) {
	var out verifiedPortable
	identity, e := age.ParseX25519Identity(key.Identity)
	if e != nil {
		return out, e
	}
	f, e := os.Open(path)
	if e != nil {
		return out, e
	}
	defer f.Close()
	reader, e := age.Decrypt(f, identity)
	if e != nil {
		return out, e
	}
	rawPath := filepath.Join(dir, "decoded.zip")
	raw, e := os.OpenFile(rawPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if e != nil {
		return out, e
	}
	defer raw.Close()
	n, e := copyBackup(ctx, raw, reader, backupSetMaxBytes+1)
	if e != nil {
		return out, e
	}
	if n > backupSetMaxBytes {
		return out, errors.New("복원 안전 용량 상한 초과")
	}
	z, e := zip.NewReader(raw, n)
	if e != nil {
		return out, e
	}
	if len(z.File) > backupMaxEntries+2 {
		return out, errors.New("백업 파일 수 한도 초과")
	}
	out.Entries = map[string]string{}
	sizes := map[string]int64{}
	hashes := map[string]string{}
	var total uint64
	for i, item := range z.File {
		if e = ctx.Err(); e != nil {
			return out, e
		}
		name := item.Name
		if name != "manifest.json" && name != "records.xlsx" && !(strings.HasPrefix(name, "photos/") && validPhotoID.MatchString(strings.TrimPrefix(name, "photos/"))) {
			return out, errors.New("백업 경로 오류")
		}
		if _, ok := out.Entries[name]; ok {
			return out, errors.New("백업 중복 파일")
		}
		limit := int64(9 << 20)
		if name == "manifest.json" {
			limit = backupManifestMaxBytes
		} else if name == "records.xlsx" {
			limit = maxBackupBytes
		}
		if item.UncompressedSize64 > uint64(limit) || item.UncompressedSize64 > uint64(backupSetMaxBytes)-total {
			return out, errors.New("백업 항목 안전 용량 초과")
		}
		total += item.UncompressedSize64
		in, e := item.Open()
		if e != nil {
			return out, e
		}
		dest := filepath.Join(dir, fmt.Sprintf("entry-%07d", i))
		file, e := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			in.Close()
			return out, e
		}
		h := sha256.New()
		count, e := copyBackup(ctx, io.MultiWriter(file, h), in, limit+1)
		in.Close()
		ce := file.Close()
		if e == nil {
			e = ce
		}
		if e != nil || count != int64(item.UncompressedSize64) {
			return out, errors.New("백업 항목 손상")
		}
		out.Entries[name] = dest
		sizes[name] = count
		hashes[name] = hex.EncodeToString(h.Sum(nil))
	}
	if out.Entries["manifest.json"] == "" || out.Entries["records.xlsx"] == "" {
		return out, errors.New("백업 필수 정보 누락")
	}
	metadata, e := os.ReadFile(out.Entries["manifest.json"])
	if e != nil {
		return out, e
	}
	if e = json.Unmarshal(metadata, &out.Manifest); e != nil {
		return out, e
	}
	m := out.Manifest
	if m.Version != 1 || m.Format != "wanryotalk-reconstructed-v3" || m.BoxID != key.BoxID || m.State.Version != 1 || !validPhotoID.MatchString(m.State.Board) {
		return out, errors.New("복구키/백업 형식 불일치")
	}
	if len(m.Checksums) != len(out.Entries)-1 {
		return out, errors.New("백업 체크섬 개수 불일치")
	}
	for name, path := range out.Entries {
		if name == "manifest.json" {
			continue
		}
		if path == "" || m.Checksums[name] != hashes[name] {
			return out, errors.New("백업 체크섬 불일치")
		}
	}
	for _, p := range backupAssets(m.State) {
		if !validPhotoID.MatchString(p.ID) || out.Entries["photos/"+p.ID] == "" {
			return out, errors.New("참조된 백업 사진/첨부 조각 누락")
		}
	}
	records := append([]ShippingRecord{}, m.State.ShippingRecords...)
	for _, post := range m.State.Manuals {
		records = append(records, ShippingRecord{Attachments: post.Files})
	}
	for _, r := range records {
		for _, a := range r.Attachments {
			if a.Size < 1 || a.Size > 1<<30 || len(a.Parts) != len(a.Hashes) || len(a.Parts) != int((a.Size+shippingAttachmentChunk-1)/shippingAttachmentChunk) {
				return out, errors.New("첨부 백업 정보 오류")
			}
			for i, p := range a.Parts {
				name := "photos/" + p.ID
				if sizes[name] != min(int64(shippingAttachmentChunk), a.Size-int64(i)*shippingAttachmentChunk) || hashes[name] != a.Hashes[i] {
					return out, errors.New("첨부 백업 조각 누락·손상")
				}
			}
		}
	}
	return out, nil
}
func copyBackup(ctx context.Context, w io.Writer, r io.Reader, limit int64) (int64, error) {
	limited := io.LimitReader(r, limit)
	buf := make([]byte, 128<<10)
	var total int64
	for {
		if e := ctx.Err(); e != nil {
			return total, e
		}
		n, e := limited.Read(buf)
		if n > 0 {
			written, we := w.Write(buf[:n])
			total += int64(written)
			if we != nil {
				return total, we
			}
			if written != n {
				return total, io.ErrShortWrite
			}
		}
		if e == io.EOF {
			return total, nil
		}
		if e != nil {
			return total, e
		}
	}
}
func backupEmpty(s State) bool {
	return len(s.TeamNotices) == 0 && len(s.MachiningJobs) == 0 && len(s.MachiningImports) == 0 && len(s.MachiningAdjustments) == 0 && len(s.Manuals) == 0 && len(s.PurchasingCalls) == 0 && len(s.AdminAccounts) == 0 && len(s.ShippingRecords) == 0 && len(s.Community) == 0 && len(s.Announcements) == 0 && len(s.Projects) == 0 && len(s.Events) == 0 && len(s.Designers) == 0 && len(s.ClientNotes) == 0 && len(s.NativeAdminLogins) == 0 && (s.NativeAdminPush == nil || len(s.NativeAdminPush.Devices) == 0) && s.Integration.Phone.Subscription == nil
}
func (a *App) applyVerifiedPortable(v verifiedPortable) error {
	if !backupEmpty(a.state) {
		return errors.New("자료가 없는 새 복구 서버에서만 복원할 수 있습니다")
	}
	// All remote parts and plaintext checksums have already been verified.
	if e := os.MkdirAll(filepath.Join(a.dir, "photos"), 0700); e != nil {
		return e
	}
	for name, path := range v.Entries {
		if !strings.HasPrefix(name, "photos/") {
			continue
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		b, e = protect(b)
		if e != nil {
			return e
		}
		if e = atomicFile(filepath.Join(a.dir, "photos", strings.TrimPrefix(name, "photos/")), b); e != nil {
			return e
		}
		_ = os.Remove(path)
	}
	return a.applyRestoredState(v.Manifest.State)
}
func (a *App) applyRestoredState(next State) error {
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
	if e := a.persist(&next); e != nil {
		return e
	}
	a.state = next
	a.sessions = map[string]Session{}
	a.initIntegration()
	a.restoreTTSConfig()
	return nil
}
