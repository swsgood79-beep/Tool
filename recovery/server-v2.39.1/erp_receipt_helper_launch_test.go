package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestReceiptHelperReadyProtocolAndLiveWindow(t *testing.T) {
	for _, line := range []string{"ALPHA_HELPER_READY", "ALPHA_HELPER_READY:0", "ALPHA_HELPER_READY:-1", "ALPHA_HELPER_READY:abc", "ALPHA_HELPER_READY:18446744073709551616"} {
		if _, ok := receiptHelperReadyHandle(line); ok {
			t.Fatal("invalid readiness", line)
		}
	}
	h, ok := receiptHelperReadyHandle("ALPHA_HELPER_READY:1234")
	if !ok || h != 1234 {
		t.Fatal("numeric window handle")
	}
	exited := make(chan struct{})
	close(exited)
	called := false
	if receiptHelperVisibleAfter(h, exited, func(uintptr) bool { called = true; return true }, time.Millisecond) || called {
		t.Fatal("exited process treated as opened")
	}
	alive := make(chan struct{})
	if receiptHelperVisibleAfter(h, alive, func(uintptr) bool { return false }, time.Millisecond) {
		t.Fatal("hidden window treated as opened")
	}
	if !receiptHelperVisibleAfter(h, alive, func(v uintptr) bool { return v == h }, time.Millisecond) {
		t.Fatal("live visible window rejected")
	}
}
func TestReceiptHelperSafeFailureStages(t *testing.T) {
	var writer receiptHelperStageWriter
	_, _ = writer.Write([]byte("private-path-and-business-data\nALPHA_HELPER_FAIL"))
	_, _ = writer.Write([]byte("ED:show\r\n"))
	if writer.Stage() != "show" {
		t.Fatal("split stage line lost")
	}
	message := receiptHelperLaunchMessage(receiptHelperLaunchError{Stage: writer.Stage()})
	if !strings.Contains(message, "창 표시") || strings.Contains(message, "private") {
		t.Fatal(message)
	}
	for _, err := range []error{fmt.Errorf("PRIVATE-PASSWORD"), receiptHelperLaunchError{Stage: "PRIVATE-PASSWORD"}} {
		if strings.Contains(receiptHelperLaunchMessage(err), "PRIVATE") {
			t.Fatal("raw error exposed")
		}
	}
	native, err := os.ReadFile("erp_receipt_helper_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(native)
	if strings.Contains(source, "HideWindow: true") || !strings.Contains(source, "CreationFlags: 0x08000000") || !strings.Contains(source, "MonitorFromWindow") || !strings.Contains(source, "owner == pid") {
		t.Fatal("visible GUI launch boundary regressed")
	}
	ps, err := os.ReadFile("erp-receipt-helper.ps1")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(string(ps), "Add-Type -TypeDefinition $filterSource") < strings.Index(string(ps), "$search.Add_Click") {
		t.Fatal("optional native filter blocks helper startup")
	}
}
func TestReceiptHelperLaunchFailureHTTPDoesNotReportOpened(t *testing.T) {
	a, id, _ := cardOutstandingFixture(t)
	owner := adminCookie(a)
	rows := syntheticReceiptRows()
	for i := range rows {
		rows[i].Order = ""
		rows[i].Key = receiptRowKey(rows[i])
	}
	entry := orderOutstandingEntry{Scope: a.outstandingScope(), State: "ready", Rows: rows, At: time.Now(), Expires: time.Now().Add(time.Minute)}
	a.outstandingCards = map[string]orderOutstandingEntry{"TEST-001": entry}
	a.receiptHelperLaunch = func(receiptHelperTask) error { return receiptHelperLaunchError{Stage: "visibility"} }
	f := url.Values{"csrf": {"admin-csrf"}, "project": {id}, "snapshot": {receiptHelperSnapshot(entry)}, "selections": {string(jsonRaw([]receiptHelperSelection{{rows[0].Key, 5}}))}, "launch": {"1"}}
	before := string(jsonRaw(a.state))
	w := call(a, "POST", "/erp-receipt-helper/prepare", f, owner)
	if w.Code != 503 || !strings.Contains(w.Body.String(), "창의 실제 표시") || strings.Contains(w.Body.String(), `"launched":true`) {
		t.Fatal(w.Code, w.Body.String())
	}
	if !a.receiptHelperLast.IsZero() || before != string(jsonRaw(a.state)) {
		t.Fatal("failed launch changed state/rate limit")
	}
}
