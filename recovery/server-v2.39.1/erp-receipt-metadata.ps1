 # Read system catalogs only. Never execute discovered modules or return source
 # bodies, default expressions, user rows, connection details or login records.
 $report.stage='receipt_metadata'
 $permission=$conn.CreateCommand()
 $permission.CommandText="SELECT CAST(COALESCE(HAS_PERMS_BY_NAME(DB_NAME(),'DATABASE','VIEW DEFINITION'),0) AS int)"
 $permission.CommandTimeout=8
 try{$report.receipt_view_definition=([int]$permission.ExecuteScalar() -eq 1)}finally{$permission.Dispose()}
 $receiptSql=@'
WITH targets AS (
 SELECT object_id,name FROM sys.objects WHERE schema_id=SCHEMA_ID('dbo')
 AND name IN ('PURORD_MST','PURORD_HIS','PDTWHS_MST','PDTWHS_HIS','INVENT_HIS','INVENT_WRK','TB_WHSINSHIS','PURFSH_HIS','PURFSH_UP')
), modules AS (
 SELECT DISTINCT o.object_id FROM sys.objects o WHERE o.is_ms_shipped=0 AND (
 o.type IN ('P','PC') OR
 (o.type IN ('TR','TA') AND o.parent_object_id IN (SELECT object_id FROM targets)) OR
 (o.type IN ('V','FN','IF','TF') AND EXISTS (
 SELECT 1 FROM sys.sql_expression_dependencies d WHERE d.referencing_id=o.object_id AND d.referenced_id IN (SELECT object_id FROM targets))))
), records AS (
 SELECT 'module' kind,SCHEMA_NAME(o.schema_id) schema_name,o.name object_name,
 COALESCE(OBJECT_NAME(o.parent_object_id),'') parent_name,CAST('' AS nvarchar(128)) member_name,
 CAST(RTRIM(o.type) AS nvarchar(128)) type_name,0 position,
 CAST(CASE WHEN sm.definition IS NOT NULL THEN 1 ELSE 0 END AS bit) flag,
 CAST(CASE WHEN tr.is_disabled=1 THEN 'disabled' ELSE '' END AS nvarchar(256)) detail
 FROM modules m JOIN sys.objects o ON o.object_id=m.object_id
 LEFT JOIN sys.sql_modules sm ON sm.object_id=o.object_id LEFT JOIN sys.triggers tr ON tr.object_id=o.object_id
 UNION ALL
 SELECT 'parameter',SCHEMA_NAME(o.schema_id),o.name,'',p.name,TYPE_NAME(p.user_type_id),p.parameter_id,p.is_output,
 CAST(p.max_length AS nvarchar(20)) FROM modules m JOIN sys.objects o ON o.object_id=m.object_id JOIN sys.parameters p ON p.object_id=o.object_id
 UNION ALL
 SELECT 'dependency',SCHEMA_NAME(o.schema_id),o.name,COALESCE(d.referenced_schema_name,''),COALESCE(d.referenced_entity_name,''),'',d.referenced_minor_id,d.is_caller_dependent,''
 FROM sys.sql_expression_dependencies d JOIN sys.objects o ON o.object_id=d.referencing_id
 WHERE d.referenced_id IN (SELECT object_id FROM targets) OR d.referencing_id IN (SELECT object_id FROM modules)
 UNION ALL
 SELECT 'column',SCHEMA_NAME(o.schema_id),o.name,'',c.name,TYPE_NAME(c.user_type_id),c.column_id,c.is_identity,
 CAST(CASE WHEN c.is_computed=1 THEN 'computed;' ELSE '' END + CASE WHEN c.default_object_id<>0 THEN 'has-default;' ELSE '' END + CASE WHEN c.is_nullable=1 THEN 'nullable;' ELSE 'required;' END AS nvarchar(256))
 FROM targets t JOIN sys.objects o ON o.object_id=t.object_id JOIN sys.columns c ON c.object_id=o.object_id
 UNION ALL
 SELECT 'index',SCHEMA_NAME(o.schema_id),o.name,COALESCE(i.name,''),c.name,'',ic.key_ordinal,i.is_unique,
 CAST(CASE WHEN i.is_primary_key=1 THEN 'primary;' ELSE '' END + CASE WHEN ic.is_included_column=1 THEN 'included;' ELSE '' END AS nvarchar(256))
 FROM targets t JOIN sys.objects o ON o.object_id=t.object_id JOIN sys.indexes i ON i.object_id=o.object_id JOIN sys.index_columns ic ON ic.object_id=i.object_id AND ic.index_id=i.index_id JOIN sys.columns c ON c.object_id=ic.object_id AND c.column_id=ic.column_id
 UNION ALL
 SELECT 'constraint',SCHEMA_NAME(o.schema_id),o.name,OBJECT_NAME(o.parent_object_id),'',RTRIM(o.type),0,CAST(0 AS bit),''
 FROM sys.objects o WHERE o.parent_object_id IN (SELECT object_id FROM targets) AND o.type IN ('D','C','F','PK','UQ')
)
SELECT TOP (5001) kind,schema_name,object_name,parent_name,member_name,type_name,position,flag,detail
FROM records ORDER BY kind,schema_name,object_name,parent_name,position,member_name
'@
 $cmd=$conn.CreateCommand();$cmd.CommandText=$receiptSql;$cmd.CommandTimeout=20
 $rd=$cmd.ExecuteReader();$rows=New-Object 'Collections.Generic.List[object]'
 try{while($rd.Read()){
  if($rows.Count -ge 5000){$report.truncated=$true;break}
  $rows.Add([ordered]@{kind=[string]$rd[0];schema=[string]$rd[1];object=[string]$rd[2];parent=[string]$rd[3];name=[string]$rd[4];type=[string]$rd[5];position=[int]$rd[6];flag=[bool]$rd[7];detail=[string]$rd[8]})
 }}finally{$rd.Close();$cmd.Dispose()}
 $report.receipt_metadata=@($rows.ToArray());$report.status='receipt_metadata';$report.stage='done'
