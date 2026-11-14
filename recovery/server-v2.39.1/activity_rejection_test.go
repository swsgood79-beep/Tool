package main

import (
	"encoding/csv"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRejectedRecordsExcludedFromActivityAndCSVButRetainedForReapplication(t *testing.T) {
	a, id := pendingReview(t, "setting")
	if err := a.rejectInspection(1, "setting", id); err != nil {
		t.Fatal(err)
	}
	var completeID int64
	if err := a.transaction(func(s *State) error {
		p := Project{ID: s.next(), Number: "EXPORT-001", Name: "유지할 정상 완료 수주", Revision: 1, InfoRevision: 1}
		p.rotate()
		completeID = p.ID
		s.Projects = append(s.Projects, p)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	p := *a.state.project(completeID)
	if err := a.requestInspection(p.ID, "setting", "완료 세팅 작업자", p.SettingForm, p.CombinedForm); err != nil {
		t.Fatal(err)
	}
	if err := a.approve(p.ID, "setting", a.state.project(p.ID).Setting.EventID); err != nil {
		t.Fatal(err)
	}
	p = *a.state.project(completeID)
	if err := a.requestInspection(p.ID, "shipping", "완료 출하 작업자", p.ShippingForm, p.CombinedForm); err != nil {
		t.Fatal(err)
	}
	if err := a.approve(p.ID, "shipping", a.state.project(p.ID).Shipping.EventID); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := a.transaction(func(s *State) error {
		s.Events = append(s.Events, Event{ID: s.next(), ProjectID: completeID, IsNote: true, Worker: "비고 작업자", Note: "유지할 비고", RequestedAt: &now})
		for i := 0; i < 120; i++ {
			s.Events = append(s.Events, Event{ID: s.next(), ProjectID: 1, Rejected: true, RejectedAt: &now, Inspection: "setting", Worker: "숨길 반려 작업자"})
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	count := len(a.state.Events)
	before, _ := json.Marshal(a.state)
	events := a.view("test")["Events"].([]Event)
	if len(events) != 3 {
		t.Fatal("rejections consumed the 100-row history window", len(events))
	}
	for _, e := range events {
		if e.Rejected {
			t.Fatal("rejection in recent history")
		}
	}
	cookie := adminCookie(a)
	w := call(a, "GET", "/", nil, cookie)
	if w.Code != 200 || strings.Contains(w.Body.String(), "숨길 반려 작업자") || !strings.Contains(w.Body.String(), "유지할 비고") {
		t.Fatal("activity render mismatch")
	}
	if out := os.Getenv("ACTIVITY_HTML_OUT"); out != "" {
		if err := os.WriteFile(out, w.Body.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	csvResponse := httptest.NewRecorder()
	a.csv(csvResponse)
	rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(csvResponse.Body.String(), "\ufeff"))).ReadAll()
	if err != nil || len(rows) != 3 {
		t.Fatal("CSV must contain header and two approved inspections", err, len(rows))
	}
	for _, row := range rows[1:] {
		if row[0] != "EXPORT-001" || !strings.HasPrefix(row[3], "완료 ") {
			t.Fatal("rejection or note exported", row)
		}
	}
	after, _ := json.Marshal(a.state)
	if string(before) != string(after) || len(a.state.Events) != count {
		t.Fatal("view/export altered stored records")
	}
	restarted, err := newApp(a.dir)
	if err != nil || len(restarted.state.Events) != count || restarted.state.project(1).RejectedInspection != "setting" {
		t.Fatal("restart lost rejected state", err)
	}
	p = *restarted.state.project(1)
	if err := restarted.requestInspection(1, "setting", "재신청 작업자", p.SettingForm, p.CombinedForm); err != nil {
		t.Fatal("hidden rejection blocked reapplication", err)
	}
	if !restarted.state.project(1).Setting.Pending || len(restarted.view("test")["Requests"].([]Event)) != 1 {
		t.Fatal("new request missing")
	}
}
