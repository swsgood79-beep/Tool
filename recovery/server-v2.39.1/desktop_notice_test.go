package main

import (
	"strings"
	"testing"
	"time"
)

func TestDesktopNoticeStructuredFieldsAndTypes(t *testing.T) {
	now := time.Date(2026, 11, 13, 6, 5, 0, 0, time.UTC)
	for _, tt := range []struct {
		name, kind, title, label string
		pending, note, batch     bool
	}{
		{"setting", "setting", "세팅검사 신청", "신청자", true, false, false},
		{"shipping", "shipping", "출하검사 신청", "신청자", true, false, false},
		{"combined", "setting", "세팅·출하 통합검사 신청", "신청자", true, false, true},
		{"completed", "shipping", "출하검사 완료", "작업자", false, false, false},
		{"note", "", "비고 메시지", "작성자", false, true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := Event{ID: 5, ProjectID: 3, Order: "ORDER-005", Project: "줄바꿈\n프로젝트", Worker: "시험 이름", Inspection: tt.kind, Pending: tt.pending, IsNote: tt.note, Note: "메모\n두번째 줄", RequestedAt: &now, Completed: &now}
			if tt.batch {
				e.BatchID = e.ID
			}
			n := desktopNotice(e)
			if n.Title != tt.title || n.workerLabel() != tt.label || n.Order != e.Order || n.Project != e.Project || n.Worker != e.Worker || n.Note != e.Note || n.When != "11.13 15:05" {
				t.Fatal("structured fields lost", n)
			}
			if tt.pending && !strings.Contains(n.Target, "/projects/3/inspection?request=5") {
				t.Fatal("request target lost")
			}
			if !tt.pending && n.Target != "/records/5/edit" {
				t.Fatal("record target lost")
			}
			if popupHeight(n) != 330 {
				t.Fatal("memo layout too short")
			}
			n.Note = ""
			if popupHeight(n) != 278 {
				t.Fatal("compact layout wrong")
			}
		})
	}
}
func TestDesktopSnapshotUsesLatestProjectNamesWithoutMarkingRead(t *testing.T) {
	a := fixture(t)
	p := a.state.Projects[0]
	a.state.Events = append(a.state.Events, Event{ID: 2, ProjectID: p.ID, Order: "old", Project: "old", Worker: "신청자", Inspection: "setting", Pending: true, Unread: true})
	s := a.desktopSnapshot()
	if len(s.Events) != 1 || s.Events[0].Order != p.Number || s.Events[0].Project != p.Name || !a.state.Events[0].Unread {
		t.Fatal("snapshot invalid", s)
	}
	if !strings.Contains(s.History, "수주번호") || !strings.Contains(s.History, "프로젝트") || !strings.Contains(s.History, "신청자") {
		t.Fatal("unlabeled history")
	}
}

func TestCompletionRecordsDoNotBecomePCAlerts(t *testing.T) {
	a := fixture(t)
	a.state.Events = []Event{{ID: 2, ProjectID: 1, Inspection: "setting", Pending: true, Unread: true}, {ID: 3, ProjectID: 1, Inspection: "shipping", Unread: true}, {ID: 4, ProjectID: 1, IsNote: true, Unread: true}}
	s := a.desktopSnapshot()
	if a.view("test")["UnreadCount"].(int) != 2 {
		t.Fatal("web badge includes completed records")
	}
	if s.Unread != 2 || len(s.Events) != 2 {
		t.Fatal("completion became notification", s)
	}
	for _, e := range s.Events {
		if e.ID == 3 {
			t.Fatal("completion included")
		}
	}
	if !strings.Contains(s.History, "출하검사 完了") && !strings.Contains(s.History, "출하검사 완료") {
		t.Fatal("completion record disappeared")
	}
}
