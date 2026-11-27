package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Machining data is independent of the live order/schedule list. Missing ERP
// rows never remove work or imply a receipt. Actor names are immutable snapshots.
type MachiningActor struct{ ID, Username, Name string }
type MachiningSource struct {
	Row                                        int
	Purchase, Order, Project, Robot, Part, Due string
	Quantity, Received                         int
}
type MachiningRun struct {
	ID                 string
	QuantityOnComplete bool `json:",omitempty"`
	Quantity           int
	Started            time.Time
	StartedBy          MachiningActor
	Completed          *time.Time      `json:",omitempty"`
	CompletedBy        *MachiningActor `json:",omitempty"`
	ReceiptAt          *time.Time      `json:",omitempty"`
	ReceiptBy          *MachiningActor `json:",omitempty"`
}
type MachiningJob struct {
	ID, Purchase, Order, Project, Part string
	Required                           int
	Sources                            []MachiningSource
	Cutting, Milling                   []MachiningRun
	MillingFirst                       []MachiningRun
	Workflow                           int
	LegacyMilling                      []string
	Revision                           int64
	LastSeen                           time.Time
	Missing, SourceChanged             bool
	ReportOutstanding                  int
}
type MachiningImport struct {
	Hash, Filename      string
	At                  time.Time
	Actor               MachiningActor
	Rows, Groups, Units int
}
type MachiningDraft struct {
	Owner          string
	Created        time.Time
	Filename, Hash string
	Rows           []MachiningSource
}

