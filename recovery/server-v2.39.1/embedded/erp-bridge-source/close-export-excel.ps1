$ErrorActionPreference='Stop'
$trace=[ordered]@{stage='load';matched=0;closed=0;remaining=0;skipped_unsaved=0;errors=0;owned_foreground_pid=0;close_verified=$false}
function Write-Trace { [IO.File]::WriteAllText($ResultPath,(ConvertTo-Json -InputObject $trace -Depth 5),(New-Object System.Text.UTF8Encoding($false))) }
try {
 Add-Type -TypeDefinition @'
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;
public static class AlphaExportExcelWindows {
 public delegate bool EnumProc(IntPtr h,IntPtr p);
 [DllImport("user32.dll")] public static extern bool EnumWindows(EnumProc c,IntPtr p);
 [DllImport("user32.dll")] public static extern bool EnumChildWindows(IntPtr h,EnumProc c,IntPtr p);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] public static extern int GetClassName(IntPtr h,StringBuilder b,int n);
 [DllImport("user32.dll")] public static extern IntPtr GetAncestor(IntPtr h,uint flag);
 [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
 [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
 [DllImport("oleacc.dll")] public static extern int AccessibleObjectFromWindow(IntPtr h,uint id,ref Guid iid,[MarshalAs(UnmanagedType.IDispatch)] out object o);
 public static string ClassOf(IntPtr h){var b=new StringBuilder(128);GetClassName(h,b,128);return b.ToString();}
 public static List<IntPtr> NativeExcelWindows(){
  var list=new List<IntPtr>();
  EnumWindows((h,p)=>{if(ClassOf(h)=="XLMAIN")EnumChildWindows(h,(c,q)=>{if(ClassOf(c)=="EXCEL7")list.Add(c);return true;},IntPtr.Zero);return true;},IntPtr.Zero);
  return list;
 }
 public static object NativeObject(IntPtr h){object o;var iid=new Guid("00020400-0000-0000-C000-000000000046");return AccessibleObjectFromWindow(h,0xfffffff0,ref iid,out o)==0?o:null;}
 public static IntPtr ExcelHwnd(long h){return new IntPtr(h & 0xffffffffL);}
 public static uint WindowPID(IntPtr h){uint pid;GetWindowThreadProcessId(h,out pid);return pid;}
}
'@
 $target=[IO.Path]::GetFullPath($ExportPath)
 function Is-Our-Workbook($book){
  try { return [String]::Equals([IO.Path]::GetFullPath([string]$book.FullName),$target,[StringComparison]::OrdinalIgnoreCase) }catch{return $false}
 }
 function Excel-Applications {
  $apps=New-Object Collections.ArrayList
  try{$active=[Runtime.InteropServices.Marshal]::GetActiveObject('Excel.Application');[void]$apps.Add($active)}catch{}
  foreach($h in [AlphaExportExcelWindows]::NativeExcelWindows()){
   try{$native=[AlphaExportExcelWindows]::NativeObject($h);if($null -ne $native){[void]$apps.Add($native.Application)}}catch{}
  }
  return ,$apps
 }
 $until=[DateTime]::UtcNow.AddSeconds(8)
 do {
  $trace.stage='locate_exact_export'
  $apps=Excel-Applications
  foreach($app in $apps){
   try {
    $books=$app.Workbooks
    for($i=$books.Count;$i -ge 1;$i--){
     $book=$books.Item($i)
     if(!(Is-Our-Workbook $book)){continue}
     $trace.matched++
     # Never discard edits, even in this tool-owned output. Leave an edited
     # workbook open and report it instead of changing any application settings.
     if(!$book.Saved){$trace.skipped_unsaved++;continue}
     $foreground=[AlphaExportExcelWindows]::GetForegroundWindow()
     $foregroundOwned=$false
     foreach($win in $book.Windows){
      $root=[AlphaExportExcelWindows]::GetAncestor(([AlphaExportExcelWindows]::ExcelHwnd([long]$win.Hwnd)),2)
      if($root -eq $foreground){$foregroundOwned=$true}
     }
     $ownedPID=0
     if($foregroundOwned){$ownedPID=[AlphaExportExcelWindows]::WindowPID($foreground)}
     $trace.stage='close_exact_export';Write-Trace
     $book.Close($false)
     $stillOpen=$false
     for($j=1;$j -le $books.Count;$j++){if(Is-Our-Workbook ($books.Item($j))){$stillOpen=$true;break}}
     if(!$stillOpen){$trace.closed++;if($foregroundOwned){$trace.owned_foreground_pid=$ownedPID}}
     else{$trace.remaining++}
    }
   } catch {$trace.errors++}
  }
  if($trace.matched -gt 0){break}
  Start-Sleep -Milliseconds 400
 }while([DateTime]::UtcNow -lt $until)
 $trace.close_verified=($trace.closed -gt 0 -and $trace.remaining -eq 0 -and $trace.skipped_unsaved -eq 0)
 $trace.stage=if($trace.close_verified){'export_closed'}elseif($trace.skipped_unsaved -gt 0){'left_unsaved_export_open'}elseif($trace.matched -eq 0){'export_not_found'}else{'close_not_confirmed'}
}catch{$trace.stage='unavailable';$trace.error_type=$_.Exception.GetType().FullName;$trace.hresult=$_.Exception.HResult}
Write-Trace
