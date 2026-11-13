package main

import (
	"errors"
	"time"
)

func (a *App) rejectInspection(id int64, kind string, request int64) error {
	return a.transaction(func(s *State) error {
		p := s.project(id)
		if p == nil || p.Hidden || p.LegacyCompleted || p.Shipping.Completed != nil {
			return errors.New("반려할 진행 중인 수주를 찾을 수 없습니다.")
		}
		kinds := []string{kind}
		if kind == "combined" {
			if !p.CombinedPending || !p.Setting.Pending || !p.Shipping.Pending {
				return errors.New("반려할 통합검사 신청이 없습니다.")
			}
			kinds = []string{"setting", "shipping"}
		} else if kind != "setting" && kind != "shipping" {
			return errors.New("검사 구분 오류")
		} else if p.CombinedPending {
			return errors.New("통합검사 신청은 통합 반려 버튼을 사용하세요.")
		}
		if kind == "shipping" && p.Setting.Completed == nil {
			return errors.New("세팅검사 상태를 먼저 확인하세요.")
		}
		if kind == "setting" && p.Shipping.Pending {
			return errors.New("출하 신청이 함께 있어 현재 검사 상태를 확인해야 합니다.")
		}
		stage := p.Setting
		if kind == "shipping" {
			stage = p.Shipping
		}
		if request == 0 || stage.EventID != request || !stage.Pending || stage.Completed != nil {
			return errors.New("이전 신청이거나 이미 처리된 검사입니다. 새로고침해 주세요.")
		}
		ids := map[int64]bool{}
		for _, k := range kinds {
			st := p.Setting
			if k == "shipping" {
				st = p.Shipping
			}
			if !st.Pending || st.Completed != nil || st.EventID == 0 {
				return errors.New("진행 중인 검사 신청이 없습니다.")
			}
			found := false
			for _, e := range s.Events {
				if e.ID == st.EventID && e.ProjectID == id && e.Inspection == k && e.Pending && !e.Rejected {
					found = true
					break
				}
			}
			if !found {
				return errors.New("검사 신청 기록이 현재 상태와 다릅니다.")
			}
			ids[st.EventID] = true
		}
		now := time.Now()
		for i := range s.Events {
			e := &s.Events[i]
			if ids[e.ID] {
				e.Pending = false
				e.Unread = false
				e.Rejected = true
				e.RejectedAt = &now
				e.Completed = nil
				e.Active = false
				e.InspectionName = stageLabel(e.Inspection) + " 반려"
				e.Revision++
			}
		}
		if kind == "setting" || kind == "combined" {
			p.Setting = Stage{}
			p.Shipping = Stage{}
		} else {
			p.Shipping = Stage{}
		}
		p.CombinedPending = false
		p.RejectedInspection = kind
		p.rotate()
		p.touch()
		return nil
	})
}
