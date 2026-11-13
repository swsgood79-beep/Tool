"""Reproducibly pack already-downloaded official NuGet Python + PyPI wheels; no secrets."""
from pathlib import Path
import hashlib,json,zipfile
root=Path(__file__).resolve().parent.parent
inputs=Path('/tmp/alpha-gtts-build')
files={}
with zipfile.ZipFile(inputs/'python.3.12.10.nupkg') as z:
 for name in z.namelist():
  if name.startswith('tools/') and not name.endswith('/'):
   files[name.removeprefix('tools/')]=z.read(name)
for p in sorted((inputs/'wheels').glob('*.whl')):
 with zipfile.ZipFile(p) as z:
  for name in z.namelist():
   if not name.endswith('/'):
    files['Lib/site-packages/'+name]=z.read(name)
sha=lambda b:hashlib.sha256(b).hexdigest()
manifest={'python':'3.12.10 x64 (official PSF NuGet python package)','inputs':{p.name:sha(p.read_bytes()) for p in [inputs/'python.3.12.10.nupkg',*sorted((inputs/'wheels').glob('*.whl'))]},'files':{n:sha(b) for n,b in sorted(files.items())},'licenses':'Python LICENSE.txt and wheel dist-info licenses preserved','download_sources':['https://api.nuget.org/v3-flatcontainer/python/3.12.10/python.3.12.10.nupkg','https://pypi.org/']}
(root/'tts_assets/runtime-manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
with zipfile.ZipFile(root/'tts_assets/python-gtts-win-x64.zip','w',zipfile.ZIP_DEFLATED,9) as z:
 for n,b in sorted(files.items()):
  info=zipfile.ZipInfo(n,date_time=(2026,1,1,0,0,0));info.compress_type=zipfile.ZIP_DEFLATED;z.writestr(info,b)
print('runtime files',len(files),'zip bytes',(root/'tts_assets/python-gtts-win-x64.zip').stat().st_size)
