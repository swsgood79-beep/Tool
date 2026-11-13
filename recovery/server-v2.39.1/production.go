package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const releaseVersion = "1.4.14"
const productionFolderName = "AlphaWorkCompleteRecovered"
const productionListen = "127.0.0.1:18791"
const productionMarker = ".alpha-recovered-production-v1"
const productionMarkerValue = "AlphaWork recovered production data v1\n"
const productionSelection = "active-production-copy.json"

var desktopIsTest = true

func productTitle(test bool) string {
	if test {
		return "알파 복구 테스트판 v" + releaseVersion
	}
	return "알파작업완료확인 v" + releaseVersion + " · 복구 운영판"
}
func desktopIdentity(test bool) (string, string) {
	if test {
		return `Local\AlphaWorkRecoveryTest-v1`, "AlphaWorkRecoveryTestMain"
	}
	return `Local\WanryoTalkNotifier-v1`, "WanryoTalkMain"
}
func prepareProductionDir(dir string) (bool, error) {
	absolute, e := filepath.Abs(dir)
	if e != nil {
		return false, e
	}
	if strings.EqualFold(filepath.Base(absolute), "WanryoTalk") || strings.EqualFold(filepath.Base(absolute), "AlphaWorkRecoveredServer") {
		return false, errors.New("기존 원본 폴더를 운영 저장 위치로 직접 사용하지 않습니다")
	}

	if real, e := filepath.EvalSymlinks(absolute); e == nil {
		if strings.EqualFold(filepath.Base(real), "WanryoTalk") || strings.EqualFold(filepath.Base(real), "AlphaWorkRecoveredServer") {
			return false, errors.New("원본 폴더를 가리키는 경로는 사용할 수 없습니다")
		}
	} else if !os.IsNotExist(e) {
		return false, e
	}
	if e = os.MkdirAll(absolute, 0700); e != nil {
		return false, e
	}
	if b, e := os.ReadFile(filepath.Join(absolute, productionMarker)); e == nil {
		if string(b) != productionMarkerValue {
			return false, errors.New("운영 자료 폴더 표시가 올바르지 않습니다")
		}
		return false, nil
	} else if !os.IsNotExist(e) {
		return false, e
	}
	entries, e := os.ReadDir(absolute)
	if e != nil {
		return false, e
	}
	if len(entries) != 0 {
		return false, errors.New("표시 없는 기존 폴더를 운영 자료로 사용하지 않습니다. 원본은 별도 복사로 가져오세요")
	}
	f, e := os.OpenFile(filepath.Join(absolute, productionMarker), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return false, e
	}
	_, e = f.WriteString(productionMarkerValue)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	return true, e
}
func selectedProductionDir(base string) (string, error) {
	b, e := os.ReadFile(filepath.Join(base, productionSelection))
	if os.IsNotExist(e) {
		return "", nil
	}
	if e != nil {
		return "", e
	}
	var s testCopySelection
	if json.Unmarshal(b, &s) != nil || s.Version != 1 || !copyFolderPattern.MatchString(s.Folder) {
		return "", errors.New("운영 자료 선택 정보가 손상됐습니다")
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
		return "", errors.New("운영 자료는 운영 전용 폴더 안에 있어야 합니다")
	}
	marker, e := os.ReadFile(filepath.Join(target, productionMarker))
	if e != nil || string(marker) != productionMarkerValue {
		return "", errors.New("표시 없는 운영 자료는 열지 않습니다")
	}
	return target, nil
}
func selectProductionDir(base, target string) error {
	name := filepath.Base(target)
	if !copyFolderPattern.MatchString(name) {
		return errors.New("잘못된 운영 복사본 이름")
	}
	b, e := json.Marshal(testCopySelection{1, name})
	if e != nil {
		return e
	}
	return atomicFile(filepath.Join(base, productionSelection), b)
}
func startProductionApp(base, source string, promote bool, original string) (*App, error) {
	if _, e := prepareProductionDir(base); e != nil {
		return nil, e
	}
	selected, e := selectedProductionDir(base)
	if e != nil {
		return nil, e
	}
	if selected != "" {
		if promote {
			return nil, errors.New("이미 운영 자료가 있습니다. 테스트 자료로 덮어쓰지 않았습니다")
		}
		return newApp(selected)
	}
	target, e := copySnapshot(source, base, nil, prepareProductionDir)
	if e != nil {
		return nil, e
	}
	app, e := newApp(target)
	if e != nil {
		return nil, e
	}
	if promote {
		// Business data comes from the explicitly chosen test copy. Restore the original
		// connection/auth settings and board so existing printed QR and mobile login remain usable.
		origin, e := importLegacy(original)
		if e != nil {
			return nil, e
		}
		if e = app.transaction(func(s *State) error {
			s.Integration = origin.Integration
			s.PasswordHash = origin.PasswordHash
			s.Salt = origin.Salt
			s.Board = origin.Board
			s.Integration.Phone.Delivered = s.NextID
			return nil
		}); e != nil {
			return nil, e
		}
	}
	if e = selectProductionDir(base, target); e != nil {
		return nil, e
	}
	return app, nil
}

