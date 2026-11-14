(()=>{
 'use strict';
 const trigger=document.getElementById('shipping-calendar-open'),dialog=document.getElementById('shipping-calendar'),list=document.querySelector('.work-list')||document.getElementById('order-items');
 if(!trigger||!dialog||!list)return;
 const admin=list.id==='order-items',rowSelector=admin?'.admin-project[data-project]:not([data-status="hidden"])':'.work-item[data-project],.delivery-completed[data-project]',business=document.getElementById(admin?'admin-business-date':'business-date');
 const days=dialog.querySelector('#calendar-days'),monthLabel=dialog.querySelector('#calendar-month'),dateLabel=dialog.querySelector('#calendar-selected-date'),total=dialog.querySelector('#calendar-selected-count'),orders=dialog.querySelector('#calendar-orders'),undatedButton=dialog.querySelector('#calendar-undated');
 const dateFormatter=new Intl.DateTimeFormat('sv-SE',{timeZone:'Asia/Seoul',year:'numeric',month:'2-digit',day:'2-digit'}),weekdays=['일','월','화','수','목','금','토'],collator=new Intl.Collator('ko-KR',{numeric:true,sensitivity:'base'});
 const validDay=s=>/^\d{4}-\d{2}-\d{2}$/.test(s||'')&&!Number.isNaN(Date.parse(s+'T00:00:00Z'))&&new Date(s+'T00:00:00Z').toISOString().slice(0,10)===s;
 const today=()=>{const server=business?.textContent.trim();return validDay(server)?server:dateFormatter.format(new Date())};
 const add=(day,n)=>{const d=new Date(day+'T00:00:00Z');d.setUTCDate(d.getUTCDate()+n);return d.toISOString().slice(0,10)};
 let month='',selected='',showUndated=false,initialized=false,queued=false,lastRender='';
 const node=(tag,cls,text)=>{const n=document.createElement(tag);if(cls)n.className=cls;if(text!==undefined)n.textContent=text;return n};
 function rows(){return [...list.querySelectorAll(rowSelector)].map(card=>({id:card.dataset.project,material:card.dataset.suppliedMaterials||'',materialColor:card.dataset.suppliedMaterialsColor||'',designerName:card.dataset.designerName||'',designerCall:card.dataset.designerCall||'',settingWorker:card.dataset.settingWorker||'',shippingWorker:card.dataset.shippingWorker||'',number:card.dataset.order||'',name:card.dataset.name||'',due:validDay(card.dataset.due)?card.dataset.due:'',stage:card.dataset.status==='done'?'done':card.dataset.stage||'setting',label:card.dataset.status==='done'?'✓ 검사 완료':(admin?(card.dataset.stage==='shipping'?'출하검사 대기':'세팅검사 대기'):(card.querySelector('.stage-chip')?.textContent||'검사 대기')).trim().replace(/^0[12]\s*/,''),card})).sort((a,b)=>collator.compare(a.number,b.number)||collator.compare(a.name,b.name))}
 function select(day){selected=day;month=day.slice(0,7);showUndated=false;render()}
 function monthMove(offset){const d=new Date(month+'-01T00:00:00Z');d.setUTCMonth(d.getUTCMonth()+offset);const m=d.toISOString().slice(0,7);if(m<'1900-01'||m>'2200-12')return;month=m;selected=rows().map(r=>r.due).filter(d=>d.startsWith(month)).sort()[0]||month+'-01';showUndated=false;render()}
 function openWork(id){
  const card=[...list.querySelectorAll(rowSelector)].find(n=>n.dataset.project===id);
  if(!card){render();return}
  close();const search=document.getElementById('project-search');if(search){search.value='';search.dispatchEvent(new Event('input',{bubbles:true}))}document.querySelector('[data-filter="all"]')?.click();
  card.dispatchEvent(new CustomEvent('alpha-open-inspection',{bubbles:true}));
  requestAnimationFrame(()=>{card.scrollIntoView({behavior:matchMedia('(prefers-reduced-motion: reduce)').matches?'auto':'smooth',block:'start'});const target=card.querySelector('.inspection-row input[name="worker"]:not(:disabled)')||card.querySelector('summary');target?.focus({preventScroll:true})});
 }
 function render(){
  if(!dialog.open)return;
  const data=rows(),signature=JSON.stringify([month,selected,showUndated,today(),data.map(r=>[r.id,r.number,r.name,r.due,r.stage,r.label,r.settingWorker,r.shippingWorker,r.designerName,r.designerCall,r.material,r.materialColor])]);if(signature===lastRender)return;lastRender=signature;const focused=document.activeElement,focusedDay=days.contains(focused)?focused.dataset.day:null,focusedProject=orders.contains(focused)?focused.dataset.project:null;const byDay=new Map();for(const row of data){if(!row.due)continue;if(!byDay.has(row.due))byDay.set(row.due,[]);byDay.get(row.due).push(row)}
  const current=today(),[year,mon]=month.split('-').map(Number),first=month+'-01',offset=new Date(first+'T00:00:00Z').getUTCDay(),end=new Date(Date.UTC(year,mon,0)).getUTCDate(),cellCount=Math.ceil((offset+end)/7)*7;
  monthLabel.textContent=`${year}년 ${mon}월`;dialog.querySelector('#calendar-prev').disabled=month==='1900-01';dialog.querySelector('#calendar-next').disabled=month==='2200-12';
  dialog.querySelector('#calendar-month-summary').textContent=`${mon}월 출고 예정 ${data.filter(r=>r.due.startsWith(month)).length}건`;
  const fragment=document.createDocumentFragment();let tr;
  for(let index=0;index<cellCount;index++){
   if(index%7===0){tr=node('tr');fragment.append(tr)}const td=node('td');tr.append(td);const day=index-offset+1;if(day<1||day>end){td.className='calendar-gap';continue}
   const iso=month+'-'+String(day).padStart(2,'0'),count=byDay.get(iso)?.length||0,b=node('button','calendar-day'+(iso===current?' is-today':'')+(!showUndated&&iso===selected?' is-selected':'')+(count?' has-orders':''));b.type='button';b.dataset.day=iso;b.setAttribute('aria-label',`${year}년 ${mon}월 ${day}일${iso===current?', 오늘':''}, 출고 예정 ${count}건`);b.setAttribute('aria-pressed',String(!showUndated&&iso===selected));if(iso===current)b.setAttribute('aria-current','date');b.tabIndex=(!showUndated&&iso===selected)||showUndated&&day===1?0:-1;
   b.append(node('span','calendar-day-number',day));if(count){const pill=node('span','calendar-day-count');pill.append(node('span','','출고 예정'),node('b','',count+'건'));b.append(pill)}
   b.addEventListener('click',()=>select(iso));b.addEventListener('keydown',e=>{const delta={ArrowLeft:-1,ArrowRight:1,ArrowUp:-7,ArrowDown:7}[e.key];if(!delta)return;e.preventDefault();const next=add(iso,delta);if(next<'1900-01-01'||next>'2200-12-31')return;select(next);days.querySelector(`[data-day="${next}"]`)?.focus()});td.append(b);
  }
  days.replaceChildren(fragment);
  const matching=data.filter(r=>showUndated?!r.due:r.due===selected);
  const d=new Date(selected+'T00:00:00Z');dateLabel.textContent=showUndated?'출고 예정일 미정':`${d.getUTCMonth()+1}월 ${d.getUTCDate()}일 ${weekdays[d.getUTCDay()]}요일`;
  total.textContent=showUndated?`날짜 미정 ${matching.length}건`:`출고 예정 ${matching.length}건`;
  orders.replaceChildren();for(const row of matching){
   const done=row.stage==='done',item=node('article','calendar-order'+(done?' is-complete':'')),info=node('div','calendar-order-info'),meta=node('div','calendar-order-meta');
   meta.append(node('span','calendar-order-stage '+row.stage,row.label));
   if(row.designerName){const callable=/^tel:[+0-9() .-]+$/.test(row.designerCall),designer=node(callable?'a':'span','designer-call calendar-designer','☎ '+row.designerName);if(callable){designer.href=row.designerCall;designer.setAttribute('aria-label','설계담당자 '+row.designerName+'에게 전화');designer.addEventListener('click',e=>e.stopPropagation())}meta.append(designer)}
   if(row.material){const b=node('button','supplied-materials');b.type='button';b.dataset.materialColor=row.materialColor;b.setAttribute('aria-haspopup','dialog');b.append(node('span','supplied-materials-text',row.material));meta.append(b)}
   const special=row.card.querySelector('.special-info-link');
   if(special){const link=node('a','special-info-link calendar-special',special.textContent.trim());link.href=special.getAttribute('href');meta.append(link)}
   info.append(node('span','calendar-order-number',row.number),node('h4','',row.name),meta);if(done){const workers=node('div','completed-workers');if(row.settingWorker)workers.append(node('p','','세팅 작업자 · '+row.settingWorker));if(row.shippingWorker)workers.append(node('p','','출하 작업자 · '+row.shippingWorker));info.append(workers)}item.append(info);
   if(admin||!done){const action=admin?'검사 화면 열기':'작업 열기',button=node(admin?'a':'button','calendar-open-work',action);if(admin)button.href='/projects/'+encodeURIComponent(row.id)+'/inspection';else{button.type='button';button.addEventListener('click',()=>openWork(row.id))}button.dataset.project=row.id;button.setAttribute('aria-label',`${row.number} ${row.name} ${action}`);item.append(button)}
   orders.append(item)
  }
  if(!matching.length)orders.append(node('p','calendar-empty',showUndated?'출고 예정일이 미정인 수주가 없습니다.':'선택한 날짜에 출고 예정인 수주가 없습니다.'));
  undatedButton.querySelector('span').textContent=data.filter(r=>!r.due).length+'건';undatedButton.setAttribute('aria-pressed',String(showUndated));if(focusedDay)days.querySelector(`[data-day="${focusedDay}"]`)?.focus({preventScroll:true});else if(focusedProject)orders.querySelector(`[data-project="${focusedProject}"]`)?.focus({preventScroll:true});
 }
 function close(){if(dialog.dataset.inline==='true')document.dispatchEvent(new CustomEvent('alpha-inline-calendar-close'));if(dialog.open){if(typeof dialog.close==='function')dialog.close();else dialog.removeAttribute('open')}if(dialog.dataset.inline!=='true')trigger.focus({preventScroll:true})}
 trigger.addEventListener('click',()=>{if(!initialized){const current=today();month=current.slice(0,7);selected=rows().map(r=>r.due).filter(d=>d>=current&&d.startsWith(month)).sort()[0]||current;initialized=true}if(document.documentElement.dataset.alphaDesign==='modern'){dialog.dataset.inline='true';dialog.setAttribute('role','region');if(!dialog.open)dialog.show()}else{dialog.removeAttribute('data-inline');dialog.removeAttribute('role');if(typeof dialog.showModal==='function')dialog.showModal();else dialog.setAttribute('open','')}render();dialog.scrollTop=0});
 dialog.querySelector('.calendar-close').addEventListener('click',close);dialog.addEventListener('click',e=>{if(dialog.dataset.inline==='true')return;const r=dialog.getBoundingClientRect();if(e.target===dialog&&(e.clientX<r.left||e.clientX>r.right||e.clientY<r.top||e.clientY>r.bottom))close()});
 dialog.querySelector('#calendar-prev').addEventListener('click',()=>monthMove(-1));dialog.querySelector('#calendar-next').addEventListener('click',()=>monthMove(1));dialog.querySelector('#calendar-today').addEventListener('click',()=>select(today()));undatedButton.addEventListener('click',()=>{showUndated=true;render()});
 function refresh(){if(!dialog.open||queued)return;queued=true;requestAnimationFrame(()=>{queued=false;render()})}
 new MutationObserver(refresh).observe(list,{subtree:true,childList:true,attributes:true,attributeFilter:['data-due','data-name','data-order','data-stage','data-status','data-revision','data-designer-name','data-designer-call','data-supplied-materials','data-supplied-materials-color']});if(business)new MutationObserver(refresh).observe(business,{childList:true,characterData:true,subtree:true});
})();
