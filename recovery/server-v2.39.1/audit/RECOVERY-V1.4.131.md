# PC1.4.131 recovery

User confirms PC says helper opened but no window. Code evidence: helper SysProcAttr.HideWindow true maps in Go syscall/exec_windows.go393-395 to STARTF_USESHOWWINDOW/SW_HIDE, unlike console-only CREATE_NO_WINDOW. Prior success used managed Shown marker without native visibility. Removed HideWindow; retained console-only flag.

Ready protocol now numeric HWND. Verify child PID, IsWindowVisible, monitor intersection and non-minimized state after1s stable live process. Noninteractive desktop guard, normal/taskbar/front WinForms setup. Fixed stage-only failure collector/message, no raw stderr/ERP data exposed. Search-only Accessibility/C# initialization lazy, base helper/clipboard independent.

Changed erp_receipt_helper_windows.go, new erp_receipt_helper_launch.go, helper PS, HTTP error mapping, tests, production. No direct ERP writes or automatic row/quantity/final-save change. Physical company Windows outcome still unverified in cloud.

Tests: numeric protocol invalid/legacy marker rejection; closed/hidden vs live visible process outcomes; stage framing/redaction; HTTP failure cannot report launched or mutate state; visible GUI startup regression; actual browser simulated hidden-window failure/retry with synthetic data. Native API mocked/not run. Full/race/vet/Windows/source rebuild proof alongside.

Next: PC update131, outstanding popup refresh/reselect→prepare→PC helper. Successful status now requires native visible child window. If rejected, Alpha displays fixed-stage reason directly, no repeated JSON capture. Root memo and PUBLIC-PC-v1.4.131.json record exact status.

Added before release: admin-card-layout-v1.4.131.css replaces absolute edit/disclosure positioning with own normal-flow grid row. Full-width multi-worker row stays above action buttons.16 light/dark/width cases validate non-overlap, full name wrapping, overflow and edit href; screenshots admin-card-*131. Legacy basic-card browser edit-position expectation updated to intended lower row. Native helper objective retained; both fixes share131.
