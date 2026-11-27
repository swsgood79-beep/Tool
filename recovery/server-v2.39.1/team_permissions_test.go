package main

import (
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"testing"
	"time"
)

func teamManager(t *testing.T, a *App, id string, menus *AdminMenus) Session {
	a.state.AdminAccounts = append(a.state.AdminAccounts, AdminAccount{ID: id, Name: "관리자 " + id, Username: id, Role: "admin", Enabled: true, Revision: 1, Menus: menus})
	c := memberInspectorCookie(a, id)
	return a.sessions[hashToken(c.Value)]
}
func TestTeamPermissionsPreserveExistingAccountsAndSessions(t *testing.T) {
	a := fixture(t)
	s := teamManager(t, a, "existing", nil)
	owner := adminCookie(a)
	before := string(jsonRaw(a.state.AdminAccounts))
	if m := adminMenus(&a.state, s); !m.Production || !m.Machining || m.Purchasing {
		t.Fatal("legacy access changed", m)
	}
	if w := call(a, "GET", "/admin-accounts", nil, owner); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if before != string(jsonRaw(a.state.AdminAccounts)) || !validAccountSession(&a.state, s) {
		t.Fatal("opening settings changed account")
	}
	u := adminAccount(&a.state, s.AccountID)
	f := url.Values{"csrf": {a.sessions[hashToken(owner.Value)].CSRF}, "action": {"menus"}, "id": {u.ID}, "revision": {"1"}, "menu_revision": {"0"}, "menu_purchasing": {"1"}, "menu_machining": {"1"}}
	if w := call(a, "POST", "/admin-accounts", f, owner); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	if !validAccountSession(&a.state, s) || adminAccount(&a.state, s.AccountID).Revision != 1 {
		t.Fatal("menu save logged existing user out")
	}
	c := memberInspectorCookie(a, s.AccountID)
	if w := call(a, "GET", "/", nil, c); w.Code != 303 || w.Header().Get("Location") != "/team-menu" {
		t.Fatal("production entry leaked", w.Code)
	}
	if w := call(a, "GET", "/inspection-requests", nil, c); w.Code != 403 {
		t.Fatal("production route bypass", w.Code)
	}
	if w := call(a, "GET", "/purchasing-admin", nil, c); w.Code != 200 {
		t.Fatal("purchase permission not applied", w.Code)
	}
	if w := call(a, "POST", "/admin-accounts", f, c); w.Code != 403 {
		t.Fatal("nonowner granted permissions")
	}
}
func TestTeamNoticesIndependentReceiptsAndRestart(t *testing.T) {
	a := fixture(t)
	one := teamManager(t, a, "one", nil)
	two := teamManager(t, a, "two", nil)
	none := teamManager(t, a, "none", &AdminMenus{Production: true})
	_, worker := machiningWorker(t, a, "notice-worker")
	if e := a.importMachining(machiningDraft(t, machiningSampleRows()), MachiningActor{}); e != nil {
		t.Fatal(e)
	}
	before := string(jsonRaw(a.state.AdminAccounts))
	id := a.state.MachiningJobs[0].ID
	var rid string
	var rev int64
	for _, phase := range []string{"cutting", "milling1", "milling2"} {
		j := machiningFind(&a.state, id)
		if e := a.actMachining(worker, id, phase, "start", "", j.Revision, 0); e != nil {
			t.Fatal(e)
		}
		j = machiningFind(&a.state, id)
		rs := j.Cutting
		if phase == "milling1" {
			rs = j.MillingFirst
		} else if phase == "milling2" {
			rs = j.Milling
		}
		rid = machiningActive(rs).ID
		rev = j.Revision
		if e := a.actMachining(worker, id, phase, "complete", rid, rev, 12); e != nil {
			t.Fatal(e)
		}
		if phase != "milling2" && len(a.state.TeamNotices) != 0 {
			t.Fatal("early-stage notification")
		}
	}
	if before != string(jsonRaw(a.state.AdminAccounts)) {
		t.Fatal("completion changed existing accounts")
	}
	if len(a.state.TeamNotices) != 1 {
		t.Fatal("missing completion notice")
	}
	n := a.state.TeamNotices[0]
	if n.Quantity != 12 || n.Completed != 12 || n.Required != 32 || !teamNoticeVisible(&a.state, n, one) || !teamNoticeVisible(&a.state, n, two) || teamNoticeVisible(&a.state, n, none) {
		t.Fatal("notice routing")
	}
	if a.actMachining(worker, id, "milling2", "complete", rid, rev, 12) == nil || len(a.state.TeamNotices) != 1 {
		t.Fatal("retry generated duplicate")
	}
	c1 := memberInspectorCookie(a, "one")
	c2 := memberInspectorCookie(a, "two")
	csrf := a.sessions[hashToken(c1.Value)].CSRF
	if w := call(a, "POST", "/team-alerts", url.Values{"csrf": {csrf}, "id": {fmtID(n.ID)}}, c1); w.Code != 200 {
		t.Fatal(w.Code)
	}
	n = a.state.TeamNotices[0]
	if !containsAccount(n.ReadBy, "one") || containsAccount(n.ReadBy, "two") {
		t.Fatal("one manager consumed another manager notice")
	}
	if w := call(a, "GET", "/team-alerts", nil, c2); w.Code != 200 {
		t.Fatal(w.Code)
	}
	restored, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	saved := machiningFind(&restored.state, id)
	if saved == nil || len(saved.Cutting) != 1 || len(saved.MillingFirst) != 1 || len(saved.Milling) != 1 || machiningDone(saved.Cutting) != 12 || machiningDone(saved.MillingFirst) != 12 || machiningDone(saved.Milling) != 12 {
		t.Fatal("acknowledging notification removed process records")
	}
	if len(restored.state.TeamNotices) != 1 || !containsAccount(restored.state.TeamNotices[0].ReadBy, "one") {
		t.Fatal("notice/read state lost on restart")
	}
	adminAccount(&a.state, "two").Menus = &AdminMenus{Production: true}
	if teamNoticeVisible(&a.state, n, two) {
		t.Fatal("revoked permission still sees notices")
	}
	if a.state.NextID < n.ID || n.At.Before(time.Now().Add(-time.Minute)) {
		t.Fatal("notice id/time invalid")
	}
}

