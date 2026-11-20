"""Package current PC+APK sources without creating or renumbering a PC release."""
from pathlib import Path
import hashlib,json,zipfile
root=Path(__file__).resolve().parent.parent
out=root.parent/'server-builds'
name='AlphaWorkComplete-Recovered-source-PC-v1.4.78-App-v1.0.11'
archive=out/(name+'.zip')
assert not archive.exists(),'Never replace an existing published source bundle'
sha=lambda b:hashlib.sha256(b).hexdigest()
files=[p for p in sorted(root.rglob('*')) if p.is_file() and p.name!='SHA256SUMS' and '__pycache__' not in p.parts and p.name not in {'local.properties','google-services.json','keystore.properties'} and not any(x in p.parts for x in {'.gradle','build'}) and p.suffix not in {'.exe','.dat','.bak','.bin','.xls','.xlsx','.jks','.keystore','.apk'} and 'service-account' not in p.name]
(root/'SHA256SUMS').write_text(''.join(f'{sha(p.read_bytes())}  {p.relative_to(root)}\n' for p in files))
with zipfile.ZipFile(archive,'w',zipfile.ZIP_DEFLATED,9) as z:
 for p in files+[root/'SHA256SUMS']:z.write(p,name+'/'+str(p.relative_to(root)))
with zipfile.ZipFile(archive) as z:assert z.testzip() is None
archive.with_name(archive.name+'.sha256').write_text(sha(archive.read_bytes())+'  '+archive.name+'\n')
print(archive.name,archive.stat().st_size,sha(archive.read_bytes()))
