package main

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestMarketPercentUsesDisplayedPriceAndDailyClose(t *testing.T) {
	now := time.Now().In(kst).Add(-time.Minute)
	at := now.Format(time.RFC3339)
	prior := now.AddDate(0, 0, -1).Format("2006-01-02")
	for _, tc := range []struct {
		name                         string
		price, supplied, base        float64
		minute, missingDate, noDaily bool
		want                         float64
		known                        bool
	}{
		{name: "provider positive but displayed index fell", price: 4984.30, supplied: .32, base: 5000, want: -.314, known: true},
		{name: "provider negative but displayed index rose", price: 5100, supplied: -.7, base: 5000, want: 2, known: true},
		{name: "minute price replaces different snapshot", price: 4984.30, supplied: .32, base: 5000, minute: true, want: -.314, known: true},
		{name: "flat market stays zero", price: 5000, supplied: .32, base: 5000, want: 0, known: true},
		{name: "missing daily close never trusts supplied rate", price: 4984.30, supplied: .32, base: 5000, noDaily: true, known: false},
		{name: "unknown quote date never guesses from fetch date", price: 4984.30, supplied: .32, base: 5000, missingDate: true, known: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := fixture(t)
			a.state.Integration.Market = MarketConfig{Enabled: true, ClientID: "fixture", ClientSecret: "fixture", Revision: 1}
			a.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				switch {
				case r.URL.Host == "api.upbit.com":
					return response(r, 200, []any{map[string]any{"market": "KRW-BTC", "trade_price": 100, "signed_change_rate": -.02}}), nil
				case r.URL.Path == "/oauth2/token":
					return response(r, 200, map[string]any{"access_token": "fixture", "expires_in": 3600}), nil
				case strings.HasSuffix(r.URL.Path, "/prices"):
					m := map[string]any{"symbol": "KOSPI", "lastPrice": tc.price, "changeRate": tc.supplied, "basePrice": 1}
					if !tc.minute && !tc.missingDate {
						m["timestamp"] = at
					}
					if tc.minute {
						m["lastPrice"] = 5200.0
					}
					n := map[string]any{}
					for key, value := range m {
						n[key] = value
					}
					n["symbol"] = "KOSDAQ"
					return response(r, 200, []any{m, n}), nil
				case strings.HasSuffix(r.URL.Path, "/candles"):
					if r.URL.Query().Get("interval") == "1m" {
						if tc.missingDate {
							return response(r, 503, map[string]any{}), nil
						}
						return response(r, 200, []any{map[string]any{"timestamp": at, "closePrice": tc.price}}), nil
					}
					if tc.noDaily {
						return response(r, 503, map[string]any{}), nil
					}
					return response(r, 200, []any{map[string]any{"date": prior, "closePrice": tc.base}, map[string]any{"date": now.Format("2006-01-02"), "closePrice": 5200}}), nil
				case strings.Contains(r.URL.Path, "exchange-rate"):
					return response(r, 200, map[string]any{"midRate": 1300}), nil
				}
				return nil, fmt.Errorf("unexpected market URL %s", r.URL)
			})}
			a.refreshMarkets(context.Background())
			for _, symbol := range []string{"KOSPI", "KOSDAQ"} {
				q := symbolRecord(a.integration.MarketResult["quotes"], symbol)
				if p, _ := numeric(q["price"]); p != tc.price {
					t.Fatal("wrong displayed price", q)
				}
				pct, known := numeric(q["percent"])
				if known != tc.known || known && math.Abs(pct-tc.want) > 1e-8 {
					t.Fatalf("percent=%v known=%v want=%v known=%v", pct, known, tc.want, tc.known)
				}
			}
			coin := symbolRecord(a.integration.MarketResult["quotes"], "KRW-BTC")
			if pct, _ := numeric(coin["percent"]); pct != -2 {
				t.Fatal("coin UTC comparison changed", coin)
			}
		})
	}
}

func TestMarketPreviousCloseUsesQuoteTradingDate(t *testing.T) {
	for _, tc := range []struct {
		name, at string
		candles  []any
		want     float64
		valid    bool
	}{
		{"holiday quote keeps its own session", "2026-11-13T15:30:00+09:00", []any{map[string]any{"date": "2026-11-12", "close": 5000}, map[string]any{"date": "2026-11-13", "close": 4900}}, 5000, true},
		{"UTC candles classified in Korea", "2026-11-19T01:00:00Z", []any{map[string]any{"timestamp": "2026-11-17T16:00:00Z", "close": 5000}, map[string]any{"timestamp": "2026-11-18T16:00:00Z", "close": 5200}}, 5000, true},
		{"equal prior close is not evidence for changing session", "2026-11-19T08:30:00+09:00", []any{map[string]any{"date": "2026-11-17", "close": 4800}, map[string]any{"date": "2026-11-18", "close": 5000}}, 5000, true},
		{"current day candle alone is not previous close", "2026-11-19T10:00:00+09:00", []any{map[string]any{"date": "2026-11-19", "close": 5200}}, 0, false},
		{"missing quote time cannot select a session", "", nil, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := fixture(t)
			a.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) { return response(r, 200, tc.candles), nil })}
			base, err := a.previousClose(context.Background(), "KOSPI", "fixture", tc.at)
			if (err == nil) != tc.valid || tc.valid && base != tc.want {
				t.Fatalf("base=%v err=%v want=%v valid=%v", base, err, tc.want, tc.valid)
			}
		})
	}
}
