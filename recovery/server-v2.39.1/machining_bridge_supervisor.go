package main

import (
	"context"
	"time"
)

// One owned child at a time. Setup interrupts automatic capture, disabling or
// shutdown cancels the owned child, and successful capture repeats after 5 min.
func (a *App) superviseERP(ctx context.Context, ticks <-chan time.Time, run func(context.Context, bool) error) {
	var cancel context.CancelFunc
	var done chan error
	var setting bool
	var next, now time.Time
	stop := func() {
		if cancel != nil {
			cancel()
			<-done
			cancel = nil
			done = nil
		}
	}
	defer stop()
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-done:
			done = nil
			cancel = nil
			a.mu.Lock()
			if setting {
				a.machiningBridgeSetting = false
			}
			a.mu.Unlock()
			setting = false
			if err != nil {
				next = now.Add(30 * time.Second)
			} else {
				next = now.Add(5 * time.Minute)
			}
		case now = <-ticks:
			a.mu.Lock()
			enabled := a.machiningBridgeEnabled()
			setup := a.machiningBridgeSetup
			a.machiningBridgeSetup = false
			if setup {
				a.machiningBridgeSetting = true
			}
			a.mu.Unlock()
			if !enabled {
				stop()
				a.mu.Lock()
				a.machiningBridgeSetting = false
				a.mu.Unlock()
				continue
			}
			if setup {
				stop()
				next = time.Time{}
			}
			if done != nil || (!setup && now.Before(next)) {
				continue
			}
			runCtx, runCancel := context.WithCancel(ctx)
			a.mu.Lock()
			a.machiningBridgeSetting = setup
			a.mu.Unlock()
			setting = setup
			cancel = runCancel
			done = make(chan error, 1)
			exited := done
			go func(setup bool) { e := run(runCtx, setup); runCancel(); exited <- e }(setup)
		}
	}
}
