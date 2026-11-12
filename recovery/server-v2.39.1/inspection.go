package main

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

func validText(s string, n int) bool {
	return strings.TrimSpace(s) != "" && utf8.RuneCountInString(s) <= n
}
func (a *App) requestInspection(id int64, kind, worker, submission, combined string) error {
	if !validText(worker, 40) {
		return errors.New("작업자 이름을 40자 이내로 입력하세요.")
	}
	return a.transaction(func(s *State) error {
		p := s.project(id)
		if p == nil || p.Hidden || p.LegacyCompleted || p.Shipping.Completed != nil {
			return errors.New("신청 가능한 수주가 아닙니다.")
		}
		if kind != "setting" && kind != "shipping" && kind != "combined" {
			return errors.New("검사 구분 오류")
		}
		if kind == "shipping" {
			if p.Setting.Completed == nil || p.Shipping.Pending || !equal(submission, p.ShippingForm) {
				return errors.New("세팅검사 완료 및 출하검사 신청 상태를 확인하세요.")
			}
		} else if p.Setting.Completed != nil || p.Setting.Pending || !equal(submission, p.SettingForm) {
			return errors.New("이미 신청되었거나 변경된 검사 화면입니다.")
		}
		if kind == "combined" && (p.Shipping.Pending || !equal(combined, p.CombinedForm)) {
			return errors.New("통합검사 신청 상태를 확인하세요.")
		}
		now := time.Now()
		kinds := []string{kind}
		if kind == "combined" {
			kinds = []string{"setting", "shipping"}
		}
		var batch int64
		for i, k := range kinds {
			eid := s.next()
			if kind == "combined" && i == 0 {
				batch = eid
			}
			stage := Stage{Worker: worker, Pending: true, Requested: &now, EventID: eid}
			if k == "setting" {
				p.Setting = stage
			} else {
				p.Shipping = stage
			}
			name := stageLabel(k)
			if batch == eid {
				name = stageLabel("combined")
			}
			s.Events = append(s.Events, Event{ID: eid, BatchID: batch, ProjectID: id, Revision: 1, Order: p.Number, Project: p.Name, Worker: worker, Inspection: k, InspectionName: name, Pending: true, Unread: i == 0, Active: true, RequestedAt: &now, Completed: &now})
		}
		p.CombinedPending = kind == "combined"
		p.touch()
		return nil
	})
}
func (a *App) approve(id int64, kind string, request int64) error {
	return a.transaction(func(s *State) error {
		p := s.project(id)
		if p == nil || p.Hidden || p.LegacyCompleted || p.Shipping.Completed != nil {
			return errors.New("검사 가능한 수주가 아닙니다.")
		}
		kinds := []string{kind}
		if kind == "combined" {
			if !p.CombinedPending || !p.Setting.Pending || !p.Shipping.Pending {
				return errors.New("통합검사 신청이 없습니다.")
			}
			kinds = []string{"setting", "shipping"}
		} else if kind != "setting" && kind != "shipping" {
			return errors.New("검사 구분 오류")
		}
		if kind == "shipping" && p.Setting.Completed == nil {
			return errors.New("세팅검사를 먼저 완료하세요.")
		}
		expected := p.Setting.EventID
		if kind == "shipping" {
			expected = p.Shipping.EventID
		}
		if request == 0 || request != expected {
			return errors.New("이전 신청이거나 이미 변경된 검사입니다.")
		}
		now := time.Now()
		for _, k := range kinds {
			st := &p.Setting
			if k == "shipping" {
				st = &p.Shipping
			}
			if !st.Pending || st.Completed != nil {
				return errors.New("진행 중인 검사 신청이 없습니다.")
			}
			st.Pending = false
			st.Completed = &now
			for i := range s.Events {
				e := &s.Events[i]
				if e.ID == st.EventID {
					e.Pending = false
					e.Completed = &now
					e.InspectionName = stageLabel(k)
					e.Revision++
				}
			}
		}
		// Partial approval splits the remaining shipping request without losing it.
		if p.CombinedPending {
			for i := range s.Events {
				e := &s.Events[i]
				if e.ID == p.Setting.EventID || e.ID == p.Shipping.EventID {
					e.BatchID = 0
					e.InspectionName = stageLabel(e.Inspection)
					e.Revision++
				}
			}
		}
		p.CombinedPending = false
		p.touch()
		return nil
	})
}
func (a *App) worker(w http.ResponseWriter, r *http.Request) bool {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		return false
	}
	kind := parts[0]
	public := false
	for _, v := range []string{"work", "status", "card", "complete", "combined", "note", "project-info", "project-photo", "markets"} {
		if kind == v {
			public = true
		}
	}
	if !public {
		return false
	}
	if !equal(parts[1], a.state.Board) {
		http.NotFound(w, r)
		return true
	}
	view := func() map[string]any {
		v := a.view(a.state.Board)
		ps := []ProjectView{}
		for _, p := range v["Projects"].([]ProjectView) {
			if !p.Hidden {
				ps = append(ps, p)
			}
		}
		v["Projects"] = ps
		return v
	}
	if r.Method == "GET" {
		switch kind {
		case "work":
			if len(parts) == 2 {
				v := view()
				v["EntryGate"] = r.URL.Query().Get("entry") == "gate"
				a.render(w, "work.html", v)
				return true
			}
		case "status":
			active := []int64{}
			stages := []any{}
			completed := []any{}
			shipments := []any{}
			for _, p := range a.state.Projects {
				if p.Hidden {
					continue
				}
				v := a.projectView(p)
				pending := []string{}
				if p.Setting.Pending {
					pending = append(pending, "setting")
				}
				if p.Shipping.Pending {
					pending = append(pending, "shipping")
				}
				item := map[string]any{"id": p.ID, "revision": p.Revision, "generation": p.Generation, "pending": pending}
				if v.Completed {
					completed = append(completed, item)
				} else {
					active = append(active, p.ID)
					stages = append(stages, item)
				}
				if v.CompletedToday {
					url := ""
					if v.HasSpecialInstructions {
						url = "/project-info/" + a.state.Board + "/" + fmtID(p.ID)
					}
					shipments = append(shipments, map[string]any{"project_id": p.ID, "number": p.Number, "name": p.Name, "info_url": url, "photo_count": len(v.Photos), "setting_worker": p.Setting.Worker, "shipping_worker": p.Shipping.Worker})
				}
			}
			jsonReply(w, map[string]any{"active": active, "stages": stages, "completed_due": completed, "shipments": shipments, "date": day(time.Now()), "schedule": "복구 서버 · 수동 납기일 기준"})
			return true
		case "card", "project-info", "project-photo":
			if len(parts) >= 3 {
				p := a.state.project(number(parts[2]))
				if p == nil {
					http.NotFound(w, r)
					return true
				}
				v := a.projectView(*p)
				if kind == "card" {
					a.render(w, "project-card", v)
				} else if kind == "project-info" {
					root := view()
					root["Projects"] = []ProjectView{v}
					a.render(w, "project-info.html", root)
				} else if len(parts) == 4 {
					a.photo(w, r, v.Photos, parts[3])
				} else {
					http.NotFound(w, r)
				}
				return true
			}
		case "markets":
			if a.state.Integration.Market.Enabled {
				jsonReply(w, a.integration.MarketResult)
			} else {
				jsonReply(w, map[string]any{"enabled": false, "quotes": []any{}, "fx": nil})
			}
			return true
		}
	} else if r.Method == "POST" && len(parts) == 2 {
		id := number(r.FormValue("project"))
		p := a.state.project(id)
		if p == nil {
			http.NotFound(w, r)
			return true
		}
		var err error
		if kind == "complete" || kind == "combined" {
			inspection := r.FormValue("inspection")
			if kind == "combined" {
				inspection = "combined"
			}
			err = a.requestInspection(id, inspection, strings.TrimSpace(r.FormValue("worker")), r.FormValue("submission"), r.FormValue("combined_submission"))
		} else if kind == "note" {
			worker, note := strings.TrimSpace(r.FormValue("worker")), strings.TrimSpace(r.FormValue("note"))
			err = a.transaction(func(s *State) error {
				p := s.project(id)
				if p.Hidden || p.Shipping.Completed != nil || !equal(r.FormValue("submission"), p.NoteForm) {
					return errors.New("비고 작성 화면을 다시 열어 주세요.")
				}
				if !validText(worker, 40) || !validText(note, 1000) {
					return errors.New("작업자 이름 또는 비고 내용을 확인하세요.")
				}
				now := time.Now()
				s.Events = append(s.Events, Event{ID: s.next(), ProjectID: id, Revision: 1, Order: p.Number, Project: p.Name, Worker: worker, Note: note, IsNote: true, InspectionName: "비고 메시지", Unread: true, Active: true, Completed: &now})
				p.NoteForm = token()
				p.touch()
				return nil
			})
		} else {
			http.Error(w, "지원하지 않는 요청", 405)
			return true
		}
		if err != nil {
			http.Error(w, err.Error(), 409)
			return true
		}
		http.Redirect(w, r, "/work/"+a.state.Board, 303)
		return true
	}
	http.Error(w, "지원하지 않는 요청", 405)
	return true
}
