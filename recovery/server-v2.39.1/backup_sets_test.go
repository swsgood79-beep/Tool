package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"filippo.io/age"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type backupTestRemote struct {
	mu                  sync.Mutex
	dir                 string
	assets              map[int64]ReleaseAsset
	paths               map[int64]string
	next                int64
	uploads, failUpload int
	corrupt             int64
	deleted             []int64
	patches             int
	failPatch           bool
}

func newBackupTestRemote(t *testing.T) *backupTestRemote {
	t.Helper()
	return &backupTestRemote{dir: t.TempDir(), assets: map[int64]ReleaseAsset{}, paths: map[int64]string{}, next: 100}
}
func (m *backupTestRemote) roundTrip(r *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r.URL.Host == "uploads.github.com" {
		m.uploads++
		if m.failUpload > 0 && m.uploads == m.failUpload {
			return response(r, 500, map[string]any{}), nil
		}
		m.next++
		id := m.next
		path := filepath.Join(m.dir, fmtID(id))
		f, e := os.Create(path)
		if e != nil {
			return nil, e
		}
		n, e := io.Copy(f, r.Body)
		f.Close()
		if e != nil {
			return nil, e
		}
		asset := ReleaseAsset{ID: id, Name: r.URL.Query().Get("name"), Label: r.URL.Query().Get("label"), Created: time.Now().UTC().Format(time.RFC3339Nano), Size: n}
		m.assets[id] = asset
		m.paths[id] = path
		return response(r, 201, asset), nil
	}
	if strings.Contains(r.URL.Path, "/releases/assets/") {
		id, _ := strconv.ParseInt(filepath.Base(r.URL.Path), 10, 64)
		if r.Method == "DELETE" {
			m.deleted = append(m.deleted, id)
			delete(m.assets, id)
			return response(r, 204, map[string]any{}), nil
		}
		asset, ok := m.assets[id]
		if !ok {
			return response(r, 404, map[string]any{}), nil
		}
		if r.Method == "PATCH" {
			if m.failPatch {
				return response(r, 500, map[string]any{}), nil
			}
			var fields map[string]string
			json.NewDecoder(r.Body).Decode(&fields)
			asset.Label = fields["label"]
			m.assets[id] = asset
			m.patches++
			return response(r, 200, asset), nil
		}
		if id == m.corrupt {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("CORRUPTED")), Header: http.Header{}, Request: r}, nil
		}
		f, e := os.Open(m.paths[id])
		if e != nil {
			return nil, e
		}
		return &http.Response{StatusCode: 200, Body: f, Header: http.Header{}, Request: r}, nil
	}
	assets := make([]ReleaseAsset, 0, len(m.assets))
	for _, a := range m.assets {
		assets = append(assets, a)
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].ID < assets[j].ID })
	if strings.Contains(r.URL.Path, "/releases/tags/") {
		return response(r, 200, Release{ID: 22, Body: "wanryotalk-reconstructed-v2", Assets: assets}), nil
	}
	if strings.HasSuffix(r.URL.Path, "/releases/22/assets") {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		start := (page - 1) * 100
		if start >= len(assets) {
			assets = nil
		} else {
			assets = assets[start:min(start+100, len(assets))]
		}
		return response(r, 200, assets), nil
	}
	return nil, fmt.Errorf("unexpected backup operation: %s %s", r.Method, r.URL)
}
func setupSegmentedBackup(t *testing.T) (*App, *backupTestRemote) {
	t.Helper()
	a := exportFixture(t)
	identity, e := age.GenerateX25519Identity()
	if e != nil {
		t.Fatal(e)
	}
	a.state.Integration.Backup.Key = RecoveryKey{1, repository, token(), identity.String()}
	a.state.Integration.Connection.GitHubToken = "TEST-ONLY"
	m := newBackupTestRemote(t)
	a.client = &http.Client{Transport: transportFunc(m.roundTrip)}
	return a, m
}
func emptyBackupTarget(t *testing.T, source *App) *App {
	t.Helper()
	a, e := newApp(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	a.client = source.client
	return a
}
func TestSegmentedBackupRoundTripAndCorruption(t *testing.T) {
	a, remote := setupSegmentedBackup(t)
	key := a.state.Integration.Backup.Key
	a.runSegmentedBackup(context.Background(), 512)
	if a.state.Integration.Backup.LastSuccess == "" || a.state.Integration.Backup.LastParts < 2 || remote.patches != 1 {
		t.Fatal(a.integration.BackupStatus, remote.uploads)
	}
	var completed ReleaseAsset
	var part int64
	for _, asset := range remote.assets {
		if asset.Label == backupSetLabel {
			completed = asset
		}
		if asset.Label == backupPartLabel {
			part = asset.ID
		}
	}
	if completed.ID == 0 {
		t.Fatal("missing committed set")
	}
	index, e := a.fetchBackupIndex(context.Background(), completed, "TEST-ONLY", key)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile(remote.paths[completed.ID])
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Contains(raw, []byte("TEST-ONLY")) || bytes.Contains(raw, []byte(a.state.Board)) {
		t.Fatal("public metadata leaked state")
	}
	fresh := emptyBackupTarget(t, a)
	if e = fresh.restoreRemoteBackup(context.Background(), key, "TEST-ONLY", 0); e != nil {
		t.Fatal(e)
	}
	if len(fresh.state.ShippingRecords) != 1 || fresh.state.Board != a.state.Board || len(fresh.state.ShippingRecords[0].Attachments) != 1 {
		t.Fatal("restored state missing")
	}
	for _, photo := range backupAssets(a.state) {
		source, sourceErr := unprotect(mustRead(t, filepath.Join(a.dir, "photos", photo.ID)))
		restored, restoreErr := unprotect(mustRead(t, filepath.Join(fresh.dir, "photos", photo.ID)))
		if sourceErr != nil || restoreErr != nil || !bytes.Equal(source, restored) {
			t.Fatal("restored bytes differ")
		}
	}
	blocked := emptyBackupTarget(t, a)
	remote.corrupt = part
	if e = blocked.restoreRemoteBackup(context.Background(), key, "TEST-ONLY", 0); e == nil || !backupEmpty(blocked.state) {
		t.Fatal("corruption changed target", e)
	}
	remote.corrupt = 0
	if entries, _ := os.ReadDir(filepath.Join(blocked.dir, "photos")); len(entries) > 0 {
		t.Fatal("corrupt restore wrote live photos")
	}
	delete(remote.assets, index.Parts[0].ID)
	if e = blocked.restoreRemoteBackup(context.Background(), key, "TEST-ONLY", 0); e == nil {
		t.Fatal("missing part accepted")
	}
}
func TestSegmentedBackupFailuresPreservePrevious(t *testing.T) {
	for _, mode := range []string{"upload", "completion", "verify"} {
		t.Run(mode, func(t *testing.T) {
			a, remote := setupSegmentedBackup(t)
			a.runSegmentedBackup(context.Background(), 1024)
			old := a.state.Integration.Backup
			oldIDs := map[int64]bool{}
			for id := range remote.assets {
				oldIDs[id] = true
			}
			if mode == "upload" {
				remote.failUpload = remote.uploads + 2
			}
			if mode == "completion" {
				remote.failPatch = true
			}
			if mode == "verify" {
				remote.corrupt = remote.next + 1
			}
			a.runSegmentedBackup(context.Background(), 1024)
			if a.state.Integration.Backup.LastSuccess != old.LastSuccess || a.state.Integration.Backup.LastAsset != old.LastAsset || a.state.Integration.Backup.LastError == "" {
				t.Fatal("failed backup replaced success", a.integration.BackupStatus)
			}
			for id := range oldIDs {
				if _, ok := remote.assets[id]; !ok {
					t.Fatal("old asset deleted", id)
				}
			}
			for _, id := range remote.deleted {
				if oldIDs[id] {
					t.Fatal("deleted previous backup")
				}
			}
			if a.integration.BackupBusy {
				t.Fatal("busy flag stuck")
			}
		})
	}
}
func TestSegmentedBackupRetentionAndLegacyRestore(t *testing.T) {
	a, remote := setupSegmentedBackup(t)
	key := a.state.Integration.Backup.Key
	legacy, e := a.portable(a.state)
	if e != nil {
		t.Fatal(e)
	}
	remote.next++
	oldID := remote.next
	path := filepath.Join(remote.dir, "legacy.age")
	os.WriteFile(path, legacy, 0600)
	remote.paths[oldID] = path
	remote.assets[oldID] = ReleaseAsset{ID: oldID, Name: "original.age", Label: "alpha-verified-v1", Created: "2020-01-01T00:00:00Z"}
	fresh := emptyBackupTarget(t, a)
	if e = fresh.restoreRemoteBackup(context.Background(), key, "TEST-ONLY", 0); e != nil {
		t.Fatal("legacy restore", e)
	}
	for i := 0; i < 5; i++ {
		a.runSegmentedBackup(context.Background(), 2048)
		if a.state.Integration.Backup.LastError != "" {
			t.Fatal(a.integration.BackupStatus)
		}
	}
	count := 0
	for _, asset := range remote.assets {
		if asset.Label == backupSetLabel {
			count++
		}
	}
	if count != 3 {
		t.Fatal("retention count", count)
	}
	if _, ok := remote.assets[oldID]; !ok {
		t.Fatal("original backup removed")
	}
	for _, asset := range remote.assets {
		if asset.Label == backupSetLabel {
			idx, e := a.fetchBackupIndex(context.Background(), asset, "TEST-ONLY", key)
			if e != nil {
				t.Fatal(e)
			}
			for _, p := range idx.Parts {
				if _, ok := remote.assets[p.ID]; !ok {
					t.Fatal("retained set incomplete")
				}
			}
		}
	}
}
func TestSegmentedBackupIndexValidationAndKeyRedownload(t *testing.T) {
	a, _ := setupSegmentedBackup(t)
	key := a.state.Integration.Backup.Key
	cookie := adminCookie(a)
	for i := 0; i < 2; i++ {
		w := call(a, "POST", "/backup/key", url.Values{"csrf": {"admin-csrf"}}, cookie)
		if w.Code != 200 {
			t.Fatal("key redownload status", w.Code)
		}
		var downloaded RecoveryKey
		if json.Unmarshal(w.Body.Bytes(), &downloaded) != nil || downloaded != key {
			t.Fatal("redownload returned different key")
		}
	}
	// Check the key route through the existing request helper below; no key rotation.
	if a.state.Integration.Backup.Key != key {
		t.Fatal("redownload rotated key")
	}
}
func TestSegmentedBackupOver256MiB(t *testing.T) {
	if os.Getenv("LARGE_BACKUP_TEST") != "1" {
		t.Skip("optional actual >256 MiB stream test")
	}
	a, remote := setupSegmentedBackup(t)
	r := &a.state.ShippingRecords[0]
	payload := make([]byte, shippingAttachmentChunk)
	if _, e := rand.Read(payload); e != nil {
		t.Fatal(e)
	}
	hash := digest(payload)
	file := ShippingAttachment{ID: token(), Name: "large-test-video.mp4", Kind: "video", MIME: "video/mp4", Size: 65 * shippingAttachmentChunk, CreatedAt: time.Now()}
	for i := 0; i < 65; i++ {
		id := token()
		b, e := protect(payload)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(a.dir, "photos", id), b, 0600); e != nil {
			t.Fatal(e)
		}
		file.Parts = append(file.Parts, Photo{ID: id, Size: shippingAttachmentChunk})
		file.Hashes = append(file.Hashes, hash)
	}
	r.Attachments = append(r.Attachments, file)
	a.runBackup(context.Background())
	if a.state.Integration.Backup.LastSuccess == "" || a.state.Integration.Backup.LastSize <= 256<<20 || a.state.Integration.Backup.LastParts != 2 {
		t.Fatal(a.integration.BackupStatus, a.state.Integration.Backup.LastSize)
	}
	fresh := emptyBackupTarget(t, a)
	if e := fresh.restoreRemoteBackup(context.Background(), a.state.Integration.Backup.Key, "TEST-ONLY", 0); e != nil {
		t.Fatal(e)
	}
	restored := fresh.state.ShippingRecords[0].Attachments[1]
	if restored.Size != file.Size || len(restored.Parts) != 65 {
		t.Fatal("large attachment lost")
	}
	for _, part := range restored.Parts {
		b, e := unprotect(mustRead(t, filepath.Join(fresh.dir, "photos", part.ID)))
		if e != nil || digest(b) != hash {
			t.Fatal("large part mismatch", e)
		}
	}
	t.Logf("verified %.1f MiB encrypted, %d parts, %d uploaded assets; restored 260 MiB attachment", float64(a.state.Integration.Backup.LastSize)/(1<<20), a.state.Integration.Backup.LastParts, remote.uploads)
}

