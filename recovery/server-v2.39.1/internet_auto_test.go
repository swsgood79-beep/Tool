package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func waitInternetAttempt(t *testing.T, c <-chan struct{}) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(2 * time.Second):
		t.Fatal("automatic connection did not start")
	}
}
func TestInternetAutoStartupRetryPauseAndRestart(t *testing.T) {
	a := fixture(t)
	a.state.Integration.Connection.Provider = "pages"
	a.state.Integration.Connection.GitHubToken = "TEST-ONLY"
	attempts := make(chan struct{}, 10)
	fakeStart := func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		attempts <- struct{}{}
		a.integration.CancelTunnel = func() {}
		a.integration.TunnelStatus = "외부 접속 연결됨"
		a.internetAuto.Attempt = time.Now()
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); a.runInternetAuto(ctx, 5*time.Millisecond, fakeStart) }()
	waitInternetAttempt(t, attempts)
	// A failed/exited initial tunnel is retried without another user click.
	a.mu.Lock()
	a.integration.CancelTunnel = nil
	a.integration.TunnelStatus = "연결 실패"
	a.internetAuto.Next = time.Now().Add(-time.Second)
	a.mu.Unlock()
	waitInternetAttempt(t, attempts)
	a.mu.Lock()
	a.pauseInternetLocked()
	a.integration.CancelTunnel = nil
	a.mu.Unlock()
	select {
	case <-attempts:
		t.Fatal("manual disconnect immediately reconnected")
	case <-time.After(30 * time.Millisecond):
	}
	cancel()
	<-done
	a.mu.Lock()
	a.integration.CancelTunnel = nil
	a.mu.Unlock()
	// A fresh launch must re-enable connection even after a deliberate disconnect.
	next, stop := context.WithCancel(context.Background())
	defer stop()
	nextDone := make(chan struct{})
	go func() { defer close(nextDone); a.runInternetAuto(next, 5*time.Millisecond, fakeStart) }()
	waitInternetAttempt(t, attempts)
	stop()
	<-nextDone
}
func TestInternetAutoMissingSettingsAndTestIsolation(t *testing.T) {
	a := fixture(t)
	a.state.Integration.Connection.Provider = "pages"
	a.state.Integration.Connection.GitHubToken = ""
	attempts := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		a.runInternetAuto(ctx, 5*time.Millisecond, func() { attempts <- struct{}{} })
	}()
	deadline := time.After(2 * time.Second)
	for {
		a.mu.Lock()
		status := a.integration.TunnelStatus
		a.mu.Unlock()
		if strings.Contains(status, "GitHub 연결키") {
			break
		}
		select {
		case <-deadline:
			cancel()
			<-done
			t.Fatal("missing settings not explained")
		case <-time.After(time.Millisecond):
		}
	}
	select {
	case <-attempts:
		t.Fatal("started a fixed QR tunnel with missing key")
	default:
	}
	a.mu.Lock()
	a.state.Integration.Connection.GitHubToken = "TEST-ONLY"
	a.internetAuto.Next = time.Time{}
	a.mu.Unlock()
	waitInternetAttempt(t, attempts)
	cancel()
	<-done
	test := fixture(t)
	test.testMode = true
	test.runInternetAuto(context.Background(), time.Millisecond, func() { t.Fatal("test mode opened external connection") })
}
func TestInternetAutoHungAttemptAndBackoff(t *testing.T) {
	for i, want := range []time.Duration{5, 10, 20, 40, 60, 60} {
		if got := internetRetryDelay(i + 1); got != want*time.Second {
			t.Fatal(got, want)
		}
	}
	a := fixture(t)
	a.internetAuto = &internetAutoState{Attempt: time.Now().Add(-3 * time.Minute)}
	cancelled := make(chan struct{})
	var once sync.Once
	a.integration.CancelTunnel = func() { once.Do(func() { close(cancelled) }) }
	a.integration.TunnelStatus = "인터넷 연결 준비 중"
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		a.runInternetAuto(ctx, 5*time.Millisecond, func() { t.Error("duplicated active attempt") })
	}()
	waitInternetAttempt(t, cancelled)
	cancel()
	<-done
}
func TestInternetAutoToolFailureClearsActiveAttempt(t *testing.T) {
	a := fixture(t)
	a.state.Integration.Connection.Provider = "quick"
	a.internetAuto = &internetAutoState{}
	if e := os.WriteFile(filepath.Join(a.dir, "cloudflared-2026.10.7.exe"), []byte("invalid test tool"), 0700); e != nil {
		t.Fatal(e)
	}
	a.startTunnel()
	deadline := time.After(2 * time.Second)
	for {
		a.mu.Lock()
		finished := a.integration.CancelTunnel == nil
		status := a.integration.TunnelStatus
		retry := a.internetAuto.Next
		a.mu.Unlock()
		if finished {
			if !strings.Contains(status, "체크섬") || !strings.Contains(status, "재연결") || retry.Before(time.Now()) {
				t.Fatal(status, retry)
			}
			break
		}
		select {
		case <-deadline:
			t.Fatal("failed tool left active attempt")
		case <-time.After(time.Millisecond):
		}
	}
}

func TestInternetAutoRestoreSuspendsAndDrainsOldTunnel(t *testing.T) {
	a := fixture(t)
	a.internetAuto = &internetAutoState{}
	stopped := make(chan struct{})
	var once sync.Once
	a.integration.CancelTunnel = func() { once.Do(func() { close(stopped) }) }
	go func() { <-stopped; a.mu.Lock(); a.integration.CancelTunnel = nil; a.mu.Unlock() }()
	release, e := a.holdInternetForRestore(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	a.mu.Lock()
	held := a.internetAuto.Restoring
	a.mu.Unlock()
	if !held {
		t.Fatal("restore does not hold auto connection")
	}
	a.startTunnel()
	a.mu.Lock()
	active := a.integration.CancelTunnel != nil
	a.mu.Unlock()
	if active {
		t.Fatal("started a new tunnel during restore")
	}
	release()
	a.mu.Lock()
	held = a.internetAuto.Restoring
	a.mu.Unlock()
	if held {
		t.Fatal("auto connection not released after restore")
	}
}
