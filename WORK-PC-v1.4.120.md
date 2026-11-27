# PC v1.4.120 recovery

Completed: owner-only read-only receipt metadata mode using existing INI. Collect module names/parameters/dependencies and receipt table identity/default presence/index/constraint metadata. Never collect module bodies/default expressions/business rows/ERP accounts/passwords; never execute modules or ERP writes. Permission limitation and truncation are explicit. UI action added to existing DB diagnostics. APK unchanged. No private uploaded diagnostic or screenshots copied to repository.

Files: erp_db_probe.go, erp_db_powershell.go, erp-db-probe.ps1, new erp-receipt-metadata.ps1, extras/machining.html and erp-db-check-v1.4.120.js, production.go.

Remaining: tests, actual synthetic desktop/mobile screenshots, Windows build/source rebuild, package120, public verification. Public update remains119 until validated. No native company SQL/Windows test performed. This metadata cannot prove final receipt semantics; ERP actor/auth and number generation still require analysis.

Next: test permissions/CSRF/decoding/bounds/read-only SQL/mode transport, browser then package. Earlier machining XLSX request remains outstanding, excluded from this diagnostic change.

Verification checkpoint: owner/worker and CSRF rejection, operating State unchanged, receipt mode context + bounded Windows stdin transport, metadata cap/warnings/unexpected fields redaction and catalog-only SQL checks pass. Actual Chromium 320/390/430/1100 all pass with synthetic metadata, restricted permission and truncated warnings shown; screenshot files receipt-diagnostic-*-v1.4.120.png. Related test run 9.497s pass. Full tests/race/vet in progress. Existing PowerShell transport regression expectation updated for expanded embedded script. Remaining native Windows/SQL Server behavior is field verification, not claimed tested.

Pre-package validation: whole Go suite pass, related race pass11.692s, Linux and Windows vet pass (empty output), Windows amd64 GUI cross-build pass. Actual Chromium4 viewport cases pass. Company PC native PowerShell/SQL Server execution not performed. Module source bodies/default expressions intentionally omitted to avoid embedded secrets; this first report maps names/dependencies and flags available definition visibility, not final semantics. Diagnostics do not validate ERP application login/session or grant any write permission. Next: package new120 only, rebuild from source ZIP, publish and fetch public files/manifest.
