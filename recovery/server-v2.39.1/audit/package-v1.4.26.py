"""Package verified release without replacing earlier archives."""
from pathlib import Path
import hashlib,json,re,struct,zipfile
root=Path(__file__).resolve().parent.parent
out=root.parent/'server-builds';version='1.4.26';stem='AlphaWorkComplete-Recovered'
exe=Path('/workspace/recovery-input')/f'{stem}-v{version}.exe'
sha=lambda b:hashlib.sha256(b).hexdigest()
b=exe.read_bytes();pe=struct.unpack_from('<I',b,0x3c)[0];assert struct.unpack_from('<H',b,pe+24+68)[0]==2
assets={n:h for n,h in json.loads((root/'audit/build-manifest-v0.2.json').read_text())['source_files'].items() if n.startswith('web/')};assert len(assets)==57
for n,h in assets.items():assert sha((root/n).read_bytes())==h,n
log=(root/f'audit/go-test-race-v{version}.txt').read_text();assert '\nPASS\n' in log and '\nFAIL' not in log
browser=json.loads((root/f'audit/browser-market-calendar-v{version}.json').read_text());assert len(browser['results'])==6 and all(all(v=='pass' for k,v in x.items() if k not in {'width','scheme','pageErrors'}) and not x['pageErrors'] for x in browser['results'])
manifest={'version':version,'change':'Restore navy market colors and visible title; add calendar grid and distinct order, selected, today and empty day colors','exe_sha256':sha(b),'exe_bytes':len(b),'target':'windows/amd64 GUI CGO_ENABLED=0','original_assets_verified':57,'go_test_race_passed':len(re.findall('^--- PASS:',log,re.M)),'optional_tests_skipped':re.findall('^--- SKIP: (.+)$',log,re.M),'go_vet_linux':'pass','go_vet_windows':'pass','browser':browser,'windows_execution':'not tested; cross-build only','operating_data':'existing %AppData%/AlphaWorkCompleteRecovered; schema unchanged; design preview is not isolated business data','preview_preference':'browser-local alpha-design-default-v2; default modern; optional legacy fallback; theme follows device with manual override','native_window':'existing Win32 layout preserved; this release promotes the modern web UI','source_files':{str(p.relative_to(root)):sha(p.read_bytes()) for p in sorted(root.glob('*.go'))},'extra_assets':{str(p.relative_to(root)):sha(p.read_bytes()) for p in sorted((root/'extras').glob('*'))}}
(root/f'audit/build-manifest-v{version}.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
files=[p for p in sorted(root.rglob('*')) if p.is_file() and p.name!='SHA256SUMS' and '__pycache__' not in p.parts and p.suffix not in {'.exe','.dat','.bak','.bin','.xlsx'}]
(root/'SHA256SUMS').write_text(''.join(f'{sha(p.read_bytes())}  {p.relative_to(root)}\n' for p in files))
source=out/f'{stem}-source-v{version}.zip';windows=out/f'{stem}-Windows-x64-v{version}.zip'
for p in [source,windows]:assert not p.exists(),f'Never overwrite {p}'
with zipfile.ZipFile(source,'w',zipfile.ZIP_DEFLATED,9) as z:
 for p in files+[root/'SHA256SUMS']:z.write(p,f'{stem}-source-v{version}/'+str(p.relative_to(root)))
with zipfile.ZipFile(windows,'w',zipfile.ZIP_DEFLATED,9) as z:
 z.writestr(stem+'.exe',b)
 for p in [root/'README.md',root/'START-HERE.md',root/f'audit/RECOVERY-V{version}.md',root/f'audit/build-manifest-v{version}.json']:z.write(p,p.name)
 z.writestr('SHA256SUMS',sha(b)+'  '+stem+'.exe\n')
for p in [source,windows]:
 with zipfile.ZipFile(p) as z:assert z.testzip() is None
 p.with_name(p.name+'.sha256').write_text(sha(p.read_bytes())+'  '+p.name+'\n');print(p.name,p.stat().st_size,sha(p.read_bytes()))
