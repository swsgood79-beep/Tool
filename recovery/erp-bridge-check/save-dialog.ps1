$ErrorActionPreference='Stop'
try {
 Add-Type -AssemblyName UIAutomationClient
 Add-Type -AssemblyName UIAutomationTypes
 $root=[System.Windows.Automation.AutomationElement]::FromHandle([IntPtr]$DialogHandle)
 $all=$root.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition)
 $filename=$null; $save=$null
 foreach($el in $all){
  $cur=$el.Current
  if($cur.ControlType -eq [System.Windows.Automation.ControlType]::Edit -and $cur.AutomationId -eq '1001'){$filename=$el}
  if($cur.ControlType -eq [System.Windows.Automation.ControlType]::Button -and $cur.AutomationId -eq '1' -and $cur.Name -match '저장|Save'){$save=$el}
 }
 if($null -eq $filename -or $null -eq $save){throw 'File name or Save control unavailable'}
 $value=$filename.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)
 if($value.Current.IsReadOnly){throw 'File name is readonly'}
 $value.SetValue($ExportPath)
 $invoke=$save.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern)
 $invoke.Invoke()
 exit 0
} catch { exit 2 }
