package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"
)

const activeCopyFile = "active-test-copy.json"

var copyFolderPattern = regexp.MustCompile(`^copy-[0-9]{8}T[0-9]{6}Z-[A-Za-z0-9_-]{43}$`)

type testCopySelection struct {
	Version int
	Folder  string
}

// Paths and file contents are read from source; source is never opened for writing.
func testCopyInventory(source string) (map[string]string, error) {
	out := map[string]string{}
	add := func(rel string) error {
		path := filepath.Join(source, filepath.FromSlash(rel))
		st, e := os.Lstat(path)
		if os.IsNotExist(e) {
			return nil
		}
		if e != nil {
			return e
		}
		if !st.Mode().IsRegular() {
			return fmt.Errorf("복사할 파일이 일반 파일이 아닙니다: %s", rel)
		}
		f, e := os.Open(path)
		if e != nil {
			return e
		}
		h := sha256.New()
		_, e = io.Copy(h, f)
		ce := f.Close()
		if e != nil {
			return e
		}
		if ce != nil {
			return ce
		}
		out[rel] = hex.EncodeToString(h.Sum(nil))
		return nil
	}
	for _, name := range originalFiles {
		if e := add(name); e != nil {
			return nil, e
		}
	}
	if _, ok := out["projects-v2.dat"]; !ok {
		return nil, errors.New("기존 수주 파일 projects-v2.dat를 찾지 못했습니다")
	}
	photoDir := filepath.Join(source, "project-photos")
	st, e := os.Lstat(photoDir)
	if e == nil {
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("사진 폴더가 일반 폴더가 아닙니다")
		}
		entries, e := os.ReadDir(photoDir)
		if e != nil {
			return nil, e
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".dat") {
				continue
			}
			id := strings.TrimSuffix(entry.Name(), ".dat")
			if !validPhotoID.MatchString(id) {
				return nil, errors.New("사진 파일 이름을 확인할 수 없습니다")
			}
			if e = add("project-photos/" + entry.Name()); e != nil {
				return nil, e
			}
		}
	} else if !os.IsNotExist(e) {
		return nil, e
	}
	return out, nil
}
func copyForTest(source, base string, hook func() error) (string, error) {
	source, e := filepath.Abs(source)
	if e != nil {
		return "", e
	}
	before, e := testCopyInventory(source)
	if e != nil {
		return "", e
	}
	if _, e = prepareTestDir(base); e != nil {
		return "", e
	}
	copies := filepath.Join(base, "copies")
	if st, err := os.Lstat(copies); err == nil {
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("복사본 저장 위치가 일반 테스트 폴더가 아닙니다")
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	name := "copy-" + time.Now().UTC().Format("20060102T150405Z") + "-" + token()
	target := filepath.Join(base, "copies", name)
	if _, e = prepareTestDir(target); e != nil {
		return "", e
	}
	for rel, expected := range before {
		path := filepath.Join(target, filepath.FromSlash(rel))
		if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
			return "", e
		}
		input, e := os.Open(filepath.Join(source, filepath.FromSlash(rel)))
		if e != nil {
			return "", e
		}
		output, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			input.Close()
			return "", e
		}
		h := sha256.New()
		_, e = io.Copy(io.MultiWriter(output, h), input)
		ce := input.Close()
		if e == nil {
			e = ce
		}
		if e == nil {
			e = output.Sync()
		}
		ce = output.Close()
		if e == nil {
			e = ce
		}
		if e != nil {
			return "", e
		}
		if hex.EncodeToString(h.Sum(nil)) != expected {
			return "", errors.New("복사하는 중 자료가 변경됐습니다. 기존 프로그램의 저장이 끝난 뒤 다시 시도하세요")
		}
	}
	if hook != nil {
		if e = hook(); e != nil {
			return "", e
		}
	}
	after, e := testCopyInventory(source)
	if e != nil {
		return "", e
	}
	if !reflect.DeepEqual(before, after) {
		return "", errors.New("복사하는 중 기존 자료가 변경됐습니다. 잠시 후 다시 시도하세요. 이전 테스트 자료는 유지됩니다")
	}
	return target, nil
}
func selectedTestDir(base string) (string, error) {
	b, e := os.ReadFile(filepath.Join(base, activeCopyFile))
	if os.IsNotExist(e) {
		return base, nil
	}
	if e != nil {
		return "", e
	}
	var s testCopySelection
	if json.Unmarshal(b, &s) != nil || s.Version != 1 || !copyFolderPattern.MatchString(s.Folder) {
		return "", errors.New("테스트 복사본 선택 정보가 손상됐습니다")
	}
	root, e := filepath.EvalSymlinks(base)
	if e != nil {
		return "", e
	}
	target, e := filepath.EvalSymlinks(filepath.Join(base, "copies", s.Folder))
	if e != nil {
		return "", e
	}
	rel, e := filepath.Rel(root, target)
	if e != nil || rel != filepath.Join("copies", s.Folder) {
		return "", errors.New("테스트 복사본은 테스트 전용 폴더 안에 있어야 합니다")
	}
	marker, e := os.ReadFile(filepath.Join(target, testMarker))
	if e != nil || string(marker) != testMarkerValue {
		return "", errors.New("표시가 없는 테스트 복사본은 열지 않습니다")
	}
	return target, nil
}
func selectTestCopy(base, target string) error {
	name := filepath.Base(target)
	if !copyFolderPattern.MatchString(name) {
		return errors.New("잘못된 복사본 이름")
	}
	b, e := json.Marshal(testCopySelection{1, name})
	if e != nil {
		return e
	}
	return atomicFile(filepath.Join(base, activeCopyFile), b)
}
