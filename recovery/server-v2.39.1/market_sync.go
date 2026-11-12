package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (a *App) getJSON(ctx context.Context, method, endpoint, bearer string, body io.Reader) (any, error) {
	r, e := http.NewRequestWithContext(ctx, method, endpoint, body)
	if e != nil {
		return nil, e
	}
	r.Header.Set("User-Agent", "WanryoTalk-Recovered/0.2")
	if bearer != "" {
		r.Header.Set("Authorization", "Bearer "+bearer)
	}
	if body != nil {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	res, e := a.httpClient().Do(r)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if res.StatusCode == 429 {
			seconds, _ := strconv.Atoi(res.Header.Get("Retry-After"))
			if seconds < 5 {
				seconds = 15
			}
			if seconds > 900 {
				seconds = 900
			}
			a.mu.Lock()
			a.integration.MarketNext = time.Now().Add(time.Duration(seconds) * time.Second)
			a.mu.Unlock()
		}
		if res.StatusCode == 401 && r.URL.Host == "openapi.tossinvest.com" {
			a.mu.Lock()
			a.integration.AccessToken = ""
			a.integration.TokenUntil = time.Time{}
			a.mu.Unlock()
		}
		return nil, fmt.Errorf("시세 HTTP %d", res.StatusCode)
	}
	var v any
	d := json.NewDecoder(io.LimitReader(res.Body, 4<<20))
	d.UseNumber()
	e = d.Decode(&v)
	return v, e
}
func numeric(v any) (float64, bool) {
	var n float64
	var e error
	switch x := v.(type) {
	case json.Number:
		n, e = x.Float64()
	case float64:
		n = x
	case string:
		n, e = strconv.ParseFloat(strings.ReplaceAll(x, ",", ""), 64)
	default:
		return 0, false
	}
	return n, e == nil && !math.IsNaN(n) && !math.IsInf(n, 0)
}
func field(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return v
		}
	}
	return nil
}
func text(v any) string { s, _ := v.(string); return s }
func maps(v any) []map[string]any {
	out := []map[string]any{}
	var walk func(any, int)
	walk = func(v any, d int) {
		if d > 12 {
			return
		}
		switch x := v.(type) {
		case map[string]any:
			out = append(out, x)
			for _, c := range x {
				walk(c, d+1)
			}
		case []any:
			for _, c := range x {
				walk(c, d+1)
			}
		}
	}
	walk(v, 0)
	return out
}
func stamp(v any) string {
	if s, ok := v.(string); ok {
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05", "20060102150405", "2006-01-02"} {
			if t, e := time.ParseInLocation(layout, s, kst); e == nil {
				return t.Format(time.RFC3339)
			}
		}
	}
	if n, ok := numeric(v); ok && n > 946684800 {
		if n > 1e12 {
			n /= 1000
		}
		if n < 7258118400 {
			return time.Unix(int64(n), 0).Format(time.RFC3339)
		}
	}
	return ""
}
func symbolRecord(v any, symbol string) map[string]any {
	for _, m := range maps(v) {
		if text(field(m, "symbol", "code", "market", "ticker")) == symbol {
			return m
		}
		if c, ok := m[symbol].(map[string]any); ok {
			return c
		}
	}
	return nil
}
func (a *App) tossToken(ctx context.Context, c MarketConfig) (string, error) {
	a.mu.Lock()
	epoch := a.epoch
	tok, until := a.integration.AccessToken, a.integration.TokenUntil
	a.mu.Unlock()
	if tok != "" && until.After(time.Now().Add(time.Minute)) {
		return tok, nil
	}
	data, e := a.getJSON(ctx, "POST", "https://openapi.tossinvest.com/oauth2/token", "", strings.NewReader(url.Values{"grant_type": {"client_credentials"}, "client_id": {c.ClientID}, "client_secret": {c.ClientSecret}}.Encode()))
	if e != nil {
		return "", e
	}
	for _, m := range maps(data) {
		if t := text(m["access_token"]); t != "" {
			seconds, _ := numeric(m["expires_in"])
			if seconds <= 60 {
				seconds = 300
			}
			a.mu.Lock()
			if a.epoch == epoch && a.state.Integration.Market.Revision == c.Revision {
				a.integration.AccessToken = t
				a.integration.TokenUntil = time.Now().Add(time.Duration(seconds) * time.Second)
			}
			a.mu.Unlock()
			return t, nil
		}
	}
	return "", errors.New("토스 토큰 응답 누락")
}
func (a *App) previousClose(ctx context.Context, symbol, bearer, at string, price float64) (float64, error) {
	key := symbol + ":" + at[:10]
	a.mu.Lock()
	base, ok := a.integration.Bases[key]
	a.mu.Unlock()
	if ok {
		return base, nil
	}
	v, e := a.getJSON(ctx, "GET", "https://openapi.tossinvest.com/api/v1/market-indicators/"+symbol+"/candles?interval=1d&count=10", bearer, nil)
	if e != nil {
		return 0, e
	}
	type candle struct {
		date  string
		close float64
	}
	candles := []candle{}
	for _, m := range maps(v) {
		p, valid := numeric(field(m, "close", "closePrice", "closingPrice"))
		ts := stamp(field(m, "dt", "observation_date", "date", "timestamp", "time"))
		if valid && p > 0 && len(ts) >= 10 {
			candles = append(candles, candle{ts[:10], p})
		}
	}
	sort.Slice(candles, func(i, j int) bool { return candles[i].date < candles[j].date })
	target := at[:10]
	if len(candles) > 0 {
		last := candles[len(candles)-1]
		if last.date < target && math.Abs(last.close-price) < 1e-7 {
			target = last.date
		}
	}
	for i := len(candles) - 1; i >= 0; i-- {
		if candles[i].date < target {
			base = candles[i].close
			a.mu.Lock()
			if len(a.integration.Bases) > 100 {
				a.integration.Bases = map[string]float64{}
			}
			a.integration.Bases[key] = base
			a.mu.Unlock()
			return base, nil
		}
	}
	return 0, errors.New("직전 거래일 종가 미제공")
}
func (a *App) refreshMarkets(ctx context.Context) {
	a.mu.Lock()
	c := a.state.Integration.Market
	epoch := a.epoch
	if !c.Enabled || time.Since(a.integration.MarketChecked) < 2*time.Second || time.Now().Before(a.integration.MarketNext) {
		a.mu.Unlock()
		return
	}
	a.integration.MarketChecked = time.Now()
	previous := a.integration.MarketResult
	a.mu.Unlock()
	now := time.Now().Format(time.RFC3339)
	quotes := []any{}
	problems := []string{}
	result := map[string]any{"enabled": true, "quotes": quotes, "checked": now, "delayed": false}
	coins, e := a.getJSON(ctx, "GET", "https://api.upbit.com/v1/ticker?markets=KRW-BTC%2CKRW-ETH", "", nil)
	if e == nil {
		for _, m := range maps(coins) {
			sym := text(m["market"])
			if sym != "KRW-BTC" && sym != "KRW-ETH" {
				continue
			}
			p, ok := numeric(m["trade_price"])
			pct, pok := numeric(m["signed_change_rate"])
			at := stamp(field(m, "trade_timestamp", "timestamp"))
			if ok && p > 0 {
				q := map[string]any{"symbol": sym, "price": p, "at": at, "checked_at": now, "source": "upbit"}
				if pok {
					q["percent"] = pct * 100
				}
				quotes = append(quotes, q)
			}
		}
	} else {
		problems = append(problems, "업비트 조회 실패")
	}
	if c.ClientID != "" && c.ClientSecret != "" {
		bearer, err := a.tossToken(ctx, c)
		if err == nil {
			v, err := a.getJSON(ctx, "GET", "https://openapi.tossinvest.com/api/v1/market-indicators/prices?symbols=KOSPI%2CKOSDAQ", bearer, nil)
			if err == nil {
				for _, symbol := range []string{"KOSPI", "KOSDAQ"} {
					m := symbolRecord(v, symbol)
					if m == nil {
						problems = append(problems, symbol+" 응답 누락")
						continue
					}
					p, ok := numeric(field(m, "lastPrice", "price", "currentPrice", "value", "close", "tradePrice"))
					if !ok || p <= 0 {
						problems = append(problems, symbol+" 현재가 누락")
						continue
					}
					at := stamp(field(m, "updatedAt", "dt", "time", "asOf", "timestamp", "observation_date"))
					basis := ""
					if at == "" {
						if minute, stamp, err := a.minuteIndex(ctx, symbol, bearer); err == nil {
							p = minute
							at = stamp
							basis = "minute"
						}
					}
					q := map[string]any{"symbol": symbol, "price": p, "at": at, "checked_at": now, "source": "toss", "basis": basis}
					percent, pok := numeric(field(m, "changeRate", "fluctuationRate", "percent"))
					if !pok {
						base, bok := numeric(field(m, "basePrice", "prevClose", "previousClose"))
						if !bok || base <= 0 {
							baseAt := at
							if baseAt == "" {
								baseAt = now
							}
							base, err = a.previousClose(ctx, symbol, bearer, baseAt, p)
							bok = err == nil
						}
						if bok {
							percent = (p - base) / base * 100
							pok = true
						}
					}
					if pok {
						q["percent"] = percent
					}
					quotes = append(quotes, q)
				}
			} else {
				problems = append(problems, "토스 지수 조회 실패")
			}
			a.mu.Lock()
			cachedFX, fxUntil := a.integration.FXCache, a.integration.FXUntil
			a.mu.Unlock()
			if cachedFX != nil && fxUntil.After(time.Now()) {
				result["fx"] = cachedFX
			} else {
				fx, err := a.getJSON(ctx, "GET", "https://openapi.tossinvest.com/api/v1/exchange-rate?baseCurrency=USD&quoteCurrency=KRW", bearer, nil)
				if err == nil {
					for _, m := range maps(fx) {
						if rate, ok := numeric(field(m, "midRate", "rate")); ok && rate > 0 {
							at := stamp(field(m, "validFrom", "updatedAt", "dt", "time", "asOf", "timestamp"))
							if at == "" {
								at = now
							}
							result["fx"] = map[string]any{"rate": rate, "at": at, "stale": false}
							a.mu.Lock()
							if a.epoch == epoch && a.state.Integration.Market.Revision == c.Revision {
								a.integration.FXCache = result["fx"].(map[string]any)
								a.integration.FXUntil = time.Now().Add(time.Minute)
							}
							a.mu.Unlock()
							break
						}
					}
				}
			}
		} else {
			problems = append(problems, "토스 인증 실패")
		}
	}
	result["quotes"] = quotes
	result["delayed"] = len(problems) > 0
	if old, ok := previous["quotes"].([]any); ok {
		present := map[string]bool{}
		for _, q := range quotes {
			present[text(q.(map[string]any)["symbol"])] = true
		}
		for _, q := range old {
			if m, ok := q.(map[string]any); ok && !present[text(m["symbol"])] {
				copy := map[string]any{}
				for k, v := range m {
					copy[k] = v
				}
				copy["stale"] = true
				quotes = append(quotes, copy)
			}
		}
		result["quotes"] = quotes
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if epoch != a.epoch || c.Revision != a.state.Integration.Market.Revision {
		return
	}
	a.integration.MarketResult = result
	a.integration.MarketStatus = strings.Join(problems, " · ")
	if len(problems) == 0 {
		a.integration.MarketStatus = "시세 조회 완료 · " + now
	}
}