func TestSegmentedBackupIndexRejectsInvalidAndPending(t *testing.T) {
	a, remote := setupSegmentedBackup(t)
	a.runSegmentedBackup(context.Background(), 1024)
	key := a.state.Integration.Backup.Key
	for _, asset := range remote.assets {
		if asset.Label != backupSetLabel {
			continue
		}
		index, e := a.fetchBackupIndex(context.Background(), asset, "TEST-ONLY", key)
		if e != nil {
			t.Fatal(e)
		}
		for _, change := range []func(*backupSetIndex){func(i *backupSetIndex) { i.Parts[0].Name = "../outside" }, func(i *backupSetIndex) { i.Bytes++ }, func(i *backupSetIndex) { i.Parts[0].SHA256 = "bad" }, func(i *backupSetIndex) { i.BoxID = token() }, func(i *backupSetIndex) { i.Parts[0].ID = -1 }} {
			copy := index
			copy.Parts = append([]backupSetPart(nil), index.Parts...)
			change(&copy)
			if copy.validate(key) == nil {
				t.Fatal("invalid index accepted")
			}
		}
	}
	rel := Release{Assets: []ReleaseAsset{{ID: 1, Name: "incomplete.set.age", Label: backupPendingLabel}, {ID: 2, Name: "part.age", Label: backupPartLabel}}}
	if len(completedBackups(rel)) != 0 {
		t.Fatal("unverified backup is selectable")
	}
	if snapshotHash(a.state) != a.state.Integration.Backup.LastHash {
		t.Fatal("backup bookkeeping creates perpetual pending changes")
	}
}
func TestSegmentedBackupPaginationAndPinnedDeletion(t *testing.T) {
	a, remote := setupSegmentedBackup(t)
	for i := int64(1); i <= 130; i++ {
		remote.assets[i] = ReleaseAsset{ID: i, Name: fmt.Sprintf("other-%d", i)}
	}
	rel, e := a.listBackupAssets(context.Background(), Release{ID: 22, Body: "wanryotalk-reconstructed-v2"}, "TEST-ONLY")
	if e != nil || len(rel.Assets) != 130 {
		t.Fatal("pagination", len(rel.Assets), e)
	}
	photo := a.state.ShippingRecords[0].Photos[0]
	a.backupPins = map[string]int{photo.ID: 1}
	a.state.ShippingRecords[0].Photos = nil
	a.pruneCommunityPhotos([]Photo{photo})
	if _, e = os.Stat(filepath.Join(a.dir, "photos", photo.ID)); e != nil {
		t.Fatal("snapshot pin did not preserve file", e)
	}
	delete(a.backupPins, photo.ID)
	a.pruneCommunityPhotos([]Photo{photo})
	if _, e = os.Stat(filepath.Join(a.dir, "photos", photo.ID)); !os.IsNotExist(e) {
		t.Fatal("unreferenced pin release did not clean file", e)
	}
}

func TestSegmentedBackupSnapshotDoesNotBlockWork(t *testing.T) {
	a, remote := setupSegmentedBackup(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	a.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "uploads.github.com" {
			once.Do(func() { close(entered); <-release })
		}
		return remote.roundTrip(r)
	})}
	board := a.state.Board
	done := make(chan struct{})
	go func() { defer close(done); a.runSegmentedBackup(context.Background(), 1024) }()
	<-entered
	work := make(chan int, 1)
	go func() { w := call(a, "GET", "/work/"+board, nil); work <- w.Code }()
	select {
	case code := <-work:
		if code != 200 {
			t.Fatal("worker view", code)
		}
	case <-time.After(2 * time.Second):
		close(release)
		<-done
		t.Fatal("backup upload held business mutex")
	}
	close(release)
	<-done
	if a.integration.BackupBusy || len(a.backupPins) != 0 {
		t.Fatal("backup pins not released")
	}
}
