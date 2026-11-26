$ErrorActionPreference='Stop'
$trace=[ordered]@{method='UIAutomation';stage='load';success=$false;filename_candidates=0;save_candidates=0}
try {
 Add-Type -AssemblyName UIAutomationClient
 Add-Type -AssemblyName UIAutomationTypes
 $root=[System.Windows.Automation.AutomationElement]::FromHandle([IntPtr]$DialogHandle)
 $rect=$root.Current.BoundingRectangle
 $trace.stage='locate_controls'
 $edits=@();$buttons=@()
 for($retry=0;$retry -lt 5;$retry++){
  $edits=@();$buttons=@()
  $all=$root.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition)
  foreach($el in $all){
   try{
    $cur=$el.Current;if(!$cur.IsEnabled -or $cur.IsOffscreen){continue}
    if($cur.ControlType -eq [System.Windows.Automation.ControlType]::Edit -and ($cur.AutomationId -in @('1001','1152') -or $cur.Name -match '파일 이름|File name') -and $cur.BoundingRectangle.Top -ge ($rect.Top+$rect.Height*0.55)){$edits+=,$el}
    if($cur.ControlType -eq [System.Windows.Automation.ControlType]::Button -and $cur.AutomationId -eq '1' -and $cur.Name -match '저장|Save'){$buttons+=,$el}
   }catch{}
  }
  if($edits.Count -eq 1 -and $buttons.Count -eq 1){break};Start-Sleep -Milliseconds 200
 }
 $trace.filename_candidates=$edits.Count;$trace.save_candidates=$buttons.Count
 if($edits.Count -ne 1 -or $buttons.Count -ne 1){throw 'controls_not_unique'}
 $trace.stage='set_filename';$value=$edits[0].GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)
 if($value.Current.IsReadOnly){throw 'readonly'}
 $value.SetValue($ExportPath)
 $trace.stage='verify_filename';if($value.Current.Value -ne $ExportPath){throw 'value_mismatch'}
 $trace.stage='invoke_save';$invoke=$buttons[0].GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern);$invoke.Invoke()
 $trace.stage='save_requested';$trace.success=$true
}catch{$trace.error_type=$_.Exception.GetType().FullName;$trace.hresult=$_.Exception.HResult}
$json=ConvertTo-Json -InputObject $trace -Depth 5
[IO.File]::WriteAllText($ResultPath,$json,(New-Object System.Text.UTF8Encoding($false)))
if($trace.success){exit 0}else{exit 2}
