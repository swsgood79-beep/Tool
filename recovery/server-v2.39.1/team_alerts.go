package main

import (
	"fmt"
	"net/http"
	"time"
)

type TeamNotice struct {
	ID                                                int64
	JobID, RunID, Order, Project, Part, Phase, Worker string
	Quantity, Completed, Required                     int
	At                                                time.Time
	Recipients, ReadBy                                []string
}

func containsAccount(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
func machiningCompletionNotice(s *State, j *MachiningJob, phase string, r MachiningRun, at time.Time) {
	if phase != "milling2" && phase != "milling" {
		return
	}
	// Called in the same durable transaction as completion; retries cannot create
	// another notification for a run that is already completed.
	rs := j.Cutting
	label := "절단"
	if phase == "milling1" {
		rs = j.MillingFirst
		label = "가공 1차"
	} else if phase == "milling2" || phase == "milling" {
		rs = j.Milling
		label = "가공 2차"
	}
	recipients := []string{""}
	for _, u := range s.AdminAccounts {
		ss := Session{AccountID: u.ID}
		if u.Enabled && managerAccount(s, ss) && adminMenus(s, ss).Machining {
			recipients = append(recipients, u.ID)
		}
	}
	id := s.next()
	s.TeamNotices = append(s.TeamNotices, TeamNotice{ID: id, JobID: j.ID, RunID: r.ID, Order: j.Order, Project: j.Project, Part: j.Part, Phase: label, Worker: r.CompletedBy.Name, Quantity: r.Quantity, Completed: machiningDone(rs), Required: j.Required, At: at, Recipients: recipients})
}
func teamNoticeVisible(s *State, n TeamNotice, session Session) bool {
	return managerAccount(s, session) && adminMenus(s, session).Machining && containsAccount(n.Recipients, session.AccountID)
}
func teamNoticeDesktop(n TeamNotice) desktopEvent {
	return desktopEvent{ID: n.ID, Title: n.Phase + " 완료", Order: n.Order, Project: n.Project, Worker: n.Worker, Note: fmt.Sprintf("%s · 이번 완료 %d개 · 누적 %d/%d개 · 남음 %d개", n.Part, n.Quantity, n.Completed, n.Required, max(0, n.Required-n.Completed)), When: n.At.In(kst).Format("01.02 15:04"), Target: "/team-menu?notice=" + fmtID(n.ID)}
}
func (a *App) teamAlertsHTTP(w http.ResponseWriter, r *http.Request, s Session) bool {
	if r.URL.Path != "/team-alerts" {
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	if !managerAccount(&a.state, s) {
		http.Error(w, "관리자 계정이 필요합니다", 403)
		return true
	}
	if r.Method == "POST" {
		id := number(r.FormValue("id"))
		found := false
		for _, n := range a.state.TeamNotices {
			if n.ID == id && teamNoticeVisible(&a.state, n, s) {
				found = true
			}
		}
		if !found {
			http.NotFound(w, r)
			return true
		}
		if e := a.commitAnnouncement(func(st *State) error {
			for i := range st.TeamNotices {
				n := &st.TeamNotices[i]
				if n.ID == id && !containsAccount(n.ReadBy, s.AccountID) {
					n.ReadBy = append(n.ReadBy, s.AccountID)
				}
			}
			return nil
		}); e != nil {
			http.Error(w, "알림 확인 저장 실패", 500)
			return true
		}
	} else if r.Method != "GET" {
		http.Error(w, "지원하지 않는 요청", 405)
		return true
	}
	list := []TeamNotice{}
	for _, n := range a.state.TeamNotices {
		if teamNoticeVisible(&a.state, n, s) && !containsAccount(n.ReadBy, s.AccountID) {
			list = append(list, n)
			if len(list) >= 100 {
				break
			}
		}
	}
	jsonReply(w, map[string]any{"notices": list, "menus": adminMenus(&a.state, s), "native_desktop": a.teamNativeDesktop && s.AccountID == "" && localRequest(r)})
	return true
}
