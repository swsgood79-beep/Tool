# PC1.4.128 recovery

Completed: admin-with-machining-menu selection and partial quantities in basic order-card outstanding popup. Full purchase key/known display PART_NO retained internally, worker API metadata redacted. Snapshot/expiry/duplicate/quantity/active project checks and rate limit. Supplier/unit/PO groups, explicit business sequence. Existing work/shipping/export/account state unchanged.

Windows helper:20-minute topmost WinForms window; vendor/PO copy; user confirms ERP vendor; optional exact labelled native EDIT input plus unique shallow toolbar Search in same PID PG03F2120. No coordinate clicks, direct DB write, automated vendor/row/quantity/final Save. Ambiguity falls back to copy. Native company launch/filter NOT verified in cloud.

Tests: synthetic two suppliers, same part different business sequences, partial quantity, auth/CSRF/stale/expiry/duplicate/rate-limit and nonmutation. Actual Chromium320/390/430/1100 selection/preview/download-facing UI, native launch mocked. Official Accessibility contract compile + PowerShell AST + actual native-input selector tested without native calls. Full/race/vet/build logs alongside.

Files: erp_receipt_helper*.go, erp-receipt-helper.ps1, erp-receipt-filter.cs, extras/order-outstanding-cards-v1.4.128.js, extras/erp-receipt-helper-v1.4.128.css, order_outstanding*, erp-db-probe.ps1, main/model/production, helper tests/audit128.

Next: PC update128, admin basic card outstanding popup select/quantity→ERP input prepare→PC helper. User manually sets correct ERP vendor and opens Data-load popup; confirm vendor then optional purchase search or copy. User checks exact PO/part/line/qty and manually saves ERP. Refresh Alpha afterwards. Do not ask for repeated diagnostics. No claim fully automatic receipt or native field verification. Final public proof in PUBLIC-PC-v1.4.128.json; root WORK-ERP-UI-RECEIPT.md records checkpoints.
