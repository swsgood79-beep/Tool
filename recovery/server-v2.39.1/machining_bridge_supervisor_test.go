package main

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManagedBridgeOwnsOneChildAndStopsWithApp(t *testing.T) {
	a := fixture(t)
	marker := filepath.Join(a.dir, "erp-bridge-enabled")
	if e := atomicFile(marker, []byte("enabled")); e != nil {
		t.Fatal(e)
	}
	type child struct {
		setup   bool
		stopped chan struct{}
	}
	started := make(chan child, 5)
	run := func(ctx context.Context, setup bool) error {
		c := child{setup, make(chan struct{})}
		started <- c
		<-ctx.Done()
		close(c.stopped)
		return ctx.Err()
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ticks := make(chan time.Time)
	finished := make(chan struct{})
	go func() { a.superviseERP(ctx, ticks, run); close(finished) }()
	receive := func() child {
		select {
		case c := <-started:
			return c
		case <-time.After(2 * time.Second):
			t.Fatal("owned child did not start")
			return child{}
		}
	}
	tick := func(n int) {
		select {
		case ticks <- time.Unix(int64(n), 0):
		case <-time.After(2 * time.Second):
			t.Fatal("supervisor unresponsive")
		}
	}
	tick(1)
	first := receive()
	if first.setup {
		t.Fatal("startup unexpectedly launched interactive setup")
	}
	tick(2)
	select {
	case <-started:
		t.Fatal("overlapping capture")
	default:
	}
	os.Remove(marker)
	tick(3)
	select {
	case <-first.stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("disable left owned child running")
	}
	atomicFile(marker, []byte("enabled"))
	a.mu.Lock()
	a.machiningBridgeSetup = true
	a.mu.Unlock()
	tick(4)
	second := receive()
	if !second.setup {
		t.Fatal("setup request did not launch interactive setup")
	}
	tick(5)
	select {
	case <-started:
		t.Fatal("duplicate setup")
	default:
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("parent shutdown did not stop helper")
	}
	select {
	case <-second.stopped:
	default:
		t.Fatal("helper survived parent shutdown")
	}
}
func TestManagedBridgeSetupOwnerRequestAndDuplicateGuard(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	csrf := a.sessions[hashToken(owner.Value)].CSRF
	f := url.Values{"csrf": {csrf}, "action": {"setup"}}
	w := call(a, "POST", "/machining-admin/bridge", f, owner)
	if w.Code != 303 || !a.machiningBridgeSetup || !a.machiningBridgeEnabled() {
		t.Fatal("setup request", w.Code)
	}
	if w := call(a, "POST", "/machining-admin/bridge", f, owner); w.Code != 409 {
		t.Fatal("duplicate setup allowed")
	}
	a.machiningBridgeSetup = false
	a.testMode = true
	owner = adminCookie(a)
	owner.Name = a.sessionCookie()
	f.Set("csrf", a.sessions[hashToken(owner.Value)].CSRF)
	if w := call(a, "POST", "/machining-admin/bridge", f, owner); w.Code != 409 || a.machiningBridgeSetup {
		t.Fatal("test mode may touch real ERP", w.Code, w.Body.String())
	}
}
