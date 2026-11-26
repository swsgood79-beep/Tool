package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type DeliveryLot struct {
	ID, Date, Form     string
	Number             int
	Inspection         Stage
	Attempted, Missing bool
}
type DeliveryView struct {
	DeliveryLot
	RecordID  string
	PhotoDone bool
}

func findDelivery(p *Project, id string) *DeliveryLot {
	for i := range p.DeliveryLots {
		if p.DeliveryLots[i].ID == id {
			return &p.DeliveryLots[i]
		}
	}
	return nil
}
func deliveryForRequest(p *Project, id int64) *DeliveryLot {
	if id == 0 {
		return nil
	}
	for i := range p.DeliveryLots {
		if p.DeliveryLots[i].Inspection.EventID == id {
			return &p.DeliveryLots[i]
		}
	}
	return nil
}
func deliveryRecord(s *State, p Project, l DeliveryLot) *ShippingRecord {
	for i := range s.ShippingRecords {
		r := &s.ShippingRecords[i]
		if r.Project.ID == p.ID && r.DeliveryGroup == p.DeliveryGroup && r.DeliveryID == l.ID && r.Project.Shipping.EventID == l.Inspection.EventID {
			return r
		}
	}
	return nil
}
func (p Project) DeliverySummary() string {
	n := 0
	for _, l := range p.DeliveryLots {
		if l.Inspection.Completed != nil {
			n++
		}
	}
	if p.Shipping.Completed != nil {
		return "모든 차수 검사·사진 완료"
	}
	label := fmt.Sprintf("분할 출고 · 검사 %d/%d차 완료", n, len(p.DeliveryLots))
	if n == len(p.DeliveryLots) {
		label += " · 사진 등록 대기"
	}
	return label
}
func (a *App) deliveryViews(p Project) []DeliveryView {
	out := []DeliveryView{}
	for _, l := range p.DeliveryLots {
		v := DeliveryView{DeliveryLot: l}
		if r := deliveryRecord(&a.state, p, l); r != nil {
			v.RecordID = r.ID
			v.PhotoDone = r.ArchivedAt != nil
		}
		out = append(out, v)
	}
	return out
}

