(()=>{'use strict';const root=document.getElementById('purchasing-app');if(!root)return;
const $=id=>document.getElementById('purchase-'+id),node=(tag,text,cls)=>{const n=document.createElement(tag);if(text!==undefined)n.textContent=text;if(cls)n.className=cls;return n};
let tab=new URL(location.href).searchParams.get('done')==='1'?'done':'pending',selected=Number(new URL(location.href).searchParams.get('call'))||0,page=1,data=null,loading=false,sending=false,generation=0,actionError=false;
window.__alphaNativeBusy=()=>sending;
const when=value=>value?new Intl.DateTimeFormat('ko-KR',{timeZone:'Asia/Seoul',month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit',hour12:false}).format(new Date(value)):'—';
function message(text){$('message').textContent=text;$('message').hidden=!text}
function setTheme(value){if(!['system','light','dark'].includes(value))value='system';if(value==='system')delete document.body.dataset.theme;else document.body.dataset.theme=value;try{localStorage.setItem('alpha-purchasing-theme',value);window.AlphaNativeTheme?.setTheme(value)}catch(_){}$('theme').value=value}
let theme='system';try{theme=window.AlphaNativeTheme?.getTheme()||localStorage.getItem('alpha-purchasing-theme')||'system'}catch(_){}setTheme(theme);$('theme').addEventListener('change',e=>setTheme(e.target.value));
function route(){const url=new URL('/purchasing',location.origin);if(selected)url.searchParams.set('call',String(selected));history.replaceState(null,'',url.pathname+url.search)}
function show(){
 $('list').hidden=tab==='menu'||selected>0;$('detail').hidden=tab==='menu'||!selected;$('menu').hidden=tab!=='menu';
 for(const b of document.querySelectorAll('.purchase-nav [data-tab]')){if(b.dataset.tab===tab)b.setAttribute('aria-current','page');else b.removeAttribute('aria-current')}
 for(const b of document.querySelectorAll('[data-list]'))b.setAttribute('aria-pressed',String(b.dataset.list===tab));
 if(!data)return;$('pending-count').textContent=data.pending_count;$('done-count').textContent=data.done_count;
 const rows=data.calls||[];$('rows').replaceChildren();
 rows.forEach((c,i)=>{const button=node('button',undefined,'purchase-row');button.type='button';const rank=node('span',String((page-1)*20+i+1).padStart(2,'0'),'purchase-rank'),info=node('span',undefined,'purchase-row-info');info.append(node('span',when(c.CreatedAt)+' · '+c.Worker),node('strong',c.ProjectName),node('small','수주 '+c.OrderNumber));button.append(rank,info,node('span',c.CompletedAt?'완료':'대기','purchase-state'+(c.CompletedAt?' done':'')));button.addEventListener('click',()=>open(c.ID));$('rows').append(button)});
 if(!rows.length)$('rows').append(node('p',tab==='done'?'완료한 호출이 없습니다.':'접수된 호출이 없습니다.','muted'));
 $('first').hidden=tab!=='pending'||!rows.length;$('page').textContent=data.page+' / '+data.pages;$('prev').disabled=data.page<=1;$('next').disabled=data.page>=data.pages;
 const c=data.selected;if(selected&&c){$('caller').textContent=c.Worker+'님의 호출';$('worker').textContent=c.Worker;$('time').textContent=when(c.CreatedAt);$('project').textContent=c.ProjectName;$('number').textContent=c.OrderNumber;$('position').textContent=c.CompletedAt?'완료':'접수 순서 '+String(data.position||1).padStart(2,'0');$('completed-row').hidden=!c.CompletedAt;$('completed-time').textContent=when(c.CompletedAt);$('finish').hidden=!!c.CompletedAt;
 const next=data.next;$('next-preview').hidden=!next;if(next){$('next-worker').textContent='다음 호출 · '+next.Worker;$('next-project').textContent=next.ProjectName}}
}
async function load(force=false){if((loading&&!force)||sending||tab==='menu'||document.hidden)return;const current=++generation;loading=true;try{
 const q=new URLSearchParams({done:tab==='done'?'1':'0',page:String(page)});if(selected)q.set('call',String(selected));const r=await fetch('/purchasing/status?'+q,{cache:'no-store'});if(!r.ok)throw Error(r.status===401||r.status===403?'로그인 상태를 확인해 주세요.':'호출 목록을 불러오지 못했습니다. 다시 연결합니다.');
 const incoming=await r.json();if(current!==generation)return;data=incoming;page=data.page;if(!actionError)message('');show()
}catch(e){if(current===generation)message(e.message)}finally{if(current===generation)loading=false}}
function open(id){if(sending)return;selected=id;actionError=false;route();load(true)}
function change(next){if(sending)return;tab=next;selected=0;page=1;actionError=false;route();show();load(true)}
for(const b of document.querySelectorAll('[data-tab]'))b.addEventListener('click',()=>change(b.dataset.tab));for(const b of document.querySelectorAll('[data-list]'))b.addEventListener('click',()=>change(b.dataset.list));
$('back').addEventListener('click',()=>change('pending'));$('first').addEventListener('click',()=>{if(data?.calls[0])open(data.calls[0].ID)});$('prev').addEventListener('click',()=>{if(page>1){page--;load()}});$('next').addEventListener('click',()=>{if(page<data.pages){page++;load()}});
$('finish').addEventListener('click',async()=>{if(sending||!selected)return;generation++;loading=false;sending=true;$('finish').disabled=true;$('finish').textContent='완료 처리 중…';root.dataset.sending='1';
 try{const r=await fetch('/purchasing/complete',{method:'POST',body:new URLSearchParams({csrf:root.dataset.csrf,id:String(selected)}),headers:{Accept:'application/json'}});if(!r.ok)throw Error('완료 처리를 저장하지 못했습니다. 다시 시도해 주세요.');const result=await r.json();if(result.ok!==true)throw Error('완료 결과를 확인하지 못했습니다.');actionError=false;selected=Number(result.next)||0;tab='pending';page=1;route();
 }catch(e){actionError=true;message(e.message)}finally{sending=false;delete root.dataset.sending;$('finish').disabled=false;$('finish').textContent='완료하고 다음 호출 보기';await load()}
});
show();load();setInterval(load,3000);document.addEventListener('visibilitychange',load);
})();
