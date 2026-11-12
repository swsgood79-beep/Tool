package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

const journalName = ".direct-write-journal.dat"

type journalEntry struct {
	Name, OldHash, NewHash string
	HadOld                 bool
}
type writeJournal struct {
	Version   int
	Directory string
	Committed bool
	Entries   []journalEntry
}

var stageName = regexp.MustCompile(`^\.alpha-direct-[A-Za-z0-9_-]{43}$`)

func writeJournalFile(dir string, j writeJournal) error {
	b, e := json.Marshal(j)
	if e != nil {
		return e
	}
	b, e = protect(b)
	if e != nil {
		return e
	}
	return atomicFile(filepath.Join(dir, journalName), b)
}
func validateJournal(j writeJournal) error {
	if j.Version != 1 || !stageName.MatchString(j.Directory) || len(j.Entries) != len(originalFiles) {
		return errors.New("직접 저장 저널 형식 오류")
	}
	seen := map[string]bool{}
	for _, e := range j.Entries {
		if !isOriginalFile(e.Name) || seen[e.Name] || len(e.NewHash) != 64 || (e.HadOld && len(e.OldHash) != 64) {
			return errors.New("직접 저장 저널 경로·해시 오류")
		}
		seen[e.Name] = true
	}
	return nil
}
func recoverDirect(dir string) error {
	b, e := os.ReadFile(filepath.Join(dir, journalName))
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	b, e = unprotect(b)
	if e != nil {
		return e
	}
	var j writeJournal
	if e = json.Unmarshal(b, &j); e != nil {
		return e
	}
	if e = validateJournal(j); e != nil {
		return e
	}
	stage := filepath.Join(dir, j.Directory)
	// Inspect every file before changing any: an unexpected third-party write is never overwritten.
	for _, entry := range j.Entries {
		b, e := os.ReadFile(filepath.Join(dir, entry.Name))
		if os.IsNotExist(e) {
			if j.Committed || entry.HadOld {
				return fmt.Errorf("복구 전 파일이 없습니다: %s", entry.Name)
			}
			continue
		}
		if e != nil {
			return e
		}
		hash := digest(b)
		if hash != entry.NewHash && (!entry.HadOld || hash != entry.OldHash) {
			return fmt.Errorf("직접 저장 중 외부 변경을 감지했습니다. 저널·백업을 보존했습니다: %s", entry.Name)
		}
		if j.Committed && hash != entry.NewHash {
			return errors.New("완료 저널과 현재 자료가 다릅니다")
		}
	}
	if !j.Committed {
		// Validate all rollback bytes before touching any current file.
		for i, entry := range j.Entries {
			if entry.HadOld {
				old, err := os.ReadFile(filepath.Join(stage, fmt.Sprintf("old-%d", i)))
				if err != nil || digest(old) != entry.OldHash {
					return fmt.Errorf("이전 파일 검증 실패: %s", entry.Name)
				}
			}
		}
		for i, entry := range j.Entries {
			path := filepath.Join(dir, entry.Name)
			if entry.HadOld {
				old, e := os.ReadFile(filepath.Join(stage, fmt.Sprintf("old-%d", i)))
				if e != nil || digest(old) != entry.OldHash {
					return fmt.Errorf("이전 파일 검증 실패: %s", entry.Name)
				}
				if e = atomicFile(path, old); e != nil {
					return e
				}
			} else {
				if e = os.Remove(path); e != nil && !os.IsNotExist(e) {
					return e
				}
			}
		}
	}
	if e = os.Remove(filepath.Join(dir, journalName)); e != nil {
		return e
	}
	return os.RemoveAll(stage)
}
func (a *App) writeOriginalGroup(n *State) error {
	if e := a.checkLegacySource(); e != nil {
		return e
	}
	if e := backupOriginals(a.dir); e != nil {
		return e
	}
	plain, e := a.projectOriginal(n)
	if e != nil {
		return e
	}
	j := writeJournal{Version: 1, Directory: ".alpha-direct-" + token()}
	stage := filepath.Join(a.dir, j.Directory)
	if e = os.Mkdir(stage, 0700); e != nil {
		return e
	}
	prepared := false
	defer func() {
		if !prepared {
			os.RemoveAll(stage)
		}
	}()
	for i, name := range originalFiles {
		data := plain[name]
		cipher, e := protect(data)
		if e != nil {
			return e
		}
		roundtrip, e := unprotect(cipher)
		if e != nil || digest(roundtrip) != digest(data) {
			return errors.New("암호화 저장 왕복 검증 실패")
		}
		entry := journalEntry{Name: name, NewHash: digest(cipher)}
		old, e := os.ReadFile(filepath.Join(a.dir, name))
		if e == nil {
			entry.HadOld = true
			entry.OldHash = digest(old)
			if e = atomicFile(filepath.Join(stage, fmt.Sprintf("old-%d", i)), old); e != nil {
				return e
			}
		} else if !os.IsNotExist(e) {
			return e
		}
		if e = atomicFile(filepath.Join(stage, fmt.Sprintf("new-%d", i)), cipher); e != nil {
			return e
		}
		j.Entries = append(j.Entries, entry)
	}
	if e = a.checkLegacySource(); e != nil {
		return e
	}
	if e = writeJournalFile(a.dir, j); e != nil {
		return e
	}
	prepared = true
	fail := func(cause error) error {
		if e := recoverDirect(a.dir); e != nil {
			return fmt.Errorf("%v; 자동 복구 중단: %w", cause, e)
		}
		return cause
	}
	for i, entry := range j.Entries {
		current, e := os.ReadFile(filepath.Join(a.dir, entry.Name))
		if entry.HadOld {
			if e != nil || digest(current) != entry.OldHash {
				return fail(errors.New("저장 직전 원본 변경 감지"))
			}
		} else if e == nil || !os.IsNotExist(e) {
			return fail(errors.New("저장 직전 새 원본 파일 충돌"))
		}
		data, e := os.ReadFile(filepath.Join(stage, fmt.Sprintf("new-%d", i)))
		if e != nil || digest(data) != entry.NewHash {
			return fail(errors.New("준비 파일 검증 실패"))
		}
		if e = atomicFile(filepath.Join(a.dir, entry.Name), data); e != nil {
			return fail(e)
		}
		if a.directHook != nil {
			if e = a.directHook("written", entry.Name); e != nil {
				return fail(e)
			}
		}
	}
	j.Committed = true
	if e = writeJournalFile(a.dir, j); e != nil {
		return fail(e)
	}
	if a.directHook != nil {
		if e = a.directHook("committed", ""); e != nil {
			a.storageNotice = "저장은 완료됐습니다: " + e.Error()
		}
	}
	n.LegacyHashes = map[string]string{}
	for _, entry := range j.Entries {
		n.LegacyHashes[entry.Name] = entry.NewHash
	}
	n.LegacySourceHash = n.LegacyHashes["projects-v2.dat"]
	if e = recoverDirect(a.dir); e != nil {
		a.storageNotice = "저장은 완료됐지만 완료 저널 정리가 필요합니다: " + e.Error()
	}
	return nil
}
