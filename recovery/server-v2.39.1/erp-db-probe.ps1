$ErrorActionPreference='Stop'
$ProgressPreference='SilentlyContinue'
[Console]::OutputEncoding=New-Object Text.UTF8Encoding($false)
$report=[ordered]@{status='failed';stage='config_path';config_reason='';connected=$false;error_code=0;columns=@();foreign_keys=@();purchase_rows=@();outstanding_rows=@();customer_candidates=@();truncated=$false}
$conn=$null
try {
 Add-Type -AssemblyName System.Data
 if([string]::IsNullOrWhiteSpace($ERPFolder)){
  $roots=@(Get-Process -Name PrimeXAR -ErrorAction SilentlyContinue | ForEach-Object {try{if($_.Path){[IO.Path]::GetDirectoryName($_.Path)}}catch{}} | Sort-Object -Unique)
  if($roots.Count -eq 1){$ERPFolder=$roots[0]}
  elseif($roots.Count -gt 1){$report.status='ambiguous';throw 'stop'}
  elseif([IO.Directory]::Exists('C:\PrimeXAR(패치)(품질)')){$ERPFolder='C:\PrimeXAR(패치)(품질)'}
  else{$report.status='config_missing';throw 'stop'}
 }
 $ini=Join-Path ([IO.Path]::GetFullPath($ERPFolder)) 'ini'
 if(![IO.Directory]::Exists($ini)){$report.status='config_missing';throw 'stop'}
 $files=@([IO.Directory]::GetFiles($ini,'DB_Information*')|Where-Object{([IO.Path]::GetFileNameWithoutExtension($_) -eq 'DB_Information' -and [IO.Path]::GetExtension($_) -eq '.ini') -or [IO.Path]::GetFileName($_) -eq 'DB_Information'})
 if($files.Count -ne 1){$report.status=if($files.Count -gt 1){'ambiguous'}else{'config_missing'};throw 'stop'}
 if((Get-Item -LiteralPath $files[0]).Length -gt 65536){$report.status='config_invalid';throw 'stop'}
 $report.stage='config_read'
 $reader=New-Object IO.StreamReader($files[0],[Text.Encoding]::Default,$true)
 try{$text=$reader.ReadToEnd()}finally{$reader.Dispose()}
 $report.config_reason='active_key_missing'
 $section='';$connections=New-Object 'Collections.Generic.List[string]'
 foreach($line in ($text -split '\r?\n')){
  $t=$line.Trim();if(!$t -or $t.StartsWith('//') -or $t.StartsWith(';') -or $t.StartsWith('#')){continue}
  if($t -match '^\[([^\]]+)\]$'){$section=$Matches[1];continue}
  if($section -eq 'DB_Information' -and $t -match '^DatabasePath\s*=(.*)$'){$connections.Add($Matches[1].Trim())}
 }
 if($connections.Count -ne 1){$report.status='config_invalid';if($connections.Count -gt 1){$report.config_reason='active_key_multiple'};throw 'stop'}
 $report.status='config_invalid';$report.stage='config_parse';$report.config_reason='setting_format'
 # SqlConnectionStringBuilder implements IDictionary. PowerShell dot-property
 # adaptation can treat property names as connection keywords (DataSource vs
 # Data Source). Call the CLR accessors explicitly, including ConnectionString.
 $builder=New-Object System.Data.SqlClient.SqlConnectionStringBuilder
 $builder.set_ConnectionString($connections[0])
 if($builder.get_AttachDBFilename() -or $builder.get_UserInstance() -or $builder.get_ContextConnection()){$report.config_reason='unsupported_options';throw 'stop'}
 if(!$builder.get_DataSource() -or !$builder.get_InitialCatalog()){$report.config_reason='missing_server_or_database';throw 'stop'}
 $report.stage='connection_options';$report.config_reason='unsupported_options'
 $builder.set_ConnectTimeout(8);$builder.set_ApplicationName('Alpha ERP Metadata Check');$builder.set_Pooling($false);$builder.set_PersistSecurityInfo($false)
 $builder.set_ApplicationIntent([System.Data.SqlClient.ApplicationIntent]::ReadOnly)
 $builder.set_Encrypt($true);$builder.set_TrustServerCertificate($true)
 $conn=New-Object System.Data.SqlClient.SqlConnection
 $conn.set_ConnectionString($builder.get_ConnectionString())
 $report.config_reason=''
 $connections.Clear();$text=$null;$builder=$null
 $report.status='failed'
 $report.stage='connect'
 $conn.Open();$report.connected=$true
 if($ReceiptContract){
 # RECEIPT_CONTRACT_SCRIPT
 }elseif($CertificateOrder){
 # CERTIFICATE_SCRIPT
 }elseif($Observe){
 # OBSERVATION_SCRIPT
 }elseif($ObservationAccess){
 # OBSERVATION_ACCESS_SCRIPT
 }elseif($ReceiptMetadata){
 # RECEIPT_METADATA_SCRIPT
 }elseif($OutstandingOrder -or $OutstandingOrders){
  $report.stage='order_outstanding'
  $orderNumbers=@($OutstandingOrders)
  if($OutstandingOrder){$orderNumbers=@($OutstandingOrder)}
  if($orderNumbers.Count -lt 1 -or $orderNumbers.Count -gt 10){throw 'invalid order count'}
  $orderParameters=@(for($index=0;$index -lt $orderNumbers.Count;$index++){"@Order$index"})
  $orderFilter='ORDER_EXPR IN ('+($orderParameters -join ',')+')'
  function Add-OutstandingParameters($command){
   for($index=0;$index -lt $orderNumbers.Count;$index++){
    [void]$command.Parameters.Add($orderParameters[$index],[Data.SqlDbType]::NVarChar,100)
    $command.Parameters[$orderParameters[$index]].Value=$orderNumbers[$index]
   }
  }
  # Discover optional line-level order and unit keys. Schema mismatch fails the
  # query rather than returning an empty success or falling back to BASE rows.
  $schema=$conn.CreateCommand()
  $schema.CommandText="SELECT COL_LENGTH('dbo.PURORD_HIS','RCVORD_NO'),COL_LENGTH('dbo.CSTMST_INFO','UNT_CD'),COL_LENGTH('dbo.PURORD_HIS','CST_CD')"
  $reader=$schema.ExecuteReader();[void]$reader.Read()
  $lineOrder=!$reader.IsDBNull(0);$customerUnit=!$reader.IsDBNull(1);$lineCustomer=!$reader.IsDBNull(2)
  $reader.Close();$schema.Dispose()
  $orderExpr='m.RCVORD_NO'
  if($lineOrder){$orderExpr="COALESCE(NULLIF(LTRIM(RTRIM(h.RCVORD_NO)),''),m.RCVORD_NO)"}
  $customerExpr='m.CST_CD'
  if($lineCustomer){$customerExpr="COALESCE(NULLIF(LTRIM(RTRIM(h.CST_CD)),''),m.CST_CD)"}
  $customerUnitExpr="CAST('' AS nvarchar(20))"
  if($customerUnit){$customerUnitExpr='c.UNT_CD'}
  $sql=@'
SELECT TOP (2001)
 CASE WHEN robot.matches=1 THEN robot.MITM_CD ELSE '' END AS robot,
 i.ITM_NM,ISNULL(h.PUR_QTY,0)-ISNULL(h.IN_QTY,0) AS outstanding,
 CAST('' AS nvarchar(1)) AS customer,h.SCH_DATE,CUSTOMER_EXPR,h.UNT_CD,ORDER_EXPR AS order_number, h.PURORD_DATE,h.PURORD_NO,h.PURORD_SEQ,h.ITM_CD
FROM dbo.PURORD_MST m JOIN dbo.PURORD_HIS h
 ON h.UNT_CD=m.UNT_CD AND h.PURORD_DATE=m.PURORD_DATE AND h.PURORD_NO=m.PURORD_NO
LEFT JOIN dbo.ITMMST_INFO i ON i.ITM_CD=h.ITM_CD AND i.PARTS_TYPE=h.PARTS_TYPE
OUTER APPLY (SELECT MIN(d.MITM_CD) AS MITM_CD,COUNT(DISTINCT NULLIF(LTRIM(RTRIM(d.MITM_CD)),'')) AS matches
 FROM dbo.DEMREQ_HIS d
 WHERE d.UNT_CD=h.UNT_CD AND d.DEMREQ_DATE=h.DEMREQ_DATE AND d.DEMREQ_NO=h.DEMREQ_NO AND d.DEMREQ_SEQ=h.DEMREQ_SEQ
 AND d.TITM_CD=h.ITM_CD AND d.PARTS_TYPE=h.PARTS_TYPE AND d.RCVORD_NO=ORDER_EXPR) robot
WHERE ORDER_FILTER AND ISNULL(h.PUR_QTY,0)>ISNULL(h.IN_QTY,0)
ORDER BY h.UNT_CD,h.PURORD_DATE,h.PURORD_NO,h.PURORD_SEQ
'@
  $cmd=$conn.CreateCommand();$cmd.CommandText=$sql.Replace('CUSTOMER_EXPR',$customerExpr).Replace('ORDER_FILTER',$orderFilter).Replace('ORDER_EXPR',$orderExpr);$cmd.CommandTimeout=20
  Add-OutstandingParameters $cmd
  $rd=$cmd.ExecuteReader();$rows=New-Object 'Collections.Generic.List[object]'
  try{while($rd.Read()){
   if($rows.Count -ge 2000){$report.truncated=$true;break}
   $rows.Add([ordered]@{robot=[string]$rd[0];item=[string]$rd[1];quantity=[decimal]$rd[2];customer=[string]$rd[3];due=[string]$rd[4];customer_code=[string]$rd[5];unit=[string]$rd[6];order=[string]$rd[7];purchase_date=[string]$rd[8];purchase=[string]$rd[9];sequence=[int]$rd[10];part=[string]$rd[11]})
  }}finally{$rd.Close();$cmd.Dispose()}
  $report.outstanding_rows=@($rows.ToArray())
  # Restrict master candidates to supplier codes on this order's outstanding
  # lines. Resolve unit-specific/shared masters without multiplying item rows.
  $customerSql=@'
SELECT DISTINCT TOP (20001) c.CST_CD,c.CST_NM,CUSTOMER_UNIT
FROM dbo.CSTMST_INFO c
WHERE EXISTS (SELECT 1 FROM dbo.PURORD_MST m JOIN dbo.PURORD_HIS h
 ON h.UNT_CD=m.UNT_CD AND h.PURORD_DATE=m.PURORD_DATE AND h.PURORD_NO=m.PURORD_NO
 WHERE ORDER_FILTER AND ISNULL(h.PUR_QTY,0)>ISNULL(h.IN_QTY,0)
 AND LTRIM(RTRIM(c.CST_CD))=LTRIM(RTRIM(CUSTOMER_EXPR)))
ORDER BY c.CST_CD,c.CST_NM
'@
  $cmd=$conn.CreateCommand();$cmd.CommandText=$customerSql.Replace('CUSTOMER_UNIT',$customerUnitExpr).Replace('CUSTOMER_EXPR',$customerExpr).Replace('ORDER_FILTER',$orderFilter).Replace('ORDER_EXPR',$orderExpr);$cmd.CommandTimeout=12
  Add-OutstandingParameters $cmd
  $rd=$cmd.ExecuteReader();$customers=New-Object 'Collections.Generic.List[object]'
  try{while($rd.Read()){
   if($customers.Count -ge 20000){$report.truncated=$true;break}
   $customers.Add([ordered]@{code=[string]$rd[0];name=[string]$rd[1];unit=[string]$rd[2]})
  }}finally{$rd.Close();$cmd.Dispose()}
  $report.customer_candidates=@($customers.ToArray());$report.status='order_outstanding';$report.stage='done' 
 }elseif($PurchasePreview){
  $report.stage='purchase_preview'
  $purchaseSql=@'
SELECT TOP (2001) h.UNT_CD,h.PURORD_DATE,h.PURORD_NO,h.PURORD_SEQ,m.RCVORD_NO,m.PROJECT,
 h.ITM_CD,h.PARTS_TYPE,h.ITM_SIZE,h.BOM_CODE,h.SCH_DATE,ISNULL(h.PUR_QTY,0),ISNULL(h.IN_QTY,0),h.WHS_STS,
 i.ITM_NM,i.ITM_SIZE,i.PART_NO,robot.MITM_CD,robot.matches
FROM dbo.PURORD_MST m JOIN dbo.PURORD_HIS h
 ON h.UNT_CD=m.UNT_CD AND h.PURORD_DATE=m.PURORD_DATE AND h.PURORD_NO=m.PURORD_NO
LEFT JOIN dbo.ITMMST_INFO i ON i.ITM_CD=h.ITM_CD AND i.PARTS_TYPE=h.PARTS_TYPE
OUTER APPLY (SELECT MIN(d.MITM_CD) AS MITM_CD,COUNT(*) AS matches FROM dbo.DEMREQ_HIS d
 WHERE d.UNT_CD=h.UNT_CD AND d.DEMREQ_DATE=h.DEMREQ_DATE AND d.DEMREQ_NO=h.DEMREQ_NO AND d.DEMREQ_SEQ=h.DEMREQ_SEQ
 AND d.TITM_CD=h.ITM_CD AND d.PARTS_TYPE=h.PARTS_TYPE AND d.RCVORD_NO=m.RCVORD_NO) robot
WHERE m.CST_CD=@Customer AND LEFT(REPLACE(REPLACE(REPLACE(LTRIM(RTRIM(h.SCH_DATE)),'-',''),'/',''),'.',''),8) BETWEEN @From AND @To
AND (@BaseOnly=0 OR (UPPER(LTRIM(RTRIM(i.ITM_NM)))='BASE' AND ISNULL(h.PUR_QTY,0)>ISNULL(h.IN_QTY,0)))
ORDER BY h.UNT_CD,h.PURORD_DATE,h.PURORD_NO,h.PURORD_SEQ
'@
  $cmd=$conn.CreateCommand();$cmd.CommandText=$purchaseSql;$cmd.CommandTimeout=12
  [void]$cmd.Parameters.Add('@Customer',[Data.SqlDbType]::VarChar,20);$cmd.Parameters['@Customer'].Value='00530'
  [void]$cmd.Parameters.Add('@From',[Data.SqlDbType]::VarChar,8);$cmd.Parameters['@From'].Value=$DateFrom
  [void]$cmd.Parameters.Add('@To',[Data.SqlDbType]::VarChar,8);$cmd.Parameters['@To'].Value=$DateTo
  [void]$cmd.Parameters.Add('@BaseOnly',[Data.SqlDbType]::Bit);$cmd.Parameters['@BaseOnly'].Value=[bool]$SyncOnly
  $rd=$cmd.ExecuteReader();$rows=New-Object 'Collections.Generic.List[object]'
  try{while($rd.Read()){if($rows.Count -ge 2000){$report.truncated=$true;break};$rows.Add([ordered]@{unit=[string]$rd[0];purchase_date=[string]$rd[1];purchase=[string]$rd[2];sequence=[int]$rd[3];order=[string]$rd[4];project=[string]$rd[5];item=[string]$rd[6];parts_type=[string]$rd[7];line_size=[string]$rd[8];bom=[string]$rd[9];due=[string]$rd[10];quantity=[int]$rd[11];received=[int]$rd[12];warehouse_status=[string]$rd[13];item_name=[string]$rd[14];master_size=[string]$rd[15];part_no=[string]$rd[16];robot=[string]$rd[17];robot_matches=[int]$rd[18]})}}finally{$rd.Close();$cmd.Dispose()}
  $report.purchase_rows=@($rows.ToArray());$report.status='purchase_preview';$report.stage='done'
 }else{
 $report.stage='columns'
 $columnSql=@'
SELECT TOP (20001) s.name AS schema_name,o.name AS object_name,o.type AS object_kind,
 c.name AS column_name,t.name AS type_name,c.is_nullable,
 CAST(CASE WHEN EXISTS(SELECT 1 FROM sys.indexes ix JOIN sys.index_columns ic ON ic.object_id=ix.object_id AND ic.index_id=ix.index_id WHERE ix.object_id=o.object_id AND ix.is_primary_key=1 AND ic.column_id=c.column_id) THEN 1 ELSE 0 END AS bit) AS is_primary_key
FROM sys.objects o JOIN sys.schemas s ON s.schema_id=o.schema_id
JOIN sys.columns c ON c.object_id=o.object_id JOIN sys.types t ON t.user_type_id=c.user_type_id
WHERE o.type IN ('U','V') AND o.is_ms_shipped=0 ORDER BY s.name,o.name,c.column_id
'@
 $cmd=$conn.CreateCommand();$cmd.CommandText=$columnSql;$cmd.CommandTimeout=12
 $rd=$cmd.ExecuteReader();$cols=New-Object 'Collections.Generic.List[object]'
 try{while($rd.Read()){if($cols.Count -ge 20000){$report.truncated=$true;break};$cols.Add([ordered]@{schema=[string]$rd[0];object=[string]$rd[1];kind=([string]$rd[2]).Trim();column=[string]$rd[3];type=[string]$rd[4];nullable=[bool]$rd[5];primary_key=[bool]$rd[6]})}}finally{$rd.Close();$cmd.Dispose()}
 $report.columns=@($cols.ToArray())
 $report.stage='relations'
 $foreignSql=@'
SELECT TOP (5001) ps.name,po.name,pc.name,rs.name,ro.name,rc.name
FROM sys.foreign_key_columns f
JOIN sys.objects po ON po.object_id=f.parent_object_id JOIN sys.schemas ps ON ps.schema_id=po.schema_id
JOIN sys.columns pc ON pc.object_id=po.object_id AND pc.column_id=f.parent_column_id
JOIN sys.objects ro ON ro.object_id=f.referenced_object_id JOIN sys.schemas rs ON rs.schema_id=ro.schema_id
JOIN sys.columns rc ON rc.object_id=ro.object_id AND rc.column_id=f.referenced_column_id
ORDER BY ps.name,po.name,pc.name
'@
 $cmd=$conn.CreateCommand();$cmd.CommandText=$foreignSql;$cmd.CommandTimeout=12
 $rd=$cmd.ExecuteReader();$keys=New-Object 'Collections.Generic.List[object]'
 try{while($rd.Read()){if($keys.Count -ge 5000){$report.truncated=$true;break};$keys.Add([ordered]@{schema=[string]$rd[0];object=[string]$rd[1];column=[string]$rd[2];ref_schema=[string]$rd[3];ref_object=[string]$rd[4];ref_column=[string]$rd[5]})}}finally{$rd.Close();$cmd.Dispose()}
 $report.foreign_keys=@($keys.ToArray());$report.status='ok';$report.stage='done'
 }
}catch{
 if($report.status -eq 'failed'){
  $ex=$_.Exception;while($ex.InnerException -and !($ex -is [Data.SqlClient.SqlException])){$ex=$ex.InnerException}
  if($ex -is [Data.SqlClient.SqlException]){$report.error_code=[int]$ex.Number
   if($ex.Number -eq 18456){$report.status='login_failed'}elseif($ex.Number -eq 4060){$report.status='database_denied'}elseif($ex.Message -match 'SSL|TLS|certificate|인증서'){$report.status='tls_failed'}elseif($report.connected){$report.status='metadata_failed'}
  }elseif($report.connected){$report.status='metadata_failed'}
 }
}finally{if($null -ne $conn){try{$conn.Close();$conn.Dispose()}catch{}}}
ConvertTo-Json -InputObject $report -Depth 7 -Compress
