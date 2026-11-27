package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTeamNativePushPartialBatchAndDedup(t *testing.T) {
	a, owner, _ := nativePushFixture(t)
	nativeEnroll(t, a, owner, token(), strings.Repeat("K", 80))
	n := TeamNotice{ID: a.state.next(), Phase: "가공 2차", Order: "TEST-ORDER", Part: "TEST-BASE", Worker: "김가공", Quantity: 10, Completed: 10, Required: 20, At: time.Now(), Recipients: []string{""}}
	a.state.TeamNotices = append(a.state.TeamNotices, n)
	calls := 0
	fail := true
	a.client = &http.Client{Transport: nativeRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "oauth2.googleapis.com" {
			return nativeResponse(200, `{"access_token":"test-access","expires_in":3600}`), nil
		}
		raw, _ := io.ReadAll(r.Body)
		var payload map[string]any
		json.Unmarshal(raw, &payload)
		data := payload["message"].(map[string]any)["data"].(map[string]any)
		if data["kind"] != "machining_complete" || data["title"] != "가공 2차 완료" || !strings.Contains(data["body"].(string), "10/20") || data["account_key"] != "" {
			t.Fatal("wrong machining payload", data)
		}
		calls++
		if fail {
			return nativeResponse(503, `{}`), nil
		}
		return nativeResponse(200, `{"name":"test"}`), nil
	})}
	a.pumpNativeMachining(context.Background())
	if calls != 1 || a.state.NativeAdminPush.Devices[0].TeamDelivered >= n.ID {
		t.Fatal("failed delivery advanced cursor")
	}
	fail = false
	a.pumpNativeMachining(context.Background())
	a.pumpNativeMachining(context.Background())
	if calls != 2 || a.state.NativeAdminPush.Devices[0].TeamDelivered != n.ID {
		t.Fatal("notification retry/dedup failed")
	}
}
