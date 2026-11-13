package main

import (
	"net/url"
	"sort"
	"strings"
	"time"
)

type desktopEvent struct {
	ID, ProjectID       int64
	Title, Body, Target string
	Pending             bool
}
type desktopView struct {
	Connection, WorkerURL, ReportPath, History, Storage string
	Unread                                              int
	Events                                              []desktopEvent
}

func (a *App) desktopSnapshot() desktopView {
	a.mu.Lock()
	defer a.mu.Unlock()
	v := desktopView{Connection: a.integration.TunnelStatus, Storage: "운영 복사본에 저장 · 기존 원본 보존 · 최초 .bak 보존"}
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
		if e.Unread {
			v.Unread++
			if e.BatchID == 0 || e.BatchID == e.ID {
				payload := notification(e)
				target := "/records/" + fmtID(e.ID) + "/edit"
				if e.Pending {
					target = "/projects/" + fmtID(e.ProjectID) + "/inspection?request=" + fmtID(e.ID)
				}
				v.Events = append(v.Events, desktopEvent{e.ID, e.ProjectID, text(payload["title"]), text(payload["body"]), target, e.Pending})
			}
		}
		if len(history) < 100 {
			when := ""
			if e.Completed != nil {
				when = e.Completed.In(kst).Format("01-02 15:04")
			}
			label := stageLabel(e.Inspection)
			if e.Pending {
				label += " 신청"
			}
			if e.IsNote {
				label = "비고"
			}
			line := when + "  " + label + " · " + e.Worker + "\r\n" + e.Order + " · " + e.Project
			if e.Note != "" {
				line += "\r\n" + e.Note
			}
			history = append(history, line)
		}
	}
	sort.Slice(v.Events, func(i, j int) bool { return v.Events[i].ID < v.Events[j].ID })
	v.History = strings.Join(history, "\r\n\r\n")
	return v
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
