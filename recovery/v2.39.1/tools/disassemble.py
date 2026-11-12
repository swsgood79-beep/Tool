#!/usr/bin/env python3
"""Static GNU objdump output, with recovered Go function boundaries."""
import argparse,bisect,gzip,json,pathlib,re,subprocess
p=argparse.ArgumentParser();p.add_argument('binary');p.add_argument('evidence');a=p.parse_args();out=pathlib.Path(a.evidence)
symbols=json.loads((out/'symbols.json').read_text());starts=[s['address'] for s in symbols]
with (out/'pe-headers.txt').open('w') as f:subprocess.run(['objdump','-x',a.binary],stdout=f,check=True)
proc=subprocess.Popen(['objdump','-d','-j','.text',a.binary],stdout=subprocess.PIPE,text=True)
last=None;lines=0
with gzip.open(out/'all-functions.asm.gz','wt') as allout,gzip.open(out/'application-functions.asm.gz','wt') as appout:
 for line in proc.stdout:
  allout.write(line);m=re.match(r'\s*([0-9a-fA-F]+):',line)
  if not m:continue
  address=int(m.group(1),16);i=bisect.bisect_right(starts,address)-1
  if i<0:continue
  s=symbols[i]
  if address>=s['end'] or not s['name'].startswith('main.'):continue
  if i!=last:appout.write('\n# '+s['name']+' @ '+hex(s['address'])+'\n');last=i
  appout.write(line);lines+=1
if proc.wait()!=0:raise SystemExit('objdump failed')
for name,args in [('strings-ascii.txt.gz',['strings','-a','-n','4']),('strings-utf16le.txt.gz',['strings','-a','-e','l','-n','4'])]:
 proc=subprocess.Popen(args+[a.binary],stdout=subprocess.PIPE)
 with gzip.open(out/name,'wb') as f:
  while chunk:=proc.stdout.read(1048576):f.write(chunk)
 if proc.wait()!=0:raise SystemExit('strings failed')
print('Application assembly lines:',lines)
