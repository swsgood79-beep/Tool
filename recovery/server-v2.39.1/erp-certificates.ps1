 $report.stage='certificates'
 $certificateSql=@'
SELECT TOP (5001) m.UNT_CD,m.RCV_ORDNO,
 COALESCE(cust.CST_NM,''),COALESCE(NULLIF(c.CLIENT,''),ord.CLIENT,''),
 COALESCE(NULLIF(c.PROJECT,''),ord.PROJECT,''),COALESCE(NULLIF(h.ITM_CD,''),m.ITM_CD,''),
 COALESCE(NULLIF(c.SERIAL_NO,''),h.SERIAL_NO,''),COALESCE(NULLIF(c.REMARK,''),h.REMARK,''),
 h.PDTCMD_DATE,h.PDTCMD_NO,h.PDTCMD_SEQ,COALESCE(c.INS_DATE,''),COALESCE(c.JUDGMENT,''),
 CAST(CASE WHEN c.UNT_CD IS NULL THEN 0 ELSE 1 END AS bit),COALESCE(h.CERSHT_NO,'')
FROM dbo.PDTCMD_MST m
JOIN dbo.PDTCMD_HIS h ON h.UNT_CD=m.UNT_CD AND h.PDTCMD_DATE=m.PDTCMD_DATE AND h.PDTCMD_NO=m.PDTCMD_NO
LEFT JOIN dbo.INSCER_MST c ON c.UNT_CD=h.UNT_CD AND c.PDTCMD_DATE=h.PDTCMD_DATE AND c.PDTCMD_NO=h.PDTCMD_NO AND c.PDTCMD_SEQ=h.PDTCMD_SEQ
LEFT JOIN dbo.RCVORD_MST ord ON ord.UNT_CD=m.UNT_CD AND ord.RCVORD_DATE=m.RCV_ORDDATE AND ord.RCVORD_NO=m.RCV_ORDNO
LEFT JOIN dbo.CSTMST_INFO cust ON cust.CST_CD=COALESCE(NULLIF(c.CST_CD,''),ord.CST_CD)
WHERE m.RCV_ORDNO=@Order
ORDER BY m.UNT_CD,h.PDTCMD_DATE,h.PDTCMD_NO,h.PDTCMD_SEQ
'@
 $cmd=$conn.CreateCommand();$cmd.CommandText=$certificateSql;$cmd.CommandTimeout=25
 [void]$cmd.Parameters.Add('@Order',[Data.SqlDbType]::VarChar,100);$cmd.Parameters['@Order'].Value=$CertificateOrder
 $rd=$cmd.ExecuteReader();$rows=New-Object 'Collections.Generic.List[object]'
 try{while($rd.Read()){
  if($rows.Count -ge 5000){$report.truncated=$true;break}
  $rows.Add([ordered]@{unit=[string]$rd[0];order=[string]$rd[1];customer=[string]$rd[2];client=[string]$rd[3];project=[string]$rd[4];item=[string]$rd[5];serial=[string]$rd[6];remark=[string]$rd[7];command_date=[string]$rd[8];command_no=[string]$rd[9];sequence=[int]$rd[10];inspection_date=[string]$rd[11];judgment=[string]$rd[12];registered=[bool]$rd[13];certificate=[string]$rd[14]})
 }}finally{$rd.Close();$cmd.Dispose()}
 $report.certificates=@($rows.ToArray());$report.status='certificates';$report.stage='done'
