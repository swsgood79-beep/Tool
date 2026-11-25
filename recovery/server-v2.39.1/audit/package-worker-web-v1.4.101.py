"""Publish current full sources for a web-only update; preserve PC100/APK11 releases."""
from pathlib import Path
import hashlib,json,zipfile
root=Path(__file__).resolve().parent.parent;out=root.parent/'server-builds';sha=lambda b:hashlib.sha256(b).hexdigest()
log=(root/'audit/worker-install-tests-v1.4.101.txt').read_text();assert 'Worker installation checks passed: 10' in log
browser=json.loads((root/'audit/browser-worker-install-v1.4.101.json').read_text());assert len(browser['results'])==10 and all(x['result']=='pass' for x in browser['results'])
assets={n:h for n,h in json.loads((root/'audit/build-manifest-v0.2.json').read_text())['source_files'].items() if n.startswith('web/')};assert len(assets)==57
for n,h in assets.items():assert sha((root/n).read_bytes())==h,n
assert 'const releaseVersion = "1.4.100"' in (root/'production.go').read_text()
manifest=json.loads((out/'pc-update.json').read_text());assert manifest['version']=='1.4.100'
windows=out/'AlphaWorkComplete-Recovered-Windows-x64-v1.4.100.zip';assert sha(windows.read_bytes())==manifest['sha256']
apk=out/'AlphaAdmin-Android-v1.0.11.apk';assert sha(apk.read_bytes())=='afc14ea37edde0d3e9f78731f1ff60bc9eb75736a8f92aacc890b61ff73162ee'
canonical=root/'worker-pwa-v1.4.101'
for p in canonical.iterdir():
 for target in [root.parents[1]/'pages/app',Path('/workspace/Tool-pages-read-v47/app')]:assert sha(p.read_bytes())==sha((target/p.name).read_bytes())
proof={'web_version':'1.4.101','pc_version':'1.4.100','pc_changed':False,'apk_version':'1.0.11','apk_changed':False,'pc_zip_sha256':manifest['sha256'],'pc_exe_sha256':manifest['exe_sha256'],'apk_sha256':sha(apk.read_bytes()),'browser_cases':10,'original_web_assets_verified':57,'runtime_sources_unchanged':True,'static_app_files':{p.name:sha(p.read_bytes()) for p in sorted(canonical.iterdir())},'scope':'Local Chromium tests; Android install prompt/intent execution and company data not exercised.'}
(root/'audit/build-worker-web-v1.4.101.json').write_text(json.dumps(proof,ensure_ascii=False,indent=2)+'\n')
files=[p for p in sorted(root.rglob('*')) if p.is_file() and p.name!='SHA256SUMS' and '__pycache__' not in p.parts and p.name not in {'local.properties','google-services.json','keystore.properties'} and not any(x in p.parts for x in {'.gradle','build'}) and p.suffix not in {'.exe','.dat','.bak','.bin','.xls','.xlsx','.jks','.keystore','.apk'} and 'service-account' not in p.name and not (p.parent==root/'audit' and p.suffix=='.png' and any('v1.4.'+str(v) in p.name for v in range(90,97)))]
(root/'SHA256SUMS').write_text(''.join(f'{sha(p.read_bytes())}  {p.relative_to(root)}\n' for p in files))
name='AlphaWorkComplete-Recovered-source-PC-v1.4.100-App-v1.0.11-Web-v1.4.101';archive=out/(name+'.zip');assert not archive.exists()
with zipfile.ZipFile(archive,'w',zipfile.ZIP_DEFLATED,9) as z:
 for p in files+[root/'SHA256SUMS']:z.write(p,name+'/'+str(p.relative_to(root)))
assert archive.stat().st_size<100*1024*1024
with zipfile.ZipFile(archive) as z:assert z.testzip() is None
archive.with_name(archive.name+'.sha256').write_text(sha(archive.read_bytes())+'  '+archive.name+'\n')
print(archive.name,archive.stat().st_size,sha(archive.read_bytes()))
