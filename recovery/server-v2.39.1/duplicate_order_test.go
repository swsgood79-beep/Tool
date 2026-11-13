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
