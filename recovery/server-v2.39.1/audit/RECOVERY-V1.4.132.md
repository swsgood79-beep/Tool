# PC1.4.132 recovery

Latest user cancels input functions and wants purchase number directly for each outstanding item. This supersedes previous helper work. Removed native preparation/launch/PS/filter, helper-only scripts/styles/state and tests. Old prepare endpoint410 for authenticated and stale anonymous clients; cannot launch a process. No helper/selection/quantity input UI or CSRF/snapshot data.

Read-only basic worker/admin popup: actual h.PURORD_NO shown full width directly above corresponding item row; no merging duplicate item names or guessed PO. Missing number explicitly unknown. Preserve4-column item/remaining qty/supplier/due table, robot wrapping, refresh/close, zero hiding, failure handling, dark palette and131 edit-worker separation. Existing query joins/line-order priority retained. Runtime cache only; no operating data, shipment/workbook/account/rights/config/APK changes.

Tests: worker/admin exact per-item PO, no-input UI/assets, retired GET/POST route410/no mutation; actual browser2 roles×4 widths×light/dark, long PO/robot, duplicates/missing PO/no overflow/refresh and admin worker non-overlap screenshots. Test harness initially passed nil cookie to helper; corrected test only, no product change. Final full/race/vet/Windows/source/public proof alongside.

Files: outstanding cards/model/customer resolver/query, main/model,132JS+CSS, input runtime deletions, tests, AGENTS/current docs. Source/checkpoints in root WORK-ERP-UI-RECEIPT.md. Next PC update132 then refresh current page; no helper/native field checks to resume. PUBLIC-PC-v1.4.132.json records public deployment.
