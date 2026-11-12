#!/usr/bin/env python3
"""Inventory every recovered asset, HTML URL/form and JS transport callsite.
Callsites retain context; dynamic URL variables are not claimed resolved.
"""
import pathlib,re,json
r=pathlib.Path(__file__).parent;e=r.parent/'v2.39.1/evidence';s=json.loads((e/'symbols.json').read_text());main=[x for x in s if x['name'].startswith('main.')];(r/'audit/main-functions.json').write_text(json.dumps(main,ensure_ascii=False,indent=2))
receivers=sorted(set(re.match(r'main\.\(\*?([^)]*)\)\.',x['name'])[1] for x in main if re.match(r'main\.\(\*?([^)]*)\)\.',x['name'])))
(r/'audit/main-receiver-type-candidates.json').write_text(json.dumps({'warning':'Receiver type names only; struct fields/layouts are not recovered by this inventory.','types':receivers},indent=2))
assets=[];refs=[];forms=[];calls=[]
for p in sorted((r/'web').iterdir()):
 assets.append({'file':p.name,'bytes':p.stat().st_size})
 if p.suffix not in ('.html','.js','.css'):continue
 text=p.read_text()
 for m in re.finditer(r'<form\b[^>]*>(.*?)</form>',text,re.S|re.I):
  tag=m.group().split('>',1)[0];method=re.search(r'method=["\']([^"\']+)',tag,re.I);action=re.search(r'action=["\']([^"\']+)',tag,re.I)
  forms.append({'file':p.name,'line':text.count('\n',0,m.start())+1,'method':method[1].upper() if method else 'GET','action':action[1] if action else '(current URL)','fields':sorted(set(re.findall(r'\bname=["\']([^"\']+)',m.group())))})
 for m in re.finditer(r'\b(?:href|src|action|formaction)=["\']([^"\']+)',text):refs.append({'file':p.name,'line':text.count('\n',0,m.start())+1,'url':m[1]})
 for m in re.finditer(r'\b(?:fetch|axios(?:\.\w+)?|WebSocket|EventSource|XMLHttpRequest)\s*\(',text):
  calls.append({'file':p.name,'line':text.count('\n',0,m.start())+1,'transport':m.group().split('(')[0].strip(),'context':text[max(0,m.start()-180):m.start()+1600]})
(r/'audit/frontend-contracts.json').write_text(json.dumps({'assets':assets,'forms':forms,'url_references':refs,'transport_calls':calls,'limits':'Dynamic template expressions and JS variables are preserved verbatim. Context provides JSON field observations, not a complete proven backend specification.'},ensure_ascii=False,indent=2))
print('Assets',len(assets),'forms',len(forms),'URL refs',len(refs),'transport calls',len(calls),'main functions',len(main),'receiver candidates',len(receivers))
