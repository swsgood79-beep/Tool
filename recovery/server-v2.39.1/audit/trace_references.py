#!/usr/bin/env python3
import pathlib,json,gzip,re,struct
r=pathlib.Path(__file__).parent;e=r.parent.parent/'v2.39.1/evidence';b=pathlib.Path('/workspace/recovery-input/AlphaWorkComplete-v2.39.1.exe').read_bytes();report=json.loads((e/'report.json').read_text());symbols=json.loads((e/'symbols.json').read_text());byaddr={x['address']:x['name'] for x in symbols}
def off(v):
 for s in report['sections']:
  if s['va']<=v<s['va']+s['size']:return s['offset']+v-s['va']
 raise ValueError()
records=[];fn='';lines=[]
for line in gzip.open(e/'application-functions.asm.gz','rt'):
 if line.startswith('# '):fn=line[2:].split(' @ ')[0]
 m=re.search(r'# 0x([0-9a-f]+)',line)
 if not m:continue
 try:p=off(int(m[1],16))
 except ValueError:continue
 text=b[p:p+200].split(b'\0')[0].decode('utf8','replace')
 if len(text)<3 or any(ord(c)<32 and c not in '\t\n\r' for c in text[:20]) or '\ufffd' in text[:20]:continue
 records.append({'function':fn,'instruction':line.strip(),'offset':p,'prefix':text})
(r/'main-string-references.json').write_text(json.dumps(records,ensure_ascii=False,indent=2));print('Refs:',len(records))
for x in records:
 if x['function'] in ['main.preparePCServer','main.main','main.encryptData','main.newBackupStore','main.openHost']:
  print(x['function'],hex(x['offset']),repr(x['prefix'][:130]))
