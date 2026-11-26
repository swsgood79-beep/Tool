"""Copy complete helper sources and reproducibly build the embedded Windows worker."""
from pathlib import Path
import hashlib,os,shutil,subprocess,json
root=Path(__file__).resolve().parent.parent
src=root.parent/'erp-bridge-check';dest=root/'embedded/erp-bridge-source';dest.mkdir(parents=True,exist_ok=True)
files=sorted(list(src.glob('*.go'))+list(src.glob('*.ps1'))+[src/'go.mod',src/'README.md'])
for old in dest.iterdir():
 if old.is_file():old.unlink()
for p in files:shutil.copy2(p,dest/p.name)
env={**os.environ,'GOOS':'windows','GOARCH':'amd64','CGO_ENABLED':'0','GOCACHE':'/workspace/toolchain/cache','GOTOOLCHAIN':'local'}
subprocess.run(['/workspace/toolchain/go/bin/go','build','-trimpath','-buildvcs=false','-ldflags=-s -w','-o',str(root/'embedded/erp-bridge.payload'),'.'],cwd=dest,env=env,check=True)
h=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
(root/'embedded/erp-bridge-manifest.json').write_text(json.dumps({'version':'0.2','entrypoints':['--managed-setup','--managed-capture'],'payload_sha256':h(root/'embedded/erp-bridge.payload'),'source_sha256':{p.name:h(dest/p.name) for p in files},'field_execution':'new saved-profile rebinding and parent-managed execution pending actual Windows ERP confirmation'},indent=2)+'\n')
print('Embedded ERP helper built with complete sources')
