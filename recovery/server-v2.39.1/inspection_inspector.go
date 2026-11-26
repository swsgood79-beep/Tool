package main

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

type InspectionInspector struct {
	AccountID, Username, Name                               string
	Source                                                  string
	RecordedAt                                              time.Time
	RecordedByAccountID, RecordedByUsername, RecordedByName string
}
type InspectionAuditTarget struct {
	Order, Project, Kind string
	Completed            *time.Time
}

type InspectionAudit struct {
	Targets                                                             []InspectionAuditTarget `json:",omitempty"`
	ID, Action, InspectorName, ActorAccountID, ActorUsername, ActorName string
	When                                                                time.Time
	Keys                                                                []string
}

func ownerDisplayName(s *State) string {
	if strings.TrimSpace(s.OwnerName) != "" {
		return s.OwnerName
	}
	return "관리자"
}
func inspectionName(i *InspectionInspector) string {
	if i == nil || strings.TrimSpace(i.Name) == "" {
		return "검사자 미기록"
	}
	return i.Name
}
func inspectionSource(i *InspectionInspector) string {
	if i == nil {
		return "미기록"
	}
	if i.Source == "backfill" {
		return "과거 기록 보완"
	}
	return "로그인 계정 자동 기록"
}
func inspectionActor(s *State, session Session) (InspectionInspector, error) {
	if !validAccountSession(s, session) || purchasingSession(s, session) || machiningSession(s, session) {
		return InspectionInspector{}, errors.New("검사 관리자 로그인이 필요합니다")
	}
	actor := InspectionInspector{AccountID: session.AccountID, Username: ownerUsername, Name: ownerDisplayName(s)}
	if u := adminAccount(s, session.AccountID); u != nil {
		actor.Username, actor.Name = u.Username, u.Name
	}
	return actor, nil
}
func inspectionKey(project int64, kind string, event int64, completed *time.Time) string {
	if event > 0 {
		return fmt.Sprintf("%d/%s/event/%d", project, kind, event)
	}
	at := ""
	if completed != nil {
		at = completed.UTC().Format(time.RFC3339Nano)
	}
	return fmt.Sprintf("%d/%s/time/%s", project, kind, at)
}

type inspectorSlot struct {
	Key, Order, Project, Kind string
	Completed                 *time.Time
	Value                     **InspectionInspector
}

func inspectorSlots(s *State) []inspectorSlot {
	var out []inspectorSlot
	add := func(p *Project) {
		for _, kind := range []string{"setting", "shipping"} {
			stage := &p.Setting
			if kind == "shipping" {
				stage = &p.Shipping
			}
			if stage.Completed != nil && !stage.Pending {
				out = append(out, inspectorSlot{inspectionKey(p.ID, kind, stage.EventID, stage.Completed), p.Number, p.Name, kind, stage.Completed, &stage.Inspector})
			}
		}
	}
	for i := range s.Projects {
		add(&s.Projects[i])
	}
	for i := range s.ShippingRecords {
		add(&s.ShippingRecords[i].Project)
	}
	for i := range s.Events {
		e := &s.Events[i]
		if e.IsNote || e.Pending || e.Rejected || e.Completed == nil || (e.Inspection != "setting" && e.Inspection != "shipping") {
			continue
		}
		out = append(out, inspectorSlot{inspectionKey(e.ProjectID, e.Inspection, e.ID, e.Completed), e.Order, e.Project, e.Inspection, e.Completed, &e.Inspector})
	}
	return out
}

type InspectorMissing struct {
	Key, Order, Project, Kind, Label string
	Completed                        *time.Time
}

