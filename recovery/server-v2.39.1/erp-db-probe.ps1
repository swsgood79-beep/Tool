$ErrorActionPreference='Stop'
$ProgressPreference='SilentlyContinue'
[Console]::OutputEncoding=New-Object Text.UTF8Encoding($false)
$report=[ordered]@{status='failed';stage='config_path';config_reason='';connected=$false;error_code=0;columns=@();foreign_keys=@();truncated=$false}
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
}catch{
 if($report.status -eq 'failed'){
  $ex=$_.Exception;while($ex.InnerException -and !($ex -is [Data.SqlClient.SqlException])){$ex=$ex.InnerException}
  if($ex -is [Data.SqlClient.SqlException]){$report.error_code=[int]$ex.Number
   if($ex.Number -eq 18456){$report.status='login_failed'}elseif($ex.Number -eq 4060){$report.status='database_denied'}elseif($ex.Message -match 'SSL|TLS|certificate|인증서'){$report.status='tls_failed'}elseif($report.connected){$report.status='metadata_failed'}
  }elseif($report.connected){$report.status='metadata_failed'}
 }
}finally{if($null -ne $conn){try{$conn.Close();$conn.Dispose()}catch{}}}
ConvertTo-Json -InputObject $report -Depth 7 -Compress
