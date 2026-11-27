# PC v1.4.120 recovery

Completed: owner-only read-only receipt metadata mode using existing INI. Collect module names/parameters/dependencies and receipt table identity/default presence/index/constraint metadata. Never collect module bodies/default expressions/business rows/ERP accounts/passwords; never execute modules or ERP writes. Permission limitation and truncation are explicit. UI action added to existing DB diagnostics. APK unchanged. No private uploaded diagnostic or screenshots copied to repository.

Files: erp_db_probe.go, erp_db_powershell.go, erp-db-probe.ps1, new erp-receipt-metadata.ps1, extras/machining.html and erp-db-check-v1.4.120.js, production.go.

Remaining: tests, actual synthetic desktop/mobile screenshots, Windows build/source rebuild, package120, public verification. Public update remains119 until validated. No native company SQL/Windows test performed. This metadata cannot prove final receipt semantics; ERP actor/auth and number generation still require analysis.

Next: test permissions/CSRF/decoding/bounds/read-only SQL/mode transport, browser then package. Earlier machining XLSX request remains outstanding, excluded from this diagnostic change.
