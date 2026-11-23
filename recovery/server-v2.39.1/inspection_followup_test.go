package main

import (
	"golang.org/x/crypto/bcrypt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestInspectorCompletionUsesApprovalTime(t *testing.T) {
	for _, mode := range []string{"combined", "separate"} {
		t.Run(mode, func(t *testing.T) {
			a := fixture(t)
			cookie := adminCookie(a)
			p := a.state.Projects[0]
			if e := a.requestInspection(p.ID, "combined", "시간 시험", p.SettingForm, p.CombinedForm); e != nil {
				t.Fatal(e)
			}
			requested := time.Now().Add(-48 * time.Hour)
			for i := range a.state.Events {
				a.state.Events[i].RequestedAt = &requested
				a.state.Events[i].Completed = &requested
			}
			before := time.Now()
			if mode == "combined" {
				approveThroughHTTP(t, a, cookie, "combined")
			} else {
				approveThroughHTTP(t, a, cookie, "setting")
				approveThroughHTTP(t, a, cookie, "shipping")
			}
			after := time.Now()
			for _, stage := range []Stage{a.state.Projects[0].Setting, a.state.Projects[0].Shipping, a.state.ShippingRecords[0].Project.Setting, a.state.ShippingRecords[0].Project.Shipping} {
				if stage.Completed == nil || stage.Completed.Before(before) || stage.Completed.After(after) || stage.Inspector == nil || !stage.Inspector.RecordedAt.Equal(*stage.Completed) {
					t.Fatalf("approval time missing: %+v", stage)
				}
			}
			for _, event := range completedEvents(a.state) {
				if event.Completed.Before(before) || event.Completed.After(after) || !event.RequestedAt.Equal(requested) {
					t.Fatal("request/approval times mixed", event)
				}
			}
			at := *a.state.Projects[0].Shipping.Completed
			approveThroughHTTPReplay := call(a, "POST", "/projects/"+fmtID(p.ID)+"/inspection", url.Values{"csrf": {"admin-csrf"}, "inspection": {"shipping"}, "request": {fmtID(a.state.Projects[0].Shipping.EventID)}}, cookie)
			if approveThroughHTTPReplay.Code == 303 || !a.state.Projects[0].Shipping.Completed.Equal(at) {
				t.Fatal("replay changed original approval time")
			}
		})
	}
}

func TestInspectorPCReopenKeepsOpenFormSession(t *testing.T) {
	a := exportFixture(t)
	cookie := adminCookie(a)
	hash, _ := bcrypt.GenerateFromPassword([]byte("native-test-password"), bcrypt.MinCost)
	a.state.PasswordHash = string(hash)
	a.localURL = "http://127.0.0.1:18791"
	page := call(a, "GET", "/inspection-inspectors", nil, cookie)
	if page.Code != 200 || !strings.Contains(page.Body.String(), `value="admin-csrf"`) {
		t.Fatal("form unavailable", page.Code)
	}
	u, _ := url.Parse(a.nativeAdminURL("/"))
	ticket := strings.TrimPrefix(u.Fragment, "ticket=")
	finish := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", a.localURL+"/mobile-admin/passkey/finish", strings.NewReader(string(jsonRaw(map[string]string{"token": ticket}))))
		r.Header.Set("Content-Type", "application/json")
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		return w
	}
	w := finish()
	if w.Code != 200 || len(w.Result().Cookies()) != 0 {
		t.Fatal("PC reopened tab replaced session", w.Code)
	}
	if w = finish(); w.Code != 403 {
		t.Fatal("reused ticket accepted")
	}
	rows := inspectorMissing(&a.state, "")
	w = call(a, "POST", "/inspection-inspectors", url.Values{"csrf": {"admin-csrf"}, "revision": {inspectorRevision(rows)}, "inspector_name": {"시험 검사자"}, "scope": {"all"}, "confirm": {"1"}}, cookie)
	if w.Code != 303 {
		t.Fatal("open form failed after reopening PC", w.Code, w.Body.String())
	}
	w = call(a, "POST", "/inspection-inspectors", url.Values{"csrf": {"forged"}}, cookie)
	if w.Code != 403 {
		t.Fatal("forged CSRF accepted")
	}
}
