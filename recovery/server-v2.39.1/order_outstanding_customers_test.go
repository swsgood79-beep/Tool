package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestOutstandingSupplierResolution(t *testing.T) {
	cases := []struct {
		name    string
		row     OrderOutstandingRow
		masters []ERPDBCustomer
		want    string
	}{
		{"same unit", OrderOutstandingRow{CustomerCode: "A", Unit: "01"}, []ERPDBCustomer{{Code: "A", Unit: "01", Name: "가상 공급사"}}, "가상 공급사"},
		{"shared unit", OrderOutstandingRow{CustomerCode: "A", Unit: "01"}, []ERPDBCustomer{{Code: "A", Unit: "00", Name: "가상 공통 공급사"}}, "가상 공통 공급사"},
		{"unit independent master", OrderOutstandingRow{CustomerCode: "A", Unit: "01"}, []ERPDBCustomer{{Code: "A", Name: "가상 공통 공급사"}}, "가상 공통 공급사"},
		{"prefer exact unit", OrderOutstandingRow{CustomerCode: "A", Unit: "01"}, []ERPDBCustomer{{Code: "A", Unit: "01", Name: "가상 공급사 A"}, {Code: "A", Unit: "02", Name: "가상 공급사 B"}}, "가상 공급사 A"},
		{"cross unit conflict", OrderOutstandingRow{CustomerCode: "A", Unit: "03"}, []ERPDBCustomer{{Code: "A", Unit: "01", Name: "가상 공급사 A"}, {Code: "A", Unit: "02", Name: "가상 공급사 B"}}, ""},
		{"same unit conflict", OrderOutstandingRow{CustomerCode: "A", Unit: "01"}, []ERPDBCustomer{{Code: "A", Unit: "01", Name: "가상 공급사 A"}, {Code: "A", Unit: "01", Name: "가상 공급사 B"}}, ""},
		{"blank local name", OrderOutstandingRow{CustomerCode: "A", Unit: "01"}, []ERPDBCustomer{{Code: "A", Unit: "01", Name: " "}, {Code: "A", Unit: "00", Name: "가상 공급사"}}, "가상 공급사"},
		{"duplicate master names", OrderOutstandingRow{CustomerCode: "A", Unit: "03"}, []ERPDBCustomer{{Code: "A", Unit: "01", Name: "가상 공급사"}, {Code: "A", Unit: "02", Name: "가상 공급사"}}, "가상 공급사"},
		{"trim keys", OrderOutstandingRow{CustomerCode: " A ", Unit: " 01 "}, []ERPDBCustomer{{Code: "A ", Unit: "01 ", Name: " 가상 공급사 "}}, "가상 공급사"},
		{"different code", OrderOutstandingRow{CustomerCode: "A", Unit: "01"}, []ERPDBCustomer{{Code: "B", Unit: "01", Name: "가상 공급사"}}, ""},
		{"preserve leading zero", OrderOutstandingRow{CustomerCode: "00530", Unit: "01"}, []ERPDBCustomer{{Code: "530", Unit: "01", Name: "가상 공급사"}}, ""},
		{"no code", OrderOutstandingRow{Unit: "01"}, []ERPDBCustomer{{Code: "A", Unit: "01", Name: "가상 공급사"}}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.row.Robot = "TEST-ROBOT"
			tc.row.Item = "BASE"
			tc.row.Quantity = 1.5
			tc.row.Due = "20261130"
			before := string(jsonRaw(tc.row))
			got := resolveOutstandingCustomers([]OrderOutstandingRow{tc.row}, tc.masters)
			if len(got) != 1 || got[0].Customer != tc.want || got[0].Quantity != 1.5 || got[0].Robot != "TEST-ROBOT" || got[0].Item != "BASE" || got[0].Due != "20261130" {
				t.Fatal(got)
			}
			if got[0].CustomerCode != "" || got[0].Unit != "" {
				t.Fatal("internal keys leaked")
			}
			if string(jsonRaw(tc.row)) != before {
				t.Fatal("input mutated")
			}
		})
	}
}
func TestOutstandingSupplierProbeDecodeThroughHTTP(t *testing.T) {
	a := fixture(t)
	worker, _ := machiningWorker(t, a, "supplier-test")
	a.state.MachiningJobs = []MachiningJob{{ID: "one", Order: "ORDER-001", Required: 20}}
	before := string(jsonRaw(a.state))
	raw := []byte(`{"status":"order_outstanding","connected":true,"outstanding_rows":[{"robot":"TEST-ROBOT","item":"BASE","quantity":1.5,"customer_code":"TEST-A","unit":"01"}],"customer_candidates":[{"code":"TEST-A","name":"가상 공통 공급사","unit":"00"}]}`)
	a.erpDBProbe = func(context.Context, string) ERPDBReport { return decodeDBProbe(raw) }
	response := call(a, "GET", "/machining/outstanding?order=ORDER-001", nil, worker)
	var result struct {
		OK   bool                  `json:"ok"`
		Rows []OrderOutstandingRow `json:"rows"`
	}
	if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &result) != nil || !result.OK || len(result.Rows) != 1 || result.Rows[0].Customer != "가상 공통 공급사" {
		t.Fatal(response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "customer_code") || strings.Contains(response.Body.String(), "customer_candidates") {
		t.Fatal("matching records leaked")
	}
	if before != string(jsonRaw(a.state)) {
		t.Fatal("ERP read mutated work")
	}
}
func TestOutstandingSupplierQueryScope(t *testing.T) {
	raw, e := os.ReadFile("erp-db-probe.ps1")
	if e != nil {
		t.Fatal(e)
	}
	s := string(raw)
	start := strings.Index(s, "if($OutstandingOrder -or $OutstandingOrders)")
	end := strings.Index(s, "}elseif($PurchasePreview)")
	if start < 0 || end < start {
		t.Fatal("missing scoped query")
	}
	s = s[start:end]
	for _, part := range []string{"COL_LENGTH('dbo.PURORD_HIS','CST_CD')", "COALESCE(NULLIF(LTRIM(RTRIM(h.CST_CD)),''),m.CST_CD)", "WHERE EXISTS", "ORDER_EXPR IN (", "WHERE ORDER_FILTER", "Add-OutstandingParameters $cmd", "LTRIM(RTRIM(c.CST_CD))=LTRIM(RTRIM(CUSTOMER_EXPR))", "customer_candidates", "customer_code", "$report.truncated=$true"} {
		if !strings.Contains(s, part) {
			t.Fatal("missing query safeguard", part)
		}
	}
	if strings.Contains(s, "LEFT JOIN dbo.CSTMST_INFO") || strings.Contains(s, "c.UNT_CD=m.UNT_CD") {
		t.Fatal("master join can drop names or multiply outstanding rows")
	}
}
