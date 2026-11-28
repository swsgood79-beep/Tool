# PC1.4.129 recovery

User report: selection disabled with generic purchase-information message.
Confirmed code defect: optional display PART_NO was required by receiptRowKey even though not in key. Actual field row cause not independently inspected. Remove optional display field prerequisite; unique unit/date/PO/business sequence remains mandatory. Add explicit sequence_known for valid zero. Missing identity fields explained in UI. Missing part is not guessed; separate ERP item code shown in preview/Windows helper. No automated row/quantity/final-save or direct DB write.

Files: erp_receipt_helper.go, order_outstanding.go/cards.go, erp-db-probe.ps1, erp-receipt-helper.ps1, extras129JS, production.go, regression tests/browser129. Synthetic fixture has missing display PART_NO selectable, explicit zero selectable, genuinely missing keys disabled and labelled. Worker metadata/token redaction preserved.

Tests: optional display/known0/missing key regression plus actual Chromium320/390/430/1100 flow. See full/race/vet/Windows/source rebuild results and PUBLIC-PC-v1.4.129.json. No native company Windows launch/filter verification claimed.

Next: existing PC update129, refresh basic admin order-card outstanding popup. No new diagnostics requested. If real identity absent, displayed exact missing fields must remain blocked. Root WORK-ERP-UI-RECEIPT.md records checkpoints/commits.