// Schedule refresh only creates/updates untouched slots. Processed slots and
// historical snapshots keep immutable IDs and never inherit another lot's work.
func reconcileDeliveries(s *State, p *Project, dates []string) {
	if p.LegacyCompleted && p.Shipping.EventID == 0 {
		return
	}
	if len(dates) < 2 && len(p.DeliveryLots) == 0 {
		return
	}
	before := digest(jsonRaw(p.DeliveryLots))
	oldPlan := map[string]string{}
	for _, l := range p.DeliveryLots {
		oldPlan[l.ID] = fmt.Sprintf("%s/%d", l.Date, l.Number)
	}
	if len(p.DeliveryLots) == 0 {
		p.DeliveryGroup = token()
		if p.Shipping.EventID > 0 {
			date := p.ShippingDueDate
			if date == "" && len(dates) > 0 {
				date = dates[0]
			}
			l := DeliveryLot{ID: token(), Date: date, Number: 1, Form: token(), Inspection: p.Shipping, Attempted: true}
			p.DeliveryLots = append(p.DeliveryLots, l)
			for i := range s.ShippingRecords {
				r := &s.ShippingRecords[i]
				if r.Project.ID == p.ID && r.Project.Shipping.EventID == l.Inspection.EventID {
					r.DeliveryGroup = p.DeliveryGroup
					r.DeliveryID = l.ID
					r.DeliveryNumber = 1
					r.Project.ShippingDueDate = l.Date
					r.Project.ShippingDueDates = nil
				}
			}
			// Split an old combined request into common setting + lot-specific shipping.
			for i := range s.Events {
				e := &s.Events[i]
				if e.ID == p.Setting.EventID || e.ID == p.Shipping.EventID {
					e.BatchID = 0
					e.InspectionName = stageLabel(e.Inspection)
					e.Revision++
				}
				if e.ID == p.Shipping.EventID {
					e.DeliveryID = l.ID
					e.DeliveryNumber = 1
					e.InspectionName = "1차 출하검사"
				}
			}
			p.CombinedPending = false
		}
	}
	wanted := map[string]bool{}
	for _, d := range dates {
		wanted[d] = true
	}
	used := map[string]bool{}
	// Reuse a not-yet-started slot for a rescheduled date before creating a new lot.
	for _, d := range dates {
		exists := false
		for i := range p.DeliveryLots {
			if p.DeliveryLots[i].Date == d {
				p.DeliveryLots[i].Missing = false
				used[p.DeliveryLots[i].ID] = true
				exists = true
				break
			}
		}
		if exists {
			continue
		}
		chosen := -1
		for i, l := range p.DeliveryLots {
			if !l.Attempted && !wanted[l.Date] && !used[l.ID] {
				chosen = i
				break
			}
		}
		if chosen >= 0 {
			p.DeliveryLots[chosen].Date = d
			p.DeliveryLots[chosen].Missing = false
			used[p.DeliveryLots[chosen].ID] = true
		} else {
			l := DeliveryLot{ID: token(), Date: d, Form: token()}
			p.DeliveryLots = append(p.DeliveryLots, l)
			used[l.ID] = true
		}
	}
	hasHistory := false
	for _, l := range p.DeliveryLots {
		hasHistory = hasHistory || l.Attempted
	}
	keep := []DeliveryLot{}
	for _, l := range p.DeliveryLots {
		if !wanted[l.Date] && !l.Attempted && !hasHistory {
			continue
		}
		l.Missing = !wanted[l.Date]
		keep = append(keep, l)
	}
	p.DeliveryLots = keep
	// Once a lot has been requested, its displayed ordinal is stable.
	anyAttempt := false
	maxNo := 0
	for _, l := range p.DeliveryLots {
		anyAttempt = anyAttempt || l.Attempted
		if l.Number > maxNo {
			maxNo = l.Number
		}
	}
	sort.SliceStable(p.DeliveryLots, func(i, j int) bool { return p.DeliveryLots[i].Date < p.DeliveryLots[j].Date })
	if !anyAttempt {
		for i := range p.DeliveryLots {
			p.DeliveryLots[i].Number = i + 1
		}
	} else {
		for i := range p.DeliveryLots {
			if p.DeliveryLots[i].Number == 0 {
				maxNo++
				p.DeliveryLots[i].Number = maxNo
			}
		}
		sort.SliceStable(p.DeliveryLots, func(i, j int) bool { return p.DeliveryLots[i].Number < p.DeliveryLots[j].Number })
	}
	for i := range p.DeliveryLots {
		l := &p.DeliveryLots[i]
		if old, ok := oldPlan[l.ID]; ok && !l.Attempted && old != fmt.Sprintf("%s/%d", l.Date, l.Number) {
			l.Form = token()
		}
	}
	for i := range s.ShippingRecords {
		r := &s.ShippingRecords[i]
		if r.DeliveryGroup == p.DeliveryGroup {
			r.DeliveryTotal = len(p.DeliveryLots)
		}
	}
	if before != digest(jsonRaw(p.DeliveryLots)) {
		p.touch()
	}
}
func refreshDeliverySummary(s *State, p *Project) {
	if len(p.DeliveryLots) == 0 {
		return
	}
	before := digest(jsonRaw(p.Shipping))
	all := p.Setting.Completed != nil
	var last Stage
	var pending *Stage
	for _, l := range p.DeliveryLots {
		st := l.Inspection
		if st.Pending && pending == nil {
			v := st
			pending = &v
		}
		if st.Completed != nil && (last.Completed == nil || st.Completed.After(*last.Completed)) {
			last = st
		}
		r := deliveryRecord(s, *p, l)
		if st.Completed == nil || r == nil || r.ArchivedAt == nil {
			all = false
		}
	}
	p.Shipping = last
	p.Shipping.Completed = nil
	p.Shipping.Pending = false
	if pending != nil {
		p.Shipping = *pending
		p.Shipping.Completed = nil
	}
	if all {
		p.Shipping = last
	}
	if before != digest(jsonRaw(p.Shipping)) {
		p.touch()
	}
}
func refreshAllDeliveries(s *State) {
	for i := range s.Projects {
		refreshDeliverySummary(s, &s.Projects[i])
	}
}
func (a *App) requestDelivery(id int64, lot, worker, form string) error {
	if !validText(worker, 40) {
		return errors.New("작업자 이름을 40자 이내로 입력하세요")
	}
	return a.transaction(func(s *State) error {
		p := s.project(id)
		if p == nil || p.Hidden || p.LegacyCompleted || p.Shipping.Completed != nil {
			return errors.New("신청할 수주를 확인해 주세요")
		}
		l := findDelivery(p, lot)
		if l == nil || p.Setting.Completed == nil || l.Inspection.Pending || l.Inspection.Completed != nil || !equal(form, l.Form) {
			return errors.New("차수와 세팅검사·신청 상태를 다시 확인해 주세요")
		}
		now := time.Now()
		eid := s.next()
		l.Inspection = Stage{Worker: worker, Pending: true, Requested: &now, EventID: eid}
		l.Attempted = true
		l.Form = token()
		s.Events = append(s.Events, Event{ID: eid, ProjectID: id, Revision: 1, Order: p.Number, Project: p.Name, Worker: worker, Inspection: "shipping", InspectionName: fmt.Sprintf("%d차 출하검사", l.Number), DeliveryID: l.ID, DeliveryNumber: l.Number, Pending: true, Unread: true, Active: true, RequestedAt: &now, Completed: &now})
		p.touch()
		return nil
	})
}
func (a *App) approveDelivery(id int64, request int64, actors ...InspectionInspector) error {
	var copies []Photo
	err := a.transaction(func(s *State) error {
		p := s.project(id)
		if p == nil || p.Hidden || p.Setting.Completed == nil {
			return errors.New("검사 가능한 수주를 확인하세요")
		}
		l := deliveryForRequest(p, request)
		if l == nil || !l.Inspection.Pending || l.Inspection.Completed != nil {
			return errors.New("이미 변경되었거나 완료된 차수 검사입니다")
		}
		now := time.Now()
		st := &l.Inspection
		st.Pending = false
		st.Completed = &now
		if len(actors) > 0 {
			actor := actors[0]
			actor.Source = "auto"
			actor.RecordedAt = now
			st.Inspector = &actor
			s.InspectionAudits = append(s.InspectionAudits, InspectionAudit{ID: token(), Action: "auto", Targets: []InspectionAuditTarget{{Order: p.Number, Project: p.Name, Kind: fmt.Sprintf("shipping/%d", l.Number), Completed: &now}}, InspectorName: actor.Name, ActorAccountID: actor.AccountID, ActorUsername: actor.Username, ActorName: actor.Name, When: now, Keys: []string{inspectionKey(p.ID, "shipping", st.EventID, &now)}})
		}
		for i := range s.Events {
			e := &s.Events[i]
			if e.ID == request {
				e.Pending = false
				e.Unread = false
				e.Completed = &now
				e.Inspector = st.Inspector
				e.Revision++
			}
		}
		snapshot := *p
		snapshot.Shipping = *st
		snapshot.DeliveryLots = nil
		snapshot.ShippingDueDate = l.Date
		snapshot.ShippingDueDates = nil
		var err error
		copies, err = a.captureShippingRecord(s, snapshot)
		if err != nil {
			return err
		}
		for i := range s.ShippingRecords {
			r := &s.ShippingRecords[i]
			if r.Project.ID == id && r.Project.Shipping.EventID == request {
				r.DeliveryGroup = p.DeliveryGroup
				r.DeliveryID = l.ID
				r.DeliveryNumber = l.Number
				r.DeliveryTotal = len(p.DeliveryLots)
			}
		}
		p.touch()
		return nil
	})
	if err != nil {
		a.pruneCommunityPhotos(copies)
	}
	return err
}
func (a *App) rejectDelivery(id int64, request int64) error {
	return a.transaction(func(s *State) error {
		p := s.project(id)
		if p == nil || p.Hidden {
			return errors.New("수주를 찾을 수 없습니다")
		}
		l := deliveryForRequest(p, request)
		if l == nil || !l.Inspection.Pending || l.Inspection.Completed != nil {
			return errors.New("진행 중인 차수 신청이 아닙니다")
		}
		now := time.Now()
		for i := range s.Events {
			e := &s.Events[i]
			if e.ID == request {
				e.Pending = false
				e.Unread = false
				e.Rejected = true
				e.RejectedAt = &now
				e.Completed = nil
				e.Active = false
				e.InspectionName = fmt.Sprintf("%d차 출하검사 반려", l.Number)
				e.Revision++
			}
		}
		l.Inspection = Stage{}
		l.Form = token()
		p.touch()
		return nil
	})
}
func splitRecordMatches(s *State, r ShippingRecord) bool {
	p := s.project(r.Project.ID)
	if p == nil || p.DeliveryGroup != r.DeliveryGroup || r.DeliveryID == "" {
		return false
	}
	l := findDelivery(p, r.DeliveryID)
	return l != nil && l.Inspection.Completed != nil && r.Project.Shipping.Completed != nil && l.Inspection.EventID == r.Project.Shipping.EventID && l.Inspection.Completed.Equal(*r.Project.Shipping.Completed)
}
func (r ShippingRecord) DeliveryTitle() string {
	if r.DeliveryNumber > 0 {
		return fmt.Sprintf("%d차 출고", r.DeliveryNumber)
	}
	return ""
}
func eventInspectionLabel(e Event) string {
	if e.DeliveryNumber > 0 {
		return fmt.Sprintf("%d차 출하검사", e.DeliveryNumber)
	}
	return stageLabel(e.Inspection)
}

