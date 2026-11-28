$ErrorActionPreference='Stop'
$ProgressPreference='SilentlyContinue'
[Console]::OutputEncoding=New-Object Text.UTF8Encoding($false)
$result=[ordered]@{status='ui_receipt';stage='ui_receipt';connected=$false;ui_receipt=[ordered]@{reason='failed';window='';elements=@();truncated=$false;read_errors=0;password_excluded=0;saved=$false;discovery=@();native_windows=0;uia_visited=0;discovery_method='';discovery_truncated=$false}}
$report=$result.ui_receipt
try {
 Add-Type -AssemblyName UIAutomationClient
 Add-Type -AssemblyName UIAutomationTypes
 $erp=@(Get-Process -Name PrimeXAR -ErrorAction SilentlyContinue)
 if($erp.Count -eq 0){$report.reason='erp_missing';throw 'stop'}
 if($erp.Count -ne 1){$report.reason='erp_ambiguous';throw 'stop'}
 # Enumerate native child windows as well as top-level windows. Classic MDI
 # dialogs may be omitted from UIA's desktop Children view or have blank Name.
 Add-Type -TypeDefinition @'
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;
public static class AlphaReceiptWindows {
 public sealed class Entry { public long Handle; public long Parent; public int Process; public bool Visible; public string Class; public string Title; }
 public sealed class ScanResult { public Entry[] Entries; public bool Truncated; }
 private delegate bool Callback(IntPtr window, IntPtr state);
 [DllImport("user32.dll")] private static extern bool EnumWindows(Callback callback, IntPtr state);
 [DllImport("user32.dll")] private static extern bool EnumChildWindows(IntPtr parent, Callback callback, IntPtr state);
 [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr window, out uint process);
 [DllImport("user32.dll")] private static extern bool IsWindowVisible(IntPtr window);
 [DllImport("user32.dll")] private static extern IntPtr GetParent(IntPtr window);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] private static extern int GetWindowText(IntPtr window,StringBuilder value,int length);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] private static extern int GetClassName(IntPtr window,StringBuilder value,int length);
 public static ScanResult Scan(int process) {
  var entries=new List<Entry>(); var seen=new HashSet<long>(); bool truncated=false;
  Callback read=delegate(IntPtr window,IntPtr state) {
   uint owner; GetWindowThreadProcessId(window,out owner);
   if(owner!=(uint)process || !seen.Add(window.ToInt64())) return true;
   if(entries.Count>=2000) {truncated=true;return false;}
   var name=new StringBuilder(512); GetClassName(window,name,name.Capacity); string cls=name.ToString();
   var title=new StringBuilder(512);
   if(!System.Text.RegularExpressions.Regex.IsMatch(cls,"edit|textbox|password",System.Text.RegularExpressions.RegexOptions.IgnoreCase)) GetWindowText(window,title,title.Capacity);
   entries.Add(new Entry {Handle=window.ToInt64(),Parent=GetParent(window).ToInt64(),Process=(int)owner,Visible=IsWindowVisible(window),Class=cls,Title=title.ToString()});return true;
  };
  EnumWindows(delegate(IntPtr window,IntPtr state) {
   uint owner;GetWindowThreadProcessId(window,out owner);
   if(owner==(uint)process) {read(window,state);EnumChildWindows(window,read,IntPtr.Zero);}
   return entries.Count<2000;
  },IntPtr.Zero);
  return new ScanResult{Entries=entries.ToArray(),Truncated=truncated || entries.Count>=2000};
 }
}
'@
 function Select-ReceiptNativeWindow($entries,[int]$process) {
  $seen=New-Object 'Collections.Generic.HashSet[long]'
  foreach($entry in $entries){
   if($entry.Process -eq $process -and $entry.Visible -and $entry.Title -match 'PG03F2120' -and $seen.Add([long]$entry.Handle)){$entry}
  }
 }
 $scan=[AlphaReceiptWindows]::Scan([int]$erp[0].Id)
 $report.discovery_truncated=$scan.Truncated
 $report.native_windows=$scan.Entries.Count
 $discovery=New-Object 'Collections.Generic.List[object]'
 foreach($entry in $scan.Entries){
  # Diagnostic titles are reduced to module codes/fixed labels. Never collect
  # arbitrary text from other ERP modules during window discovery.
  $code=[regex]::Match($entry.Title,'(?i)PG[0-9A-Z]{5,20}').Value
  $label=if($entry.Title -match '입고등록'){'입고등록'}else{''}
  if($entry.Visible -and ($code -or $label -or $entry.Parent -eq 0) -and $discovery.Count -lt 100){
   $discovery.Add([ordered]@{source='win32';handle=[string]$entry.Handle;parent=[string]$entry.Parent;module=$code;label=$label;class=$entry.Class;visible=$entry.Visible})
  }
 }
 $report.discovery=@($discovery.ToArray())
 $nativeTargets=@(Select-ReceiptNativeWindow $scan.Entries ([int]$erp[0].Id))
 $targets=New-Object 'Collections.Generic.List[object]'
 if($nativeTargets.Count -gt 1){$report.reason='window_ambiguous';throw 'stop'}
 if($nativeTargets.Count -eq 1){
  $targets.Add([System.Windows.Automation.AutomationElement]::FromHandle([IntPtr]$nativeTargets[0].Handle));$report.discovery_method='win32'
 }else{
  $pidCondition=New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::ProcessIdProperty,[int]$erp[0].Id)
  $windows=[System.Windows.Automation.AutomationElement]::RootElement.FindAll([System.Windows.Automation.TreeScope]::Children,$pidCondition)
  $walk=[System.Windows.Automation.TreeWalker]::RawViewWalker
  $pending=New-Object 'Collections.Generic.Queue[object]'
  for($i=0;$i -lt $windows.Count;$i++){$pending.Enqueue(@{element=$windows[$i];depth=0})}
  $seen=New-Object 'Collections.Generic.HashSet[string]';$visited=0
  while($pending.Count -gt 0 -and $visited -lt 1200){
   $entry=$pending.Dequeue();$visited++
   try{
    $element=$entry.element;$cur=$element.Current
    if($cur.ProcessId -ne $erp[0].Id -or $cur.IsPassword){continue}
    $type=$cur.ControlType.ProgrammaticName
    if($type -match '(Window|Pane|Custom)$' -and $cur.Name -match 'PG03F2120' -and !$cur.IsOffscreen){
     $identity=if($cur.NativeWindowHandle -ne 0){[string]$cur.NativeWindowHandle}else{[string]::Join(',', $element.GetRuntimeId())}
     if($seen.Add($identity)){$targets.Add($element)}
     continue
    }
    if($entry.depth -lt 10 -and $type -notmatch '(Edit|DataGrid|Table|DataItem|ListItem|Document|Text)$'){
     $child=$walk.GetFirstChild($element);$siblings=0
     while($null -ne $child -and $siblings -lt 300 -and $pending.Count -lt 1800){$pending.Enqueue(@{element=$child;depth=$entry.depth+1});$siblings++;$child=$walk.GetNextSibling($child)}
     if($null -ne $child){$report.discovery_truncated=$true}
    }
   }catch{$report.read_errors++}
  }
  $report.uia_visited=$visited
  if($pending.Count -gt 0){$report.discovery_truncated=$true}
  if($targets.Count -gt 0){$report.discovery_method='uia_descendants'}
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
   $item=[ordered]@{index=$index;parent=[int]$entry.parent;depth=[int]$entry.depth;type=$cur.ControlType.ProgrammaticName;class=$cur.ClassName;automation_id=$cur.AutomationId;name='';value='';patterns=@();enabled=$cur.IsEnabled;offscreen=$cur.IsOffscreen;row=-1;column=-1;grid_rows=-1;grid_columns=-1;toggle='';selected=$false;legacy_role=0;legacy_state=0;legacy_child_id=0}
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
   $pattern=$null
   try{if($element.TryGetCurrentPattern([System.Windows.Automation.LegacyIAccessiblePattern]::Pattern,[ref]$pattern)){
    $legacy=$pattern.Current
    $item.legacy_role=[uint32]$legacy.Role;$item.legacy_state=[uint32]$legacy.State;$item.legacy_child_id=[int]$legacy.ChildId
    if(!$cur.IsOffscreen){if(!$item.name){$item.name=[string]$legacy.Name};if(!$item.value){$item.value=[string]$legacy.Value}}
   }}catch{$report.read_errors++}
   foreach($field in @('name','value')){if($item[$field].Length -gt 1500){$item[$field]=$item[$field].Substring(0,1500);$report.truncated=$true}}
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
