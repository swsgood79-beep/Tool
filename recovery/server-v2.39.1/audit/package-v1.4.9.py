#!/usr/bin/env python3
from pathlib import Path
import hashlib,json,re,struct,zipfile
root=Path(__file__).resolve().parent.parent
out=root.parent/'server-builds'
exe=Path('/workspace/recovery-input/AlphaWorkComplete-Recovered-v1.4.9.exe')
sha=lambda b:hashlib.sha256(b).hexdigest()
b=exe.read_bytes();pe=struct.unpack_from('<I',b,0x3c)[0];assert struct.unpack_from('<H',b,pe+24+68)[0]==2
old=json.loads((root/'audit/build-manifest-v0.2.json').read_text());assets={n:h for n,h in old['source_files'].items() if n.startswith('web/')};assert len(assets)==57
for n,h in assets.items():assert sha((root/n).read_bytes())==h,n
log=(root/'audit/go-test-race-v1.4.9.txt').read_text();assert '\nPASS\n' in log and '\nFAIL' not in log and 'SKIP' not in log
browser=json.loads((root/'audit/browser-isolation-v1.1.json').read_text());assert browser['originalSimulationFilesUnchanged']=='pass'
copied=json.loads((root/'audit/browser-copy-v1.2.json').read_text());assert copied['originalSourceFilesUnchanged']=='pass'
added=json.loads((root/'audit/browser-missing-features-v1.3.json').read_text());assert added['exportedManualMemoFormulaAndExtraSheetPreserved']=='pass'
matching=json.loads((root/'audit/browser-workbook-match-v1.3.1.json').read_text());assert matching['noDuplicateRowsAndOriginalMemoPreserved']=='pass'
calendar=json.loads((root/'audit/browser-calendar-designer-v1.4.2.json').read_text());assert len(calendar['results'])==4
filters=json.loads((root/'audit/browser-order-filters-v1.4.5.json').read_text());assert len(filters['results'])==2
directory=json.loads((root/'audit/browser-worker-directory-v1.4.7.json').read_text());assert len(directory['results'])==2
market=json.loads((root/'audit/browser-market-status-v1.4.8.json').read_text());assert market['frontendStatusJSONPath']=='pass'
supplied=json.loads((root/'audit/browser-supplied-materials-v1.4.9.json').read_text());assert len(supplied['results'])==4
production=json.loads((root/'audit/browser-production-v1.4.json').read_text());assert production['originalSourceUnchanged']=='pass'
manifest={'version':'1.4.9 supplied materials from schedule column N','exe_sha256':sha(b),'exe_bytes':len(b),'target':'windows/amd64 GUI CGO_ENABLED=0','toolchain':'go1.27.4','go_test_race_passed':len(re.findall('^--- PASS:',log,re.M)),'go_vet_linux':'pass','go_vet_windows':'pass','original_assets_verified':57,'default_data':'%AppData%\\AlphaWorkCompleteRecovered','default_listen':'127.0.0.1:18791','automatic_original_import':'first production launch only, read-only snapshot','existing_unmarked_data_folder':'reject before reading or writing','remote_connections_push_backup':'enabled in production according to settings; blocked in -test','cookies_mutex_and_native_classes':'production shares original mutex; -test identities separate','browser_coexistence':browser,'windows_native_execution':'NOT tested; cross-build only','explicit_copy_option':'-test -copy-original; production has separate root and one-time automatic copy','copy_browser_workflow':copied,'added_feature_browser':added,'extra_assets':{str(p.relative_to(root)):sha(p.read_bytes()) for p in sorted((root/'extras').glob('*'))},'workbook_match_browser':matching,'actual_report_file':'provided and read-only parsed; excluded from all artifacts; company PC database not inspected','supplied_materials_browser':supplied,'market_status_browser_v1_4_8_historical':market,'live_onedrive_download':'NOT verified; development proxy blocked connection','worker_directory_browser_v1_4_7_historical':directory,'order_filters_browser_v1_4_5_historical':filters,'calendar_browser_v1_4_2_historical':calendar,'production_browser_v1_4_historical':production,'first_start_validation':'unit tests; Windows dialog not executed here','test_mode':'explicit -test; separate data, port, cookies, classes, mutex and no external network','source_files':{p.name:sha(p.read_bytes()) for p in sorted(root.glob('*.go'))}}
(root/'audit/build-manifest-v1.4.9.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
files=[p for p in sorted(root.rglob('*')) if p.is_file() and p.name!='SHA256SUMS' and '__pycache__' not in p.parts and p.suffix not in {'.exe','.dat','.bak','.bin','.xlsx'}]
(root/'SHA256SUMS').write_text(''.join(f'{sha(p.read_bytes())}  {p.relative_to(root)}\n' for p in files))
source=out/'AlphaWorkComplete-Recovered-source-v1.4.9.zip';windows=out/'AlphaWorkComplete-Recovered-Windows-x64-v1.4.9.zip'
for p in [source,windows]:assert not p.exists(),f'Never overwrite existing ZIP: {p}'
with zipfile.ZipFile(source,'w',zipfile.ZIP_DEFLATED,9) as z:
 for p in files+[root/'SHA256SUMS']:z.write(p,'AlphaWorkComplete-Recovered-source-v1.4.9/'+str(p.relative_to(root)))
with zipfile.ZipFile(windows,'w',zipfile.ZIP_DEFLATED,9) as z:
 z.writestr('AlphaWorkComplete-Recovered.exe',b)
 for p in [root/'README.md',root/'audit/RECOVERY-V1.4.1.md',root/'audit/build-manifest-v1.4.9.json']:z.write(p,p.name)
 z.write(root/'START-HERE.md','START-HERE.md')
 z.writestr('SHA256SUMS',sha(b)+'  AlphaWorkComplete-Recovered.exe\n')
for p in [source,windows]:
 with zipfile.ZipFile(p) as z:assert z.testzip() is None
 p.with_name(p.name+'.sha256').write_text(sha(p.read_bytes())+'  '+p.name+'\n');print(p.name,p.stat().st_size,sha(p.read_bytes()))