func machiningSession(s *State, session Session) bool {
	u := adminAccount(s, session.AccountID)
	return u != nil && u.Role == "machining"
}
func machiningActor(s *State, session Session) MachiningActor {
	if u := adminAccount(s, session.AccountID); u != nil {
		return MachiningActor{u.ID, u.Username, u.Name}
	}
	return MachiningActor{"", ownerUsername, ownerDisplayName(s)}
}
func machiningKey(purchase, order, part string) string {
	return digest(jsonRaw([]string{purchase, order, part}))
}
func machiningFind(s *State, id string) *MachiningJob {
	for i := range s.MachiningJobs {
		if s.MachiningJobs[i].ID == id {
			return &s.MachiningJobs[i]
		}
	}
	return nil
}
func machiningDone(runs []MachiningRun) int {
	n := 0
	for _, r := range runs {
		if r.Completed != nil {
			n += r.Quantity
		}
	}
	return n
}
func machiningActive(runs []MachiningRun) *MachiningRun {
	for i := range runs {
		if runs[i].Completed == nil {
			return &runs[i]
		}
	}
	return nil
}
func machiningUnreceived(s *State) int {
	n := 0
	for _, j := range s.MachiningJobs {
		for _, r := range j.Milling {
			if r.Completed != nil && r.ReceiptAt == nil {
				n += r.Quantity
			}
		}
	}
	return n
}
func machiningGroups(rows []MachiningSource) []MachiningJob {
	out := []MachiningJob{}
	indices := map[string]int{}
	for _, r := range rows {
		key := machiningKey(r.Purchase, r.Order, r.Part)
		i, ok := indices[key]
		if !ok {
			i = len(out)
			indices[key] = i
			out = append(out, MachiningJob{Workflow: 2, ID: key, Purchase: r.Purchase, Order: r.Order, Project: r.Project, Part: r.Part})
		}
		out[i].Sources = append(out[i].Sources, r)
		out[i].Required += r.Quantity - r.Received
	}
	for i := range out {
		out[i].ReportOutstanding = out[i].Required
	}
	return out
}
func (a *App) importMachining(d MachiningDraft, actor MachiningActor) error {
	if n := len(a.state.MachiningImports); n > 0 && a.state.MachiningImports[n-1].Hash == d.Hash {
		return nil
	}
	groups := machiningGroups(d.Rows)
	if len(groups) == 0 {
		return errors.New("가져올 BASE 미입고 항목이 없습니다")
	}
	return a.commitAnnouncement(func(s *State) error {
		if len(s.MachiningJobs)+len(groups) > 50000 {
			return errors.New("가공 작업목록 한도를 확인해 주세요")
		}
		seen := map[string]bool{}
		now := time.Now()
		units := 0
		for _, g := range groups {
			seen[g.ID] = true
			units += g.Required
			j := machiningFind(s, g.ID)
			if j == nil {
				g.Revision = 1
				g.LastSeen = now
				s.MachiningJobs = append(s.MachiningJobs, g)
				continue
			}
			// Never reset process history, even if the ERP report's outstanding amount shrinks.
			if len(j.Cutting) == 0 && len(j.Milling) == 0 && len(j.MillingFirst) == 0 {
				j.Required = g.Required
			}
			j.ReportOutstanding = g.Required
			received := 0
			for _, run := range j.Milling {
				if run.ReceiptAt != nil {
					received += run.Quantity
				}
			}
			j.SourceChanged = j.Required-received != g.Required
			j.Sources = g.Sources
			j.Project = g.Project
			j.LastSeen = now
			j.Missing = false
			j.Revision++
		}
		for i := range s.MachiningJobs {
			if !seen[s.MachiningJobs[i].ID] {
				s.MachiningJobs[i].Missing = true
			}
		}
		s.MachiningImports = append(s.MachiningImports, MachiningImport{Hash: d.Hash, Filename: d.Filename, At: now, Actor: actor, Rows: len(d.Rows), Groups: len(groups), Units: units})
		return nil
	})
}
func (a *App) actMachining(session Session, id, phase, action, runID string, revision int64, quantity int) error {
	if !validAccountSession(&a.state, session) || !machiningSession(&a.state, session) {
		return errors.New("가공팀 계정으로 로그인해 주세요")
	}
	actor := machiningActor(&a.state, session)
	return a.commitAnnouncement(func(s *State) error {
		j := machiningFind(s, id)
		if j == nil {
			return errors.New("가공 작업을 찾을 수 없습니다")
		}
		if j.Revision != revision {
			return errors.New("다른 화면에서 작업이 변경됐습니다. 새로고침 후 확인해 주세요")
		}
		machiningUpgrade(j)
		runs := &j.Cutting
		available := j.Required - machiningDone(j.Cutting)
		if phase == "milling1" {
			runs = &j.MillingFirst
			available = machiningDone(j.Cutting) - machiningDone(j.MillingFirst) - machiningFirstCredit(j, false)
		} else if phase == "milling2" || phase == "milling" {
			if phase == "milling" && action != "complete" {
				return errors.New("가공 공정이 변경됐습니다. 화면을 새로고침해 주세요")
			}
			runs = &j.Milling
			available = machiningFirstDone(j) - machiningDone(j.Milling)
		} else if phase != "cutting" {
			return errors.New("공정을 확인해 주세요")
		}
		active := machiningActive(*runs)
		now := time.Now()
		switch action {
		case "start":
			if j.SourceChanged {
				return errors.New("ERP 수량이 변경됐습니다. 관리자에게 수량 확인을 요청해 주세요")
			}
			if active != nil {
				return errors.New("이미 진행 중인 작업입니다")
			}
			if quantity < 0 || quantity > available || available < 1 {
				return fmt.Errorf("시작할 수량은 1~%d개입니다", available)
			}
			*runs = append(*runs, MachiningRun{ID: token(), Quantity: quantity, QuantityOnComplete: quantity == 0, Started: now, StartedBy: actor})
		case "complete":
			if phase == "milling" && !machiningLegacy(j, runID) {
				return errors.New("가공 공정을 다시 확인해 주세요")
			}
			if active == nil || active.ID != runID {
				return errors.New("진행 중인 작업을 다시 확인해 주세요")
			}
			if active.StartedBy.ID != actor.ID {
				return errors.New("작업을 시작한 계정에서 완료해 주세요")
			}
			if active.QuantityOnComplete {
				if quantity < 1 || quantity > available {
					return fmt.Errorf("완료 수량은 1~%d개로 입력해 주세요", available)
				}
				active.Quantity = quantity
			} else if quantity != 0 {
				// Existing pre-update runs keep their start time and can complete
				// a smaller actual batch; old clients can still submit zero for full.
				limit := active.Quantity
				if !(phase == "milling" || phase == "milling2") || !machiningLegacy(j, active.ID) {
					if available < limit {
						limit = available
					}
				}
				if quantity < 1 || quantity > limit {
					return fmt.Errorf("완료 수량은 1~%d개로 입력해 주세요", limit)
				}
				active.Quantity = quantity
			}
			active.Completed = &now
			active.CompletedBy = &actor
			machiningCompletionNotice(s, j, phase, *active, now)
		default:
			return errors.New("작업 요청을 확인해 주세요")
		}
		j.Revision++
		return nil
	})
}
func (a *App) machiningAdminAction(session Session, id, action, runID string, revision int64, quantity int) error {
	if !validAccountSession(&a.state, session) || machiningSession(&a.state, session) || purchasingSession(&a.state, session) {
		return errors.New("관리자 로그인이 필요합니다")
	}
	actor := machiningActor(&a.state, session)
	return a.commitAnnouncement(func(s *State) error {
		j := machiningFind(s, id)
		if j == nil || j.Revision != revision {
			return errors.New("작업 정보가 변경됐습니다. 다시 확인해 주세요")
		}
		now := time.Now()
		if action == "receipt" {
			for i := range j.Milling {
				r := &j.Milling[i]
				if r.ID == runID && r.Completed != nil && r.ReceiptAt == nil {
					r.ReceiptAt = &now
					r.ReceiptBy = &actor
					j.Revision++
					return nil
				}
			}
			return errors.New("입고 확인할 완료 작업을 찾을 수 없습니다")
		}
		if action == "quantity" {
			min := machiningDone(j.Cutting)
			if r := machiningActive(j.Cutting); r != nil {
				min += r.Quantity
			}
			if quantity < min || quantity < 1 || quantity > 1000000 {
				return errors.New("수량은 이미 시작·완료한 절단 수량 이상이어야 합니다")
			}
			// Retain every deliberate correction, rather than silently rewriting totals.
			s.MachiningAdjustments = append(s.MachiningAdjustments, MachiningAdjustment{JobID: id, Before: j.Required, After: quantity, At: now, Actor: actor})
			j.Required = quantity
			j.SourceChanged = false
			j.Revision++
			return nil
		}
		return errors.New("관리 요청을 확인해 주세요")
	})
}

type MachiningAdjustment struct {
	JobID         string
	Before, After int
	At            time.Time
	Actor         MachiningActor
}

func machiningSorted(s *State) []MachiningJob {
	jobs := append([]MachiningJob{}, s.MachiningJobs...)
	sort.SliceStable(jobs, func(i, j int) bool {
		a, b := jobs[i], jobs[j]
		return strings.Join([]string{a.Order, a.Purchase, a.Part}, "/") < strings.Join([]string{b.Order, b.Purchase, b.Part}, "/")
	})
	return jobs
}
