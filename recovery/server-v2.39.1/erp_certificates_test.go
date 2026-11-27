package main

import (
	"bytes"
	"context"
	"github.com/xuri/excelize/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"
)

const certificatePath = "/machining-admin/certificates"

func certificateWait(t *testing.T, a *App) {
	t.Helper()
	end := time.Now().Add(2 * time.Second)
	for {
		a.mu.Lock()
		busy := a.erpCertificates.Busy
		a.mu.Unlock()
		if !busy {
			return
		}
		if time.Now().After(end) {
			t.Fatal("stalled")
		}
		time.Sleep(time.Millisecond)
	}
}
func syntheticCertificates(order string) ERPDBReport {
	return ERPDBReport{Status: "certificates", Connected: true, Certificates: []ERPCertificateRow{{Unit: "100", Order: order, Customer: "가상 거래처", Client: "가상 최종고객", Project: "가상 설비", Item: "TEST-ROBOT-AR160-PU1-800-75B-H-EL-LONG-PART-NUMBER", Serial: "0000042", Remark: "=HYPERLINK(\"https://example.invalid\",\"test\")", CommandDate: "20261112", CommandNo: "TEST-CMD-001", Sequence: 1, Registered: true, InspectionDate: "20261120", Judgment: "TEST", Certificate: "TEST-TEMPLATE"}, {Unit: "100", Order: order, Item: "가상 미등록 품번", CommandDate: "20261112", CommandNo: "TEST-CMD-001", Sequence: 2}}}
}
func TestERPCertificateScopeSnapshotAndXLSX(t *testing.T) {
	a := fixture(t)
	owner := adminCookie(a)
	worker, _ := machiningWorker(t, a, "cert-worker")
	csrf := a.sessions[hashToken(owner.Value)].CSRF
	for _, p := range []string{certificatePath, certificatePath + "/status", certificatePath + "/export"} {
		if w := call(a, "GET", p, nil, worker); w.Code != 403 {
			t.Fatal("worker access", p, w.Code)
		}
	}
	if w := call(a, "POST", certificatePath, url.Values{"order": {"TEST-001"}}, owner); w.Code != 403 {
		t.Fatal("csrf")
	}
	before := string(jsonRaw(a.state))
	a.erpCertificateProbe = func(ctx context.Context, _ string) ERPDBReport {
		return syntheticCertificates(ctx.Value(erpCertificateKey{}).(string))
	}
	f := url.Values{"csrf": {csrf}, "order": {"TEST-001"}}
	if w := call(a, "POST", certificatePath, f, owner); w.Code != 202 {
		t.Fatal(w.Code, w.Body.String())
	}
	certificateWait(t, a)
	a.mu.Lock()
	id := a.erpCertificates.ID
	a.mu.Unlock()
	w := call(a, "GET", certificatePath+"/export?id="+id, nil, owner)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	x, e := excelize.OpenReader(bytes.NewReader(w.Body.Bytes()))
	if e != nil {
		t.Fatal(e)
	}
	defer x.Close()
	sheet := "검사성적서 목록"
	for cell, want := range map[string]string{"G2": "0000042", "N2": "등록", "N3": "미등록", "B2": "TEST-001"} {
		v, _ := x.GetCellValue(sheet, cell)
		if v != want {
			t.Fatal(cell, v)
		}
	}
	formula, _ := x.GetCellFormula(sheet, "H2")
	if formula != "" {
		t.Fatal("ERP text became formula")
	}
	if before != string(jsonRaw(a.state)) {
		t.Fatal("operating data changed")
	}
	if w := call(a, "GET", certificatePath+"/export?id=stale", nil, owner); w.Code != 409 {
		t.Fatal("stale token")
	}
	a.mu.Lock()
	a.erpCertificates.At = time.Now().Add(-11 * time.Minute)
	a.mu.Unlock()
	if w := call(a, "GET", certificatePath+"/export?id="+id, nil, owner); w.Code != 409 {
		t.Fatal("expired export")
	}
}
func TestERPCertificateFailuresClearPreviousResult(t *testing.T) {
	for _, mode := range []string{"failed", "truncated", "wrong-order", "duplicate", "zero"} {
		t.Run(mode, func(t *testing.T) {
			a := fixture(t)
			owner := adminCookie(a)
			csrf := a.sessions[hashToken(owner.Value)].CSRF
			a.erpCertificates = &erpCertificateJob{ID: "old", OK: true, At: time.Now(), Rows: syntheticCertificates("OLD").Certificates}
			a.erpCertificateProbe = func(ctx context.Context, _ string) ERPDBReport {
				r := syntheticCertificates("TEST")
				switch mode {
				case "failed":
					r.Status = "metadata_failed"
				case "truncated":
					r.Truncated = true
				case "wrong-order":
					r.Certificates[0].Order = "OTHER"
				case "duplicate":
					r.Certificates = append(r.Certificates, r.Certificates[0])
				case "zero":
					r.Certificates = nil
				}
				return r
			}
			call(a, "POST", certificatePath, url.Values{"csrf": {csrf}, "order": {"TEST"}}, owner)
			certificateWait(t, a)
			a.mu.Lock()
			j := a.erpCertificates
			ok := j.OK
			id := j.ID
			rows := len(j.Rows)
			a.mu.Unlock()
			if (mode == "zero") != ok || rows != 0 {
				t.Fatal("false success or stale rows", mode, ok, rows)
			}
			w := call(a, "GET", certificatePath+"/export?id="+id, nil, owner)
			if mode != "zero" && w.Code != 409 {
				t.Fatal("partial export")
			}
		})
	}
}
func TestERPCertificateQueryReadOnlyAndKeys(t *testing.T) {
	q := regexp.MustCompile(`(?s)\$certificateSql=@'\s*(.*?)\s*'@`).FindStringSubmatch(erpCertificateScript)
	if len(q) != 2 {
		t.Fatal("missing SQL")
	}
	if regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE|EXEC|CREATE|DROP|ALTER|MERGE)\b`).MatchString(q[1]) {
		t.Fatal("write SQL")
	}
	for _, v := range []string{"TOP (5001)", "m.RCV_ORDNO=@Order", "h.UNT_CD=m.UNT_CD", "h.PDTCMD_DATE=m.PDTCMD_DATE", "h.PDTCMD_NO=m.PDTCMD_NO", "c.PDTCMD_SEQ=h.PDTCMD_SEQ", "ord.RCVORD_DATE=m.RCV_ORDDATE", "ord.UNT_CD=m.UNT_CD"} {
		if !strings.Contains(q[1], v) {
			t.Fatal(v)
		}
	}
	cmd := erpDBPowerShellCommand(context.WithValue(context.Background(), erpCertificateKey{}, "TEST'123"), "")
	b, _ := io.ReadAll(cmd.Stdin)
	if !strings.Contains(string(b), "$CertificateOrder='TEST''123'") || strings.Contains(string(b), "# CERTIFICATE_SCRIPT") {
		t.Fatal("transport escaping")
	}
	r := decodeDBProbe(jsonRaw(ERPDBReport{Status: "certificates", Certificates: make([]ERPCertificateRow, 5001)}))
	if len(r.Certificates) != 5000 || !r.Truncated {
		t.Fatal("cap")
	}
	for _, s := range []string{"", "TEST\n001", strings.Repeat("x", 101)} {
		if validCertificateOrder(s) {
			t.Fatal("bad order")
		}
	}
}
func TestERPCertificateBrowser122(t *testing.T) {
	if os.Getenv("ALPHA122_BROWSER") != "1" {
		t.Skip("optional Chromium")
	}
	a := fixture(t)
	owner := adminCookie(a)
	a.erpCertificateProbe = func(ctx context.Context, _ string) ERPDBReport {
		r := syntheticCertificates(ctx.Value(erpCertificateKey{}).(string))
		r.Certificates[0].Remark = "가상 시험용 비고"
		return r
	}
	server := httptest.NewServer(http.HandlerFunc(a.ServeHTTP))
	defer server.Close()
	cmd := exec.Command("node", "audit/browser-certificates-v1.4.122.cjs", server.URL, string(jsonRaw(map[string]string{"name": owner.Name, "value": owner.Value})))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
