package main

import (
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"
)

// The first date stays in the legacy scalar field for date-input compatibility.
// Additional dates are schedule metadata, never separate inspection cycles.
func (p Project) DueDates() []string {
	out := []string{}
	seen := map[string]bool{}
	for _, d := range append([]string{p.ShippingDueDate}, p.ShippingDueDates...) {
		t, e := time.Parse("2006-01-02", d)
		if e == nil && t.Format("2006-01-02") == d && !seen[d] {
			out = append(out, d)
			seen[d] = true
		}
	}
	sort.Strings(out)
	return out
}
func (p Project) ShippingDueLabel() string { return strings.Join(p.DueDates(), " · ") }
func (p Project) SplitShipment() bool      { return len(p.DueDates()) > 1 || len(p.DeliveryLots) > 1 }
func (p Project) ShippingDueJSON() string  { b, _ := json.Marshal(p.DueDates()); return string(b) }
func (p Project) DueOn(date string) bool {
	for _, d := range p.DueDates() {
		if d == date {
			return true
		}
	}
	return false
}
func setProjectDueDates(p *Project, dates []string) {
	p.ShippingDueDate = ""
	p.ShippingDueDates = nil
	if len(dates) > 0 {
		p.ShippingDueDate = dates[0]
	}
	if len(dates) > 1 {
		p.ShippingDueDates = append([]string{}, dates...)
	}
}

var splitScheduleFullDate = regexp.MustCompile(`\d{4}[-/.]\d{1,2}[-/.]\d{1,2}`)

func scheduleCellDates(value string) ([]string, error) {
	d, unknown, e := scheduleDate(value)
	if e == nil {
		if unknown {
			return nil, nil
		}
		return []string{d}, nil
	}
	// Accept explicit complete dates separated by commas/newlines/etc. Never
	// guess a year or expand a date range from ambiguous free text.
	matches := splitScheduleFullDate.FindAllString(value, -1)
	rest := splitScheduleFullDate.ReplaceAllString(value, "")
	if len(matches) < 2 || strings.Trim(rest, " \t\r\n,;/·、") != "" {
		return nil, errors.New("날짜 확인 필요")
	}
	out := []string{}
	seen := map[string]bool{}
	for _, s := range matches {
		d, _, e := scheduleDate(s)
		if e != nil {
			return nil, e
		}
		if !seen[d] {
			out = append(out, d)
			seen[d] = true
		}
	}
	sort.Strings(out)
	return out, nil
}
