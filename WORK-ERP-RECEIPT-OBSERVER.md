# ERP receipt observer recovery

User authorized implementing an observer for normal ERP receipt steps, not automating ERP writes. Existing release PC120 remains unchanged. Work branch work-erp-receipt-observer.

Implemented first increment: owner-only existing db-check endpoint mode observe-access. Existing INI reused; fixed SELECT queries HAS_PERMS_BY_NAME for old and SQL2022 granular XE permissions and checks XE availability. Does not change rights/create sessions/collect business rows or secrets. Output explicitly says recording has NOT started.

Files: erp-observation-access.ps1, erp-db-probe.ps1, erp_db_probe.go, erp_db_powershell.go, transport test expectation.

Remaining: access UI, permission tests, source preview and native SQL collector. Recorder needs explicit scope to company PC + ERP database/client, ready handshake before operator receipt, bounded duration, stop/cleanup on cancellation, dropped/truncated-event reporting, private protected local storage. A permissions check alone must not be represented as observation. No recorder/public release yet. Do not tell user to run a business receipt before readiness.

Private receipt metadata previously uploaded established PC_PDTWHSMST_UPDATE/UPDATE2 and PC_PDTWHSHIS_UPDATE as candidates only. Never publish original company metadata. Trace must prove actual calls, params and transactional behavior; static dependencies do not prove that.

Permission-discovery tests: catalog permission SELECT does not execute DDL (permission names inside string literals ignored by SQL-token assertion), owner and CSRF rejection, context transport, false permissions retained, unknown secret field removed, State unchanged. Related TestObservation/TestERPDB/TestReceiptMetadata suite pass. Not recording yet; public PC120 unchanged. New test file erp_observation_access_test.go. Next implementation remains ready/start/stop/cleanup scoped XE collector and synthetic browser validation before any release.

Recorder increment: bounded90-second XE ring-buffer capture with database ID + company hostname + single PrimeXAR client PID filters. Captures completed RPC/batch/procedure statements only (not proof of committed row values). Ready file handshake before UI recording. Stop file ends recording; finally drops exact owned session. Protected local recovery journal blocks new starts after uncertain cleanup; explicit owner recovery action drops journal's owned session. Abrupt OS loss may leave an XE session until recovery; STARTUP_STATE OFF, bounded2MiB ring. No ERP writes/grants. Report DPAPI protect/unprotect verification and atomic local save before download. Auth-related SQL text omitted; results still confidential business data. No user data copied to repo.

Files added erp_observation.go, erp-observation.ps1, extras/erp-observer-v1.4.121.js, machining template; updated models/dispatch/context script. Production code version121 draft, public metadata120 retained. Tests/browser/native SQL validation and packaging remain.

Lifecycle verification checkpoint: duplicate/unauthorized/CSRF/early download rejected; ready handshake then stop then DPAPI-style roundtrip local backup/download passes; journal retained on uncertain cleanup and recovery removes exact prior session journal; 0-row/truncated report warnings verified; State unchanged. Actual synthetic Chromium320/390/430/1100 ready/start/stop/download pass14.870s. Screenshot observer-recording/finished-*-v1.4.121.png. SQL XE execution on real company Windows/SQL remains unverified. Parser/source/Windows cross-build only must not be called field-tested.

Pre-package: fullGo65.727s, related race11.184s, Linux/Windows vet pass, Windows GUI cross-build pass. Additional backup-failure and stop-before-ready test passed0.325s. Linux PowerShell7 AST parse of fully expanded embedded script passed; native Windows PowerShell5/SQL XE not executed. Four-width screenshots synthetic only. Added ERP-RECEIPT-OBSERVER.md precise scope/retention/recovery/confidentiality instructions. Final current version121 draft. Next sourceZIP rebuild, new immutable artifacts, public metadata/download verification.

Source ZIP rebuild verified byte-identical Windows EXE, full archive checksums and embedded helper exact rebuild pass; APK1.0.11 hash unchanged. Windows ZIP0027fec7795971e4287fc91ea24458b352ccc4c03efc4946b2f66a7d7ce8b7d9; sourceZIP ae54295a9a966718a31ffdf100a3c0c9c768d597833385e1a4db14ed50508f2d. Ready to publish new121+pc-update together. Public verification pending at this commit.
