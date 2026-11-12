#!/usr/bin/env python3
"""Read-only PE/Go1.27 metadata and embed.FS extraction. No binary execution."""
import argparse,struct,json,hashlib,pathlib,gzip,re

def main():
 ap=argparse.ArgumentParser();ap.add_argument('binary');ap.add_argument('output');a=ap.parse_args();b=pathlib.Path(a.binary).read_bytes();out=pathlib.Path(a.output);out.mkdir(parents=True,exist_ok=True)
 pe=struct.unpack_from('<I',b,60)[0];assert b[:2]==b'MZ' and b[pe:pe+4]==b'PE\0\0'
 n=struct.unpack_from('<H',b,pe+6)[0];optsize=struct.unpack_from('<H',b,pe+20)[0];opt=pe+24;assert struct.unpack_from('<H',b,opt)[0]==0x20b
 base=struct.unpack_from('<Q',b,opt+24)[0];sections=[]
 for i in range(n):
  p=opt+optsize+i*40;vs,va,rs,ro=struct.unpack_from('<IIII',b,p+8);sections.append(dict(name=b[p:p+8].rstrip(b'\0').decode(),va=base+va,virtual_size=vs,offset=ro,size=rs))
 def vaoff(va):
  for s in sections:
   if s['va']<=va<s['va']+s['size']:return s['offset']+va-s['va']
  raise ValueError('unmapped VA')
 def offva(off):
  for s in sections:
   if s['offset']<=off<s['offset']+s['size']:return s['va']+off-s['offset']
  raise ValueError('unmapped offset')
 def uvar(p):
  v=0;shift=0
  while True:
   c=b[p];p+=1;v|=(c&127)<<shift
   if c<128:return v,p
   shift+=7;assert shift<70
 def gostr(p):
  size,p=uvar(p);return b[p:p+size],p+size
 bi=b.index(b'\xff Go buildinf:');assert b[bi+15]&2
 version,p=gostr(bi+32);mod,p=gostr(p);mod=mod[16:-16].decode();build='Go version: '+version.decode()+'\n'+mod
 (out/'go-build-info.txt').write_text(build)
 modules=[]
 for line in mod.splitlines():
  parts=line.split('\t')
  if parts[0]=='dep':modules.append({'path':parts[1],'version':parts[2],'sum':parts[3] if len(parts)>3 else ''})
 (out/'modules.json').write_text(json.dumps(modules,indent=2))
 pcl=b.index(b'\xf1\xff\xff\xff\0\0\x01\x08');nf,nfiles,text,fn,cu,ft,pt,pc=struct.unpack_from('<8Q',b,pcl+8)
 text=text or next(s['va'] for s in sections if s['name']=='.text');table=pcl+pc;symbols=[]
 for i in range(nf):
  entry,fo=struct.unpack_from('<II',b,table+i*8);nxt=struct.unpack_from('<I',b,table+(i+1)*8)[0];f=table+fo;nameoff=struct.unpack_from('<i',b,f+4)[0];p=pcl+fn+nameoff;end=b.index(0,p);name=b[p:end].decode('utf-8','replace')
  line=struct.unpack_from('<i',b,f+36)[0];source=None
  pcfile,cuindex=struct.unpack_from('<I',b,f+20)[0],struct.unpack_from('<I',b,f+32)[0]
  if pcfile and cuindex!=0xffffffff:
   delta,z=uvar(pcl+pt+pcfile);fileidx=-1+(~(delta>>1) if delta&1 else delta>>1)
   if fileidx>=0:
    fileoff=struct.unpack_from('<I',b,pcl+cu+4*(cuindex+fileidx))[0]
    if fileoff!=0xffffffff:
     fp=pcl+ft+fileoff;source=b[fp:b.index(0,fp)].decode('utf-8','replace')
  symbols.append(dict(name=name,address=text+entry,end=text+nxt,size=nxt-entry,source_file=source,source_line=line,func_offset=f))
 (out/'symbols.json').write_text(json.dumps(symbols,ensure_ascii=False,indent=2))
 (out/'application-functions.txt').write_text('\n'.join(f"{s['address']:016x} {s['size']:7d} {s['source_line']:5d} {s['name']} {s['source_file'] or ''}" for s in symbols if s['name'].startswith('main.'))+'\n')
 files=[s.decode('utf-8','replace') for s in b[pcl+ft:pcl+pt].split(b'\0') if s]
 (out/'source-file-paths.txt').write_text('\n'.join(files)+'\n')
 # embed.file is {name string, data string, hash [16]byte}. Locate by exact
 # name pointer/length, mapped addresses, size bounds and Go1.27 compiler hash.
 candidates=[]
 for m in re.finditer(rb'(?:web|assets)/',b):
  try:nameva=offva(m.start())
  except ValueError:continue
  needle=struct.pack('<Q',nameva);start=0
  while True:
   p=b.find(needle,start)
   if p<0:break
   start=p+1
   if p+48>len(b):continue
   namelen=struct.unpack_from('<Q',b,p+8)[0]
   if not 5<namelen<240:continue
   rawpath=b[m.start():m.start()+namelen]
   if not re.fullmatch(rb'(?:web|assets)/[A-Za-z0-9_./-]+',rawpath):continue
   path=rawpath.decode()
   ptr,size=struct.unpack_from('<QQ',b,p+16)
   if size<=0 or size>100_000_000:continue
   try:q=vaoff(ptr)
   except ValueError:continue
   content=b[q:q+size]
   if len(content)!=size:continue
   # Go1.27 cmd/internal/hash: Sum32 for <=1KiB, New32 otherwise.
   compiler_hash=bytearray(hashlib.sha256(content if size<=1024 else b'\x01'+content).digest())
   if size<=1024:compiler_hash[0]^=255
   if bytes(compiler_hash[:16])!=b[p+32:p+48]:continue
   recorded_hash=b[p+32:p+48].hex()
   parts=pathlib.PurePosixPath(path)
   if '..' in parts.parts:continue
   dst=out/'assets'/parts;dst.parent.mkdir(parents=True,exist_ok=True)
   if dst.exists():assert dst.read_bytes()==content
   else:dst.write_bytes(content)
   candidates.append(dict(path=path,offset=q,length=size,table_offset=p,sha256=hashlib.sha256(content).hexdigest(),recorded_embed_hash=recorded_hash,embed_hash_verified=True))
 unique={x['path']:x for x in candidates}
 (out/'asset-manifest.json').write_text(json.dumps(list(unique.values()),ensure_ascii=False,indent=2))
 # Evidence of all printable UTF-8/ASCII runs, not claimed semantic string boundaries.
 with gzip.open(out/'strings-utf8.txt.gz','wt',encoding='utf-8') as z:
  for s in re.findall(r'[^\x00-\x1f\x7f-\x9f\ufffd]{4,}',b.decode('utf-8','replace')):z.write(s+'\n')
 packages=sorted(set(s['name'][:s['name'].find('.',s['name'].rfind('/')+1)] for s in symbols if s['name'].find('.',s['name'].rfind('/')+1)>0))
 (out/'package-prefixes.txt').write_text('\n'.join(packages)+'\n')
 report={'binary_sha256':hashlib.sha256(b).hexdigest(),'binary_size':len(b),'go_version':version.decode(),'image_base':hex(base),'sections':sections,'pclntab_offset':pcl,'functions':nf,'source_files_header':nfiles,'source_files_recovered':len(files),'assets_recovered':len(unique),'application_functions':sum(s['name'].startswith('main.') for s in symbols),'limits':'Embedded bytes copied from validated pointers and lengths and verified with Go1.27 compiler hashes. Ordinary SHA-256 generated separately for reproducibility. No original Go backend source recovered.'}
 (out/'report.json').write_text(json.dumps(report,indent=2));print(json.dumps(report,ensure_ascii=False))
if __name__=='__main__':main()
