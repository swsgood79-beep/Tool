 # Permission discovery only. No trace sessions are created and no grants applied.
 $report.stage='observation_access'
 $sql=@'
SELECT
 CAST(COALESCE(HAS_PERMS_BY_NAME(NULL,NULL,'VIEW SERVER STATE'),0) AS int),
 CAST(COALESCE(HAS_PERMS_BY_NAME(NULL,NULL,'VIEW SERVER PERFORMANCE STATE'),0) AS int),
 CAST(COALESCE(HAS_PERMS_BY_NAME(NULL,NULL,'ALTER ANY EVENT SESSION'),0) AS int),
 CAST(COALESCE(HAS_PERMS_BY_NAME(NULL,NULL,'CREATE ANY EVENT SESSION'),0) AS int),
 CAST(COALESCE(HAS_PERMS_BY_NAME(NULL,NULL,'ALTER ANY EVENT SESSION ENABLE'),0) AS int),
 CAST(COALESCE(HAS_PERMS_BY_NAME(NULL,NULL,'DROP ANY EVENT SESSION'),0) AS int),
 CAST(CASE WHEN OBJECT_ID('sys.dm_xe_sessions') IS NOT NULL THEN 1 ELSE 0 END AS int)
'@
 $cmd=$conn.CreateCommand();$cmd.CommandText=$sql;$cmd.CommandTimeout=8
 $rd=$cmd.ExecuteReader()
 try{
  if(!$rd.Read()){throw 'missing permission result'}
  $report.observation_access=[ordered]@{
   view_server_state=([int]$rd[0] -eq 1)
   view_server_performance_state=([int]$rd[1] -eq 1)
   alter_any_event_session=([int]$rd[2] -eq 1)
   create_any_event_session=([int]$rd[3] -eq 1)
   alter_any_event_session_enable=([int]$rd[4] -eq 1)
   drop_any_event_session=([int]$rd[5] -eq 1)
   extended_events_available=([int]$rd[6] -eq 1)
  }
 }finally{$rd.Close();$cmd.Dispose()}
 $report.status='observation_access';$report.stage='done'
