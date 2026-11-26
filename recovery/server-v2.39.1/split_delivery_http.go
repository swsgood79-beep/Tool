package main

import (
	"net/http"
	"strings"
)

func (a *App) deliveryHTTP(w http.ResponseWriter, r *http.Request) bool {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 0 || parts[0] != "delivery-request" {
		return false
	}
	if len(parts) != 4 || !equal(parts[1], a.state.Board) {
		http.NotFound(w, r)
		return true
	}
	if r.Method != "POST" {
		http.Error(w, "POST 요청이 필요합니다", 405)
		return true
	}
	if !equal(r.FormValue("csrf"), a.state.Board) {
		http.Error(w, "신청 화면을 다시 열어 주세요", 403)
		return true
	}
	err := a.requestDelivery(number(parts[2]), parts[3], strings.TrimSpace(r.FormValue("worker")), r.FormValue("submission"))
	if err != nil {
		http.Error(w, err.Error(), 409)
	} else {
		http.Redirect(w, r, "/work/"+a.state.Board, 303)
	}
	return true
}

type DeliveryArchiveLink struct {
	Number           int
	Date, ID, Status string
}

func splitSiblings(s *State, r ShippingRecord) []DeliveryArchiveLink {
	out := []DeliveryArchiveLink{}
	if p := s.project(r.Project.ID); p != nil && p.DeliveryGroup == r.DeliveryGroup {
		for _, l := range p.DeliveryLots {
			v := DeliveryArchiveLink{Number: l.Number, Date: l.Date, Status: "검사 신청 전"}
			if l.Inspection.Pending {
				v.Status = "검사 대기"
			}
			if l.Inspection.Completed != nil {
				v.Status = "검사 완료 · 사진 대기"
			}
			if rec := deliveryRecord(s, *p, l); rec != nil {
				v.ID = rec.ID
				if rec.ArchivedAt != nil {
					v.Status = "검사·사진 완료"
				}
			}
			out = append(out, v)
		}
		return out
	}
	for _, rec := range s.ShippingRecords {
		if rec.DeliveryGroup == r.DeliveryGroup {
			status := "검사 완료 · 사진 대기"
			if rec.ArchivedAt != nil {
				status = "검사·사진 완료"
			}
			out = append(out, DeliveryArchiveLink{rec.DeliveryNumber, rec.Project.ShippingDueDate, rec.ID, status})
		}
	}
	return out
}
func splitArchiveDone(s *State, group string) bool {
	if group == "" {
		return false
	}
	count, total := 0, 0
	for _, r := range s.ShippingRecords {
		if r.DeliveryGroup == group {
			count++
			if r.DeliveryTotal > total {
				total = r.DeliveryTotal
			}
			if r.ArchivedAt == nil {
				return false
			}
		}
	}
	return total > 0 && count >= total
}

// A single overview entry per finished order, with links to the individual lots.
func splitListRecords(s *State, pending bool) []ShippingRecord {
	out := []ShippingRecord{}
	seen := map[string]bool{}
	for _, r := range s.ShippingRecords {
		if r.DeliveryID == "" {
			out = append(out, r)
			continue
		}
		if pending {
			if r.ArchivedAt == nil {
				out = append(out, r)
			}
			continue
		}
		if !splitArchiveDone(s, r.DeliveryGroup) || seen[r.DeliveryGroup] {
			continue
		}
		var latest ShippingRecord
		for _, v := range s.ShippingRecords {
			if v.DeliveryGroup == r.DeliveryGroup && (latest.ArchivedAt == nil || v.ArchivedAt.After(*latest.ArchivedAt)) {
				latest = v
			}
		}
		seen[r.DeliveryGroup] = true
		out = append(out, latest)
	}
	return out
}
func splitUnfinishedRecords(s *State) []ShippingRecord {
	out := []ShippingRecord{}
	seen := map[string]bool{}
	for _, r := range s.ShippingRecords {
		if r.DeliveryGroup != "" && !seen[r.DeliveryGroup] && !splitArchiveDone(s, r.DeliveryGroup) {
			seen[r.DeliveryGroup] = true
			out = append(out, r)
		}
	}
	return out
}
