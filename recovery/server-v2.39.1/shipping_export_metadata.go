package main

import (
	"regexp"
	"strings"
	"time"
)

// Customer names may contain spaces. The customer's naming convention uses a
// whitespace-delimited axis count, for example "비엔에스 4축 (선진행, 예가)".
var shippingExportAxisSuffix = regexp.MustCompile(`\s+[0-9]+\s*축(?:$|[\s(（,，])`)

func shippingExportClientName(record ShippingRecord) string {
	name := strings.TrimSpace(record.Project.Name)
	if loc := shippingExportAxisSuffix.FindStringIndex(name); loc != nil {
		if client := strings.TrimSpace(name[:loc[0]]); client != "" {
			return client
		}
	}
	return strings.TrimSpace(record.ClientName)
}

// Fill only absent archive fields from the exact same live order and inspection
// cycle. Never replace a recorded value or infer a customer from project text.
func shippingExportMetadata(s *State, record ShippingRecord) ShippingRecord {
	r := record
	filled := []string{}
	missing := []string{}
	live := s.project(r.Project.ID)
	same := live != nil && live.ID == r.Project.ID && strings.EqualFold(strings.TrimSpace(live.Number), strings.TrimSpace(r.Project.Number)) && r.Project.Shipping.EventID > 0 && live.Shipping.EventID == r.Project.Shipping.EventID && live.Shipping.Completed != nil && r.Project.Shipping.Completed != nil && live.Shipping.Completed.Equal(*r.Project.Shipping.Completed)
	if r.DeliveryID != "" {
		same = splitRecordMatches(s, r)
	}
	if strings.TrimSpace(r.ClientName) == "" && same {
		for _, c := range s.ClientNotes {
			if live.ClientID != "" && c.ID == live.ClientID {
				r.ClientName = c.Name
				if r.ClientName != "" {
					filled = append(filled, "고객사")
				}
				break
			}
		}
	}
	if strings.TrimSpace(r.Project.ShippingDueDate) == "" && same {
		if d, err := time.Parse("2006-01-02", live.ShippingDueDate); err == nil && d.Format("2006-01-02") == live.ShippingDueDate {
			r.Project.ShippingDueDate = live.ShippingDueDate
			r.Project.ShippingDueDates = append([]string(nil), live.ShippingDueDates...)
			filled = append(filled, "출고 예정일")
		}
	}
	if strings.TrimSpace(r.ClientName) == "" {
		missing = append(missing, "고객사")
	}
	if strings.TrimSpace(r.Project.ShippingDueDate) == "" {
		missing = append(missing, "출고 예정일")
	}
	notes := []string{}
	if len(filled) > 0 {
		notes = append(notes, "보관 당시 빈 값을 현재 동일 수주에서 보완: "+strings.Join(filled, ", "))
	}
	if len(missing) > 0 {
		notes = append(notes, "원자료 미등록: "+strings.Join(missing, ", "))
	}
	r.ExportSourceNote = strings.Join(notes, " · ")
	return r
}
