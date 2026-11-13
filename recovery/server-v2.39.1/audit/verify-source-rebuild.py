#!/usr/bin/env python3
"""Verify the delivered source archive builds the delivered Windows EXE; never touch operating data."""
import argparse,hashlib,json,os,subprocess,tempfile,zipfile
from pathlib import Path
from datetime import datetime,timezone
p=argparse.ArgumentParser();p.add_argument('--version',default='1.4.13');p.add_argument('--go',default='/workspace/toolchain/go/bin/go');args=p.parse_args()
module=Path(__file__).resolve().parent.parent;out=module.parent/'server-builds';stem='AlphaWorkComplete-Recovered'
source=out/f'{stem}-source-v{args.version}.zip';windows=out/f'{stem}-Windows-x64-v{args.version}.zip';proof=out/f'{stem}-source-v{args.version}-rebuild.json'
assert not proof.exists(),'Never overwrite an existing verification record'
sha=lambda b:hashlib.sha256(b).hexdigest()
with zipfile.ZipFile(windows) as z:expected=sha(z.read(stem+'.exe'));assert z.testzip() is None
with tempfile.TemporaryDirectory(prefix='alpha-source-rebuild-',dir='/tmp') as temp:
 with zipfile.ZipFile(source) as z:
  assert z.testzip() is None
  sums=next(n for n in z.namelist() if n.endswith('/SHA256SUMS'));prefix=sums.removesuffix('SHA256SUMS')
  for line in z.read(sums).decode().splitlines():
   digest,name=line.split('  ',1);assert sha(z.read(prefix+name))==digest,name
  for name in z.namelist():assert not Path(name).is_absolute() and '..' not in Path(name).parts
  z.extractall(temp)
 root=Path(temp)/prefix;exe=Path(temp)/'rebuilt.exe';env={**os.environ,'GOOS':'windows','GOARCH':'amd64','CGO_ENABLED':'0','GOTOOLCHAIN':'local','GOCACHE':'/workspace/toolchain/cache','GOPATH':'/workspace/toolchain/gopath'}
 subprocess.run([args.go,'build','-trimpath','-buildvcs=false','-ldflags=-s -w -H=windowsgui','-o',str(exe),'.'],cwd=root,env=env,check=True)
 actual=sha(exe.read_bytes());assert actual==expected,'Rebuild succeeded but the EXE differs; do not claim an identical rebuild'
 record={'version':args.version,'checked_utc':datetime.now(timezone.utc).isoformat(),'source_zip':source.name,'source_zip_sha256':sha(source.read_bytes()),'delivered_exe_sha256':expected,'rebuilt_exe_sha256':actual,'identical':True,'toolchain':subprocess.check_output([args.go,'version'],text=True).strip(),'target':'windows/amd64; CGO_ENABLED=0','source_archive_all_file_checksums':'pass','dependency_note':'go.mod/go.sum with external modules in cache; this is not a fully offline dependency bundle','windows_execution':'not performed; cross-build comparison only'}
 proof.write_text(json.dumps(record,ensure_ascii=False,indent=2)+'\n');print(proof.name,'identical Windows EXE verified')
