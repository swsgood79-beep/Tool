package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func TestNativePasswordFormContractAndAccountUsername(t *testing.T) {
	a := accountFixture(t)
	a.state.Integration.MobileEnabled = true
	c := adminCookie(a)
	createAccount(t, a, c, "qa_choice")
	server := httptest.NewTLSServer(a)
	defer server.Close()
	for _, tc := range []struct {
		username, password string
		owner              bool
	}{{"wssong", accountOwnerPassword, true}, {"qa_choice", accountMemberPassword, false}} {
		client := server.Client()
		jar, _ := cookiejar.New(nil)
		client.Jar = jar
		client.CheckRedirect = func(r *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
		r, e := client.Get(server.URL + "/login")
		if e != nil {
			t.Fatal(e)
		}
		body, _ := io.ReadAll(r.Body)
		r.Body.Close()
		m := regexp.MustCompile(`name="csrf"\s+value="([A-Za-z0-9_-]{43})"`).FindSubmatch(body)
		if len(m) != 2 {
			t.Fatal("native login nonce contract changed")
		}
		root, _ := url.Parse(server.URL)
		for _, v := range jar.Cookies(root) {
			if v.Name == a.nonceCookie() {
				t.Fatal("login nonce incorrectly visible at root; native POST must use /login cookies")
			}
		}
		r, e = client.PostForm(server.URL+"/login", url.Values{"csrf": {string(m[1])}, "login_mode": {"account"}, "username": {tc.username}, "password": {tc.password}})
		if e != nil {
			t.Fatal(e)
		}
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
		if r.StatusCode != 303 {
			t.Fatal("native password login", tc.username, r.StatusCode)
		}
		r, e = client.Get(server.URL + "/native-admin/bootstrap")
		if e != nil {
			t.Fatal(e)
		}
		var j map[string]any
		if e = json.NewDecoder(r.Body).Decode(&j); e != nil {
			t.Fatal(e)
		}
		r.Body.Close()
		if r.StatusCode != 200 || j["owner"] != tc.owner || j["account_username"] != tc.username {
			t.Fatal("incorrect password-account binding", j["owner"], j["account_username"])
		}
		data := string(jsonRaw(j))
		if strings.Contains(data, a.state.PasswordHash) {
			t.Fatal("password hash exposed in app identity")
		}
	}
}
