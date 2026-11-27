# ERP receipt observer recovery

User authorized implementing an observer for normal ERP receipt steps, not automating ERP writes. Existing release PC120 remains unchanged. Work branch work-erp-receipt-observer.

Implemented first increment: owner-only existing db-check endpoint mode observe-access. Existing INI reused; fixed SELECT queries HAS_PERMS_BY_NAME for old and SQL2022 granular XE permissions and checks XE availability. Does not change rights/create sessions/collect business rows or secrets. Output explicitly says recording has NOT started.

Files: erp-observation-access.ps1, erp-db-probe.ps1, erp_db_probe.go, erp_db_powershell.go, transport test expectation.

Remaining: access UI, permission tests, source preview and native SQL collector. Recorder needs explicit scope to company PC + ERP database/client, ready handshake before operator receipt, bounded duration, stop/cleanup on cancellation, dropped/truncated-event reporting, private protected local storage. A permissions check alone must not be represented as observation. No recorder/public release yet. Do not tell user to run a business receipt before readiness.

Private receipt metadata previously uploaded established PC_PDTWHSMST_UPDATE/UPDATE2 and PC_PDTWHSHIS_UPDATE as candidates only. Never publish original company metadata. Trace must prove actual calls, params and transactional behavior; static dependencies do not prove that.
