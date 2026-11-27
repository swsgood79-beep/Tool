 $report.stage='observation'
 $ob=[ordered]@{events=@();truncated=$false;dropped=0;cleanup=$false;ready=$false;reason='preparing'}
 $report.observation=$ob;$created=$false
 function Observer-Command([string]$sql){$c=$conn.CreateCommand();$c.CommandText=$sql;$c.CommandTimeout=8;try{[void]$c.ExecuteNonQuery()}finally{$c.Dispose()}}
 try{
  if($ObserveSession -notmatch '^AlphaReceipt_[a-f0-9]{24}$'){throw 'invalid observer name'}
  $c=$conn.CreateCommand();$c.CommandTimeout=8
  $c.CommandText="SELECT DB_ID(),COALESCE(HAS_PERMS_BY_NAME(NULL,NULL,'ALTER ANY EVENT SESSION'),0),COALESCE(HAS_PERMS_BY_NAME(NULL,NULL,'VIEW SERVER STATE'),0),COALESCE(HAS_PERMS_BY_NAME(NULL,NULL,'CREATE ANY EVENT SESSION'),0),COALESCE(HAS_PERMS_BY_NAME(NULL,NULL,'ALTER ANY EVENT SESSION ENABLE'),0),COALESCE(HAS_PERMS_BY_NAME(NULL,NULL,'DROP ANY EVENT SESSION'),0),COALESCE(HAS_PERMS_BY_NAME(NULL,NULL,'VIEW SERVER PERFORMANCE STATE'),0)"
  $rd=$c.ExecuteReader()
  try{[void]$rd.Read();$db=[int]$rd[0];$canManage=([int]$rd[1] -eq 1) -or (([int]$rd[3] -eq 1) -and ([int]$rd[4] -eq 1) -and ([int]$rd[5] -eq 1));$canRead=([int]$rd[2] -eq 1) -or ([int]$rd[6] -eq 1)}finally{$rd.Close();$c.Dispose()}
  if(!$canManage -or !$canRead){$ob.reason='trace_permission_denied';throw 'stop'}
  if($ObserveRecover){Observer-Command "IF EXISTS(SELECT 1 FROM sys.server_event_sessions WHERE name=N'$ObserveSession') DROP EVENT SESSION [$ObserveSession] ON SERVER";$ob.cleanup=$true;$ob.reason='recovered';throw 'recovery_done'}
  $erp=@(Get-Process -Name PrimeXAR -ErrorAction SilentlyContinue)
  if($erp.Count -ne 1){$ob.reason='erp_process_missing_or_ambiguous';throw 'stop'}
  $clientPid=[int]$erp[0].Id;$hostName=[Environment]::MachineName.Replace("'","''")
  $filter="([sqlserver].[database_id]=($db) AND [sqlserver].[client_pid]=($clientPid) AND [sqlserver].[client_hostname]=N'$hostName')"
  $create=@"
CREATE EVENT SESSION [$ObserveSession] ON SERVER
ADD EVENT sqlserver.rpc_completed(ACTION(sqlserver.session_id,sqlserver.sql_text) WHERE $filter),
ADD EVENT sqlserver.sql_batch_completed(ACTION(sqlserver.session_id,sqlserver.sql_text) WHERE $filter),
ADD EVENT sqlserver.sp_statement_completed(ACTION(sqlserver.session_id) WHERE $filter)
ADD TARGET package0.ring_buffer(SET max_memory=2048)
WITH(MAX_MEMORY=2048 KB,EVENT_RETENTION_MODE=ALLOW_SINGLE_EVENT_LOSS,MAX_DISPATCH_LATENCY=1 SECONDS,TRACK_CAUSALITY=ON,STARTUP_STATE=OFF)
"@
  Observer-Command $create;$created=$true
  Observer-Command "ALTER EVENT SESSION [$ObserveSession] ON SERVER STATE=START"
  [IO.File]::WriteAllText((Join-Path $ObserveDirectory 'phase'),'ready');$ob.ready=$true;$ob.reason='recording'
  $end=[DateTime]::UtcNow.AddSeconds(90)
  while([DateTime]::UtcNow -lt $end -and ![IO.File]::Exists((Join-Path $ObserveDirectory 'stop'))){Start-Sleep -Milliseconds 200}
  # Flush before snapshot; the snapshot is a boundary, not proof of commit.
  Start-Sleep -Milliseconds 1500
  $c=$conn.CreateCommand();$c.CommandTimeout=8
  $c.CommandText="SELECT CAST(t.target_data AS nvarchar(max)),s.dropped_event_count FROM sys.dm_xe_sessions s JOIN sys.dm_xe_session_targets t ON t.event_session_address=s.address WHERE s.name=@Name AND t.target_name='ring_buffer'"
  [void]$c.Parameters.Add('@Name',[Data.SqlDbType]::NVarChar,128);$c.Parameters['@Name'].Value=$ObserveSession
  $rd=$c.ExecuteReader()
  try{if(!$rd.Read()){$ob.reason='target_unavailable';throw 'stop'};$xmlText=[string]$rd[0];$ob.dropped=[long]$rd[1];if($xmlText.Length -gt 3000000){$ob.truncated=$true;$ob.reason='target_output_limit';throw 'stop'}}finally{$rd.Close();$c.Dispose()}
  [xml]$xml=$xmlText;if($xml.RingBufferTarget.truncated -eq '1'){$ob.truncated=$true}
  $rows=New-Object 'Collections.Generic.List[object]'
  foreach($event in $xml.RingBufferTarget.event){
   if($rows.Count -ge 2000){$ob.truncated=$true;break}
   $n=$event.SelectSingleNode("data[@name='statement']/value");if(!$n){$n=$event.SelectSingleNode("data[@name='batch_text']/value")};if(!$n){$n=$event.SelectSingleNode("action[@name='sql_text']/value")}
   $statement=if($n){[string]$n.InnerText}else{''}
   if($statement -match '(?i)password|pwd\s*=|USER_PW|MAIL_PW|USERSEC_INFO|CREATE\s+LOGIN|ALTER\s+LOGIN|sp_password'){$statement='[authentication-related text omitted]'}
   if($statement.Length -gt 16000){$statement=$statement.Substring(0,16000);$ob.truncated=$true}
   $sid=$event.SelectSingleNode("action[@name='session_id']/value");$dur=$event.SelectSingleNode("data[@name='duration']/value")
   $rows.Add([ordered]@{at=[string]$event.timestamp;name=[string]$event.name;session=$(if($sid){[int]$sid.InnerText}else{0});text=$statement;duration=$(if($dur){[long]$dur.InnerText}else{0})})
  }
  $ob.events=@($rows.ToArray());$ob.reason='captured'
 }catch{
  if($ob.reason -eq 'preparing' -or $ob.reason -eq 'recording'){$ob.reason='observer_failed'}
  $ex=$_.Exception;while($ex.InnerException -and !($ex -is [Data.SqlClient.SqlException])){$ex=$ex.InnerException};if($ex -is [Data.SqlClient.SqlException]){$report.error_code=[int]$ex.Number}
 }finally{
  if($created){try{Observer-Command "DROP EVENT SESSION [$ObserveSession] ON SERVER";$ob.cleanup=$true}catch{$ob.cleanup=$false;$ob.reason='cleanup_failed'}}elseif(!$ObserveRecover){$ob.cleanup=$true}
  $report.status='observation';$report.stage='done'
 }
