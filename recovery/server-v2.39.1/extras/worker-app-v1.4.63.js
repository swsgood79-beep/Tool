(()=>{'use strict';const cfg=document.getElementById('worker-app-config');if(!cfg)return;const board=cfg.dataset.board,parentOrigin='https://swsgood79-beep.github.io',valid=/^[A-Za-z0-9_-]{43}$/;if(!valid.test(board||''))return;
if(parent===window){const append=()=>{const menu=document.querySelector('.design-menu');if(!menu||menu.querySelector('.worker-app-install'))return;const a=document.createElement('a');a.className='worker-app-install';a.href=parentOrigin+'/Tool/app/#k='+board;a.textContent='알파 앱 설치 · QR 없이 실행';a.rel='noopener';a.target='_blank';menu.append(a)};append();addEventListener('DOMContentLoaded',append,{once:true});return}
document.addEventListener('click',e=>{const a=e.target.closest('a');if(!a)return;const u=new URL(a.href,location.href);if(u.origin===location.origin&&u.pathname.startsWith('/mobile-admin')){e.preventDefault();window.open(u.href,'_blank','noopener')}},true);
const themes=['system','light','dark'];const cookie=value=>{try{document.cookie='alpha-app-theme-pwa-v1='+value+'; Path=/; Max-Age=31536000; Secure; SameSite=None; Partitioned'}catch(_){}};
document.addEventListener('alpha-theme-change',e=>{if(!themes.includes(e.detail))return;cookie(e.detail);parent.postMessage({type:'alpha-app-theme',board,value:e.detail},parentOrigin)});
addEventListener('message',e=>{if(e.source!==parent||e.origin!==parentOrigin||e.data?.board!==board||e.data.type!=='alpha-app-theme-apply'||!themes.includes(e.data.value))return;cookie(e.data.value);document.dispatchEvent(new CustomEvent('alpha-set-theme',{detail:e.data.value}))});
parent.postMessage({type:'alpha-app-theme-request',board,value:document.documentElement.dataset.alphaTheme||'system'},parentOrigin);
const dirty=new Set();let posting=false,availableRelease=cfg.dataset.release,failures=0;
function busy(){return posting||!!document.querySelector('form[data-sending],.worker-inspection-dialog[open],.worker-special-review-dialog[open]')||[...dirty].some(f=>f.isConnected&&f.getClientRects().length)}
function state(nonce=''){parent.postMessage({type:'alpha-app-state',board,release:cfg.dataset.release,availableRelease,unavailable:failures>=2,path:location.pathname+location.search,dirty:!!busy(),nonce},parentOrigin)}
for(const type of ['input','change'])document.addEventListener(type,e=>{const f=e.target.closest('form');if(f&&(f.method||'get').toLowerCase()==='post'&&(e.isTrusted||e.target.type==='file')){dirty.add(f);state()}},true);
document.addEventListener('reset',e=>{dirty.delete(e.target);state()},true);document.addEventListener('submit',e=>{queueMicrotask(()=>{if(!e.defaultPrevented&&(e.target.method||'get').toLowerCase()==='post'){posting=true;state()}})},true);
addEventListener('message',e=>{if(e.source!==parent||e.origin!==parentOrigin||e.data?.board!==board)return;if(e.data.type==='alpha-app-check')state(e.data.nonce)});
addEventListener('pageshow',()=>{posting=false;state()});addEventListener('pagehide',()=>parent.postMessage({type:'alpha-app-leaving',board},parentOrigin));
async function version(){try{const r=await fetch('/ready/'+board,{cache:'no-store'});if(!r.ok)throw Error();const data=await r.json();if(typeof data.release==='string')availableRelease=data.release;failures=0}catch(_){failures++}state()}
state();setInterval(()=>state(),1000);setInterval(version,10000);version();
})();
