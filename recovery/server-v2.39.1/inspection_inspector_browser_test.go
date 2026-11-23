package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestInspectorBrowser(t *testing.T) {
	if os.Getenv("INSPECTOR_BROWSER") != "1" {
		t.Skip("optional inspector browser")
	}
	a := exportFixture(t)
	owner := adminCookie(a)
	u := createAccount(t, a, owner, "browser90")
	member := memberInspectorCookie(a, u.ID)
	base, e := clone(a.state)
	if e != nil {
		t.Fatal(e)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/audit/reset-inspectors" {
			a.mu.Lock()
			defer a.mu.Unlock()
			next, _ := clone(base)
			next.LegacyHashes = a.state.LegacyHashes
			next.LegacySourceHash = a.state.LegacySourceHash
			next.LegacySnapshots = a.state.LegacySnapshots
			if e := a.persist(&next); e != nil {
				t.Error(e)
			} else {
				a.state = next
			}
			w.WriteHeader(204)
			return
		}
		if r.URL.Path == "/audit/new-inspection" {
			a.mu.Lock()
			defer a.mu.Unlock()
			if e := a.transaction(func(s *State) error {
				p := &s.Projects[0]
				p.Setting = Stage{}
				p.Shipping = Stage{}
				p.CombinedPending = false
				p.LegacyCompleted = false
				p.Hidden = false
				p.rotate()
				return nil
			}); e != nil {
				t.Error(e)
			}
			p := a.state.Projects[0]
			if e := a.requestInspection(p.ID, "combined", "작업자 시험", p.SettingForm, p.CombinedForm); e != nil {
				t.Error(e)
			}
			w.WriteHeader(204)
			return
		}
		a.ServeHTTP(w, r)
	}))
	defer srv.Close()
	cfg, _ := json.Marshal(map[string]any{"owner": map[string]string{"name": owner.Name, "value": owner.Value}, "member": map[string]string{"name": member.Name, "value": member.Value}, "memberName": u.Name, "project": a.state.Projects[0].ID, "record": a.state.ShippingRecords[0].ID})
	cmd := exec.Command("node", "audit/browser-inspector-v1.4.90.cjs", srv.URL, string(cfg))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e = cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
