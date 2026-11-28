package main

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestERPDBPowerShellCommandLineBounded(t *testing.T) {
	contexts := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{"metadata", context.Background(), "$PurchasePreview=$false"},
		{"preview", context.WithValue(context.Background(), erpDBPreviewKey{}, erpDBPreviewRange{From: "2026-11-01", To: "2026-12-31"}), "$DateFrom='20261101'"},
		{"sync", context.WithValue(context.WithValue(context.Background(), erpDBPreviewKey{}, erpDBPreviewRange{From: "2026-11-01", To: "2026-12-31"}), erpDBSyncKey{}, true), "$SyncOnly=$true"},
		{"batch", context.WithValue(context.Background(), erpOrderOutstandingBatchKey{}, []string{"TEST-001", "TEST-수주'002"}), "$OutstandingOrders=@('TEST-001','TEST-수주''002')"},
		{"outstanding", context.WithValue(context.Background(), erpOrderOutstandingKey{}, "TEST-수주'001"), "$OutstandingOrder='TEST-수주''001'"},
	}
	var encoded string
	for _, tc := range contexts {
		t.Run(tc.name, func(t *testing.T) {
			cmd := erpDBPowerShellCommand(tc.ctx, `C:\가상 ERP'폴더`)
			total := 0
			for _, arg := range cmd.Args {
				total += len(utf16.Encode([]rune(arg))) + 3
			}
			if total >= 2048 {
				t.Fatalf("process command grew to %d characters", total)
			}
			if len(cmd.Args) != 6 || cmd.Args[4] != "-EncodedCommand" {
				t.Fatal(cmd.Args)
			}
			if encoded != "" && encoded != cmd.Args[5] {
				t.Fatal("query data entered command arguments")
			}
			encoded = cmd.Args[5]
			raw, e := io.ReadAll(cmd.Stdin)
			if e != nil {
				t.Fatal(e)
			}
			script := string(raw)
			if !strings.HasSuffix(script, strings.Replace(strings.Replace(strings.Replace(strings.Replace(strings.Replace(erpDBScript, "# RECEIPT_METADATA_SCRIPT", erpReceiptMetadataScript, 1), "# OBSERVATION_ACCESS_SCRIPT", erpObservationAccessScript, 1), "# OBSERVATION_SCRIPT", erpObservationScript, 1), "# CERTIFICATE_SCRIPT", erpCertificateScript, 1), "# RECEIPT_CONTRACT_SCRIPT", erpReceiptContractScript, 1)) || !strings.Contains(script, tc.want) || !strings.Contains(script, "$ERPFolder='C:\\가상 ERP''폴더'") {
				t.Fatal("script/prefix or Unicode path was changed")
			}
			// This is the released 1.4.117 regression: the existing complete query is
			// already too large to pass as a UTF-16 Base64 command-line argument.
			legacyLength := base64.StdEncoding.EncodedLen(len(utf16.Encode([]rune(script))) * 2)
			if legacyLength <= 32767 {
				t.Fatal("regression fixture must exercise the oversized released script")
			}
		})
	}
	decoded, e := base64.StdEncoding.DecodeString(encoded)
	if e != nil {
		t.Fatal(e)
	}
	units := make([]uint16, len(decoded)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(decoded[i*2:])
	}
	bootstrap := string(utf16.Decode(units))
	if bootstrap != erpDBBootstrap || !strings.Contains(bootstrap, "[Console]::In.ReadToEnd()") || !strings.Contains(bootstrap, "UTF8Encoding") || !strings.Contains(bootstrap, "[ScriptBlock]::Create($source)") {
		t.Fatal("invalid stdin bootstrap")
	}
}
func TestERPDBPowerShellStdinDeliveredIntact(t *testing.T) {
	input := erpDBPowerShellCommand(context.Background(), `C:\가상 ERP'경로`)
	raw, e := io.ReadAll(input.Stdin)
	if e != nil {
		t.Fatal(e)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestERPDBStdinChild$")
	child.Env = append(os.Environ(), "ALPHA_ERP_STDIN_CHILD=1")
	child.Stdin = strings.NewReader(string(raw))
	output, e := child.Output()
	if e != nil {
		t.Fatal(e)
	}
	var result struct {
		Bytes  int
		Digest string
	}
	if json.Unmarshal(output, &result) != nil || result.Bytes != len(raw) || result.Digest != digest(raw) {
		t.Fatal("large Unicode script was not delivered intact")
	}
}
func TestERPDBStdinChild(t *testing.T) {
	if os.Getenv("ALPHA_ERP_STDIN_CHILD") != "1" {
		return
	}
	raw, e := io.ReadAll(os.Stdin)
	if e != nil {
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(struct {
		Bytes  int
		Digest string
	}{len(raw), digest(raw)})
	os.Exit(0)
}
func TestERPDBFailureStagesSurviveDecoding(t *testing.T) {
	for _, stage := range []string{"process_job", "process_start", "process_attach", "process_wait", "process_exit", "response_decode"} {
		report := decodeDBProbe(jsonRaw(ERPDBReport{Status: "failed", Stage: stage, ErrorCode: 206}))
		if report.Stage != stage || report.ErrorCode != 206 {
			t.Fatal(report)
		}
	}
	report := decodeDBProbe(jsonRaw(ERPDBReport{Status: "failed", Stage: "process_start", ErrorCode: 206}))
	if !strings.Contains(dbProbeMessage(report), "Windows 오류 206") {
		t.Fatal(dbProbeMessage(report))
	}
	if _, e := machiningDBRows(report); e == nil || !strings.Contains(e.Error(), "Windows 오류 206") {
		t.Fatal("sync hid launch failure", e)
	}
	if report = decodeDBProbe([]byte("invalid private process output")); report.Stage != "response_decode" {
		t.Fatal(report)
	}
}
