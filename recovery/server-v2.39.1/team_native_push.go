package main

import (
	"context"
	"fmt"
)

func teamNativeData(n TeamNotice) map[string]string {
	return map[string]string{"kind": "machining_complete", "title": "가공 2차 완료", "body": nativePreview(fmt.Sprintf("이번 완료 %d개 · 누적 %d/%d개 · 남음 %d개\n수주 %s\n%s\n품번 %s\n작업자 %s", n.Quantity, n.Completed, n.Required, max(0, n.Required-n.Completed), n.Order, n.Project, n.Part, n.Worker)), "project_id": "0", "request_id": "0", "tag": "alpha-machining-" + fmtID(n.ID)}
}
func (a *App) pumpNativeMachining(ctx context.Context) {
	a.mu.Lock()
	if a.testMode || !nativePushReady(a.state.NativeAdminPush) {
		a.mu.Unlock()
		return
	}
	cfg := *a.state.NativeAdminPush
	devices := append([]NativeAdminDevice(nil), cfg.Devices...)
	notices := append([]TeamNotice(nil), a.state.TeamNotices...)
	epoch := a.epoch
	a.mu.Unlock()
	for _, d := range devices {
		if d.Role == "purchasing" || d.Role == "machining" {
			continue
		}
		a.mu.Lock()
		valid := nativeAccountValid(&a.state, d)
		a.mu.Unlock()
		if !valid {
			continue
		}
		delivered := d.TeamDelivered
		status := ""
		disable := false
		sent := 0
		for _, n := range notices {
			if n.ID <= delivered {
				continue
			}
			a.mu.Lock()
			currentNotice := false
			for _, current := range a.state.TeamNotices {
				if current.ID == n.ID && current.JobID == n.JobID && current.RunID == n.RunID {
					currentNotice = true
					break
				}
			}
			allowed := currentNotice && nativeAccountValid(&a.state, d) && teamNoticeVisible(&a.state, n, Session{AccountID: d.AccountID})
			a.mu.Unlock()
			if allowed {
				bad, e := a.sendNativeData(ctx, cfg, d, teamNativeData(n))
				if e != nil {
					disable = bad
					status = e.Error()
					break
				}
				status = "가공 2차 완료 알림 전송 완료"
				sent++
			}
			delivered = n.ID
			if sent >= 32 {
				break
			}
		}
		if delivered == d.TeamDelivered && status == "" {
			continue
		}
		a.mu.Lock()
		current := nativeDevice(&a.state, d.ID)
		if current != nil && equal(current.SecretHash, d.SecretHash) && a.epoch == epoch && a.state.NativeAdminPush.Revision == cfg.Revision {
			a.commitAnnouncement(func(s *State) error {
				v := nativeDevice(s, d.ID)
				v.TeamDelivered = max(v.TeamDelivered, delivered)
				if status != "" {
					v.LastStatus = status
				}
				if disable {
					v.Enabled = false
				}
				return nil
			})
		}
		a.mu.Unlock()
	}
}
