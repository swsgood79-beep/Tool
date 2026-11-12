#!/usr/bin/env python3
"""Recover main package struct metadata from Go1.27 PE RTTI; never execute input."""
import json,struct,pathlib,argparse
p=argparse.ArgumentParser();p.add_argument('binary');p.add_argument('evidence');a=p.parse_args();b=pathlib.Path(a.binary).read_bytes();report=json.loads((pathlib.Path(a.evidence)/'report.json').read_text());ss=report['sections']
def off(v):
 for s in ss:
  if s['va']<=v<s['va']+s['size']:return s['offset']+v-s['va']
 raise ValueError('unmapped address')
def va(p):
 for s in ss:
  if s['offset']<=p<s['offset']+s['size']:return s['va']+p-s['offset']
 raise ValueError('unmapped file offset')
def var(p):
 value=shift=0
 for _ in range(10):
  x=b[p];p+=1;value|=(x&127)<<shift
  if x<128:return value,p
  shift+=7
 raise ValueError('bad varint')
def name(p):
 flags=b[p];n,q=var(p+1)
 if n>100000:raise ValueError('name too long')
 text=b[q:q+n].decode();q+=n;tag=''
 if flags&2:
  n,q=var(q);tag=b[q:q+n].decode()
 return text,tag,flags
pc=va(report['pclntab_offset']);pos=b.find(struct.pack('<Q',pc));assert pos>=0
base=struct.unpack_from('<Q',b,pos+37*8)[0];end=struct.unpack_from('<Q',b,pos+39*8)[0]
assert end>base
def typename(p):
 n=struct.unpack_from('<i',b,p+40)[0];text,_,_=name(off(base+n))
 return text[1:] if b[p+20]&2 else text
found=[]
for p in range(off(base),off(end)-80,8):
 if b[p+23]&31!=25 or not b[p+20]&4 or b[p+21] not in (1,2,4,8):continue
 try:
  text=typename(p)
  if not text.startswith('main.'):continue
  size=struct.unpack_from('<Q',b,p)[0];pkg,fields,n,cap=struct.unpack_from('<4Q',b,p+48)
  if n>1000 or cap!=n:continue
  pkgname=name(off(pkg))[0] if pkg else '';items=[]
  for i in range(n):
   np,tp,offset=struct.unpack_from('<3Q',b,off(fields)+i*24);field,tag,flags=name(off(np));assert field and offset<=size
   items.append({'name':field,'type':typename(off(tp)),'tag':tag,'offset':offset,'exported':bool(flags&1),'embedded':bool(flags&8)})
  found.append({'name':text,'package':pkgname,'size':size,'address':hex(va(p)),'fields':items})
 except (ValueError,UnicodeError,struct.error,IndexError,AssertionError):continue
out=pathlib.Path(__file__).parent;found.sort(key=lambda x:x['name']);(out/'main-struct-layouts.json').write_text(json.dumps(found,ensure_ascii=False,indent=2));print('Validated main struct descriptors:',len(found));print('Fields:',sum(len(x['fields']) for x in found))
