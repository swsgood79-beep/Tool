package main

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func receiptQuery(a *App, keys ...string) *httptest.ResponseRecorder {
	return call(a, "POST", "/purchasing-receipts/"+a.state.Board, url.Values{"csrf": {a.state.Board}, "key": keys})
}
func TestPurchasingReceiptPrivacyAndExpiry(t *testing.T) {
	a := accountFixture(t)
	_, buyer := purchaseAccount(t, a, adminCookie(a), "receiptbuyer")
	p := a.state.Projects[0]
	ka, kb, alias := token(), token(), token()
	if w := purchaseSend(a, p.ID, "김승준", ka); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := receiptQuery(a, token()); w.Code != 200 || strings.Contains(w.Body.String(), "김승준") {
		t.Fatal("other phone leaked", w.Body.String())
	}
	purchaseSend(a, p.ID, "강준필", kb)
	purchaseSend(a, p.ID, "김승준", alias)
	var data struct {
		Calls []struct {
			Worker    string
			Completed *time.Time
		}
	}
	w := receiptQuery(a, ka, alias)
	json.Unmarshal(w.Body.Bytes(), &data)
	if len(data.Calls) != 1 || data.Calls[0].Worker != "김승준" || strings.Contains(w.Body.String(), ka) || strings.Contains(w.Body.String(), kb) {
		t.Fatal("receipt keys or another sender leaked", w.Body.String())
	}
	if w := purchaseFinish(a, buyer, a.state.PurchasingCalls[0].ID); w.Code != 200 {
		t.Fatal(w.Code)
	}
	w = receiptQuery(a, ka)
	json.Unmarshal(w.Body.Bytes(), &data)
	if len(data.Calls) != 1 || data.Calls[0].Completed == nil {
		t.Fatal("completed receipt absent", w.Body.String())
	}
	old := time.Now().Add(-61 * time.Second)
	a.state.PurchasingCalls[0].CompletedAt = &old
	w = receiptQuery(a, ka, kb)
	json.Unmarshal(w.Body.Bytes(), &data)
	if len(data.Calls) != 1 || data.Calls[0].Worker != "강준필" {
		t.Fatal("expiry removed another pending sender", w.Body.String())
	}
	if receiptQuery(a, "invalid").Code != 400 {
		t.Fatal("invalid capability")
	}
	if call(a, "GET", "/purchasing-receipts/"+a.state.Board, nil).Code != 405 {
		t.Fatal("method")
	}
	if call(a, "POST", "/purchasing-receipts/"+a.state.Board, url.Values{"csrf": {"wrong"}, "key": {ka}}).Code != 403 {
		t.Fatal("csrf")
	}
	if call(a, "POST", "/purchasing-receipts/wrong", nil).Code != 404 {
		t.Fatal("board")
	}
	if w := receiptQuery(a); w.Code != 200 || strings.Contains(w.Body.String(), "강준필") {
		t.Fatal("empty keys exposed calls")
	}
}
