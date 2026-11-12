#!/usr/bin/env python3
"""Independently validate recovered bytes/records/symbol ranges against original PE."""
import argparse,hashlib,json,pathlib,struct
p=argparse.ArgumentParser();p.add_argument('binary');p.add_argument('evidence');a=p.parse_args()
b=pathlib.Path(a.binary).read_bytes();e=pathlib.Path(a.evidence);report=json.loads((e/'report.json').read_text());assets=json.loads((e/'asset-manifest.json').read_text());symbols=json.loads((e/'symbols.json').read_text())
assert hashlib.sha256(b).hexdigest()==report['binary_sha256']
def off(v):
 for s in report['sections']:
  if s['va']<=v<s['va']+s['size']:return s['offset']+v-s['va']
 raise ValueError('unmapped address')
for x in assets:
 data=(e/'assets'/x['path']).read_bytes();p=x['table_offset'];np,n,dp,d=struct.unpack_from('<4Q',b,p)
 assert b[off(np):off(np)+n].decode()==x['path']
 assert off(dp)==x['offset'] and d==x['length']
 assert data==b[x['offset']:x['offset']+x['length']]
 assert hashlib.sha256(data).hexdigest()==x['sha256']
 h=bytearray(hashlib.sha256(data if d<=1024 else b'\x01'+data).digest())
 if d<=1024:h[0]^=255
 assert h[:16]==b[p+32:p+48]
text=next(x for x in report['sections'] if x['name']=='.text')
assert len(symbols)==report['functions']
assert any(x['name']=='main.main' for x in symbols)
assert all(text['va']<=x['address']<x['end']<=text['va']+text['size'] for x in symbols)
assert all(x['end']<=y['address'] for x,y in zip(symbols,symbols[1:]))
result={'binary_sha256_verified':True,'assets_byte_identical':len(assets),'compiler_hashes_verified':len(assets),'function_ranges_verified':len(symbols),'backend_source_recovered':False}
(e/'verification.json').write_text(json.dumps(result,indent=2));print(json.dumps(result))
