package main

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func scheduleMockResponse(r *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}
}
func TestScheduleDirectDownloadBeforeAPI(t *testing.T) {
	a := fixture(t)
	book := scheduleFixture(t)
	calls := 0
	a.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host != "onedrive.live.com" || r.URL.Query().Get("download") != "1" {
			t.Fatal("not direct", r.URL.Host)
		}
		if r.Header.Get("Cache-Control") != "no-cache" || !strings.Contains(r.Header.Get("User-Agent"), "AlphaWorkComplete/") {
			t.Fatal("download headers absent")
		}
		return scheduleMockResponse(r, 200, string(book)), nil
	})}
	b, e := a.readSchedule(context.Background(), ScheduleConfig{Mode: "url", URL: "https://onedrive.live.com/view?resid=fixture&authkey=fixture"})
	if e != nil || string(b) != string(book) || calls != 1 {
		t.Fatal("direct download failed", e, calls)
	}
}
func TestScheduleModernRedeemFallbackAndStorageRedirect(t *testing.T) {
	a := fixture(t)
	book := scheduleFixture(t)
	short := "https://1drv.ms/x/c/fixture/share?e=fixture"
	raw := "https://onedrive.live.com/:x:/g/personal/fixture/view?redeem=" + base64.RawURLEncoding.EncodeToString([]byte(short))
	seen := []string{}
	a.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		seen = append(seen, r.URL.Host)
		switch r.URL.Host {
		case "onedrive.live.com", "1drv.ms":
			return scheduleMockResponse(r, 400, "bad"), nil
		case "api.onedrive.com":
			share := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1.0/shares/u!"), "/root/content")
			decoded, err := base64.RawURLEncoding.DecodeString(share)
			if err != nil || string(decoded) != short {
				t.Fatal("original short share not used")
			}
			res := scheduleMockResponse(r, 302, "")
			res.Header.Set("Location", "https://fixture.storage.live.com/download.xlsx")
			return res, nil
		case "fixture.storage.live.com":
			return scheduleMockResponse(r, 200, string(book)), nil
		default:
			t.Fatal("unexpected host", r.URL.Host)
			return nil, nil
		}
	})}
	b, e := a.readSchedule(context.Background(), ScheduleConfig{Mode: "url", URL: raw})
	if e != nil || string(b) != string(book) || len(seen) != 4 {
		t.Fatal("fallback failed", e, seen)
	}
}
func TestScheduleHTMLFallbackAndFailureKeepsDatesPrivate(t *testing.T) {
	a := fixture(t)
	book := scheduleFixture(t)
	raw := "https://onedrive.live.com/view?authkey=private-fixture"
	a.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "onedrive.live.com" {
			return scheduleMockResponse(r, 200, "<html>sign in</html>"), nil
		}
		return scheduleMockResponse(r, 200, string(book)), nil
	})}
	if _, e := a.readSchedule(context.Background(), ScheduleConfig{Mode: "url", URL: raw}); e != nil {
		t.Fatal("HTML did not fall back", e)
	}
	a.state.Integration.Schedule = ScheduleConfig{Mode: "url", URL: raw, Enabled: true}
	a.integration.ScheduleStatus["success"] = "2026-11-12T10:00:00+09:00"
	a.state.Projects[0].ShippingDueDate = "2026-11-20"
	a.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		return scheduleMockResponse(r, 400, "private response must not display"), nil
	})}
	a.syncSchedule(context.Background(), true)
	message := text(a.integration.ScheduleStatus["error"])
	if text(a.integration.ScheduleStatus["success"]) != "2026-11-12T10:00:00+09:00" {
		t.Fatal("failure lost last successful check")
	}
	if a.state.Projects[0].ShippingDueDate != "2026-11-20" || !strings.Contains(message, "HTTP 400") || strings.Contains(message, "private-fixture") || strings.Contains(message, "private response") {
		t.Fatal("failed download changed data or leaked link", message)
	}
}
func TestScheduleRejectsForeignRedirectAndRedeem(t *testing.T) {
	a := fixture(t)
	raw := "https://onedrive.live.com/view?redeem=" + url.QueryEscape(base64.RawURLEncoding.EncodeToString([]byte("https://evil.example/file")))
	c, e := scheduleDownloadCandidates(raw)
	if e != nil || len(c) != 2 {
		t.Fatal("foreign redeem used", e)
	}
	a.client = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if !scheduleURLAllowed(r.URL.String()) {
			t.Fatal("foreign host reached")
		}
		res := scheduleMockResponse(r, 302, "")
		res.Header.Set("Location", "https://evil.example/file")
		return res, nil
	})}
	if _, e := a.readSchedule(context.Background(), ScheduleConfig{Mode: "url", URL: raw}); e == nil {
		t.Fatal("foreign redirect allowed")
	}
	if !scheduleURLAllowed("https://fixture.storage.live.com/download") {
		t.Fatal("Microsoft storage redirect rejected")
	}
	if scheduleURLAllowed("https://fixture.storage.live.com.evil.example/file") {
		t.Fatal("lookalike allowed")
	}
}