func inspectorMissing(s *State, q string) []InspectorMissing {
	rows := map[string]InspectorMissing{}
	known := map[string]bool{}
	q = strings.ToLower(strings.TrimSpace(q))
	for _, slot := range inspectorSlots(s) {
		if *slot.Value != nil && strings.TrimSpace((*slot.Value).Name) != "" {
			known[slot.Key] = true
		}
		if q != "" && !strings.Contains(strings.ToLower(slot.Order+" "+slot.Project), q) {
			continue
		}
		rows[slot.Key] = InspectorMissing{slot.Key, slot.Order, slot.Project, slot.Kind, stageLabel(slot.Kind), slot.Completed}
	}
	out := []InspectorMissing{}
	for key, row := range rows {
		if !known[key] {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind == "setting"
		}
		return out[i].Key < out[j].Key
	})
	return out
}
func inspectorRevision(rows []InspectorMissing) string { return digest(jsonRaw(rows)) }
func (a *App) backfillInspectors(r *http.Request, actor InspectionInspector) error {
	name := strings.TrimSpace(r.FormValue("inspector_name"))
	if !validText(name, 40) {
		return errors.New("검사자 이름을 40자 이내로 입력하세요")
	}
	if r.FormValue("confirm") != "1" {
		return errors.New("실제 검사자를 확인했다는 항목에 체크해 주세요")
	}
	return a.transaction(func(s *State) error {
		rows := inspectorMissing(s, r.FormValue("q"))
		if !equal(r.FormValue("revision"), inspectorRevision(rows)) {
			return errors.New("검사 기록이 변경되었습니다. 목록을 새로 확인한 뒤 보완해 주세요")
		}
		allowed := map[string]bool{}
		for _, row := range rows {
			allowed[row.Key] = true
		}
		keys := map[string]bool{}
		switch r.FormValue("scope") {
		case "all":
			for key := range allowed {
				keys[key] = true
			}
		case "selected":
			for _, key := range r.Form["record"] {
				if !allowed[key] {
					return errors.New("보완할 수 없거나 이미 검사자가 기록된 항목입니다")
				}
				keys[key] = true
			}
		default:
			return errors.New("보완 범위를 선택해 주세요")
		}
		if len(keys) == 0 {
			return errors.New("보완할 항목을 선택해 주세요")
		}
		now := time.Now()
		value := InspectionInspector{Name: name, Source: "backfill", RecordedAt: now, RecordedByAccountID: actor.AccountID, RecordedByUsername: actor.Username, RecordedByName: actor.Name}
		if name == actor.Name {
			value.AccountID, value.Username = actor.AccountID, actor.Username
		}
		targetsByKey := map[string]InspectionAuditTarget{}
		for _, row := range rows {
			targetsByKey[row.Key] = InspectionAuditTarget{row.Order, row.Project, row.Kind, row.Completed}
		}
		auditKeys := make([]string, 0, len(keys))
		for key := range keys {
			auditKeys = append(auditKeys, key)
		}
		sort.Strings(auditKeys)
		targets := []InspectionAuditTarget{}
		for _, key := range auditKeys {
			targets = append(targets, targetsByKey[key])
		}
		for _, slot := range inspectorSlots(s) {
			if keys[slot.Key] && (*slot.Value == nil || strings.TrimSpace((*slot.Value).Name) == "") {
				copy := value
				*slot.Value = &copy
			}
		}
		s.InspectionAudits = append(s.InspectionAudits, InspectionAudit{ID: token(), Action: "backfill", Targets: targets, InspectorName: name, ActorAccountID: actor.AccountID, ActorUsername: actor.Username, ActorName: actor.Name, When: now, Keys: auditKeys})
		return nil
	})
}
func (a *App) inspectorHTTP(w http.ResponseWriter, r *http.Request, session Session) bool {
	if r.URL.Path != "/inspection-inspectors" {
		return false
	}
	if session.AccountID != "" {
		http.Error(w, "대표 관리자만 과거 검사자를 보완할 수 있습니다", 403)
		return true
	}
	actor, e := inspectionActor(&a.state, session)
	if e != nil {
		http.Error(w, e.Error(), 403)
		return true
	}
	problem := ""
	saved := r.URL.Query().Get("saved") == "1"
	if r.Method == http.MethodPost {
		if e = a.backfillInspectors(r, actor); e == nil {
			http.Redirect(w, r, "/inspection-inspectors?saved=1", http.StatusSeeOther)
			return true
		}
		problem = e.Error()
	} else if r.Method != http.MethodGet {
		http.Error(w, "지원하지 않는 요청", 405)
		return true
	}
	q := r.FormValue("q")
	rows := inspectorMissing(&a.state, q)
	page := int(number(r.URL.Query().Get("page")))
	if page < 1 {
		page = 1
	}
	pages := max(1, (len(rows)+49)/50)
	if page > pages {
		page = pages
	}
	start := (page - 1) * 50
	v := a.view(session.CSRF)
	v["Rows"], v["Count"], v["Query"], v["Revision"], v["DefaultInspector"], v["Error"], v["Saved"] = rows[start:min(start+50, len(rows))], len(rows), q, inspectorRevision(rows), actor.Name, problem, saved
	v["Page"], v["Pages"], v["Previous"], v["Next"] = page, pages, page-1, page+1
	audits := a.state.InspectionAudits
	if len(audits) > 50 {
		audits = audits[len(audits)-50:]
	}
	recent := make([]InspectionAudit, 0, len(audits))
	for i := len(audits) - 1; i >= 0; i-- {
		recent = append(recent, audits[i])
	}
	v["Audits"] = recent
	a.render(w, "inspection-inspectors.html", v)
	return true
}

func inspectionActorByCSRF(s *State, sessions map[string]Session, csrf string) (InspectionInspector, error) {
	for _, session := range sessions {
		if equal(session.CSRF, csrf) {
			return inspectionActor(s, session)
		}
	}
	return InspectionInspector{}, errors.New("관리자 세션이 없습니다")
}
