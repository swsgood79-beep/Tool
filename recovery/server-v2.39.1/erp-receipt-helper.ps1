$ErrorActionPreference='Stop'
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName Accessibility
$filterSource=@'
# FILTER_SOURCE
'@
Add-Type -TypeDefinition $filterSource -ReferencedAssemblies @('System.dll','System.Core.dll',[Accessibility.IAccessible].Assembly.Location)
$task=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($TaskBase64))|ConvertFrom-Json
if([DateTimeOffset]::Parse($task.expires) -le [DateTimeOffset]::Now){throw 'expired'}
$form=New-Object Windows.Forms.Form
$form.Text='Alpha · ERP 입고 입력 도우미'
$form.Size=New-Object Drawing.Size(540,700)
$form.MinimumSize=New-Object Drawing.Size(430,500)
$form.TopMost=$true
$form.StartPosition='CenterScreen'
$form.Font=New-Object Drawing.Font('맑은 고딕',10)
$layout=New-Object Windows.Forms.TableLayoutPanel
$layout.Dock='Fill';$layout.ColumnCount=1;$layout.RowCount=7
foreach($height in @(48,42,50,42,52,0,70)){
 $style=New-Object Windows.Forms.RowStyle
 if($height -eq 0){$style.SizeType='Percent';$style.Height=100}else{$style.SizeType='Absolute';$style.Height=$height}
 [void]$layout.RowStyles.Add($style)
}
$groups=New-Object Windows.Forms.ComboBox;$groups.Dock='Fill';$groups.DropDownStyle='DropDownList'
foreach($group in $task.groups){[void]$groups.Items.Add(($group.customer+' / 발주 '+$group.purchase))}
$layout.Controls.Add($groups,0,0)
$vendor=New-Object Windows.Forms.Label;$vendor.Dock='Fill';$layout.Controls.Add($vendor,0,1)
$buttons=New-Object Windows.Forms.FlowLayoutPanel;$buttons.Dock='Fill'
$copyVendor=New-Object Windows.Forms.Button;$copyVendor.Text='거래처 코드 복사';$copyVendor.Width=150;$copyVendor.Height=38
$copyPurchase=New-Object Windows.Forms.Button;$copyPurchase.Text='발주번호 복사';$copyPurchase.Width=140;$copyPurchase.Height=38
$buttons.Controls.AddRange(@($copyVendor,$copyPurchase));$layout.Controls.Add($buttons,0,2)
$confirmed=New-Object Windows.Forms.CheckBox;$confirmed.Text='현재 ERP 거래처가 위 거래처와 같습니다';$confirmed.Dock='Fill';$layout.Controls.Add($confirmed,0,3)
$search=New-Object Windows.Forms.Button;$search.Text='열린 ERP 창에 발주번호 입력 · 검색';$search.Dock='Fill';$search.Enabled=$false;$layout.Controls.Add($search,0,4)
$lines=New-Object Windows.Forms.TextBox;$lines.Multiline=$true;$lines.ReadOnly=$true;$lines.WordWrap=$true;$lines.ScrollBars='Vertical';$lines.Dock='Fill';$layout.Controls.Add($lines,0,5)
$status=New-Object Windows.Forms.Label;$status.Text='ERP에서 거래처 지정 → Data 불러오기를 여세요. 품목 체크·수량 입력·최종 저장은 직접 합니다.';$status.Dock='Fill';$layout.Controls.Add($status,0,6)
$form.Controls.Add($layout)
$groups.Add_SelectedIndexChanged({
 $group=$task.groups[$groups.SelectedIndex];$vendor.Text=$group.customer+' (코드 '+$group.customer_code+') / 사업장 '+$group.unit
 $confirmed.Checked=$false
 $text=New-Object 'Collections.Generic.List[string]'
 foreach($line in $group.lines){if($line.part){$text.Add([string]$line.part)}else{$text.Add('표시용 품번 미등록')};if($line.item_code){$text.Add('ERP 품목 코드: '+$line.item_code)};$text.Add($line.item+' · 입고할 수량 '+$line.quantity+' / 잔량 '+$line.remaining);$text.Add('수주 '+$line.order+' / 발주일 '+$line.purchase_date+' / 발주 행 '+$line.sequence);$text.Add('')}
 $lines.Text=$text -join "`r`n"
})
$copyVendor.Add_Click({[Windows.Forms.Clipboard]::SetText([string]$task.groups[$groups.SelectedIndex].customer_code)})
$copyPurchase.Add_Click({[Windows.Forms.Clipboard]::SetText([string]$task.groups[$groups.SelectedIndex].purchase)})
$confirmed.Add_CheckedChanged({$search.Enabled=$confirmed.Checked})
$search.Add_Click({
 $search.Enabled=$false
 try{
  if([DateTimeOffset]::Parse($task.expires) -le [DateTimeOffset]::Now){$status.Text='준비 목록이 만료됐습니다. Alpha에서 새로고침 후 다시 준비해 주세요';return}
  $processes=@(Get-Process -Name PrimeXAR -ErrorAction SilentlyContinue)
  if($processes.Count -ne 1){$status.Text='ERP 한 개만 실행해 주세요. 발주번호 복사는 계속 사용할 수 있습니다';return}
  $status.Text=[AlphaReceiptFilter]::Apply([int]$processes[0].Id,[string]$task.groups[$groups.SelectedIndex].purchase)
 }catch{$status.Text='자동 검색을 완료하지 못했습니다. 발주번호 복사로 ERP에서 직접 검색해 주세요'}
 finally{$search.Enabled=$confirmed.Checked}
})
$timer=New-Object Windows.Forms.Timer;$timer.Interval=30000
$timer.Add_Tick({if([DateTimeOffset]::Parse($task.expires) -le [DateTimeOffset]::Now){$form.Close()}})
$form.Add_Shown({[Console]::WriteLine('ALPHA_HELPER_READY');[Console]::Out.Flush();$timer.Start()})
$groups.SelectedIndex=0
[Windows.Forms.Application]::Run($form)
$timer.Dispose();$form.Dispose()
