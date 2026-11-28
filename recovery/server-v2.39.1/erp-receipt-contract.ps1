 $report.stage='receipt_contract'
 $sourceSql=@'
SELECT TOP (3) o.name,m.definition
FROM sys.objects o LEFT JOIN sys.sql_modules m ON m.object_id=o.object_id
WHERE o.schema_id=SCHEMA_ID('dbo') AND o.type='P'
AND o.name IN ('PC_PDTWHSMST_UPDATE2','PC_PDTWHSHIS_UPDATE','PC_PDTWHSMST_AMT_UPDATE')
ORDER BY o.name
'@
 $cmd=$conn.CreateCommand();$cmd.CommandText=$sourceSql;$cmd.CommandTimeout=10
 $rd=$cmd.ExecuteReader();$rows=New-Object 'Collections.Generic.List[object]'
 try{while($rd.Read()){
  $name=[string]$rd[0];$body=if($rd.IsDBNull(1)){''}else{[string]$rd[1]};$visible=($body.Length -gt 0);$redacted=$false
  if($body -match '(?i)password|pwd\s*=|USER_PW|MAIL_PW|OPENROWSET|OPENDATASOURCE|CREATE\s+LOGIN|ALTER\s+LOGIN'){$body='';$redacted=$true}
  if($body.Length -gt 200000){$body='';$report.truncated=$true}
  $rows.Add([ordered]@{name=$name;definition=$body;visible=$visible;redacted=$redacted})
 }}finally{$rd.Close();$cmd.Dispose()}
 $report.receipt_definitions=@($rows.ToArray());$report.status='receipt_contract';$report.stage='done'
