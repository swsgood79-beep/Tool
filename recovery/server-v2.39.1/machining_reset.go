package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type MachiningResetAudit struct {
	Target, Executor, Backup string
	At                       time.Time
	Runs, Notices            int
}
type MachiningResetRow struct {
	Job, Order, Part, Phase, Run, Reason string
	Quantity                             int
	Started                              time.Time
	Completed                            *time.Time
}
type MachiningResetPreview struct {
	Account, Fingerprint     string
	Rows                     []MachiningResetRow
	Removable, Held, Notices int
}

// Downstream quantities have no run-level provenance. Hold the whole job when
// a different actor has later-stage work, rather than inventing a dependency.
func machiningResetPlan(s *State, account string) MachiningResetPreview {
	p := MachiningResetPreview{Account: account}
	p.Fingerprint = digest(jsonRaw([]any{account, s.MachiningJobs, s.TeamNotices, s.AdminAccounts, s.MachiningResets}))
	for _, j := range s.MachiningJobs {
		stages := [][]MachiningRun{j.Cutting, j.MillingFirst, j.Milling}
		earliest := 3
		reason := ""
		for _, source := range j.Sources {
			if source.Received > 0 {
				reason = "ERP 입고 수량이 있는 작업"
			}
		}
		for phase, runs := range stages {
			for _, run := range runs {
				if run.StartedBy.ID == account || run.CompletedBy != nil && run.CompletedBy.ID == account {
					earliest = min(earliest, phase)
				}
				if run.ReceiptAt != nil {
					reason = "ERP 입고 확인이 있는 작업"
				}
			}
		}
		for phase, runs := range stages {
			for _, run := range runs {
				if phase > earliest && (run.StartedBy.ID != account || run.CompletedBy != nil && run.CompletedBy.ID != account) {
					reason = "다른 작업자의 후속 공정과 연결된 작업"
				}
			}
		}
		for phase, runs := range stages {
			for _, run := range runs {
				if run.StartedBy.ID != account && (run.CompletedBy == nil || run.CompletedBy.ID != account) {
					continue
				}
				hold := reason
				if run.StartedBy.ID != account || run.CompletedBy != nil && run.CompletedBy.ID != account {
					hold = "여러 작업자의 기록이 연결된 공정"
				}
				p.Rows = append(p.Rows, MachiningResetRow{Job: j.ID, Order: j.Order, Part: j.Part, Phase: []string{"절단", "가공 1차", "가공 2차"}[phase], Run: run.ID, Reason: hold, Quantity: run.Quantity, Started: run.Started, Completed: run.Completed})
				if hold != "" {
					p.Held++
				} else {
					p.Removable++
					for _, n := range s.TeamNotices {
						if n.JobID == j.ID && n.RunID == run.ID {
							p.Notices++
						}
					}
				}
			}
		}
	}
	return p
}
func resetRunKey(job, run string) string { return job + "\x00" + run }
func (a *App) resetMachining(account, fingerprint string, actor MachiningActor) error {
	if account == "" || adminAccount(&a.state, account) == nil {
		return fmt.Errorf("대상 계정을 다시 선택해 주세요")
	}
	p := machiningResetPlan(&a.state, account)
	if !equal(p.Fingerprint, fingerprint) {
		return fmt.Errorf("미리보기 이후 기록이 변경됐습니다. 다시 미리보기를 확인해 주세요")
	}
	if p.Removable == 0 {
		return fmt.Errorf("초기화할 수 있는 기록이 없습니다")
	}
	now := time.Now()
	name := "machining-test-reset-" + now.UTC().Format("20060102T150405.000000000") + "-" + token()[:12] + ".dpapi"
	backupDir := filepath.Join(a.dir, "machining-test-backups")
	// Back up only the affected machining domain. Credentials and ERP data are
	// never exported. DPAPI uses the same Windows identity as the state store.
	raw := jsonRaw(struct {
		Preview  MachiningResetPreview
		Executor MachiningActor
		At       time.Time
		Jobs     []MachiningJob
		Notices  []TeamNotice
	}{p, actor, now, a.state.MachiningJobs, a.state.TeamNotices})
	encrypted, err := protect(raw)
	if err != nil {
		return fmt.Errorf("백업 암호화 실패. 초기화하지 않았습니다")
	}
	if err = os.MkdirAll(backupDir, 0700); err != nil {
		return fmt.Errorf("백업 폴더 저장 실패. 초기화하지 않았습니다")
	}
	path := filepath.Join(backupDir, name)
	if err = atomicFile(path, encrypted); err != nil {
		return fmt.Errorf("백업 저장 실패. 초기화하지 않았습니다")
	}
	// Read back and decrypt before allowing the durable state transaction.
	saved, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("백업 확인 실패")
	}
	restored, err := unprotect(saved)
	if err != nil || !equal(digest(raw), digest(restored)) {
		return fmt.Errorf("백업 검증 실패")
	}
	remove := map[string]bool{}
	for _, row := range p.Rows {
		if row.Reason == "" {
			remove[resetRunKey(row.Job, row.Run)] = true
		}
	}
	return a.commitAnnouncement(func(s *State) error {
		for i := range s.MachiningJobs {
			j := &s.MachiningJobs[i]
			changed := false
			for _, runs := range []*[]MachiningRun{&j.Cutting, &j.MillingFirst, &j.Milling} {
				kept := (*runs)[:0:0]
				for _, run := range *runs {
					if remove[resetRunKey(j.ID, run.ID)] {
						changed = true
					} else {
						kept = append(kept, run)
					}
				}
				*runs = kept
			}
			legacy := j.LegacyMilling[:0:0]
			for _, id := range j.LegacyMilling {
				if !remove[resetRunKey(j.ID, id)] {
					legacy = append(legacy, id)
				}
			}
			j.LegacyMilling = legacy
			if changed {
				j.Revision++
			}
		}
		notices := []TeamNotice{}
		for _, n := range s.TeamNotices {
			if !remove[resetRunKey(n.JobID, n.RunID)] {
				notices = append(notices, n)
			}
		}
		s.TeamNotices = notices
		s.MachiningResets = append(s.MachiningResets, MachiningResetAudit{Target: account, Executor: actor.Username, Backup: name, At: now, Runs: p.Removable, Notices: p.Notices})
		return nil
	})
}
func (a *App) machiningResetHTTP(w http.ResponseWriter, r *http.Request, s Session) bool {
	if r.URL.Path != "/machining-admin/test-reset" {
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	if s.AccountID != "" {
		http.Error(w, "대표 관리자만 이용할 수 있습니다", 403)
		return true
	}
	if r.Method != "GET" && r.Method != "POST" {
		http.Error(w, "지원하지 않는 요청입니다", 405)
		return true
	}
	account := r.FormValue("account")
	message := ""
	if r.Method == "POST" {
		if !equal(r.FormValue("csrf"), s.CSRF) {
			http.Error(w, "요청 확인 실패", 403)
			return true
		}
		if r.FormValue("confirmed") != "yes" {
			http.Error(w, "확인 체크가 필요합니다", 400)
			return true
		}
		if err := a.resetMachining(account, r.FormValue("fingerprint"), machiningActor(&a.state, s)); err != nil {
			message = err.Error()
			w.WriteHeader(409)
		} else {
			message = "백업 후 초기화했습니다"
		}
	}
	var preview *MachiningResetPreview
	if account != "" && adminAccount(&a.state, account) != nil {
		p := machiningResetPlan(&a.state, account)
		preview = &p
	}
	a.renderTemplate(w, templates, "machining-reset.html", map[string]any{"CSRF": s.CSRF, "Accounts": a.state.AdminAccounts, "Account": account, "Preview": preview, "Message": message, "History": a.state.MachiningResets})
	return true
}
