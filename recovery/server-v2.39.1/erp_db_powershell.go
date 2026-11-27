package main

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

//go:embed erp-observation.ps1
var erpObservationScript string

//go:embed erp-observation-access.ps1
var erpObservationAccessScript string

//go:embed erp-receipt-metadata.ps1
var erpReceiptMetadataScript string

//go:embed erp-db-probe.ps1
var erpDBScript string

// Keep the Windows process command line bounded. The complete UTF-8 script is
// delivered through stdin, so query growth cannot exceed CreateProcess's limit.
const erpDBBootstrap = `[Console]::InputEncoding=New-Object Text.UTF8Encoding($false);$source=[Console]::In.ReadToEnd();& ([ScriptBlock]::Create($source))`

func erpDBPowerShellCommand(ctx context.Context, folder string) *exec.Cmd {
	prefix := "$SyncOnly=$false\n$PurchasePreview=$false\n"
	if dates, ok := erpDBPreviewOptions(ctx); ok {
		prefix = "$PurchasePreview=$true\n$DateFrom='" + strings.ReplaceAll(dates.From, "-", "") + "'\n$DateTo='" + strings.ReplaceAll(dates.To, "-", "") + "'\n"
	}
	if syncOnly, _ := ctx.Value(erpDBSyncKey{}).(bool); syncOnly {
		prefix += "$SyncOnly=$true\n"
	}
	if order, ok := ctx.Value(erpOrderOutstandingKey{}).(string); ok {
		prefix += "$OutstandingOrder='" + strings.ReplaceAll(order, "'", "''") + "'\n"
	}
	if orders, ok := ctx.Value(erpOrderOutstandingBatchKey{}).([]string); ok {
		values := make([]string, len(orders))
		for i, order := range orders {
			values[i] = "'" + strings.ReplaceAll(order, "'", "''") + "'"
		}
		prefix += "$OutstandingOrders=@(" + strings.Join(values, ",") + ")\n"
	}
	if receipt, _ := ctx.Value(erpDBReceiptKey{}).(bool); receipt {
		prefix += "$ReceiptMetadata=$true\n"
	}
	if observe, _ := ctx.Value(erpDBObserveKey{}).(bool); observe {
		prefix += "$ObservationAccess=$true\n"
	}
	if options, ok := ctx.Value(erpObservationKey{}).(erpObservationOptions); ok {
		if options.Recovery {
			prefix += "$ObserveRecover=$true\n"
		}
		prefix += "$Observe=$true\n$ObserveDirectory='" + strings.ReplaceAll(options.Directory, "'", "''") + "'\n$ObserveSession='" + options.Session + "'\n"
	}
	script := prefix + "$ERPFolder='" + strings.ReplaceAll(folder, "'", "''") + "'\n" + strings.Replace(strings.Replace(erpDBScript, "# RECEIPT_METADATA_SCRIPT", erpReceiptMetadataScript, 1), "# OBSERVATION_ACCESS_SCRIPT", erpObservationAccessScript, 1)
	script = strings.Replace(script, "# OBSERVATION_SCRIPT", erpObservationScript, 1)
	units := utf16.Encode([]rune(erpDBBootstrap))
	encoded := make([]byte, len(units)*2)
	for i, unit := range units {
		binary.LittleEndian.PutUint16(encoded[i*2:], unit)
	}
	command := exec.CommandContext(ctx, filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe"), "-NoLogo", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(encoded))
	command.Stdin = strings.NewReader(script)
	return command
}
