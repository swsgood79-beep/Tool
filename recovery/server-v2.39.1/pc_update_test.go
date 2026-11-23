package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type pcTestTransport func(*http.Request) (*http.Response, error)

func (f pcTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func pcTestPE() []byte {
	b := make([]byte, 512)
	copy(b, "MZ")
	binary.LittleEndian.PutUint32(b[60:], 128)
	copy(b[128:], "PE\x00\x00")
	binary.LittleEndian.PutUint16(b[132:], 0x8664)
	binary.LittleEndian.PutUint16(b[148:], 240)
	binary.LittleEndian.PutUint16(b[152:], 0x20b)
	binary.LittleEndian.PutUint16(b[220:], 2)
	binary.LittleEndian.PutUint32(b[260:], 16)
	return b
}
func pcTestHash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func pcTestZIP(t *testing.T, names []string, payload []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, n := range names {
		f, e := z.Create(n)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = f.Write(payload); e != nil {
			t.Fatal(e)
		}
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func pcTestUpdater(t *testing.T, zipData, exe []byte) *pcUpdater {
	t.Helper()
	u := newPCUpdater()
	v, _ := pcVersion(releaseVersion)
	next := fmt.Sprintf("%d.%d.%d", v[0], v[1], v[2]+1)
	m := pcUpdateManifest{Version: next, URL: pcUpdateBase + "AlphaWorkComplete-Recovered-Windows-x64-v" + next + ".zip", SHA256: pcTestHash(zipData), EXESHA256: pcTestHash(exe), Bytes: int64(len(zipData))}
	// ZIP compression may be tiny for a header-only synthetic PE. Pad the ZIP
	// archive comment, which remains covered by the archive digest.
	if len(zipData) < 1024 {
		zipData = append(zipData, make([]byte, 1024-len(zipData))...)
		m.SHA256 = pcTestHash(zipData)
		m.Bytes = int64(len(zipData))
	}
	u.manifest = m
	u.view.Available = true
	u.client = &http.Client{Transport: pcTestTransport(func(r *http.Request) (*http.Response, error) {
		var data []byte
		if strings.Contains(r.URL.Path, "pc-update.json") {
			data, _ = json.Marshal(u.manifest)
		} else {
			data = zipData
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data)), ContentLength: int64(len(data)), Header: make(http.Header)}, nil
	})}
	return u
}
func TestPCUpdateManifestAndVersion(t *testing.T) {
	for _, tc := range []struct {
		a, b  string
		newer bool
	}{{"1.4.87", "1.4.86", true}, {"1.5.0", "1.4.99", true}, {"1.4.86", "1.4.86", false}, {"1.4.85", "1.4.86", false}, {"1.04.87", "1.4.86", false}, {"1.4.87-x", "1.4.86", false}} {
		if pcNewer(tc.a, tc.b) != tc.newer {
			t.Fatal(tc)
		}
	}
	u := pcTestUpdater(t, pcTestZIP(t, []string{"AlphaWorkComplete-Recovered.exe"}, pcTestPE()), pcTestPE())
	u.check(context.Background())
	if !u.snapshot().Available || u.snapshot().Busy {
		t.Fatal(u.snapshot())
	}
	for _, url := range []string{"http://raw.githubusercontent.com/file.zip", pcUpdateBase + "../private.zip", "https://example.org/a.zip", pcUpdateBase + "AlphaWorkComplete-Recovered-Windows-x64-v1.4.85.zip"} {
		m := u.manifest
		m.URL = url
		if m.validate() == nil {
			t.Fatal("untrusted URL", url)
		}
	}
	m := u.manifest
	m.Bytes = pcUpdateMaxBytes + 1
	if m.validate() == nil {
		t.Fatal("oversize accepted")
	}
}
func TestPCUpdateDownloadIntegrityAndIsolation(t *testing.T) {
	pe := pcTestPE()
	for _, tc := range []struct {
		name                   string
		names                  []string
		badZIPHash, badEXEHash bool
		payload                []byte
		pass                   bool
	}{
		{name: "verified", names: []string{"AlphaWorkComplete-Recovered.exe", "README.md"}, payload: pe, pass: true},
		{name: "ZIP checksum", names: []string{"AlphaWorkComplete-Recovered.exe"}, payload: pe, badZIPHash: true},
		{name: "EXE checksum", names: []string{"AlphaWorkComplete-Recovered.exe"}, payload: pe, badEXEHash: true},
		{name: "traversal", names: []string{"../AlphaWorkComplete-Recovered.exe"}, payload: pe},
		{name: "duplicate", names: []string{"AlphaWorkComplete-Recovered.exe", "AlphaWorkComplete-Recovered.exe"}, payload: pe},
		{name: "invalid PE", names: []string{"AlphaWorkComplete-Recovered.exe"}, payload: []byte("not a Windows executable")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "Alpha.exe")
			os.WriteFile(target, []byte("old program"), 0700)
			data := filepath.Join(dir, "projects-v2.dat")
			os.WriteFile(data, []byte("business data"), 0600)
			u := pcTestUpdater(t, pcTestZIP(t, tc.names, tc.payload), tc.payload)
			if tc.badZIPHash {
				u.manifest.SHA256 = strings.Repeat("0", 64)
			}
			if tc.badEXEHash {
				u.manifest.EXESHA256 = strings.Repeat("0", 64)
			}
			stage, e := u.download(context.Background(), target)
			if (e == nil) != tc.pass {
				t.Fatal(stage, e)
			}
			if string(mustRead(t, target)) != "old program" || string(mustRead(t, data)) != "business data" {
				t.Fatal("live file changed during download")
			}
			if tc.pass {
				if !bytes.Equal(mustRead(t, filepath.Join(stage, "new.exe")), pe) {
					t.Fatal("missing new executable")
				}
			} else {
				entries, _ := os.ReadDir(dir)
				for _, entry := range entries {
					if strings.HasPrefix(entry.Name(), ".alpha-pc-update-") {
						t.Fatal("failed download left stage", entry.Name())
					}
				}
				if u.snapshot().Busy {
					t.Fatal("failure blocked retry")
				}
			}
		})
	}
}
func TestPCUpdateReplacementRollbackAndStartup(t *testing.T) {
	for _, mode := range []string{"success", "new exits", "missing new", "still starting"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			stage, e := os.MkdirTemp(dir, ".alpha-pc-update-")
			if e != nil {
				t.Fatal(e)
			}
			target := filepath.Join(dir, "Alpha.exe")
			os.WriteFile(target, []byte("old"), 0700)
			if mode != "missing new" {
				os.WriteFile(filepath.Join(stage, "new.exe"), []byte("new"), 0700)
			}
			p := pcUpdatePlan{Target: target, Stage: stage, Args: pcRestartArgs(filepath.Join(dir, "data"), "127.0.0.1:18791")}
			var launches []string
			launch := func(path string, args []string, ready string) error {
				v := string(mustRead(t, path))
				launches = append(launches, v)
				if ready == "" {
					return nil
				}
				if mode == "new exits" {
					return errors.New("launch failed")
				}
				if mode == "still starting" {
					return errPCUpdateStillStarting
				}
				return pcUpdateReadyFile(ready)
			}
			e = pcReplaceAndRestart(p, launch)
			switch mode {
			case "success":
				if e != nil || string(mustRead(t, target)) != "new" || string(mustRead(t, filepath.Join(stage, "previous.exe"))) != "old" {
					t.Fatal(e, launches)
				}
			case "still starting":
				if !errors.Is(e, errPCUpdateStillStarting) || len(launches) != 1 || string(mustRead(t, target)) != "new" {
					t.Fatal("must not kill/restart a live startup", e, launches)
				}
			default:
				if e == nil || string(mustRead(t, target)) != "old" || launches[len(launches)-1] != "old" {
					t.Fatal("rollback failed", e, launches)
				}
			}
		})
	}
}
func TestPCUpdatePlanAndRestartSettings(t *testing.T) {
	dir := t.TempDir()
	stage, _ := os.MkdirTemp(dir, ".alpha-pc-update-")
	target := filepath.Join(dir, "Alpha.exe")
	os.WriteFile(target, []byte("old"), 0700)
	os.WriteFile(filepath.Join(stage, "new.exe"), []byte("new"), 0700)
	p := pcUpdatePlan{Target: target, Stage: stage, OldSHA256: pcTestHash([]byte("old")), NewSHA256: pcTestHash([]byte("new")), ParentPID: 12, Args: pcRestartArgs(filepath.Join(dir, "D-drive-data"), "127.0.0.1:18791")}
	if e := p.validate(filepath.Join(stage, "plan.json")); e != nil {
		t.Fatal(e)
	}
	p.Args = []string{"-empty"}
	if p.validate(filepath.Join(stage, "plan.json")) == nil {
		t.Fatal("replayed destructive startup flag")
	}
	if pcUpdateReadyFile(filepath.Join(dir, "projects-v2.dat")) == nil {
		t.Fatal("accepted arbitrary ready path")
	}
}

