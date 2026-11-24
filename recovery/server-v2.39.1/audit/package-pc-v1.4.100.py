"""Package unified manual/talk navigation without replacing earlier PC/APK releases."""
from pathlib import Path
import hashlib,json,struct,zipfile
root=Path(__file__).resolve().parent.parent;out=root.parent/'server-builds';version='1.4.100';stem='AlphaWorkComplete-Recovered'
sha=lambda b:hashlib.sha256(b).hexdigest()
binary=(Path('/workspace/recovery-input')/(stem+'-v'+version+'.exe')).read_bytes();pe=struct.unpack_from('<I',binary,0x3c)[0];assert struct.unpack_from('<H',binary,pe+24+68)[0]==2
log=(root/'audit/go-test-final-v1.4.100.txt').read_text();assert 'ok  ' in log and 'FAIL' not in log
for name in ['go-vet-linux-v1.4.100.txt','go-vet-windows-v1.4.100.txt']:assert not (root/'audit'/name).read_text().strip()
assert '\nPASS\n' in (root/'audit/manual-browser-v1.4.100.txt').read_text()
browser=json.loads((root/'audit/browser-manual-board-v1.4.100.json').read_text());assert len(browser['results'])==8 and all(x['result']=='pass' for x in browser['results'])
assert 'const releaseVersion = "'+version+'"' in (root/'production.go').read_text()
assets={n:h for n,h in json.loads((root/'audit/build-manifest-v0.2.json').read_text())['source_files'].items() if n.startswith('web/')};assert len(assets)==57
for n,h in assets.items():assert sha((root/n).read_bytes())==h,n
apk=out/'AlphaAdmin-Android-v1.0.11.apk';apksha=sha(apk.read_bytes());assert apksha=='afc14ea37edde0d3e9f78731f1ff60bc9eb75736a8f92aacc890b61ff73162ee'
update=json.loads((root/'android-admin/update.json').read_text());assert update['sha256']==apksha and update['version_code']==12 and update['version_name']=='1.0.11'
manifest={'pc_version':version,'apk_version':'1.0.11','apk_changed':False,'apk_sha256':apksha,'change':'Unified manual board talk shell: shared five category tabs and active manual state on list/compose/detail; aligned toolbar/width; per-role links and app/worker talk selection; existing uploads/data unchanged.','go_test_all':'pass','go_vet_linux':'pass','go_vet_windows':'pass','manual_talk_browser':browser,'original_assets_verified':57,'windows_gui_exe_sha256':sha(binary),'windows_gui_exe_bytes':len(binary),'windows_execution':'cross-build only; actual company data modification/Windows/Android/Microsoft Excel not executed','source_files':{str(p.relative_to(root)):sha(p.read_bytes()) for p in sorted(root.glob('*.go'))}}
(root/'audit/build-manifest-v1.4.100.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
files=[p for p in sorted(root.rglob('*')) if p.is_file() and p.name!='SHA256SUMS' and '__pycache__' not in p.parts and p.name not in {'local.properties','google-services.json','keystore.properties'} and not any(x in p.parts for x in {'.gradle','build'}) and p.suffix not in {'.exe','.dat','.bak','.bin','.xls','.xlsx','.jks','.keystore','.apk'} and 'service-account' not in p.name and not (p.parent==root/'audit' and p.suffix=='.png' and any('v1.4.'+str(v) in p.name for v in range(90,97)))]
(root/'SHA256SUMS').write_text(''.join(f'{sha(p.read_bytes())}  {p.relative_to(root)}\n' for p in files))
name=f'{stem}-source-PC-v{version}-App-v1.0.11';source=out/(name+'.zip');windows=out/f'{stem}-Windows-x64-v{version}.zip'
for p in [source,windows]:assert not p.exists(),f'Never replace {p}'
with zipfile.ZipFile(source,'w',zipfile.ZIP_DEFLATED,9) as z:
 for p in files+[root/'SHA256SUMS']:z.write(p,name+'/'+str(p.relative_to(root)))
with zipfile.ZipFile(windows,'w',zipfile.ZIP_DEFLATED,9) as z:
 z.writestr(stem+'.exe',binary)
 for p in [root/'README.md',root/'START-HERE.md',root/'ANDROID-ADMIN-SETUP.md',root/'CHANGELOG.md',root/'PURCHASING-SETUP.md',root/'SHIPMENT-ATTACHMENTS.md',root/'SHIPMENT-EXCEL.md',root/'PC-AUTO-UPDATE.md',root/'BACKUP-SETS.md',root/'INTERNET-AUTO.md',root/'INSPECTION-INSPECTORS.md',root/'audit/RECOVERY-V1.4.100.md',root/'audit/build-manifest-v1.4.100.json']:z.write(p,p.name)
 z.write(root/'docs/history/START-HERE-v1.4.74.md','docs/history/START-HERE-v1.4.74.md');z.write(root/'android-admin/README.md','android-admin/README.md')
 z.write(root/'docs/licenses/fix-webm-duration-MIT.txt','licenses/fix-webm-duration-MIT.txt');z.write(root/'docs/licenses/fix-webm-duration-source.json','licenses/fix-webm-duration-source.json')
 z.writestr('SHA256SUMS',sha(binary)+'  '+stem+'.exe\n')
for p in [source,windows]:
 with zipfile.ZipFile(p) as z:assert z.testzip() is None
 p.with_name(p.name+'.sha256').write_text(sha(p.read_bytes())+'  '+p.name+'\n');print(p.name,p.stat().st_size,sha(p.read_bytes()))

# Always publish this stable endpoint with every subsequent PC release.
current={'version':version,'url':'https://raw.githubusercontent.com/swsgood79-beep/Tool/recovery-v2.39.1/recovery/server-builds/'+windows.name,'sha256':sha(windows.read_bytes()),'exe_sha256':sha(binary),'bytes':windows.stat().st_size,'notes':'매뉴얼도 토크 메뉴 유지 · 제목·검색·목록 배치 통일 · APK 1.0.11 유지'}
(out/'pc-update.json').write_text(json.dumps(current,ensure_ascii=False,indent=2)+'\n')
print('Latest PC update metadata:',json.dumps(current,ensure_ascii=False))
