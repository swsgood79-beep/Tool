package main

import (
	"context"
	"net/http"
	"sort"
)

func purchasingPushData(c PurchasingCall, test bool) map[string]string {
	if test {
		return map[string]string{"kind": "purchasing_call", "call_id": "0", "title": "알파 구매팀", "body": "구매팀 호출 알림 연결 테스트입니다.", "tag": "alpha-purchasing-test"}
	}
	return map[string]string{"kind": "purchasing_call", "call_id": fmtID(c.ID), "title": nativePreview(c.Worker + "님이 구매팀을 호출했습니다"), "body": nativePreview(c.ProjectName + "\n수주 " + c.OrderNumber), "tag": "alpha-purchasing-call-" + fmtID(c.ID)}
}
func (a *App) pumpPurchasing(ctx context.Context) {
	a.mu.Lock()
	if a.testMode || !nativePushReady(a.state.NativeAdminPush) {
		a.mu.Unlock()
		return
	}
	cfg := *a.state.NativeAdminPush
	cfg.Devices = append([]NativeAdminDevice(nil), cfg.Devices...)
	calls := append([]PurchasingCall(nil), a.state.PurchasingCalls...)
	epoch := a.epoch
	a.mu.Unlock()
	sort.Slice(calls, func(i, j int) bool { return calls[i].ID < calls[j].ID })
	for _, d := range cfg.Devices {
		if d.Role != "purchasing" {
			continue
		}
		a.mu.Lock()
		valid := nativeAccountValid(&a.state, d)
		a.mu.Unlock()
		if !valid {
			continue
		}
		delivered, status, disable, testDone := d.Delivered, "", false, false
		if d.TestPending {
			bad, err := a.sendNativeData(ctx, cfg, d, purchasingPushData(PurchasingCall{}, true))
			disable = bad
			if err != nil {
				status = err.Error()
			} else {
				status = "구매팀 테스트 알림 전송 완료"
				testDone = true
			}
		}
		if status == "" || testDone {
			sent := 0
			for _, c := range calls {
				if c.ID <= delivered {
					continue
				}
				a.mu.Lock()
				current := nativeDevice(&a.state, d.ID)
				call := purchasingFind(&a.state, c.ID)
				active := current != nil && equal(current.SecretHash, d.SecretHash) && nativeAccountValid(&a.state, *current) && a.epoch == epoch && a.state.NativeAdminPush.Revision == cfg.Revision
				eligible := active && call != nil && call.RecipientID == d.AccountID && call.CompletedAt == nil
				a.mu.Unlock()
				if !active {
					break
				}
				if eligible {
					bad, err := a.sendNativeData(ctx, cfg, d, purchasingPushData(c, false))
					if err != nil {
						status = err.Error()
						disable = bad
						break
					}
					status = "구매팀 호출 알림 전송 완료"
					sent++
				}
				delivered = c.ID
				if sent >= 32 {
					break
				}
			}
		}
		if delivered == d.Delivered && status == "" {
			continue
		}
		a.mu.Lock()
		current := nativeDevice(&a.state, d.ID)
		if current != nil && equal(current.SecretHash, d.SecretHash) && a.epoch == epoch && a.state.NativeAdminPush.Revision == cfg.Revision {
			a.commitAnnouncement(func(s *State) error {
				v := nativeDevice(s, d.ID)
				v.Delivered = max(v.Delivered, delivered)
				if status != "" {
					v.LastStatus = status
				}
				if testDone {
					v.TestPending = false
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

func (a *App) purchasingSettingsHTTP(w http.ResponseWriter, r *http.Request) bool {
	session, ok := a.admin(w, r)
	if !ok {
		return true
	}
	if session.AccountID != "" {
		http.Error(w, "대표 관리자만 구매팀 연결을 설정할 수 있습니다.", 403)
		return true
	}
	problem := ""
	if r.Method == "POST" {
		var err error
		switch r.FormValue("action") {
		case "recipient":
			id := r.FormValue("recipient")
			u := adminAccount(&a.state, id)
			if id != "" && (u == nil || !u.Enabled || u.Role != "purchasing") {
				problem = "사용 중인 구매팀 계정을 선택해 주세요."
			} else {
				err = a.commitAnnouncement(func(s *State) error { s.PurchasingRecipient = id; return nil })
			}

		default:
			problem = "설정 요청을 확인해 주세요."
		}
		if err != nil {
			problem = err.Error()
		}
		if problem == "" {
			http.Redirect(w, r, "/purchasing-settings?saved=1", 303)
			return true
		}
	}
	rows := []AdminAccount{}
	for _, u := range a.state.AdminAccounts {
		if u.Role == "purchasing" {
			u.PasswordHash = ""
			rows = append(rows, u)
		}
	}
	ready := nativePushReady(a.state.NativeAdminPush)
	v := a.view(session.CSRF)
	v["PurchasingAccounts"], v["Recipient"], v["Error"], v["Ready"], v["Saved"], v["Package"] = rows, a.state.PurchasingRecipient, problem, ready, r.URL.Query().Get("saved") == "1", nativeAndroidPackage
	a.render(w, "purchasing-settings.html", v)
	return true
}
