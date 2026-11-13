package main

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestDuplicateOrderShowsCompletedAndHiddenWithoutChangingData(t *testing.T) {
	for _, hidden := range []bool{false, true} {
		t.Run(map[bool]string{false: "completed", true: "hidden"}[hidden], func(t *testing.T) {
			a := fixture(t)
			now := time.Now()
			p := &a.state.Projects[0]
			p.Number = "test"
			p.Name = `기존 <시험> 수주`
			p.Hidden = hidden
			p.Shipping.Completed = &now
			cookie := adminCookie(a)
			r := call(a, "POST", "/projects", url.Values{"csrf": {a.sessions[hashToken(cookie.Value)].CSRF}, "number": {"TEST"}, "name": {"새 등록"}}, cookie)
			s := r.Body.String()
			if r.Code != 409 || !strings.Contains(s, "기존 수주 열기") || !strings.Contains(s, "기존 &lt;시험&gt; 수주") || !strings.Contains(s, "/projects/1/inspection") {
				t.Fatalf("missing duplicate details: %d %s", r.Code, s)
			}
			if len(a.state.Projects) != 1 || a.state.Projects[0].Name != `기존 <시험> 수주` {
				t.Fatal("existing order overwritten")
			}
			r = call(a, "GET", "/projects/find?number=TEST", nil, cookie)
			if r.Code != 200 || !strings.Contains(r.Body.String(), "test") {
				t.Fatal("lookup failed")
			}
			r = call(a, "GET", "/projects/find?number=absent", nil, cookie)
			if r.Code != 404 {
				t.Fatal("unknown found")
			}
		})
	}
}
func TestDuplicateLookupNeedsAdminLogin(t *testing.T) {
	a := fixture(t)
	r := call(a, "GET", "/projects/find?number=T-001", nil)
	if r.Code == 200 || strings.Contains(r.Body.String(), "복원 서버 자동시험") {
		t.Fatal("order exposed without login")
	}
}

func TestDeletedTestOrderNumberCanBeRegisteredAgain(t *testing.T) {
	a := fixture(t)
	if e := a.transaction(func(s *State) error {
		s.Projects[0].Number = "TEST"
		now := time.Now()
		s.Projects[0].Shipping.Completed = &now
		s.Events = append(s.Events, Event{ID: s.next(), ProjectID: s.Projects[0].ID, Order: "TEST", Project: "시험", Inspection: "shipping", Completed: &now})
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	cookie := adminCookie(a)
	csrf := a.sessions[hashToken(cookie.Value)].CSRF
	r := call(a, "POST", "/projects/1/delete", url.Values{"csrf": {csrf}, "confirm": {"1"}}, cookie)
	if r.Code != 303 {
		t.Fatal("delete failed", r.Body.String())
	}
	r = call(a, "POST", "/projects", url.Values{"csrf": {csrf}, "number": {"test"}, "name": {"새 시험 수주"}}, cookie)
	if r.Code != 303 || len(a.state.Projects) != 1 || a.state.Projects[0].Number != "test" || a.state.Projects[0].Shipping.Completed != nil {
		t.Fatal("number could not be reused", r.Code, r.Body.String())
	}
	if len(a.state.Events) != 1 {
		t.Fatal("old completion record lost")
	}
}
