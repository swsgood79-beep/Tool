package main

import (
	"net/url"
	"sort"
	"strings"
	"time"
)

type desktopEvent struct {
	Rejected                                       bool
	ID, ProjectID                                  int64
	Title, Body, Target                            string
	Pending                                        bool
	Inspection, Order, Project, Worker, Note, When string
	IsNote                                         bool
}
type desktopView struct {
	Connection, WorkerURL, ReportPath, History, Storage string
	Connected                                           bool
	Unread                                              int
	Events                                              []desktopEvent
}

func (a *App) desktopSnapshot() desktopView {
	a.mu.Lock()
	defer a.mu.Unlock()
	v := desktopView{Connection: a.integration.TunnelStatus, Storage: "운영 복사본에 저장 · 기존 원본 보존 · 최초 .bak 보존"}
	v.Connected = !a.testMode && a.integration.CancelTunnel != nil && a.integration.TunnelStatus == "외부 접속 연결됨" && a.state.Integration.Connection.PublicURL != ""
	if a.testMode {
		v.Connection = "격리 테스트 · 외부 접속/알림/백업 꺼짐"
		v.Storage = "테스트 전용 자료 · 기존 원본은 읽거나 쓰지 않음"
	}
	if v.Connection == "" {
		v.Connection = "인터넷 연결 대기"
	}
	if a.storageNotice != "" {
		v.Storage = a.storageNotice
	}
	if a.state.Integration.Connection.Provider == "pages" {
		v.WorkerURL = pagesOrigin + "/Tool/#k=" + a.state.Board
	} else if a.state.Integration.Connection.PublicURL != "" {
		v.WorkerURL = a.state.Integration.Connection.PublicURL + "/work/" + a.state.Board
	} else {
		v.WorkerURL = a.localURL + "/work/" + a.state.Board
	}
	v.ReportPath = a.dir
	if a.reportError != "" {
		v.Storage += " · 엑셀 저장 재시도 필요"
	}
	if a.state.PasswordHash != "" && !strings.HasPrefix(a.state.PasswordHash, "$2") {
		v.Storage += " · 관리자 로그인 후 비밀번호 호환 전환"
	}
	history := []string{}
	for i := len(a.state.Events) - 1; i >= 0; i-- {
		e := a.state.Events[i]
		if p := a.state.project(e.ProjectID); p != nil {
			e.Order = p.Number
			e.Project = p.Name
		}
		if e.Unread && notifyEvent(e) {
			v.Unread++
			if e.BatchID == 0 || e.BatchID == e.ID {
				v.Events = append(v.Events, desktopNotice(e))
			}
		}
		if len(history) < 100 {
			when := ""
			if e.Rejected && e.RejectedAt != nil {
				when = e.RejectedAt.In(kst).Format("01-02 15:04")
			} else if e.Completed != nil {
				when = e.Completed.In(kst).Format("01-02 15:04")
			}
			n := desktopNotice(e)
			line := "[" + n.Title + "]  " + when + "\r\n수주번호  " + n.Order + "\r\n프로젝트  " + n.Project + "\r\n" + n.workerLabel() + "  " + n.Worker
			if n.Note != "" {
				line += "\r\n비고  " + n.Note
			}
			history = append(history, line)
		}
	}
	sort.Slice(v.Events, func(i, j int) bool { return v.Events[i].ID < v.Events[j].ID })
	v.History = strings.Join(history, "\r\n\r\n")
	return v
}

// Inspection completion is a stored record, not a new alert. Notes retain their existing alerts.
func notifyEvent(e Event) bool { return !e.Rejected && (e.Pending || e.IsNote) }

// Structured PC fields come from the event, never from splitting a display string.
func desktopNotice(e Event) desktopEvent {
	kind := e.Inspection
	if e.BatchID != 0 && e.BatchID == e.ID {
		kind = "combined"
	}
	title := stageLabel(kind)
	if e.IsNote {
		title = "비고 메시지"
	} else if e.Rejected {
		title += " 반려"
	} else if e.Pending {
		title += " 신청"
	} else {
		title += " 완료"
	}
	target := "/records/" + fmtID(e.ID) + "/edit"
	if e.Pending {
		target = "/projects/" + fmtID(e.ProjectID) + "/inspection?request=" + fmtID(e.ID)
	}
	when := ""
	at := e.Completed
	if e.Rejected {
		at = e.RejectedAt
	} else if e.Pending && e.RequestedAt != nil {
		at = e.RequestedAt
	}
	if at != nil {
		when = at.In(kst).Format("01.02 15:04")
	}
	payload := notification(e)
	return desktopEvent{Rejected: e.Rejected, ID: e.ID, ProjectID: e.ProjectID, Title: title, Body: text(payload["body"]), Target: target, Pending: e.Pending, Inspection: kind, Order: e.Order, Project: e.Project, Worker: e.Worker, Note: e.Note, When: when, IsNote: e.IsNote}
}
func (e desktopEvent) workerLabel() string {
	if e.Pending || e.Rejected {
		return "신청자"
	}
	if e.IsNote {
		return "작성자"
	}
	return "작업자"
}
func popupHeight(e desktopEvent) int {
	if e.Note != "" {
		return 330
	}
	return 278
}

func (a *App) nativeAdminURL(next string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.state.PasswordHash == "" || !strings.HasPrefix(a.state.PasswordHash, "$2") {
		return a.localURL + "/login?next=" + url.QueryEscape(next)
	}
	ticket := token()
	a.integration.Handoffs[hashToken(ticket)] = Handoff{Next: safeNext(next), Expires: time.Now().Add(time.Minute)}
	return a.localURL + "/native-admin#ticket=" + ticket
}

// Timer behavior is separate from unread state: dismissing a popup never marks an event read.
type popupClock struct {
	Remaining time.Duration
	Last      time.Time
}

func (p *popupClock) tick(now time.Time, hover bool) bool {
	if !p.Last.IsZero() && !hover {
		delta := now.Sub(p.Last)
		if delta > 0 {
			p.Remaining -= delta
		}
	}
	p.Last = now
	return p.Remaining <= 0
}

const nativeEnterJS = `(()=>{'use strict';const t=new URLSearchParams(location.hash.slice(1)).get('ticket')||'';history.replaceState(null,'',location.pathname);const s=document.getElementById('native-result');if(!/^[A-Za-z0-9_-]{43}$/.test(t)){s.textContent='연결 시간이 지났습니다. PC 프로그램에서 다시 열어 주세요.';return}fetch('/mobile-admin/passkey/finish',{method:'POST',credentials:'same-origin',headers:{'Content-Type':'application/json'},body:JSON.stringify({token:t})}).then(async r=>{if(!r.ok)throw Error();const d=await r.json();if(d.ok!==true)throw Error();const u=new URL(d.next,location.origin);if(u.origin!==location.origin)throw Error();location.replace(u.pathname+u.search)}).catch(()=>{s.textContent='PC 프로그램에서 관리자 화면을 다시 열어 주세요.'})})();`
