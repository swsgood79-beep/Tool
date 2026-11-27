package main

import (
	"context"
	"io"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestObservationPermissionDiscoveryDoesNotStartTrace(t *testing.T) {
	q := regexp.MustCompile(`(?s)\$sql=@'\s*(.*?)\s*'@`).FindStringSubmatch(erpObservationAccessScript)
	if len(q) != 2 {
		t.Fatal("missing query")
	}
	if !strings.HasPrefix(q[1], "SELECT") || regexp.MustCompile(`(?i)\b(CREATE|ALTER|DROP|GRANT|REVOKE|INSERT|UPDATE|DELETE|EXEC)\b`).MatchString(regexp.MustCompile(`'[^']*'`).ReplaceAllString(q[1], "")) {
		t.Fatal("permission query mutates server")
	}
	cmd := erpDBPowerShellCommand(context.WithValue(context.Background(), erpDBObserveKey{}, true), "")
	b, e := io.ReadAll(cmd.Stdin)
	if e != nil {
		t.Fatal(e)
	}
	s := string(b)
	if !strings.Contains(s, "$ObservationAccess=$true") || strings.Contains(s, "# OBSERVATION_ACCESS_SCRIPT") {
		t.Fatal("transport")
	}
}
func TestObservationPermissionScopeAndResult(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	worker, _ := machiningWorker(t, a, "observer")
	csrf := a.sessions[hashToken(owner.Value)].CSRF
	f := url.Values{"mode": {"observe-access"}, "csrf": {csrf}}
	if w := call(a, "POST", "/machining-admin/db-check", f, worker); w.Code != 403 {
		t.Fatal("worker access")
	}
	if w := call(a, "POST", "/machining-admin/db-check", url.Values{"mode": {"observe-access"}}, owner); w.Code != 403 {
		t.Fatal("csrf")
	}
	before := string(jsonRaw(a.state))
	a.erpDBProbe = func(ctx context.Context, _ string) ERPDBReport {
		if on, _ := ctx.Value(erpDBObserveKey{}).(bool); !on {
			t.Error("no context")
		}
		return ERPDBReport{Status: "observation_access", Connected: true, ObservationAccess: &ERPDBObservationAccess{}}
	}
	if w := call(a, "POST", "/machining-admin/db-check", f, owner); w.Code != 202 {
		t.Fatal(w.Code)
	}
	deadline := time.Now().Add(time.Second)
	for {
		a.mu.Lock()
		busy := a.erpDBBusy
		a.mu.Unlock()
		if !busy {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timeout")
		}
		time.Sleep(time.Millisecond)
	}
	w := call(a, "GET", "/machining-admin/db-check/download", nil, owner)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"alter_any_event_session":false`) {
		t.Fatal("result")
	}
	if string(jsonRaw(a.state)) != before {
		t.Fatal("operating data changed")
	}
	r := decodeDBProbe([]byte(`{"status":"observation_access","stage":"observation_access","observation_access":{"view_server_state":true,"password":"synthetic-secret"}}`))
	if strings.Contains(string(jsonRaw(r)), "synthetic-secret") {
		t.Fatal("secret")
	}
	if !strings.Contains(dbProbeMessage(r), "아직") {
		t.Fatal("false ready")
	}
}
