package main

import (
	"fmt"
	"strings"
	"time"
)

// Read the same cached provider results used by the worker screen; never start extra API calls.
func (a *App) marketAdminStatus() map[string]any {
	enabled := a.state.Integration.Market.Enabled
	result := a.integration.MarketResult
	checked := text(result["checked"])
	quotes, _ := result["quotes"].([]any)
	available := map[string]bool{}
	stale := map[string]bool{}
	for _, value := range quotes {
		if q, ok := value.(map[string]any); ok {
			if price, ok := numeric(q["price"]); ok && price > 0 {
				symbol := text(q["symbol"])
				available[symbol] = true
				stale[symbol], _ = q["stale"].(bool)
			}
		}
	}
	state, message := "waiting", "아직 시세 확인 전입니다. 잠시 후 다시 확인하세요."
	details := []string{}
	hasKeys := a.state.Integration.Market.ClientID != "" && a.state.Integration.Market.ClientSecret != ""
	fresh := 0
	missing := false
	for _, item := range []struct{ symbol, label string }{{"KOSPI", "코스피"}, {"KOSDAQ", "코스닥"}, {"KRW-BTC", "비트코인"}, {"KRW-ETH", "이더리움"}} {
		label := "확인 대기"
		if !hasKeys && (item.symbol == "KOSPI" || item.symbol == "KOSDAQ") {
			label = "토스 키 미등록"
		} else if available[item.symbol] {
			if stale[item.symbol] {
				label = "이전 값 유지"
				missing = true
			} else {
				label = "정상"
				fresh++
			}
		} else {
			missing = true
		}
		details = append(details, item.label+" "+label)
	}
	if hasKeys {
		fx, _ := result["fx"].(map[string]any)
		rate, ok := numeric(fx["rate"])
		if ok && rate > 0 {
			details = append(details, "환율 정상")
		} else {
			details = append(details, "환율 확인 대기")
			missing = true
		}
	}
	if !enabled {
		state, message = "disabled", "시장 정보 표시가 꺼져 있습니다."
	} else if checked != "" {
		delayed, _ := result["delayed"].(bool)
		if fresh > 0 && !delayed && !missing {
			state, message = "ok", "시세 조회 정상"
		} else if len(available) > 0 {
			state, message = "warning", "일부 시세 확인 중 · 실패한 항목은 이전 값이 표시될 수 있습니다."
		} else {
			state, message = "error", "시세를 확인하지 못했습니다."
		}
		if state != "ok" && a.integration.MarketStatus != "" && !strings.HasPrefix(a.integration.MarketStatus, "시세 조회 완료") {
			message += " " + a.integration.MarketStatus
		}
		if t, e := time.Parse(time.RFC3339, checked); e == nil {
			message += fmt.Sprintf(" · 마지막 확인 %s", t.In(kst).Format("01.02 15:04:05"))
		}
	}
	return map[string]any{"enabled": enabled, "ok": state == "ok", "state": state, "message": message, "status": a.integration.MarketStatus, "checked": checked, "summary": strings.Join(details, " · ")}
}
