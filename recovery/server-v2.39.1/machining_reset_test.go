package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func resetFixture(t *testing.T) (*App, *http.Cookie, *http.Cookie) {
	a := fixture(t)
	owner := adminCookie(a)
	worker, _ := machiningWorker(t, a, "target")
	machiningWorker(t, a, "other")
	now := time.Now()
	who := MachiningActor{ID: "target", Username: "target", Name: "동일 이름"}
	other := MachiningActor{ID: "other", Username: "other", Name: "동일 이름"}
	run := func(id string, actor MachiningActor) MachiningRun {
		return MachiningRun{ID: id, Quantity: 10, Started: now, StartedBy: actor, Completed: &now, CompletedBy: &actor}
	}
	a.state.MachiningJobs = []MachiningJob{
		{ID: "safe", Order: "ORDER-001", Part: "TEST-BASE", Required: 20, Workflow: 2, Revision: 1, Cutting: []MachiningRun{run("cut", who)}, MillingFirst: []MachiningRun{run("first", who)}, Milling: []MachiningRun{run("last", who)}},
		{ID: "downstream", Order: "ORDER-001", Required: 20, Workflow: 2, Revision: 1, Cutting: []MachiningRun{run("target-cut", who)}, MillingFirst: []MachiningRun{run("other-first", other)}},
		{ID: "receipt", Order: "ORDER-001", Required: 20, Workflow: 2, Revision: 1, Milling: []MachiningRun{run("received", who)}},
		{ID: "other", Order: "ORDER-001", Required: 20, Workflow: 2, Revision: 1, Cutting: []MachiningRun{run("other-cut", other)}},
	}
	a.state.MachiningJobs[2].Milling[0].ReceiptAt = &now
	a.state.TeamNotices = []TeamNotice{{ID: 1, JobID: "safe", RunID: "last"}, {ID: 2, JobID: "receipt", RunID: "received"}, {ID: 3, JobID: "other", RunID: "other-cut"}}
	return a, owner, worker
}
func resetForm(a *App, c *http.Cookie) url.Values {
	p := machiningResetPlan(&a.state, "target")
	return url.Values{"csrf": {a.sessions[hashToken(c.Value)].CSRF}, "account": {"target"}, "fingerprint": {p.Fingerprint}, "confirmed": {"yes"}}
}
func TestMachiningResetAuthorizationAndCSRF(t *testing.T) {
	a, owner, worker := resetFixture(t)
	before := string(jsonRaw(a.state))
	for _, method := range []string{"GET", "POST"} {
		if w := call(a, method, "/machining-admin/test-reset", resetForm(a, worker), worker); w.Code != 403 {
			t.Fatal(w.Code)
		}
	}
	f := resetForm(a, owner)
	f.Set("csrf", "wrong")
	if w := call(a, "POST", "/machining-admin/test-reset", f, owner); w.Code != 403 {
		t.Fatal(w.Code)
	}
	f = resetForm(a, owner)
	f.Del("confirmed")
	if w := call(a, "POST", "/machining-admin/test-reset", f, owner); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if string(jsonRaw(a.state)) != before {
		t.Fatal("rejected requests mutated state")
	}
}
func TestMachiningResetPreservesOtherWorkAndBacksUp(t *testing.T) {
	a, owner, _ := resetFixture(t)
	p := machiningResetPlan(&a.state, "target")
	if p.Removable != 3 || p.Held != 2 || p.Notices != 1 {
		t.Fatal(p)
	}
	unchanged := string(jsonRaw(a.state.MachiningJobs[1:]))
	accounts := string(jsonRaw(a.state.AdminAccounts))
	projects := string(jsonRaw(a.state.Projects))
	sessions := string(jsonRaw(a.sessions))
	oldJobs := string(jsonRaw(a.state.MachiningJobs))
	if w := call(a, "POST", "/machining-admin/test-reset", resetForm(a, owner), owner); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	j := a.state.MachiningJobs[0]
	if len(j.Cutting)+len(j.MillingFirst)+len(j.Milling) != 0 || j.Required != 20 || j.Revision != 2 {
		t.Fatal(j)
	}
	if unchanged != string(jsonRaw(a.state.MachiningJobs[1:])) || accounts != string(jsonRaw(a.state.AdminAccounts)) || projects != string(jsonRaw(a.state.Projects)) || sessions != string(jsonRaw(a.sessions)) {
		t.Fatal("unrelated state changed")
	}
	if len(a.state.TeamNotices) != 2 || len(a.state.MachiningResets) != 1 {
		t.Fatal("notice/audit mismatch")
	}
	audit := a.state.MachiningResets[0]
	b, e := os.ReadFile(filepath.Join(a.dir, "machining-test-backups", audit.Backup))
	if e != nil {
		t.Fatal(e)
	}
	b, e = unprotect(b)
	if e != nil {
		t.Fatal(e)
	}
	var backup struct{ Jobs []MachiningJob }
	if json.Unmarshal(b, &backup) != nil || string(jsonRaw(backup.Jobs)) != oldJobs {
		t.Fatal("invalid backup")
	}
	if strings.Contains(string(b), "PasswordHash") {
		t.Fatal("credentials included")
	}
	restored, e := newApp(a.dir)
	if e != nil {
		t.Fatal(e)
	}
	a = restored
	if len(a.state.MachiningResets) != 1 || len(a.state.MachiningJobs[0].Cutting) != 0 {
		t.Fatal("reset not durable")
	}
}
func TestMachiningResetStaleAndBackupFailure(t *testing.T) {
	a, owner, _ := resetFixture(t)
	f := resetForm(a, owner)
	a.state.MachiningJobs[0].Revision++
	before := string(jsonRaw(a.state))
	if w := call(a, "POST", "/machining-admin/test-reset", f, owner); w.Code != 409 {
		t.Fatal(w.Code)
	}
	if string(jsonRaw(a.state)) != before {
		t.Fatal("stale mutation")
	}
	if e := os.WriteFile(filepath.Join(a.dir, "machining-test-backups"), []byte("blocked"), 0600); e != nil {
		t.Fatal(e)
	}
	if w := call(a, "POST", "/machining-admin/test-reset", resetForm(a, owner), owner); w.Code != 409 {
		t.Fatal(w.Code)
	}
	if string(jsonRaw(a.state)) != before {
		t.Fatal("backup failure mutation")
	}
}
