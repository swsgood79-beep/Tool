package main

import (
	"context"
	"errors"
	"time"
)

type MinuteQuote struct {
	Price float64
	At    string
	Until time.Time
}

func (a *App) minuteIndex(ctx context.Context, symbol, bearer string) (float64, string, error) {
	a.mu.Lock()
	epoch := a.epoch
	q, ok := a.integration.Minutes[symbol]
	a.mu.Unlock()
	if ok && q.Until.After(time.Now()) {
		return q.Price, q.At, nil
	}
	data, e := a.getJSON(ctx, "GET", "https://openapi.tossinvest.com/api/v1/market-indicators/"+symbol+"/candles?interval=1m&count=10", bearer, nil)
	if e != nil {
		return 0, "", e
	}
	latest := time.Time{}
	var price float64
	var at string
	for _, m := range maps(data) {
		p, ok := numeric(field(m, "closePrice", "close"))
		s := stamp(field(m, "timestamp", "dt", "time"))
		t, e := time.Parse(time.RFC3339, s)
		if !ok || p <= 0 || e != nil || t.After(time.Now().Add(10*time.Minute)) {
			continue
		}
		if t.After(latest) {
			latest = t
			price = p
			at = s
		}
	}
	if price <= 0 {
		return 0, "", errors.New("분봉 값·시각 미제공")
	}
	a.mu.Lock()
	if epoch == a.epoch {
		a.integration.Minutes[symbol] = MinuteQuote{price, at, time.Now().Add(15 * time.Second)}
	}
	a.mu.Unlock()
	return price, at, nil
}
