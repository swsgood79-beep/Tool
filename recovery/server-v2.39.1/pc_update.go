package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const pcUpdateBase = "https://raw.githubusercontent.com/swsgood79-beep/Tool/recovery-v2.39.1/recovery/server-builds/"
const pcUpdateManifestURL = pcUpdateBase + "pc-update.json"
const pcUpdateMaxBytes = 100 << 20

type pcUpdateManifest struct {
	Version   string `json:"version"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	EXESHA256 string `json:"exe_sha256"`
	Bytes     int64  `json:"bytes"`
	Notes     string `json:"notes"`
}
type pcUpdateView struct {
	Message, Version string
	Busy, Available  bool
	Percent          int
}
type pcUpdater struct {
	mu       sync.Mutex
	client   *http.Client
	view     pcUpdateView
	manifest pcUpdateManifest
}

func newPCUpdater() *pcUpdater {
	return &pcUpdater{client: &http.Client{Timeout: 15 * time.Minute, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) > 3 || !strings.HasPrefix(r.URL.String(), pcUpdateBase) {
			return errors.New("업데이트 주소가 변경되었습니다")
		}
		return nil
	}}, view: pcUpdateView{Message: "새 버전을 자동 확인합니다"}}
}
func pcVersion(s string) ([3]int, error) {
	var v [3]int
	p := strings.Split(s, ".")
	if len(p) != 3 {
		return v, errors.New("올바르지 않은 버전")
	}
	for i, x := range p {
		n, e := strconv.Atoi(x)
		if e != nil || n < 0 || n > 999999 || strconv.Itoa(n) != x {
			return v, errors.New("올바르지 않은 버전")
		}
		v[i] = n
	}
	return v, nil
}
func pcNewer(a, b string) bool {
	x, e := pcVersion(a)
	y, f := pcVersion(b)
	if e != nil || f != nil {
		return false
	}
	for i := range x {
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return false
}
func pcDigest(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == sha256.Size && s == strings.ToLower(s)
}
func (m pcUpdateManifest) validate() error {
	if _, e := pcVersion(m.Version); e != nil {
		return e
	}
	expected := pcUpdateBase + "AlphaWorkComplete-Recovered-Windows-x64-v" + m.Version + ".zip"
	if m.URL != expected || !pcDigest(m.SHA256) || !pcDigest(m.EXESHA256) || m.Bytes < 1024 || m.Bytes > pcUpdateMaxBytes || len(m.Notes) > 2000 {
		return errors.New("업데이트 배포 정보를 확인할 수 없습니다")
	}
	return nil
}
func (u *pcUpdater) snapshot() pcUpdateView { u.mu.Lock(); defer u.mu.Unlock(); return u.view }
func (u *pcUpdater) begin(message string) bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.view.Busy {
		return false
	}
	u.view.Busy = true
	u.view.Message = message
	u.view.Percent = 0
	return true
}
func (u *pcUpdater) failed(e error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.view.Busy = false
	u.view.Message = "업데이트 실패 · 기존 버전으로 계속 사용 중: " + e.Error()
}
func (u *pcUpdater) check(ctx context.Context) {
	if !u.begin("새 버전 확인 중…") {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, pcUpdateManifestURL+"?t="+strconv.FormatInt(time.Now().Unix(), 10), nil)
	if e != nil {
		u.failed(e)
		return
	}
	req.Header.Set("Cache-Control", "no-cache")
	resp, e := u.client.Do(req)
	if e != nil {
		u.failed(errors.New("인터넷 연결 후 다시 확인해 주세요"))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		u.failed(fmt.Errorf("배포 정보 응답 %d", resp.StatusCode))
		return
	}
	raw, e := io.ReadAll(io.LimitReader(resp.Body, 32769))
	var m pcUpdateManifest
	if e != nil || len(raw) > 32768 || json.Unmarshal(raw, &m) != nil {
		u.failed(errors.New("배포 정보를 읽지 못했습니다"))
		return
	}
	if e = m.validate(); e != nil {
		u.failed(e)
		return
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.manifest = m
	u.view.Busy = false
	u.view.Version = m.Version
	u.view.Available = pcNewer(m.Version, releaseVersion)
	if u.view.Available {
		u.view.Message = "새 버전 v" + m.Version + " · 업데이트를 누르면 내려받은 뒤 재시작합니다"
	} else {
		u.view.Message = "최신 PC 버전 v" + releaseVersion + " 사용 중"
	}
}
func pcHashFile(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	_, e = io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), e
}
func pcValidateEXE(path string) error {
	f, e := pe.Open(path)
	if e != nil {
		return errors.New("실행 파일 형식을 확인할 수 없습니다")
	}
	defer f.Close()
	h, ok := f.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || f.Machine != pe.IMAGE_FILE_MACHINE_AMD64 || h.Subsystem != pe.IMAGE_SUBSYSTEM_WINDOWS_GUI {
		return errors.New("Windows x64 프로그램이 아닙니다")
	}
	return nil
}

// Stage on the executable's volume. No changes to the live executable or data.
func (u *pcUpdater) download(ctx context.Context, target string) (stage string, err error) {
	if !u.begin("업데이트 다운로드 준비 중…") {
		return "", errors.New("이미 업데이트를 진행 중입니다")
	}
	defer func() {
		if err != nil {
			u.failed(err)
			if stage != "" {
				os.RemoveAll(stage)
			}
		}
	}()
	u.mu.Lock()
	m := u.manifest
	u.mu.Unlock()
	if e := m.validate(); e != nil {
		return "", e
	}
	if !pcNewer(m.Version, releaseVersion) {
		return "", errors.New("새 버전을 다시 확인해 주세요")
	}
	stage, err = os.MkdirTemp(filepath.Dir(target), ".alpha-pc-update-")
	if err != nil {
		return "", errors.New("프로그램 폴더에 쓰기 권한이 없습니다. ZIP을 문서 등 쓰기 가능한 폴더에 풀어 실행해 주세요")
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, m.URL, nil)
	if e != nil {
		return stage, e
	}
	resp, e := u.client.Do(req)
	if e != nil {
		return stage, errors.New("다운로드 연결 실패. 다시 시도해 주세요")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return stage, fmt.Errorf("다운로드 응답 %d", resp.StatusCode)
	}
	if resp.ContentLength > 0 && resp.ContentLength != m.Bytes {
		return stage, errors.New("배포 파일 크기가 다릅니다")
	}
	archive := filepath.Join(stage, "package.zip")
	f, e := os.OpenFile(archive, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return stage, e
	}
	hash := sha256.New()
	reader := io.LimitReader(resp.Body, m.Bytes+1)
	buf := make([]byte, 128<<10)
	var count int64
	for {
		n, re := reader.Read(buf)
		if n > 0 {
			_, e = f.Write(buf[:n])
			if e != nil {
				f.Close()
				return stage, e
			}
			hash.Write(buf[:n])
			count += int64(n)
			u.mu.Lock()
			u.view.Percent = int(count * 100 / m.Bytes)
			u.view.Message = fmt.Sprintf("업데이트 다운로드 %d%% · 기존 프로그램은 계속 실행 중", min(u.view.Percent, 100))
			u.mu.Unlock()
		}
		if re == io.EOF {
			break
		}
		if re != nil {
			f.Close()
			return stage, re
		}
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return stage, e
	}
	if e = f.Close(); e != nil {
		return stage, e
	}
	if count != m.Bytes || hex.EncodeToString(hash.Sum(nil)) != m.SHA256 {
		return stage, errors.New("다운로드 검증 실패. 기존 프로그램을 유지합니다")
	}
	z, e := zip.OpenReader(archive)
	if e != nil {
		return stage, e
	}
	defer z.Close()
	found := 0
	for _, item := range z.File {
		if item.Name != "AlphaWorkComplete-Recovered.exe" {
			continue
		}
		found++
		if found != 1 || item.UncompressedSize64 > pcUpdateMaxBytes {
			return stage, errors.New("실행 파일 크기/구성이 다릅니다")
		}
		src, e := item.Open()
		if e != nil {
			return stage, e
		}
		dest, e := os.OpenFile(filepath.Join(stage, "new.exe"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
		if e != nil {
			src.Close()
			return stage, e
		}
		_, e = io.Copy(dest, io.LimitReader(src, pcUpdateMaxBytes+1))
		src.Close()
		if e == nil {
			e = dest.Sync()
		}
		ce := dest.Close()
		if e == nil {
			e = ce
		}
		if e != nil {
			return stage, e
		}
	}
	if found != 1 {
		return stage, errors.New("실행 파일이 없습니다")
	}
	h, e := pcHashFile(filepath.Join(stage, "new.exe"))
	if e != nil || h != m.EXESHA256 {
		return stage, errors.New("실행 파일 검증 실패")
	}
	if e = pcValidateEXE(filepath.Join(stage, "new.exe")); e != nil {
		return stage, e
	}
	os.Remove(archive)
	u.mu.Lock()
	u.view.Message = "검증 완료 · 프로그램 재시작 준비 중"
	u.mu.Unlock()
	return stage, nil
}

// Keep restart parameters explicit. Never replay first-run copy/empty/promote flags.
func pcRestartArgs(data, listen string) []string {
	return []string{"-data", data, "-listen", listen, "-no-browser"}
}
func pcSafeStage(path string) bool {
	return filepath.IsAbs(path) && strings.HasPrefix(filepath.Base(path), ".alpha-pc-update-")
}
