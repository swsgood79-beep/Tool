package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAccountValidationFailureAndOwnerControls(t *testing.T) {
	a := accountFixture(t)
	owner := adminCookie(a)
	u := createAccount(t, a, owner, "qa02")
	id := u.ID
	for _, username := range []string{"qa02", "admin", "<script>", "aa"} {
		w := accountManageCall(a, owner, url.Values{"action": {"create"}, "name": {"추가"}, "username": {username}, "password": {accountMemberPassword}, "confirm": {accountMemberPassword}})
		if w.Code == 303 || len(a.state.AdminAccounts) != 1 {
			t.Fatal("invalid or duplicate account accepted", username)
		}
	}
	cookie, _ := accountLogin(t, a, "qa02", accountMemberPassword)
	before := jsonRaw(a.state.AdminAccounts)
	w := accountManageCall(a, owner, url.Values{"action": {"disable"}, "id": {id}, "revision": {"99"}})
	if w.Code == 303 || !bytes.Equal(before, jsonRaw(a.state.AdminAccounts)) {
		t.Fatal("stale revision changed account")
	}
	a.directHook = func(phase, name string) error {
		if phase == "written" {
			return errors.New("storage failure")
		}
		return nil
	}
	w = accountManageCall(a, owner, url.Values{"action": {"disable"}, "id": {id}, "revision": {"1"}})
	if w.Code == 303 || !bytes.Equal(before, jsonRaw(a.state.AdminAccounts)) || call(a, "GET", "/", nil, cookie).Code != 200 {
		t.Fatal("failed save revoked account")
	}
	a.directHook = nil
	w = call(a, "GET", "/admin-accounts", nil, owner)
	if strings.Contains(w.Body.String(), a.state.AdminAccounts[0].PasswordHash) || strings.Contains(w.Body.String(), accountMemberPassword) {
		t.Fatal("password leaked in management HTML")
	}
	ownerHash := a.state.PasswordHash
	w = accountManageCall(a, owner, url.Values{"action": {"disable"}, "id": {""}, "revision": {"0"}})
	if w.Code == 303 || a.state.PasswordHash != ownerHash {
		t.Fatal("legacy owner disabled via member control")
	}
}

func TestRemoteNamedLoginRespectsExistingAccessSwitch(t *testing.T) {
	a := accountFixture(t)
	owner := adminCookie(a)
	createAccount(t, a, owner, "qa02")
	a.state.Integration.Connection.PublicURL = "https://accounts-test.trycloudflare.com"
	remote := func(cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "https://accounts-test.trycloudflare.com/", nil)
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		return w
	}
	member, _ := accountLogin(t, a, "qa02", accountMemberPassword)
	if w := remote(member); w.Code != 403 {
		t.Fatal("remote switch bypass", w.Code)
	}
	a.state.Integration.MobileEnabled = true
	if w := remote(member); w.Code != 200 {
		t.Fatal("enabled remote member denied", w.Code, w.Body.String())
	}
}