func TestPCUpdateCompletedBackupRetention(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "Alpha.exe")
	for i := 0; i < 4; i++ {
		stage, e := os.MkdirTemp(dir, ".alpha-pc-update-")
		if e != nil {
			t.Fatal(e)
		}
		pcWriteJSON(filepath.Join(stage, "plan.json"), pcUpdatePlan{Target: target, Stage: stage})
		os.WriteFile(filepath.Join(stage, "previous.exe"), []byte("previous"), 0600)
		os.WriteFile(filepath.Join(stage, "helper.exe"), []byte("helper"), 0600)
		os.WriteFile(filepath.Join(stage, "complete"), []byte("complete"), 0600)
		at := time.Now().Add(time.Duration(i) * time.Hour)
		os.Chtimes(filepath.Join(stage, "complete"), at, at)
		if i == 0 {
			os.WriteFile(filepath.Join(stage, "user-file.txt"), []byte("keep me"), 0600)
		}
	}
	pending, _ := os.MkdirTemp(dir, ".alpha-pc-update-")
	os.WriteFile(filepath.Join(pending, "new.exe"), []byte("pending"), 0600)
	pcCleanCompletedUpdates(target)
	entries, _ := os.ReadDir(dir)
	backups, unknown := 0, 0
	for _, entry := range entries {
		stage := filepath.Join(dir, entry.Name())
		if _, e := os.Stat(filepath.Join(stage, "previous.exe")); e == nil {
			backups++
		}
		if _, e := os.Stat(filepath.Join(stage, "user-file.txt")); e == nil {
			unknown++
		}
	}
	if backups != 2 || unknown != 1 || string(mustRead(t, filepath.Join(pending, "new.exe"))) != "pending" {
		t.Fatal("retention removed pending or unknown files", backups, unknown)
	}
}

func TestPCUpdateCancelledDownloadAllowsRetry(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "Alpha.exe")
	os.WriteFile(target, []byte("old"), 0700)
	u := pcTestUpdater(t, pcTestZIP(t, []string{"AlphaWorkComplete-Recovered.exe"}, pcTestPE()), pcTestPE())
	entered := make(chan struct{})
	u.client = &http.Client{Transport: pcTestTransport(func(r *http.Request) (*http.Response, error) {
		close(entered)
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, e := u.download(ctx, target); done <- e }()
	<-entered
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = u.snapshot()
			}
		}()
	}
	cancel()
	wg.Wait()
	if e := <-done; e == nil || u.snapshot().Busy || string(mustRead(t, target)) != "old" {
		t.Fatal("cancelled update changed running program", e)
	}
}
