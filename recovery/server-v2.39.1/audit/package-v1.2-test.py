#!/usr/bin/env python3
from pathlib import Path
import hashlib,json,re,struct,zipfile
root=Path(__file__).resolve().parent.parent
out=root.parent/'server-builds'
exe=Path('/workspace/recovery-input/AlphaWorkRecoveryTest-v1.2.exe')
sha=lambda b:hashlib.sha256(b).hexdigest()
b=exe.read_bytes();pe=struct.unpack_from('<I',b,0x3c)[0];assert struct.unpack_from('<H',b,pe+24+68)[0]==2
old=json.loads((root/'audit/build-manifest-v0.2.json').read_text());assets={n:h for n,h in old['source_files'].items() if n.startswith('web/')};assert len(assets)==57
for n,h in assets.items():assert sha((root/n).read_bytes())==h,n
log=(root/'audit/go-test-race-v1.2-test.txt').read_text();assert '\nPASS\n' in log and '\nFAIL' not in log and 'SKIP' not in log
browser=json.loads((root/'audit/browser-isolation-v1.1.json').read_text());assert browser['originalSimulationFilesUnchanged']=='pass'
copied=json.loads((root/'audit/browser-copy-v1.2.json').read_text());assert copied['originalSourceFilesUnchanged']=='pass'
manifest={'version':'1.2 isolated test with explicit original-data copy','exe_sha256':sha(b),'exe_bytes':len(b),'target':'windows/amd64 GUI CGO_ENABLED=0','toolchain':'go1.27.4','go_test_race_passed':len(re.findall('^--- PASS:',log,re.M)),'go_vet_linux':'pass','go_vet_windows':'pass','original_assets_verified':57,'default_data':'%AppData%\\AlphaWorkRecoveryTest-v1.1','default_listen':'127.0.0.1:28791','automatic_original_import':False,'existing_unmarked_data_folder':'reject before reading or writing','remote_connections_push_backup':'blocked','cookies_mutex_and_native_classes':'separate from production','browser_coexistence':browser,'windows_native_execution':'NOT tested; cross-build only','explicit_copy_option':'-copy-original; source opened read-only; writes restricted to new test copy','copy_browser_workflow':copied,'source_files':{p.name:sha(p.read_bytes()) for p in sorted(root.glob('*.go'))}}
(root/'audit/build-manifest-v1.2-test.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
files=[p for p in sorted(root.rglob('*')) if p.is_file() and p.name!='SHA256SUMS' and '__pycache__' not in p.parts and p.suffix not in {'.exe','.dat','.bak','.bin','.xlsx'}]
(root/'SHA256SUMS').write_text(''.join(f'{sha(p.read_bytes())}  {p.relative_to(root)}\n' for p in files))
source=out/'AlphaWorkRecoveryTest-source-v1.2.zip';windows=out/'AlphaWorkRecoveryTest-Windows-x64-v1.2.zip'
for p in [source,windows]:assert not p.exists(),f'Never overwrite existing ZIP: {p}'
with zipfile.ZipFile(source,'w',zipfile.ZIP_DEFLATED,9) as z:
 for p in files+[root/'SHA256SUMS']:z.write(p,'AlphaWorkRecoveryTest-source-v1.2/'+str(p.relative_to(root)))
with zipfile.ZipFile(windows,'w',zipfile.ZIP_DEFLATED,9) as z:
 z.writestr('AlphaWorkRecoveryTest.exe',b)
 for p in [root/'README.md',root/'audit/RECOVERY-V1.1-TEST.md',root/'audit/build-manifest-v1.2-test.json']:z.write(p,p.name)
 for p in [root/'copy-existing-data.cmd',root/'continue-test.cmd']:z.write(p,p.name)
 z.writestr('SHA256SUMS',sha(b)+'  AlphaWorkRecoveryTest.exe\n')
for p in [source,windows]:
 with zipfile.ZipFile(p) as z:assert z.testzip() is None
 p.with_name(p.name+'.sha256').write_text(sha(p.read_bytes())+'  '+p.name+'\n');print(p.name,p.stat().st_size,sha(p.read_bytes()))