func (a *App) modeNotice() string {
	if a.testMode {
		return notice
	}
	return `<aside class="recovery-banner">알파작업완료확인 v1.4.14 · 복구 운영판 <a href="/recovery-status">구현·검증 상태</a> · <a href="/">관리자</a></aside>`
}
func (a *App) modeStatus() string {
	if a.testMode {
		return statusPage
	}
	s := strings.ReplaceAll(statusPage, "격리 테스트판 v1.4.14", "복구 운영판 v1.4.14")
	s = strings.ReplaceAll(s, "이 버전은 전용 테스트 폴더·포트·트레이·로그인 쿠키를 사용합니다. 기존 운영 자료를 자동으로 읽지 않습니다. 아래 원본 저장 형식 설명은 테스트 폴더 안에서만 적용되며 외부 연동은 차단되어 있습니다.", "이 운영판은 별도 운영 폴더의 복사본에 저장하며 기존 WanryoTalk 원본은 보존합니다. 외부 접속·알림·시세·일정·백업은 저장된 설정에 따라 사용할 수 있습니다. 이전 프로그램을 종료한 뒤 운영하세요.")
	return s
}

func exitMessage(test bool) string {
	if test {
		return "테스트판을 종료하시겠습니까? 기존 알파 프로그램은 계속 실행됩니다."
	}
	return "종료하면 휴대폰 접속과 알림이 멈춥니다. 운영판을 종료하시겠습니까?"
}

func startEmptyProductionApp(base string) (*App, error) {
	if _, e := prepareProductionDir(base); e != nil {
		return nil, e
	}
	selected, e := selectedProductionDir(base)
	if e != nil {
		return nil, e
	}
	if selected != "" {
		return nil, errors.New("이미 운영 자료가 있어 새 빈 자료로 바꾸지 않았습니다")
	}
	copies := filepath.Join(base, "copies")
	if st, e := os.Lstat(copies); e == nil {
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("운영 복사본 위치가 일반 폴더가 아닙니다")
		}
	} else if !os.IsNotExist(e) {
		return nil, e
	}
	target := filepath.Join(copies, "copy-"+time.Now().UTC().Format("20060102T150405Z")+"-"+token())
	if _, e = prepareProductionDir(target); e != nil {
		return nil, e
	}
	app, e := newApp(target)
	if e != nil {
		return nil, e
	}
	if e = selectProductionDir(base, target); e != nil {
		return nil, e
	}
	return app, nil
}

func latestBusinessSource(home string) (string, error) {
	selected, e := selectedProductionDir(filepath.Join(home, productionFolderName))
	if e != nil {
		return "", e
	}
	if selected != "" {
		return selected, nil
	}
	return filepath.Join(home, "WanryoTalk"), nil
}

// Called only before the first operating copy is selected; updates never choose again.
func firstProductionChoice(home string, ask func(string) int) (promote, empty bool, err error) {
	original := filepath.Join(home, "WanryoTalk", "projects-v2.dat")
	_, originalErr := os.Stat(original)
	testDir, e := selectedTestDir(filepath.Join(home, testFolderName))
	if e != nil {
		return false, false, e
	}
	_, testErr := os.Stat(filepath.Join(testDir, "projects-v2.dat"))
	if originalErr != nil && !os.IsNotExist(originalErr) {
		return false, false, originalErr
	}
	if testErr != nil && !os.IsNotExist(testErr) {
		return false, false, testErr
	}
	if originalErr == nil {
		message := "처음 시작하는 자료를 선택해 주세요.\n\n[예] 기존 WanryoTalk 자료·사진·설정을 복사해 시작\n"
		if testErr == nil {
			message += "[아니요] 테스트판에서 수정한 업무 자료로 시작 (연결 설정은 원본에서 가져옴)\n"
		} else {
			message += "[아니요] 시작 취소\n"
		}
		message += "[취소] 시작 취소\n\n원본은 보존하며 이후 실행·업데이트는 운영 자료를 자동으로 이어서 엽니다."
		switch ask(message) {
		case 6:
			return false, false, nil
		case 7:
			if testErr == nil {
				return true, false, nil
			}
		}
	} else {
		if ask("기존 WanryoTalk 수주 파일을 찾지 못했습니다.\n\n[예] 빈 자료로 시작 (새 PC에서는 관리자 설정에서 암호화 백업 복원)\n[아니요/취소] 종료\n\n기존 자료를 가져오려면 원래 PC의 같은 Windows 계정에서 실행하세요.") == 6 {
			return false, true, nil
		}
	}
	return false, false, errors.New("자료 선택을 취소했습니다. 기존 자료는 변경하지 않았습니다")
}
