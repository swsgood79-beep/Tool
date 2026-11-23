package main

import (
	"context"
	"errors"
	"time"
)

// Runtime only: a deliberate disconnect lasts until this program exits.
// Every normal launch (including the updater's restart) enables connection again.
type internetAutoState struct {
	Context         context.Context
	Running, Paused bool
	Restoring       bool
	Attempt, Next   time.Time
	Failures        int
}

func internetRetryDelay(failures int) time.Duration {
	if failures < 1 {
		failures = 1
	}
	return time.Duration(min(60, 5<<min(failures-1, 4))) * time.Second
}
func (a *App) resumeInternetLocked() {
	if a.internetAuto == nil {
		a.internetAuto = &internetAutoState{}
	}
	a.internetAuto.Paused = false
	a.internetAuto.Next = time.Time{}
	a.internetAuto.Failures = 0
}
func (a *App) pauseInternetLocked() {
	if a.internetAuto == nil {
		a.internetAuto = &internetAutoState{}
	}
	a.internetAuto.Paused = true
	if a.integration.CancelTunnel != nil {
		a.integration.CancelTunnel()
	}
	a.integration.TunnelStatus = "인터넷 연결 끔 · 다음 실행 시 자동 연결"
}
func (a *App) startInternetAuto(ctx context.Context) {
	go a.runInternetAuto(ctx, 5*time.Second, a.startTunnel)
}
func (a *App) runInternetAuto(ctx context.Context, interval time.Duration, start func()) {
	a.mu.Lock()
	if a.testMode || (a.internetAuto != nil && a.internetAuto.Running) {
		a.mu.Unlock()
		return
	}
	a.resumeInternetLocked()
	a.internetAuto.Context = ctx
	a.internetAuto.Running = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.internetAuto.Running = false
		a.internetAuto.Paused = true
		if a.integration.CancelTunnel != nil {
			a.integration.CancelTunnel()
		}
	}()
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		a.mu.Lock()
		auto := a.internetAuto
		shouldStart := false
		now := time.Now()
		if !auto.Paused && !auto.Restoring {
			if a.integration.CancelTunnel == nil && !now.Before(auto.Next) {
				c := a.state.Integration.Connection
				if c.Provider == "pages" && c.GitHubToken == "" {
					a.integration.TunnelStatus = "자동 연결 대기 · 고정 QR 설정에서 GitHub 연결키를 확인해 주세요"
					auto.Next = now.Add(30 * time.Second)
				} else {
					auto.Next = now.Add(5 * time.Second)
					shouldStart = true
				}
			} else if a.integration.CancelTunnel != nil && a.integration.TunnelStatus == "외부 접속 연결됨" {
				auto.Failures = 0
			} else if a.integration.CancelTunnel != nil && !auto.Attempt.IsZero() && now.Sub(auto.Attempt) > 2*time.Minute {
				a.integration.TunnelStatus = "연결 준비 시간 초과 · 자동 재연결 대기"
				auto.Next = now.Add(internetRetryDelay(auto.Failures))
				a.integration.CancelTunnel()
				auto.Attempt = now
			}
		}
		a.mu.Unlock()
		if shouldStart {
			start()
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// Restoring replaces integration runtime. Drain the old tunnel before that reset
// so its deferred cleanup cannot cancel or clear a newly started connection.
func (a *App) holdInternetForRestore(ctx context.Context) (func(), error) {
	a.mu.Lock()
	if a.internetAuto == nil {
		a.internetAuto = &internetAutoState{}
	}
	a.internetAuto.Restoring = true
	if a.integration.CancelTunnel != nil {
		a.integration.CancelTunnel()
	}
	a.mu.Unlock()
	release := func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.internetAuto.Restoring = false
		a.internetAuto.Next = time.Time{}
	}
	wait, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		a.mu.Lock()
		active := a.integration.CancelTunnel != nil
		a.mu.Unlock()
		if !active {
			return release, nil
		}
		select {
		case <-wait.Done():
			return release, errors.New("인터넷 연결 종료를 기다리는 중입니다. 잠시 후 복원을 다시 실행해 주세요")
		case <-tick.C:
		}
	}
}
