"""Package the PC-only receipt feature; preserve shared APK and older archives."""
from pathlib import Path
import hashlib,json,struct,zipfile
root=Path(__file__).resolve().parent.parent
out=root.parent/'server-builds';version='1.4.80';stem='AlphaWorkComplete-Recovered'
sha=lambda b:hashlib.sha256(b).hexdigest()
exe=Path('/workspace/recovery-input')/(stem+'-v'+version+'.exe');binary=exe.read_bytes()
pe=struct.unpack_from('<I',binary,0x3c)[0]
assert struct.unpack_from('<H',binary,pe+24+68)[0]==2
assert 'ok  ' in (root/'audit/go-test-final-v1.4.80.txt').read_text()
assert 'FAIL' not in (root/'audit/go-test-final-v1.4.80.txt').read_text()
assert not (root/'audit/go-vet-linux-v1.4.80.txt').read_text().strip()
assert not (root/'audit/go-vet-windows-v1.4.80.txt').read_text().strip()
assert 'PASS' in (root/'audit/purchasing-receipts-race-v1.4.80.txt').read_text()
assert '\nPASS\n' in (root/'audit/purchasing-receipts-browser-v1.4.80.txt').read_text()
browser=json.loads((root/'audit/browser-purchasing-receipts-v1.4.80.json').read_text())
assert len(browser['results'])==4 and all(x['result']=='pass' for x in browser['results'])
assets={n:h for n,h in json.loads((root/'audit/build-manifest-v0.2.json').read_text())['source_files'].items() if n.startswith('web/')}
assert len(assets)==57
for n,h in assets.items():assert sha((root/n).read_bytes())==h,n
apk=out/'AlphaAdmin-Android-v1.0.11.apk';apksha=sha(apk.read_bytes())
assert apksha=='afc14ea37edde0d3e9f78731f1ff60bc9eb75736a8f92aacc890b61ff73162ee'
update=json.loads((root/'android-admin/update.json').read_text())
assert update['sha256']==apksha and update['version_code']==12 and update['version_name']=='1.0.11'
assert not (root/'shipping_export.go').exists(), 'Excel work is deliberately outside this release'
manifest={'pc_version':version,'apk_version':'1.0.11','apk_changed':False,'apk_sha256':apksha,'change':'Device-local purchasing call receipt and completion feedback; same-order other senders stay usable; completed feedback expires after 60 seconds. Not push delivery/read acknowledgement. Excel export deferred.', 'go_test_all':'pass','go_vet_linux':'pass','go_vet_windows':'pass','receipt_race':'pass','browser':browser,'original_assets_verified':57,'windows_gui_exe_sha256':sha(binary),'windows_gui_exe_bytes':len(binary),'windows_execution':'cross-build only; real Windows/Android runtime and live FCM not executed','source_files':{str(p.relative_to(root)):sha(p.read_bytes()) for p in sorted(root.glob('*.go'))}}
(root/'audit/build-manifest-v1.4.80.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
files=[p for p in sorted(root.rglob('*')) if p.is_file() and p.name!='SHA256SUMS' and '__pycache__' not in p.parts and p.name not in {'local.properties','google-services.json','keystore.properties'} and not any(x in p.parts for x in {'.gradle','build'}) and p.suffix not in {'.exe','.dat','.bak','.bin','.xls','.xlsx','.jks','.keystore','.apk'} and 'service-account' not in p.name]
(root/'SHA256SUMS').write_text(''.join(f'{sha(p.read_bytes())}  {p.relative_to(root)}\n' for p in files))
name=f'{stem}-source-PC-v{version}-App-v1.0.11'
source=out/(name+'.zip');windows=out/f'{stem}-Windows-x64-v{version}.zip'
for p in [source,windows]:assert not p.exists(),f'Never replace {p}'
with zipfile.ZipFile(source,'w',zipfile.ZIP_DEFLATED,9) as z:
 for p in files+[root/'SHA256SUMS']:z.write(p,name+'/'+str(p.relative_to(root)))
with zipfile.ZipFile(windows,'w',zipfile.ZIP_DEFLATED,9) as z:
 z.writestr(stem+'.exe',binary)
 for p in [root/'README.md',root/'START-HERE.md',root/'ANDROID-ADMIN-SETUP.md',root/'CHANGELOG.md',root/'PURCHASING-SETUP.md',root/'audit/RECOVERY-V1.4.80.md',root/'audit/build-manifest-v1.4.80.json']:z.write(p,p.name)
 z.write(root/'docs/history/START-HERE-v1.4.74.md','docs/history/START-HERE-v1.4.74.md')
 z.write(root/'android-admin/README.md','android-admin/README.md')
 z.writestr('SHA256SUMS',sha(binary)+'  '+stem+'.exe\n')
for p in [source,windows]:
 with zipfile.ZipFile(p) as z:assert z.testzip() is None
 p.with_name(p.name+'.sha256').write_text(sha(p.read_bytes())+'  '+p.name+'\n')
 print(p.name,p.stat().st_size,sha(p.read_bytes()))
