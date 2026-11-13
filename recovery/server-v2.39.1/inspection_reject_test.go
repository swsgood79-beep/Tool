package main

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pendingReview(t *testing.T, kind string) (*App, int64) {
	t.Helper()
	a := fixture(t)
	a.state.Projects[0].SpecialInstructions = "보존 특이사항"
	a.state.Projects[0].SuppliedMaterials = "보존 사급품"
	p := a.state.Projects[0]
	if kind == "shipping" {
		if e := a.requestInspection(p.ID, "setting", "세팅 작업자", p.SettingForm, p.CombinedForm); e != nil {
			t.Fatal(e)
		}
		if e := a.approve(p.ID, "setting", a.state.Projects[0].Setting.EventID); e != nil {
			t.Fatal(e)
		}
	}
	p = a.state.Projects[0]
	token := p.SettingForm
	if kind == "shipping" {
		token = p.ShippingForm
	}
	if e := a.requestInspection(p.ID, kind, "신청 작업자", token, p.CombinedForm); e != nil {
		t.Fatal(e)
	}
	id := a.state.Projects[0].Setting.EventID
	if kind == "shipping" {
		id = a.state.Projects[0].Shipping.EventID
	}
	return a, id
}
func rejectHTML(t *testing.T, a *App, kind, name string) {
	t.Helper()
	if out := os.Getenv("REJECT_HTML_OUT"); out != "" {
		dir := filepath.Join(out, kind)
		if e := os.MkdirAll(dir, 0700); e != nil {
			t.Fatal(e)
		}
		path := "/projects/1/inspection"
		if strings.HasPrefix(name, "worker") {
			path = "/work/" + a.state.Board + "?entry=skip"
		}
		r := call(a, "GET", path, nil, adminCookie(a))
		if r.Code != 200 {
			t.Fatal("review render failed", r.Code, r.Body.String())
		}
		if e := os.WriteFile(filepath.Join(dir, name), r.Body.Bytes(), 0600); e != nil {
			t.Fatal(e)
		}
	}
}
func TestRejectInspectionAllowsNewRequestAndPreservesApprovedStages(t *testing.T) {
	for _, kind := range []string{"setting", "shipping", "combined"} {
		t.Run(kind, func(t *testing.T) {
			a, id := pendingReview(t, kind)
			before := a.state.Projects[0]
			rejectHTML(t, a, kind, "pending.html")
			if e := a.rejectInspection(1, kind, id); e != nil {
				t.Fatal(e)
			}
			p := a.state.Projects[0]
			if p.RejectedInspection != kind || p.Setting.Pending || p.Shipping.Pending || p.CombinedPending || p.Shipping.Completed != nil || p.SpecialInstructions != "보존 특이사항" || p.SuppliedMaterials != "보존 사급품" {
				t.Fatal("wrong rejection state", p)
			}
			if kind == "shipping" {
				if p.Setting.Completed == nil || p.Setting.EventID != before.Setting.EventID {
					t.Fatal("shipping rejection lost setting approval")
				}
			} else if p.Setting.Completed != nil {
				t.Fatal("rejected setting completed")
			}
			rejected := 0
			for _, e := range a.state.Events {
				if e.Rejected {
					rejected++
					if e.Pending || e.Unread || e.Completed != nil || e.RejectedAt == nil || notifyEvent(e) {
						t.Fatal("rejected event became completion/alert")
					}
				}
			}
			want := 1
			if kind == "combined" {
				want = 2
			}
			if rejected != want {
				t.Fatal("missing rejection history")
			}
			wantCompleted := 0
			if kind == "shipping" {
				wantCompleted = 1
			}
			if len(completedEvents(a.state)) != wantCompleted {
				t.Fatal("rejected request exported as complete")
			}
			if len(a.view("test")["Requests"].([]Event)) != 0 {
				t.Fatal("rejected request stayed in queue")
			}
			reopened, e := newApp(a.dir)
			if e != nil || reopened.state.Projects[0].RejectedInspection != kind {
				t.Fatal("restart lost rejection", e)
			}
			legacy, e := importLegacy(a.dir)
			if e != nil || legacy.Projects[0].RejectedInspection != kind {
				t.Fatal("original fields lost rejection", e)
			}
			found := false
			for _, event := range legacy.Events {
				if event.ID == id && event.Rejected && event.RejectedAt != nil {
					found = true
				}
			}
			if !found {
				t.Fatal("original event projection lost rejection")
			}
			rejectHTML(t, a, kind, "rejected.html")
			rejectHTML(t, a, kind, "worker-rejected.html")
			if e := a.approve(1, kind, id); e == nil {
				t.Fatal("stale approve accepted")
			}
			token := before.SettingForm
			if kind == "shipping" {
				token = before.ShippingForm
			}
			if e := a.requestInspection(1, kind, "재신청자", token, before.CombinedForm); e == nil {
				t.Fatal("old submission token reused")
			}
			p = a.state.Projects[0]
			token = p.SettingForm
			if kind == "shipping" {
				token = p.ShippingForm
			}
			if e := a.requestInspection(1, kind, "재신청자", token, p.CombinedForm); e != nil {
				t.Fatal("new request denied", e)
			}
			newID := a.state.Projects[0].Setting.EventID
			if kind == "shipping" {
				newID = a.state.Projects[0].Shipping.EventID
			}
			if newID == id || a.state.Projects[0].RejectedInspection != "" {
				t.Fatal("new request not independent")
			}
			if e := a.rejectInspection(1, kind, id); e == nil {
				t.Fatal("old rejection changed new request")
			}
			if e := a.approve(1, kind, newID); e != nil {
				t.Fatal("new request cannot be approved", e)
			}
			if kind == "setting" && a.state.Projects[0].Setting.Completed == nil {
				t.Fatal("setting not saved")
			}
			if kind != "setting" && a.state.Projects[0].Shipping.Completed == nil {
				t.Fatal("shipping not saved")
			}
		})
	}
}
func TestRejectEndpointRequiresAdminAndCSRF(t *testing.T) {
	a, id := pendingReview(t, "setting")
	form := url.Values{"inspection": {"setting"}, "request": {fmtID(id)}, "action": {"reject"}}
	r := call(a, "POST", "/projects/1/inspection", form)
	if r.Code == 303 && r.Header().Get("Location") != "/login" {
		t.Fatal("anonymous rejected inspection")
	}
	if !a.state.Projects[0].Setting.Pending {
		t.Fatal("unauthorized state change")
	}
	cookie := adminCookie(a)
	form.Set("csrf", "wrong")
	r = call(a, "POST", "/projects/1/inspection", form, cookie)
	if r.Code != 403 || !a.state.Projects[0].Setting.Pending {
		t.Fatal("bad CSRF accepted")
	}
	form.Set("csrf", a.sessions[hashToken(cookie.Value)].CSRF)
	r = call(a, "POST", "/projects/1/inspection", form, cookie)
	if r.Code != 303 || a.state.Projects[0].Setting.Pending {
		t.Fatal("admin rejection failed", r.Code, r.Body.String())
	}
	r = call(a, "GET", "/records/"+fmtID(id)+"/edit", nil, cookie)
	if r.Code != 200 || !strings.Contains(r.Body.String(), "반려된 검사 신청") {
		t.Fatal("rejection history page missing")
	}
}
func TestConcurrentApproveAndRejectOnlyOneSucceeds(t *testing.T) {
	a, id := pendingReview(t, "setting")
	cookie := adminCookie(a)
	csrf := a.sessions[hashToken(cookie.Value)].CSRF
	codes := make(chan int, 2)
	for _, action := range []string{"approve", "reject"} {
		go func(action string) {
			r := call(a, "POST", "/projects/1/inspection", url.Values{"csrf": {csrf}, "inspection": {"setting"}, "request": {fmtID(id)}, "action": {action}}, cookie)
			codes <- r.Code
		}(action)
	}
	c1, c2 := <-codes, <-codes
	if !((c1 == 303 && c2 == 409) || (c1 == 409 && c2 == 303)) {
		t.Fatal("both decisions accepted", c1, c2)
	}
	p := a.state.Projects[0]
	if (p.Setting.Completed != nil) == (p.RejectedInspection != "") {
		t.Fatal("mixed final state")
	}
}
func TestCombinedCannotBeRejectedWithSingleStageForm(t *testing.T) {
	a, id := pendingReview(t, "combined")
	before, _ := json.Marshal(a.state)
	if e := a.rejectInspection(1, "setting", id); e == nil {
		t.Fatal("partial rejection accepted")
	}
	after, _ := json.Marshal(a.state)
	if string(before) != string(after) {
		t.Fatal("invalid action mutated combined request")
	}
}

