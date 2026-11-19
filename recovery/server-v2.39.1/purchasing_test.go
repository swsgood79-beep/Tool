package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func purchaseAccount(t *testing.T, a *App, owner *http.Cookie, user string) (*AdminAccount, *http.Cookie) {
	t.Helper()
	w := accountManageCall(a, owner, url.Values{"action": {"create"}, "name": {"구매팀 " + user}, "username": {user}, "role": {"purchasing"}, "password": {accountMemberPassword}, "confirm": {accountMemberPassword}})
	if w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	u := adminAccountByUsername(&a.state, user)
	cookie, login := accountLogin(t, a, user, accountMemberPassword)
	if cookie == nil || login.Code != 303 || login.Header().Get("Location") != "/purchasing" {
		t.Fatal("purchase login target", login.Code, login.Header())
	}
	return u, cookie
}
func purchaseSend(a *App, project int64, worker, key string) *httptest.ResponseRecorder {
	return call(a, "POST", "/purchasing-call/"+a.state.Board+"/"+fmtID(project), url.Values{"csrf": {a.state.Board}, "worker": {worker}, "key": {key}})
}
func purchaseFinish(a *App, cookie *http.Cookie, id int64) *httptest.ResponseRecorder {
	return nativeCall(a, cookie, "/purchasing/complete", url.Values{"id": {fmtID(id)}})
}
func TestPurchasingPermissionsQueueAndPersistence(t *testing.T) {
	a := accountFixture(t)
	owner := adminCookie(a)
	u, buyer := purchaseAccount(t, a, owner, "buyer01")
	other, otherCookie := purchaseAccount(t, a, owner, "buyer02")
	a.state.Integration.MobileEnabled = true
	for _, path := range []string{"/projects/new", "/inspection-requests", "/admin-accounts", "/android-admin-settings", "/client-notes", "/shipping-archive-admin", "/native-admin/view?mode=worker", "/work/" + a.state.Board, "/community/" + a.state.Board, "/purchasing-settings", "/my-password"} {
		if w := call(a, "GET", path, nil, buyer); w.Code != 403 {
			t.Fatal("purchase GET leaked", path, w.Code)
		}
	}
	for _, path := range []string{"/projects/new", "/connection", "/settings", "/admin-accounts", "/native-admin/view", "/work/" + a.state.Board} {
		if w := nativeCall(a, buyer, path, url.Values{}); w.Code != 403 {
			t.Fatal("purchase POST leaked", path, w.Code)
		}
	}
	if w := call(a, "GET", "/", nil, buyer); w.Code != 303 || w.Header().Get("Location") != "/purchasing" {
		t.Fatal("purchase root")
	}
	if w := call(a, "GET", "/purchasing", nil, owner); w.Code != 403 {
		t.Fatal("wrong role")
	}
	p := a.state.Projects[0]
	before := jsonRaw(a.state.Projects)
	events := jsonRaw(a.state.Events)
	key := token()
	if w := purchaseSend(a, p.ID, "김승준", key); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	first := a.state.PurchasingCalls[0]
	if first.RecipientID != u.ID {
		t.Fatal("wrong recipient")
	}
	alias := token()
	if w := purchaseSend(a, p.ID, "김승준", alias); w.Code != 200 || len(a.state.PurchasingCalls) != 1 {
		t.Fatal("duplicate pending")
	}
	if w := purchaseSend(a, p.ID, "강준필", token()); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if !bytes.Equal(before, jsonRaw(a.state.Projects)) || !bytes.Equal(events, jsonRaw(a.state.Events)) {
		t.Fatal("call affected inspection")
	}
	if w := purchaseFinish(a, otherCookie, first.ID); w.Code != 400 {
		t.Fatal("other buyer completed call")
	}
	if w := call(a, "POST", "/purchasing/complete", url.Values{"csrf": {"wrong"}, "id": {fmtID(first.ID)}}, buyer); w.Code != 403 {
		t.Fatal("csrf")
	}
	if w := call(a, "GET", "/purchasing/status?call="+fmtID(first.ID), nil, otherCookie); w.Code != 404 {
		t.Fatal("other buyer details")
	}
	if w := call(a, "GET", "/purchasing/status?page=99999999999999", nil, buyer); w.Code != 200 {
		t.Fatal("page clamp")
	}
	a.directHook = func(phase, name string) error {
		if phase == "written" {
			return errors.New("injected persistence failure")
		}
		return nil
	}
	if w := purchaseFinish(a, buyer, first.ID); w.Code != 400 || a.state.PurchasingCalls[0].CompletedAt != nil {
		t.Fatal("complete rollback")
	}
	a.directHook = nil
	if w := purchaseFinish(a, buyer, first.ID); w.Code != 200 || a.state.PurchasingCalls[0].CompletedAt == nil {
		t.Fatal("complete")
	}
	if w := purchaseSend(a, p.ID, "김승준", key); w.Code != 200 || len(a.state.PurchasingCalls) != 2 {
		t.Fatal("retry revived completed call")
	}
	if w := purchaseSend(a, p.ID, "김승준", alias); w.Code != 200 || len(a.state.PurchasingCalls) != 2 {
		t.Fatal("duplicate alias replay after completion")
	}
	if w := call(a, "GET", "/purchasing/status?call="+fmtID(first.ID), nil, buyer); strings.Contains(w.Body.String(), key) || strings.Contains(w.Body.String(), alias) {
		t.Fatal("idempotency keys exposed")
	}
	if w := purchaseFinish(a, buyer, first.ID); w.Code != 200 || len(purchasingList(&a.state, u.ID, true)) != 1 {
		t.Fatal("repeat finish")
	}
	a.state.PurchasingRecipient = other.ID
	if w := purchaseSend(a, p.ID, "새 작업자", token()); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if len(purchasingList(&a.state, other.ID, false)) != 1 || len(purchasingList(&a.state, u.ID, false)) != 1 {
		t.Fatal("recipient routing")
	}
	if err := a.commitAnnouncement(func(s *State) error { s.Projects = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	fresh, err := newApp(a.dir)
	if err != nil || len(fresh.state.PurchasingCalls) != 3 || adminAccountByUsername(&fresh.state, "buyer01").Role != "purchasing" {
		t.Fatal("restart durability", err)
	}
	if len(purchasingList(&fresh.state, u.ID, false)) != 1 {
		t.Fatal("source removal hid queue")
	}
}
func TestPurchasingNativePersistentLoginAndPushIsolation(t *testing.T) {
	a, owner, _ := nativePushFixture(t)
	u, buyer := purchaseAccount(t, a, owner, "buyerpush")
	_, otherCookie := purchaseAccount(t, a, owner, "buyother")
	purchaseDevice, adminDevice, otherDevice := token(), token(), token()
	nativeEnroll(t, a, owner, adminDevice, strings.Repeat("A", 80))
	nativeEnroll(t, a, buyer, purchaseDevice, strings.Repeat("B", 80))
	nativeEnroll(t, a, otherCookie, otherDevice, strings.Repeat("C", 80))
	if nativeDevice(&a.state, purchaseDevice).Role != "purchasing" {
		t.Fatal("device role")
	}
	grant := enrollNativeLogin(t, a, buyer, purchaseDevice)
	restored := restoredCookie(t, nativeLoginCall(a, nil, "/native-admin/session/restore", grant))
	if w := call(a, "GET", "/purchasing", nil, restored); w.Code != 200 {
		t.Fatal("restored purchase role")
	}
	if w := call(a, "GET", "/inspection-requests", nil, restored); w.Code != 403 {
		t.Fatal("restored role escalated")
	}
	p := a.state.Projects[0]
	purchaseSend(a, p.ID, "작업자", token())
	a.requestInspection(p.ID, "combined", "검사 작업자", p.SettingForm, p.CombinedForm)
	sent := []map[string]any{}
	fail := true
	a.client = &http.Client{Transport: nativeRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "oauth2.googleapis.com" {
			return nativeResponse(200, `{"access_token":"mock-access","expires_in":3600}`), nil
		}
		if r.URL.Host != "fcm.googleapis.com" {
			t.Fatal("unexpected host")
		}
		raw, _ := io.ReadAll(r.Body)
		var out map[string]any
		json.Unmarshal(raw, &out)
		msg := out["message"].(map[string]any)
		if msg["token"] == strings.Repeat("B", 80) && fail {
			return nativeResponse(503, `{}`), nil
		}
		sent = append(sent, msg)
		return nativeResponse(200, `{"name":"projects/test/messages/1"}`), nil
	})}
	before := nativeDevice(&a.state, purchaseDevice).Delivered
	a.pumpPurchasing(context.Background())
	if nativeDevice(&a.state, purchaseDevice).Delivered != before {
		t.Fatal("failed call marked delivered")
	}
	fail = false
	a.pumpNativeAdmin(context.Background())
	a.pumpPurchasing(context.Background())
	count := len(sent)
	a.pumpNativeAdmin(context.Background())
	a.pumpPurchasing(context.Background())
	if len(sent) != count {
		t.Fatal("duplicate notification")
	}
	seenAdmin, seenPurchase := false, false
	for _, msg := range sent {
		data := msg["data"].(map[string]any)
		tok := msg["token"].(string)
		if tok == strings.Repeat("C", 80) {
			t.Fatal("other buyer notified")
		}
		if tok == strings.Repeat("B", 80) {
			seenPurchase = true
			if data["kind"] != "purchasing_call" || data["account_key"] != u.ID {
				t.Fatal("buyer got inspection or wrong account", data)
			}
		}
		if tok == strings.Repeat("A", 80) {
			seenAdmin = true
			if data["kind"] == "purchasing_call" {
				t.Fatal("admin got purchasing call")
			}
		}
	}
	if !seenAdmin || !seenPurchase {
		t.Fatal("missing isolated push", sent)
	}
	fresh, err := newApp(a.dir)
	if err != nil {
		t.Fatal(err)
	}
	fresh.state.Integration.MobileEnabled = true
	again := restoredCookie(t, nativeLoginCall(fresh, nil, "/native-admin/session/restore", grant))
	if w := call(fresh, "GET", "/purchasing", nil, again); w.Code != 200 {
		t.Fatal("cold restart did not preserve purchasing login")
	}
	if w := nativeCall(fresh, again, "/logout", url.Values{}); w.Code != 303 {
		t.Fatal("logout", w.Code)
	}
	if w := nativeLoginCall(fresh, nil, "/native-admin/session/restore", grant); w.Code != 403 {
		t.Fatal("logout grant restored")
	}
	if !nativeDevice(&fresh.state, adminDevice).Enabled {
		t.Fatal("purchase logout affected admin")
	}
}
func TestPurchasingBrowser(t *testing.T) {
	if os.Getenv("PURCHASING_BROWSER") != "1" {
		t.Skip("optional actual purchasing queue browser")
	}
	a := accountFixture(t)
	owner := adminCookie(a)
	_, buyer := purchaseAccount(t, a, owner, "buyerbrowser")
	a.state.Integration.MobileEnabled = true
	a.state.Projects[0].Name = "월드에프엘와이 4축 1종"
	a.state.Projects[0].Number = "A261028-002"
	p := a.state.Projects[0]
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg := map[string]any{"cookie": map[string]string{"name": buyer.Name, "value": buyer.Value}, "owner": map[string]string{"name": owner.Name, "value": owner.Value}, "board": a.state.Board, "project": p.ID}
	cmd := exec.Command("node", "audit/browser-purchasing-v1.4.78.cjs", srv.URL, string(jsonRaw(cfg)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if len(a.state.PurchasingCalls) != 8 || len(purchasingList(&a.state, a.state.PurchasingRecipient, false)) != 0 {
		t.Fatal("queue after browser", len(a.state.PurchasingCalls))
	}
}
