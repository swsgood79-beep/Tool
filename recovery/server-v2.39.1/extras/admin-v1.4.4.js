(() => {
 'use strict';
 const koreanDate=new Intl.DateTimeFormat('sv-SE',{timeZone:'Asia/Seoul',year:'numeric',month:'2-digit',day:'2-digit'});
 const normal=value=>String(value).normalize('NFKC').toLocaleLowerCase('ko-KR').replace(/\s+/g,'');
 function paginate({prefix,selector,filterAttribute,typeAttribute,size}) {
  const query=document.getElementById(`admin-${prefix==='order'?'order':'event'}-search`);
  const container=document.getElementById(`${prefix}-items`);
  if(!query||!container)return;
  const initialHTML=container.innerHTML;let items=[...container.querySelectorAll(selector)];const buttons=[...document.querySelectorAll(`[${filterAttribute}]`)];
  let filter='all',page=0;
  const prev=document.getElementById(`${prefix}-prev`),next=document.getElementById(`${prefix}-next`),label=document.getElementById(`${prefix}-page`);
  let day=koreanDate.format(new Date());
  const isToday=item=>item.dataset.today==='true'&&container.dataset.day===day;
  const eligible=(item,value='all')=>prefix!=='order'||(value==='today'?isToday(item):item.dataset.status!=='done');
  const dueUI=window.AlphaDue;
  const inFilter=(item,value)=>(prefix==='order'&&['due_today','due_tomorrow'].includes(value)?item.dataset.status!=='hidden'&&dueUI.isDue(item.dataset.due,value,day):eligible(item,value)&&(value==='all'||(prefix==='order'&&value==='today'?isToday(item):prefix==='order'&&['due_today','due_tomorrow'].includes(value)?item.dataset.status==='active'&&dueUI.isDue(item.dataset.due,value,day):prefix==='order'&&['setting','shipping'].includes(value)?item.dataset.status==='active'&&item.dataset.stage===value:item.getAttribute(typeAttribute)===value)));
  function draw(){
   if(prefix==='order')items=dueUI.order(container,items);
   const text=normal(query.value);
   const matches=items.filter(item=>normal(item.dataset.search).includes(text)&&inFilter(item,filter));
   const pages=Math.max(1,Math.ceil(matches.length/size));page=Math.min(page,pages-1);
   const visible=new Set(matches.slice(page*size,(page+1)*size));items.forEach(item=>item.hidden=!visible.has(item));
   if(prefix==='order'){dueUI.paint(document.querySelector('.delivery-panel'),items,filter,day,n=>n.dataset.status!=='hidden');dueUI.group(container,items,filter)}
   buttons.forEach(button=>{const value=button.getAttribute(filterAttribute);button.setAttribute('aria-pressed',String(value===filter));const count=button.querySelector('.filter-count');if(count)count.textContent=items.filter(item=>inFilter(item,value)).length});
   label.textContent=`${matches.length}건 · ${page+1} / ${pages} 페이지`;prev.disabled=page===0;next.disabled=page>=pages-1;
   container.querySelector('.admin-no-results').hidden=matches.length>0||items.length===0;
   if(prefix==='order')document.getElementById('order-count').textContent=`전체 ${items.filter(item=>eligible(item,'all')).length}건`;
  }
  query.addEventListener('input',()=>{page=0;container.scrollTop=0;draw()});
  buttons.forEach(button=>button.addEventListener('click',()=>{filter=button.getAttribute(filterAttribute);page=0;container.scrollTop=0;draw()}));
  prev.addEventListener('click',()=>{page=Math.max(0,page-1);container.scrollTop=0;draw()});next.addEventListener('click',()=>{page++;container.scrollTop=0;draw()});
  if(prefix==='order'){const rollover=()=>{const nextDay=koreanDate.format(new Date());if(nextDay!==day){day=nextDay;page=0;draw();const date=document.getElementById('admin-business-date');if(date)date.textContent=day}};setInterval(rollover,1000);document.addEventListener('visibilitychange',()=>{if(!document.hidden)rollover()})}
  draw();
  if(prefix==='order'){let previousHTML=initialHTML,pending=false;async function refresh(){if(pending)return;pending=true;try{if(document.hidden)return;const response=await fetch('/orders/panel',{cache:'no-store',credentials:'same-origin'});if(!response.ok)return;const doc=new DOMParser().parseFromString(await response.text(),'text/html'),fresh=doc.getElementById('order-items');if(!fresh)return;const hint=document.getElementById('admin-schedule-hint');if(hint)hint.textContent=fresh.dataset.scheduleHint||'';container.dataset.scheduleHint=fresh.dataset.scheduleHint||'';const html=fresh.innerHTML;if(html===previousHTML&&container.dataset.day===fresh.dataset.day)return;previousHTML=html;const open=new Set([...container.querySelectorAll('details[open]')].map(n=>n.dataset.project));container.replaceChildren(...[...fresh.childNodes].map(n=>document.importNode(n,true)));container.dataset.day=fresh.dataset.day;items=[...container.querySelectorAll(selector)];items.forEach(n=>{if(open.has(n.dataset.project))n.open=true});draw()}catch(_){}finally{pending=false;setTimeout(refresh,5000)}}setTimeout(refresh,5000);document.addEventListener('visibilitychange',()=>{if(!document.hidden)refresh()})}
 }
 paginate({prefix:'order',selector:'.project-row',filterAttribute:'data-order-filter',typeAttribute:'data-status',size:8});
 paginate({prefix:'event',selector:'.event-row',filterAttribute:'data-event-filter',typeAttribute:'data-type',size:10});
})();
