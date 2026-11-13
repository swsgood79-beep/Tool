package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarketStatusEndpointMatchesFrontendAndCachedQuotes(t *testing.T) {
	a := fixture(t)
	cookie := adminCookie(a)
	a.state.Integration.Market.Enabled = true
	a.integration.MarketResult = map[string]any{"checked": "2026-11-13T07:17:16Z", "quotes": []any{map[string]any{"symbol": "KRW-BTC", "price": 100.0}, map[string]any{"symbol": "KRW-ETH", "price": 10.0}}}
	a.integration.MarketStatus = "시세 조회 완료 · 2026-11-13T16:17:16+09:00"
	a.client = &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("status should not call providers")
		return nil, nil
	})}
	for _, path := range []string{"/market-settings/status.json", "/market-settings/status"} {
		r := call(a, "GET", path, nil, cookie)
		var data map[string]any
		if r.Code != 200 || json.Unmarshal(r.Body.Bytes(), &data) != nil || data["state"] != "ok" || data["ok"] != true || !strings.Contains(text(data["message"]), "16:17:16") {
			t.Fatal("wrong status", r.Code, r.Body.String())
		}
	}
	r := call(a, "GET", "/market-settings", nil, cookie)
	if out := os.Getenv("MARKET_HTML_OUT"); out != "" {
		if err := os.WriteFile(filepath.Join(out, "market.html"), r.Body.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(a.marketAdminStatus())
		if err := os.WriteFile(filepath.Join(out, "market-normal.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Contains(r.Body.String(), `<p class="error" role="alert">시세 조회 완료`) {
		t.Fatal("success shown as error")
	}
	a.integration.MarketResult["delayed"] = true
	a.integration.MarketResult["quotes"] = []any{map[string]any{"symbol": "KRW-BTC", "price": 100.0, "stale": true}}
	if a.marketAdminStatus()["state"] != "warning" {
		t.Fatal("stale data marked healthy")
	}
	a.state.Integration.Market.Enabled = false
	if a.marketAdminStatus()["state"] != "disabled" {
		t.Fatal("disabled marked healthy")
	}
	if r := call(a, "GET", "/market-settings/status.json", nil); r.Code == 200 {
		t.Fatal("status exposed without login")
	}
}
