package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const testFolderName = "AlphaWorkRecoveryTest-v1.1"
const testListen = "127.0.0.1:28791"
const testMarker = ".alpha-isolated-test-v1"
const testMarkerValue = "AlphaWorkRecovery isolated test data v1\n"

type testNoNetwork struct{}

func (testNoNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("격리 테스트판: 외부 네트워크 전송이 차단되어 있습니다")
}
func blockedTestRoute(p string) bool {
	for _, prefix := range []string{"/phone", "/passkeys/", "/mobile-", "/backup", "/connection"} {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

// An unmarked existing data folder is never imported, copied, backed up or modified.
// The marker is only created in a completely empty directory. Keep it to reopen test data.
func prepareTestDir(dir string) (bool, error) {
	absolute, e := filepath.Abs(dir)
	if e != nil {
		return false, e
	}
	if strings.EqualFold(filepath.Base(absolute), "WanryoTalk") || strings.EqualFold(filepath.Base(absolute), "AlphaWorkRecoveredServer") {
		return false, errors.New("기존 자료 폴더를 테스트판에서 사용할 수 없습니다")
	}
	real, e := filepath.EvalSymlinks(absolute)
	if e == nil {
		if strings.EqualFold(filepath.Base(real), "WanryoTalk") || strings.EqualFold(filepath.Base(real), "AlphaWorkRecoveredServer") {
			return false, errors.New("기존 운영/복구 자료 폴더를 테스트판에서 사용할 수 없습니다")
		}
	} else if !os.IsNotExist(e) {
		return false, e
	}
	if e = os.MkdirAll(absolute, 0700); e != nil {
		return false, e
	}
	marker := filepath.Join(absolute, testMarker)
	if data, e := os.ReadFile(marker); e == nil {
		if string(data) != testMarkerValue {
			return false, errors.New("테스트 폴더 표시가 올바르지 않습니다")
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
		return false, fmt.Errorf("비어 있지 않은 기존 폴더는 읽거나 수정하지 않습니다. 빈 테스트 폴더를 지정하세요: %s", absolute)
	}
	f, e := os.OpenFile(marker, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return false, e
	}
	_, e = f.WriteString(testMarkerValue)
	if e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e == nil {
		e = closeErr
	}
	return true, e
}
func newTestApp(dir string) (*App, error) {
	fresh, e := prepareTestDir(dir)
	if e != nil {
		return nil, e
	}
	return openTestApp(dir, fresh)
}
func openTestApp(dir string, fresh bool) (*App, error) {
	a, e := newApp(dir)
	if e != nil {
		return nil, e
	}
	a.testMode = true
	a.client = &http.Client{Transport: testNoNetwork{}, Timeout: 5 * time.Second}
	// Clear any copied credentials/endpoints even in an explicitly marked test folder.
	e = a.transaction(func(s *State) error {
		s.Integration = IntegrationConfig{}
		if fresh {
			for i, name := range []string{"테스트 수주 · 세팅검사 신청해 보기", "테스트 수주 · 통합검사 신청해 보기"} {
				p := Project{ID: s.next(), Number: fmt.Sprintf("TEST-%03d", i+1), Name: name, Revision: 1, InfoRevision: 1, ShippingDueDate: day(time.Now().AddDate(0, 0, i)), SpecialInstructions: "테스트용 안내입니다. 실제 회사 수주가 아닙니다."}
				p.rotate()
				s.Projects = append(s.Projects, p)
			}
		}
		return nil
	})
	if e != nil {
		return nil, e
	}
	return a, nil
}

func (a *App) sessionCookie() string {
	if a.testMode {
		return "aw_recovery_test_session"
	}
	return "aw_recovered_session"
}
func (a *App) nonceCookie() string {
	if a.testMode {
		return "aw_recovery_test_nonce"
	}
	return "aw_login_nonce"
}
