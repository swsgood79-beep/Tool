# Read structural metadata only. No Invoke/Select/Value/Text patterns are executed.
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
$labels = @('Excel','엑셀','검색','조회','창닫기','창 닫기','발주현황','발주일','발주일자','발주번호','거래처','수주번호','PROJECT','품번','품명','규격','OPTION','수량','납기예정일','입고수량','품번구분')
$root = [System.Windows.Automation.AutomationElement]::FromHandle([IntPtr]$TargetHandle)
$walker = [System.Windows.Automation.TreeWalker]::ControlViewWalker
$queue = New-Object 'System.Collections.Generic.Queue[object]'
$queue.Enqueue(@{ Element=$root; Depth=0 })
$items = New-Object 'System.Collections.Generic.List[object]'
while ($queue.Count -gt 0 -and $items.Count -lt 300) {
 $entry=$queue.Dequeue(); $el=$entry.Element; $depth=$entry.Depth
 try {
  $cur=$el.Current; $type=$cur.ControlType.ProgrammaticName; $name=''
  # Names are queried only on non-data controls and reduced to fixed labels.
  if ($type -notmatch '(Edit|DataItem|ListItem|TreeItem|Document|Text)$') {
   $candidate=(($cur.Name -replace '&','') -replace '\s+',' ').Trim()
   foreach ($label in $labels) { if ($candidate -ieq $label) { $name=$label; break } }
  }
  $auto=$cur.AutomationId
  if ($auto -notmatch '^(?:[0-9]{1,20}|(?:btn|cmd|txt|edt|cbo|cmb|grd|spr|tbl|tab|pnl|lbl|grid|button|label|textBox|dateTimePicker|toolStrip)[A-Za-z0-9_.:]{0,50})$') { $auto='' }
  $bounds=$cur.BoundingRectangle
  $items.Add([ordered]@{Depth=$depth; Type=$type; Class=$cur.ClassName; AutomationId=$auto; Label=$name; Enabled=$cur.IsEnabled; Offscreen=$cur.IsOffscreen; Bounds=@($bounds.X,$bounds.Y,$bounds.Width,$bounds.Height); Patterns=@($el.GetSupportedPatterns() | ForEach-Object {$_.ProgrammaticName})})
  if ($depth -lt 9 -and $type -notmatch '(DataGrid|Table|DataItem|List|ListItem|Tree|TreeItem|Edit|Document)$') {
   $child=$walker.GetFirstChild($el); $siblings=0
   while ($null -ne $child -and $siblings -lt 200 -and $queue.Count -lt 600) {
    $queue.Enqueue(@{Element=$child;Depth=$depth+1}); $siblings++; $child=$walker.GetNextSibling($child)
   }
  }
 } catch { }
}
$data=[ordered]@{Available=$true; Count=$items.Count; Controls=@($items.ToArray())}
$json=ConvertTo-Json -InputObject $data -Depth 8
[IO.File]::WriteAllText($Destination,$json,(New-Object System.Text.UTF8Encoding($false)))
