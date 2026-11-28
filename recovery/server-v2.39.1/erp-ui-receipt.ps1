$ErrorActionPreference='Stop'
$ProgressPreference='SilentlyContinue'
[Console]::OutputEncoding=New-Object Text.UTF8Encoding($false)
$result=[ordered]@{status='ui_receipt';stage='ui_receipt';connected=$false;ui_receipt=[ordered]@{reason='failed';window='';elements=@();truncated=$false;read_errors=0;password_excluded=0;saved=$false}}
$report=$result.ui_receipt
try {
 Add-Type -AssemblyName UIAutomationClient
 Add-Type -AssemblyName UIAutomationTypes
 $erp=@(Get-Process -Name PrimeXAR -ErrorAction SilentlyContinue)
 if($erp.Count -eq 0){$report.reason='erp_missing';throw 'stop'}
 if($erp.Count -ne 1){$report.reason='erp_ambiguous';throw 'stop'}
 $pidCondition=New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::ProcessIdProperty,[int]$erp[0].Id)
 $windows=[System.Windows.Automation.AutomationElement]::RootElement.FindAll([System.Windows.Automation.TreeScope]::Children,$pidCondition)
 $targets=New-Object 'Collections.Generic.List[object]'
 for($i=0;$i -lt $windows.Count;$i++) {
  try{if($windows[$i].Current.Name -match 'PG03F2120' -and !$windows[$i].Current.IsOffscreen){$targets.Add($windows[$i])}}catch{$report.read_errors++}
 }
 if($targets.Count -eq 0){$report.reason='window_missing';throw 'stop'}
 if($targets.Count -ne 1){$report.reason='window_ambiguous';throw 'stop'}
 $root=$targets[0];$report.window='PG03F2120'
 $walker=[System.Windows.Automation.TreeWalker]::RawViewWalker
 $queue=New-Object 'Collections.Generic.Queue[object]'
 $queue.Enqueue(@{element=$root;depth=0;parent=-1})
 $items=New-Object 'Collections.Generic.List[object]'
 while($queue.Count -gt 0 -and $items.Count -lt 1200){
  $entry=$queue.Dequeue();$element=$entry.element
  try {
   $cur=$element.Current
   if($cur.ProcessId -ne $erp[0].Id){continue}
   # Password controls and their subtrees are excluded before reading Name/Value.
   if($cur.IsPassword){$report.password_excluded++;continue}
   $index=$items.Count
   $item=[ordered]@{index=$index;parent=[int]$entry.parent;depth=[int]$entry.depth;type=$cur.ControlType.ProgrammaticName;class=$cur.ClassName;automation_id=$cur.AutomationId;name='';value='';patterns=@();enabled=$cur.IsEnabled;offscreen=$cur.IsOffscreen;row=-1;column=-1;grid_rows=-1;grid_columns=-1;toggle='';selected=$false}
   if(!$cur.IsOffscreen){$item.name=[string]$cur.Name}
   $item.patterns=@($element.GetSupportedPatterns()|ForEach-Object{$_.ProgrammaticName})
   foreach($field in @('name','class','automation_id')){if($item[$field].Length -gt 1500){$item[$field]=$item[$field].Substring(0,1500);$report.truncated=$true}}
   # All pattern access is read-only. No Invoke, SetValue, Toggle, Select,
   # focus change, scrolling, keyboard messages or mouse movement is performed.
   $pattern=$null
   try{if(!$cur.IsOffscreen -and $element.TryGetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern,[ref]$pattern)){$item.value=[string]$pattern.Current.Value}}catch{$report.read_errors++}
   if($item.value.Length -gt 1500){$item.value=$item.value.Substring(0,1500);$report.truncated=$true}
   $pattern=$null
   try{if($element.TryGetCurrentPattern([System.Windows.Automation.GridItemPattern]::Pattern,[ref]$pattern)){$item.row=[int]$pattern.Current.Row;$item.column=[int]$pattern.Current.Column}}catch{$report.read_errors++}
   $pattern=$null
   try{if($element.TryGetCurrentPattern([System.Windows.Automation.GridPattern]::Pattern,[ref]$pattern)){$item.grid_rows=[int]$pattern.Current.RowCount;$item.grid_columns=[int]$pattern.Current.ColumnCount}}catch{$report.read_errors++}
   $pattern=$null
   try{if($element.TryGetCurrentPattern([System.Windows.Automation.TogglePattern]::Pattern,[ref]$pattern)){$item.toggle=[string]$pattern.Current.ToggleState}}catch{$report.read_errors++}
   $pattern=$null
   try{if($element.TryGetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern,[ref]$pattern)){$item.selected=[bool]$pattern.Current.IsSelected}}catch{$report.read_errors++}
   $items.Add($item)
   if($entry.depth -ge 18){$report.truncated=$true;continue}
   $child=$walker.GetFirstChild($element);$siblings=0
   while($null -ne $child -and $siblings -lt 600 -and $queue.Count -lt 1800){
    $queue.Enqueue(@{element=$child;depth=($entry.depth+1);parent=$index});$siblings++;$child=$walker.GetNextSibling($child)
   }
   if($null -ne $child){$report.truncated=$true}
  }catch{$report.read_errors++}
 }
 if($queue.Count -gt 0){$report.truncated=$true}
 $report.elements=@($items.ToArray());$report.reason='captured'
}catch{}
$result.stage='done'
ConvertTo-Json -InputObject $result -Depth 8 -Compress
