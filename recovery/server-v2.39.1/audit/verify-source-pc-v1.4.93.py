"""Check current combined sources rebuild the already published PC 1.4.93 EXE."""
from pathlib import Path
import hashlib,json,os,subprocess,tempfile,zipfile
root=Path(__file__).resolve().parent.parent;out=root.parent/'server-builds'
name='AlphaWorkComplete-Recovered-source-PC-v1.4.93-App-v1.0.11'
sha=lambda b:hashlib.sha256(b).hexdigest()
archive=out/(name+'.zip');proof=out/(name+'-rebuild.json');assert not proof.exists()
with zipfile.ZipFile(out/'AlphaWorkComplete-Recovered-Windows-x64-v1.4.93.zip') as z:expected=sha(z.read('AlphaWorkComplete-Recovered.exe'))
with tempfile.TemporaryDirectory(prefix='alpha-current-source-',dir='/tmp') as temp:
 with zipfile.ZipFile(archive) as z:
  assert z.testzip() is None
  for n in z.namelist():assert not Path(n).is_absolute() and '..' not in Path(n).parts
  for line in z.read(name+'/SHA256SUMS').decode().splitlines():
   digest,path=line.split('  ',1);assert sha(z.read(name+'/'+path))==digest,path
  z.extractall(temp)
 folder=Path(temp)/name;exe=Path(temp)/'rebuilt.exe'
 env={**os.environ,'GOOS':'windows','GOARCH':'amd64','CGO_ENABLED':'0','GOTOOLCHAIN':'local','GOCACHE':'/workspace/toolchain/cache','GOPATH':'/workspace/toolchain/gopath'}
 subprocess.run(['/workspace/toolchain/go/bin/go','build','-trimpath','-buildvcs=false','-ldflags=-s -w -H=windowsgui','-o',str(exe),'.'],cwd=folder,env=env,check=True)
 actual=sha(exe.read_bytes());assert actual==expected,'Current PC sources must match the existing PC 1.4.93 binary'
 apk=out/'AlphaAdmin-Android-v1.0.11.apk';manifest=json.loads((folder/'android-admin/update.json').read_text());assert manifest['version_code']==12 and manifest['sha256']==sha(apk.read_bytes())
 record={'pc_version':'1.4.93','app_version':'1.0.11','source_zip':archive.name,'source_zip_sha256':sha(archive.read_bytes()),'published_pc_exe_sha256':expected,'rebuilt_pc_exe_sha256':actual,'identical_to_existing_pc_release':True,'apk_sha256':sha(apk.read_bytes()),'full_source_checksums':'pass','windows_execution':'cross-build verification only'}
 proof.write_text(json.dumps(record,ensure_ascii=False,indent=2)+'\n');print('Current source rebuild exactly matches published PC 1.4.93; APK 1.0.11 preserved.')
