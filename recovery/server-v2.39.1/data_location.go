package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type DataLocation struct {
	Version           int
	Directory, Source string
}

const locationFile = "data-location-v1.json"
const pendingLocationFile = "pending-data-location-v1.json"

func locationBase(home string) string { return filepath.Join(home, productionFolderName) }
func readLocation(home, name string) (DataLocation, error) {
	var v DataLocation
	b, err := os.ReadFile(filepath.Join(locationBase(home), name))
	if os.IsNotExist(err) {
		return v, nil
	}
	if err != nil {
		return v, err
	}
	if json.Unmarshal(b, &v) != nil || v.Version != 1 || !filepath.IsAbs(v.Directory) {
		return v, errors.New("자료 저장 위치 설정이 올바르지 않습니다.")
	}
	return v, nil
}
func checkLocationPath(source, target string) error {
	if !filepath.IsAbs(target) {
		return errors.New("D:\\AlphaWorkCompleteData처럼 전체 경로를 입력해 주세요.")
	}
	if strings.EqualFold(filepath.Base(target), "WanryoTalk") || strings.EqualFold(filepath.Base(target), "AlphaWorkRecoveredServer") {
		return errors.New("원본 프로그램 폴더 대신 새 전용 폴더를 지정해 주세요.")
	}
	source = filepath.Clean(source)
	target = filepath.Clean(target)
	if filepath.Dir(target) == target {
		return errors.New("드라이브 전체가 아닌 전용 폴더를 지정해 주세요.")
	}
	for _, pair := range [][2]string{{source, target}, {target, source}} {
		rel, err := filepath.Rel(pair[0], pair[1])
		if err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			return errors.New("현재 자료 폴더나 그 안/상위 폴더로는 변경할 수 없습니다.")
		}
	}
	for p := target; ; p = filepath.Dir(p) {
		st, e := os.Lstat(p)
		if e == nil && st.Mode()&os.ModeSymlink != 0 {
			return errors.New("연결된 폴더 대신 실제 저장 경로를 지정해 주세요.")
		}
		if e != nil && !os.IsNotExist(e) {
			return e
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	if st, err := os.Stat(target); err == nil {
		if !st.IsDir() {
			return errors.New("자료 저장 위치가 폴더가 아닙니다.")
		}
		entries, e := os.ReadDir(target)
		if e != nil {
			return e
		}
		if len(entries) > 0 {
			return errors.New("새 폴더 또는 비어 있는 폴더를 지정해 주세요. 기존 자료는 덮어쓰지 않습니다.")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}
func storageInventory(root string) (map[string]string, error) {
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("자료 안의 연결된 파일/폴더는 복사하지 않습니다.")
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == "server.lock" {
			return nil
		}
		st, e := d.Info()
		if e != nil {
			return e
		}
		if !st.Mode().IsRegular() {
			return errors.New("일반 파일이 아닌 자료가 있습니다.")
		}
		rel, e := filepath.Rel(root, p)
		if e != nil {
			return e
		}
		f, e := os.Open(p)
		if e != nil {
			return e
		}
		h := sha256.New()
		_, e = io.Copy(h, f)
		f.Close()
		if e != nil {
			return e
		}
		out[rel] = fmt.Sprintf("%x", h.Sum(nil))
		return nil
	})
	return out, err
}
func migrateDataDirectory(source, target string) error {
	source = filepath.Clean(source)
	target = filepath.Clean(target)
	if err := checkLocationPath(source, target); err != nil {
		return err
	}
	marker, err := os.ReadFile(filepath.Join(source, productionMarker))
	if err != nil || string(marker) != productionMarkerValue {
		return errors.New("현재 운영 자료 폴더를 확인할 수 없습니다.")
	}
	unlock, err := lockData(filepath.Join(source, "server.lock"))
	if err != nil {
		return err
	}
	defer unlock()
	if err = recoverDirect(source); err != nil {
		return err
	}
	before, err := storageInventory(source)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	stage := target + ".copy-" + token()
	if err = os.Mkdir(stage, 0700); err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	for rel := range before {
		dst := filepath.Join(stage, rel)
		if err = os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
			return err
		}
		src, e := os.Open(filepath.Join(source, rel))
		if e != nil {
			return e
		}
		f, e := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			src.Close()
			return e
		}
		_, e = io.Copy(f, src)
		if e == nil {
			e = f.Sync()
		}
		f.Close()
		src.Close()
		if e != nil {
			return e
		}
	}
	copied, err := storageInventory(stage)
	if err != nil || !reflect.DeepEqual(before, copied) {
		return errors.New("자료 복사 검증에 실패했습니다. 기존 자료는 그대로입니다.")
	}
	after, err := storageInventory(source)
	if err != nil || !reflect.DeepEqual(before, after) {
		return errors.New("복사 중 원본 자료가 바뀌었습니다. 다시 시도해 주세요.")
	}
	state, ok, err := loadDirect(stage)
	if err != nil || !ok || state.Board == "" {
		return errors.New("복사한 자료를 읽을 수 없습니다. 기존 경로를 유지합니다.")
	}
	if st, e := os.Stat(target); e == nil && st.IsDir() {
		if err = os.Remove(target); err != nil {
			return err
		}
	}
	return os.Rename(stage, target)
}
func resolveDataLocation(home string) (string, error) {
	active, err := readLocation(home, locationFile)
	if err != nil {
		return "", err
	}
	pending, err := readLocation(home, pendingLocationFile)
	if err != nil {
		return "", err
	}
	if pending.Directory != "" {
		if !filepath.IsAbs(pending.Source) {
			return "", errors.New("기존 자료 위치가 올바르지 않습니다.")
		}
		if err = migrateDataDirectory(pending.Source, pending.Directory); err != nil {
			atomicFile(filepath.Join(locationBase(home), "storage-migration-error.txt"), []byte(err.Error()))
			return pending.Source, nil
		}
		if err = atomicFile(filepath.Join(locationBase(home), locationFile), jsonRaw(DataLocation{Version: 1, Directory: pending.Directory})); err != nil {
			return "", err
		}
		os.Remove(filepath.Join(locationBase(home), pendingLocationFile))
		os.Remove(filepath.Join(locationBase(home), "storage-migration-error.txt"))
		active = pending
	}
	if active.Directory != "" {
		marker, err := os.ReadFile(filepath.Join(active.Directory, productionMarker))
		if err != nil || string(marker) != productionMarkerValue {
			return "", errors.New("설정한 자료 드라이브가 없거나 운영 자료를 읽을 수 없습니다. 드라이브 연결과 저장 위치를 확인해 주세요.")
		}
		if _, err = os.Stat(filepath.Join(active.Directory, "projects-v2.dat")); err != nil {
			return "", errors.New("선택한 자료 파일을 찾을 수 없습니다. 빈 자료로 시작하지 않았습니다.")
		}
	}
	return active.Directory, nil
}
func (a *App) storageHTTP(w http.ResponseWriter, r *http.Request, s Session) bool {
	if r.URL.Path != "/data-storage" {
		return false
	}
	if !localRequest(r) || a.testMode || a.storageHome == "" {
		http.Error(w, "회사 PC의 운영 관리자 화면에서만 저장 위치를 설정할 수 있습니다.", 403)
		return true
	}
	home := a.storageHome
	if r.Method == "POST" {
		if r.FormValue("cancel") == "1" {
			os.Remove(filepath.Join(locationBase(home), pendingLocationFile))
			os.Remove(filepath.Join(locationBase(home), "storage-migration-error.txt"))
			http.Redirect(w, r, "/data-storage", 303)
			return true
		}
		target := filepath.Clean(strings.TrimSpace(r.FormValue("directory")))
		source, err := filepath.Abs(a.dir)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return true
		}
		if err := checkLocationPath(source, target); err != nil {
			http.Error(w, err.Error(), 400)
			return true
		}
		if err := os.MkdirAll(locationBase(home), 0700); err != nil {
			http.Error(w, err.Error(), 500)
			return true
		}
		if err := atomicFile(filepath.Join(locationBase(home), pendingLocationFile), jsonRaw(DataLocation{1, target, source})); err != nil {
			http.Error(w, err.Error(), 500)
			return true
		}
		http.Redirect(w, r, "/data-storage", 303)
		return true
	}
	if r.Method != "GET" {
		http.Error(w, "지원하지 않는 요청입니다.", 405)
		return true
	}
	pending, err := readLocation(home, pendingLocationFile)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return true
	}
	var size int64
	filepath.WalkDir(a.dir, func(p string, d os.DirEntry, e error) error {
		if e == nil && !d.IsDir() {
			if st, e := d.Info(); e == nil && st.Mode().IsRegular() {
				size += st.Size()
			}
		}
		return nil
	})
	msg, _ := os.ReadFile(filepath.Join(locationBase(home), "storage-migration-error.txt"))
	v := a.view(s.CSRF)
	proposed := pending.Directory
	if proposed == "" {
		proposed = `D:\AlphaWorkCompleteData`
	}
	v["ProposedDirectory"] = proposed
	v["Directory"], v["PendingDirectory"], v["StorageError"], v["StorageMB"] = a.dir, pending.Directory, string(msg), fmt.Sprintf("%.1f", float64(size)/(1<<20))
	a.render(w, "data-storage.html", v)
	return true
}