func TestRejectShippingAfterPartialCombinedApprovalKeepsSetting(t *testing.T) {
	a, id := pendingReview(t, "combined")
	if e := a.approve(1, "setting", id); e != nil {
		t.Fatal(e)
	}
	shippingID := a.state.Projects[0].Shipping.EventID
	if e := a.rejectInspection(1, "shipping", shippingID); e != nil {
		t.Fatal(e)
	}
	p := a.state.Projects[0]
	if p.Setting.Completed == nil || p.Shipping.Completed != nil || p.Shipping.Pending || len(completedEvents(a.state)) != 1 {
		t.Fatal("partial combined rejection reset completed setting")
	}
}

func TestRejectedHistoryDeletionDoesNotCancelNewRequest(t *testing.T) {
	a, id := pendingReview(t, "setting")
	if e := a.rejectInspection(1, "setting", id); e != nil {
		t.Fatal(e)
	}
	p := a.state.Projects[0]
	if e := a.requestInspection(1, "setting", "재신청자", p.SettingForm, p.CombinedForm); e != nil {
		t.Fatal(e)
	}
	current := a.state.Projects[0].Setting.EventID
	cookie := adminCookie(a)
	r := call(a, "POST", "/records/"+fmtID(id)+"/edit", url.Values{"csrf": {a.sessions[hashToken(cookie.Value)].CSRF}, "action": {"delete"}, "confirm": {"1"}, "revision": {"2"}}, cookie)
	if r.Code != 303 || !a.state.Projects[0].Setting.Pending || a.state.Projects[0].Setting.EventID != current {
		t.Fatal("history deletion changed new request", r.Code, r.Body.String())
	}
}
