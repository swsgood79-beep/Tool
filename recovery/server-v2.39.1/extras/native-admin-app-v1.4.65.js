(()=>{'use strict';const app=document.getElementById('alpha-admin-app');if(!app)return;
const meta=document.getElementById('alpha-design'),home='/mobile-admin?design-tab=orders';
const heading=document.querySelector('.design-orders-title h1');if(heading)heading.textContent='진행 중인 수주';
// Any worker-style back link returns to the authenticated app work/calendar tab.
for(const a of document.querySelectorAll('a[href]')){let u;try{u=new URL(a.href,location.href)}catch(_){continue}if(u.origin===location.origin&&u.pathname==='/work/'+app.dataset.board){u.pathname='/mobile-admin';u.searchParams.delete('entry');if(u.searchParams.get('design-tab')!=='calendar'&&u.searchParams.get('design-tab')!=='menu')u.searchParams.set('design-tab','orders');a.href=u.pathname+u.search}}
const link=document.querySelector('.design-bottom-nav [data-tab=inspection]');let busy=false;
function draw(n){if(!link)return;let badge=link.querySelector('.native-inspection-badge');if(!badge){badge=document.createElement('span');badge.className='native-inspection-badge';badge.setAttribute('aria-hidden','true');link.append(badge)}badge.textContent=String(n);badge.hidden=n===0;link.setAttribute('aria-label',n?'검사, 신청 대기 '+n+'건':'검사')}
async function update(){if(busy||document.hidden)return;const local=document.getElementById('inspection-request-items');if(local){draw(local.querySelectorAll('[data-request]').length);return}busy=true;const abort=new AbortController(),timeout=setTimeout(()=>abort.abort(),8000);try{const r=await fetch('/inspection-requests/panel',{credentials:'same-origin',cache:'no-store',signal:abort.signal});if(!r.ok)return;const doc=new DOMParser().parseFromString(await r.text(),'text/html'),items=doc.getElementById('inspection-request-items');if(items)draw(items.querySelectorAll('[data-request]').length)}catch(_){}finally{clearTimeout(timeout);busy=false}}
update();setInterval(update,5000);document.addEventListener('visibilitychange',()=>{if(!document.hidden)update()});
})();