func TestTeamAlertBrowser(t *testing.T) {
	if os.Getenv("TEAM_BROWSER") != "1" {
		t.Skip("optional Chromium")
	}
	a := fixture(t)
	owner := adminCookie(a)
	s := teamManager(t, a, "qa02", nil)
	teamManager(t, a, "qa03", &AdminMenus{Production: true})
	member := memberInspectorCookie(a, s.AccountID)
	_, worker := machiningWorker(t, a, "machine01")
	adminAccount(&a.state, "qa02").Name = "김관리"
	adminAccount(&a.state, "qa03").Name = "박관리"
	adminAccount(&a.state, "machine01").Name = "김가공"
	rows := machiningSampleRows()
	rows[0][8] = 10
	rows[1][8] = 10
	if e := a.importMachining(machiningDraft(t, rows), MachiningActor{}); e != nil {
		t.Fatal(e)
	}
	id := a.state.MachiningJobs[0].ID
	for _, phase := range []string{"cutting", "milling1", "milling2"} {
		j := machiningFind(&a.state, id)
		a.actMachining(worker, id, phase, "start", "", j.Revision, 0)
		j = machiningFind(&a.state, id)
		rs := j.Cutting
		if phase == "milling1" {
			rs = j.MillingFirst
		} else if phase == "milling2" {
			rs = j.Milling
		}
		q := 20
		if phase == "milling2" {
			q = 10
		}
		if e := a.actMachining(worker, id, phase, "complete", machiningActive(rs).ID, j.Revision, q); e != nil {
			t.Fatal(e)
		}
	}
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg := map[string]any{"owner": map[string]string{"name": owner.Name, "value": owner.Value}, "member": map[string]string{"name": member.Name, "value": member.Value}}
	cmd := exec.Command("node", "audit/browser-team-v1.4.115.cjs", srv.URL, string(jsonRaw(cfg)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
