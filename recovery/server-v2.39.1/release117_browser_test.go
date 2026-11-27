package main

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestRelease117SupplierBrowser(t *testing.T) {
	if os.Getenv("ALPHA117_BROWSER") != "1" {
		t.Skip("optional Chromium verification")
	}
	a := fixture(t)
	worker, _ := machiningWorker(t, a, "supplier-browser")
	a.state.MachiningJobs = []MachiningJob{{ID: "synthetic", Order: "TEST-ORDER-001", Project: "가상 수주 · 거래처 연결 시험", Part: "TEST-BASE", Required: 20, Workflow: 2, Revision: 1}}
	a.erpDBProbe = func(context.Context, string) ERPDBReport {
		return decodeDBProbe([]byte(`{"status":"order_outstanding","connected":true,"outstanding_rows":[
  {"robot":"TEST-ROBOT-PR2-400(530)-20N-N-EL-V3-LONG-ROBOT-PART-NUMBER-2026","item":"가상 BASE","quantity":10,"due":"20261130","customer_code":"TEST-A","unit":"PROD"},
  {"robot":"TEST-ROBOT-PR2-400(530)-20N-N-EL-V3-LONG-ROBOT-PART-NUMBER-2026","item":"가상 BRACKET","quantity":1.5,"due":"20261201","customer_code":"TEST-B","unit":"PROD"},
  {"robot":"TEST-ROBOT-PR2-400(530)-20N-N-EL-V3-LONG-ROBOT-PART-NUMBER-2026","item":"가상 PLATE","quantity":3,"due":"20261202","customer_code":"TEST-C","unit":"PROD"}],
  "customer_candidates":[
  {"code":"TEST-A","unit":"COMMON","name":"가상 공통 거래처"},
  {"code":"TEST-B","unit":"PROD","name":"가상 가공 거래처"},
  {"code":"TEST-B","unit":"OTHER","name":"가상 다른 사업장 업체"},
  {"code":"TEST-C","unit":"01","name":"가상 후보 업체 A"},
  {"code":"TEST-C","unit":"02","name":"가상 후보 업체 B"}]} `))
	}
	srv := httptest.NewServer(a)
	defer srv.Close()
	cfg := map[string]any{"worker": map[string]string{"name": worker.Name, "value": worker.Value}}
	cmd := exec.Command("node", "audit/browser-supplier-v1.4.117.cjs", srv.URL, string(jsonRaw(cfg)))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if e := cmd.Run(); e != nil {
		t.Fatal(e)
	}
}