func buildSplitShippingExport(ctx context.Context, dir string, records []ShippingRecord, now time.Time, read func(Photo) ([]byte, error), progress func(int), limits shippingExportLimits) (int64, int, int, error) {
	if len(records) < 2 || len(records) > limits.Records || records[0].DeliveryGroup == "" {
		return 0, 0, 0, errors.New("분할 출고 엑셀 범위를 확인해 주세요")
	}
	b := newExportBook("목차")
	defer b.File.Close()
	entries := []exportEntry{}
	warnings := []exportWarning{}
	photoBytes := 0
	sort.SliceStable(records, func(i, j int) bool { return records[i].DeliveryNumber < records[j].DeliveryNumber })
	for i, r := range records {
		if r.DeliveryGroup != records[0].DeliveryGroup || r.ArchivedAt == nil {
			return 0, 0, 0, errors.New("동일 수주의 보관된 차수만 내보낼 수 있습니다")
		}
		if err := ctx.Err(); err != nil {
			return 0, 0, 0, err
		}
		photos, w := prepareExportPhotos(ctx, r, read)
		for _, ph := range photos {
			photoBytes += len(ph.Data)
		}
		if photoBytes > limits.PhotoBytes {
			return 0, 0, 0, errors.New("한 수주 엑셀의 사진 용량 한도를 넘었습니다")
		}
		entry := exportEntry{Record: r, Sheet: exportSheetName(i+1, r), Warnings: len(w)}
		b.detail(entry, photos)
		entries = append(entries, entry)
		warnings = append(warnings, w...)
		if progress != nil {
			progress(i + 1)
		}
	}
	b.index("목차", "분할 출고 기록 · "+records[0].Project.Name, entries, false, now)
	b.warnings(warnings)
	b.File.SetActiveSheet(0)
	if err := b.save(filepath.Join(dir, "export.xlsx")); err != nil {
		return 0, 0, 0, err
	}
	st, err := os.Stat(filepath.Join(dir, "export.xlsx"))
	if err != nil {
		return 0, 0, 0, err
	}
	return st.Size(), 1, len(warnings), nil
}

func (v ProjectView) DeliveryCalendarJSON() string {
	out := []map[string]any{}
	for _, l := range v.LotViews {
		label := "출하검사 대기"
		if l.Inspection.Pending {
			label = "출하검사 확인 대기"
		}
		if l.Inspection.Completed != nil {
			label = "검사 완료 · 사진 대기"
		}
		if l.PhotoDone {
			label = "검사·사진 완료"
		}
		out = append(out, map[string]any{"date": l.Date, "number": l.Number, "done": l.PhotoDone, "label": label, "worker": l.Inspection.Worker})
	}
	return string(jsonRaw(out))
}

func (p Project) DeliveryStageLabel() string {
	if p.Hidden {
		return "숨김"
	}
	if p.Shipping.Completed != nil {
		return "전체 완료"
	}
	if p.Setting.Completed == nil {
		if p.Setting.Pending {
			return "세팅검사 확인 대기"
		}
		return "세팅검사 대기"
	}
	all := true
	for _, l := range p.DeliveryLots {
		if l.Inspection.Pending {
			return "차수별 출하검사 확인 대기"
		}
		all = all && l.Inspection.Completed != nil
	}
	if all {
		return "출하사진 등록 대기"
	}
	return "차수별 출하검사 대기"
}
