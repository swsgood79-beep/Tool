# PC1.4.130 recovery

User screenshot: pale helper panel/control rows with dark-mode light text. Confirmed hardcoded new helper backgrounds combined with inherited dark theme; modern table striping affected control rows.

Changed: new extras/erp-receipt-helper-v1.4.130.css uses existing --ui tokens for all panel, text, button/disabled button, part/control rows, labels and missing-key notes. Explicit receipt-controls class in extras/order-outstanding-cards-v1.4.130.js; order_outstanding_cards.go asset routing and release version130. No backend receipt/permission/account/data changes.

Actual-browser regression:320/390/430/1100 widths × light, system-dark, app-dark with light OS, forced-light with dark OS; computed foreground/background contrast plus selection/prepare/long code/no overflow and screenshots. Synthetic data only; native launch mocked. Full/race/vet/Windows/source rebuild and public proof recorded alongside.

Next: PC update130 then refresh phone Alpha page (APK unchanged1.0.11). No new diagnostic needed. Existing manual ERP vendor/row/qty/final-save workflow boundary and native company field verification pending remain. Root WORK-ERP-UI-RECEIPT.md and PUBLIC-PC-v1.4.130.json record status.
