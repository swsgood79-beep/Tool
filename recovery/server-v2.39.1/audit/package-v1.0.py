#!/usr/bin/env python3
"""Package already-built, tested recovery v1.0 without company runtime data."""
from pathlib import Path
import hashlib,json,re,struct,zipfile,subprocess
root=Path(__file__).resolve().parent.parent
exe=root.parent.parent.parent/'recovery-input/AlphaWorkRecoveredServer-v1.0.exe'
# /workspace/Tool/recovery/server-v2.39.1 -> execution input is outside repository
exe=Path('/workspace/recovery-input/AlphaWorkRecoveredServer-v1.0.exe')
out=root.parent/'server-builds';out.mkdir(exist_ok=True)
def sha(b):return hashlib.sha256(b).hexdigest()
binary=exe.read_bytes();pe=struct.unpack_from('<I',binary,0x3c)[0];subsystem=struct.unpack_from('<H',binary,pe+24+68)[0]
assert binary[:2]==b'MZ' and subsystem==2,'Must be a Windows GUI PE'
old=json.loads((root/'audit/build-manifest-v0.2.json').read_text())
web={n:h for n,h in old['source_files'].items() if n.startswith('web/')}
assert len(web)==57
for n,h in web.items():assert sha((root/n).read_bytes())==h,n+' original web asset changed'
tests=(root/'audit/go-test-race-v1.0.txt').read_text();assert '\nPASS\n' in tests and '\nFAIL' not in tests
passed=len(re.findall(r'^--- PASS:',tests,re.M));assert 'SKIP' not in tests
files={str(p.relative_to(root)):sha(p.read_bytes()) for p in sorted(root.glob('*.go'))}
for n in ['go.mod','go.sum','build-windows.ps1']:files[n]=sha((root/n).read_bytes())
manifest={'version':'1.0 direct-write / native Windows UI reconstruction','original_equivalence_100_percent':'NOT certified; independent implementation; remaining differences in README','toolchain':'go1.27.4 linux/amd64','target':'windows/amd64 CGO_ENABLED=0','command':"go build -trimpath -buildvcs=false -ldflags='-s -w -H=windowsgui' -o AlphaWorkRecoveredServer.exe .",'pe_subsystem':subsystem,'exe_sha256':sha(binary),'exe_bytes':len(binary),'source_files':files,'original_web_assets_verified':57,'go_test_race_passed':passed,'go_vet_linux':'pass','go_vet_windows':'pass','browser_workflow':'pass','browser_settings':'pass','provided_schedule_parsed_orders':52,'provided_schedule_conflicts':0,'windows_native_execution':'NOT tested; cross-build, ABI and static checks only','actual_company_data_and_backup':'NOT tested; schema fixtures only','phone_hardware_and_delivery':'NOT tested','external_live_accounts':'NOT tested; mock transports used','production_pointer':'NOT modified','original_direct_files':['projects-v2.dat','connection-v1.dat','phone-push-v1.dat','designers-v1.dat','client-notes-v1.dat','backup-v1.dat'],'initial_backup':'one-time .bak, never replace existing backup','transaction_journal':'prepared -> rollback; committed -> retain','recovery_state':'_RecoveredV1 inside projects-v2.dat; no new shadow state file'}
(root/'audit/build-manifest-v1.0.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
excluded={'SHA256SUMS'}
paths=[p for p in sorted(root.rglob('*')) if p.is_file() and p.name not in excluded and '__pycache__' not in p.parts and p.suffix not in {'.exe','.dat','.bak','.bin','.xlsx'}]
# All packaged files are source, recovered assets or explicit audit artifacts.
(root/'SHA256SUMS').write_text(''.join(f'{sha(p.read_bytes())}  {p.relative_to(root)}\n' for p in paths))
source=out/'AlphaWorkRecoveredServer-source-v1.0.zip'
windows=out/'AlphaWorkRecoveredServer-Windows-x64-v1.0.zip'
for dest in [source,windows]:assert not dest.exists(),f'Preserve existing artifact: {dest}'
with zipfile.ZipFile(source,'w',zipfile.ZIP_DEFLATED,9) as z:
 for p in paths+[root/'SHA256SUMS']:z.write(p,'AlphaWorkRecoveredServer-source-v1.0/'+str(p.relative_to(root)))
with zipfile.ZipFile(windows,'w',zipfile.ZIP_DEFLATED,9) as z:
 z.writestr('AlphaWorkRecoveredServer.exe',binary)
 z.write(root/'README.md','README.md')
 z.write(root/'audit/build-manifest-v1.0.json','build-manifest-v1.0.json')
 z.write(root/'audit/RECOVERY-V1.0.md','RECOVERY-V1.0.md')
 z.writestr('SOURCE-URL.txt','https://github.com/swsgood79-beep/Tool/tree/recovery-v2.39.1/recovery/server-v2.39.1\n')
 z.writestr('SHA256SUMS',sha(binary)+'  AlphaWorkRecoveredServer.exe\n')
for p in [source,windows]:
 with zipfile.ZipFile(p) as z:assert z.testzip() is None
 p.with_name(p.name+'.sha256').write_text(sha(p.read_bytes())+'  '+p.name+'\n')
 print(p.name,p.stat().st_size,sha(p.read_bytes()))
print('Tests:',passed,'original assets:',len(web),'Windows GUI subsystem:',subsystem)
