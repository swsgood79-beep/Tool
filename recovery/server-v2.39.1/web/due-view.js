(() => {
 'use strict';
 const add=(day,n)=>{const d=new Date(day+'T00:00:00Z');d.setUTCDate(d.getUTCDate()+n);return d.toISOString().slice(0,10)};
 const weekday=day=>new Date(day+'T00:00:00Z').getUTCDay();
 const label=day=>{const d=new Date(day+'T00:00:00Z');return `${['일','월','화','수','목','금','토'][d.getUTCDay()]}요일 ${d.getUTCMonth()+1}/${d.getUTCDate()}`};
 const range=day=>{const start=add(day,1);let end=start;while([0,6].includes(weekday(end)))end=add(end,1);return {start,end}};
 const isDue=(value,stage,day)=>{if(!value)return false;if(stage==='due_today')return value===day;const {start,end}=range(day);return value>=start&&value<=end};
 const collator=new Intl.Collator('ko-KR',{numeric:true,sensitivity:'base'});
 const compare=(a,b)=>{const x=a.dataset.due||'',y=b.dataset.due||'';if(x!==y)return !x?1:!y?-1:x<y?-1:1;return collator.compare(a.dataset.order||'',b.dataset.order||'')||collator.compare(a.dataset.name||'',b.dataset.name||'')||Number(a.dataset.project)-Number(b.dataset.project)};
 function order(container,items){
  const focused=document.activeElement,selection=focused&&typeof focused.selectionStart==='number'?[focused.selectionStart,focused.selectionEnd]:null;
  const sorted=[...items].sort(compare),members=new Set(items);
  let cursor=[...container.children].find(n=>members.has(n))||null;
  for(const item of sorted){if(item===cursor){cursor=cursor.nextElementSibling;while(cursor&&!members.has(cursor))cursor=cursor.nextElementSibling}else container.insertBefore(item,cursor)}
  if(focused?.isConnected&&document.activeElement!==focused){focused.focus({preventScroll:true});if(selection)focused.setSelectionRange(...selection)}
  return sorted;
 }
 function paint(panel,items,stage,day,eligible=()=>true){
  if(!panel)return;
  const due=stage==='due_today'||stage==='due_tomorrow';
  const candidates=items.filter(eligible), {start,end}=range(day);
  const upcoming=candidates.filter(n=>isDue(n.dataset.due,'due_tomorrow',day));
  const weekend=upcoming.some(n=>[0,6].includes(weekday(n.dataset.due)));
  const title=panel.querySelector('.delivery-title'),hint=panel.querySelector('.delivery-description');
  if(due){title.textContent=stage==='due_today'?`오늘 ${label(day)}`:weekend?`주말~${label(end)}`:label(end);
   const count=candidates.filter(n=>isDue(n.dataset.due,stage,day)).length;
   title.textContent+=` · ${count}건`;
   hint.textContent=stage==='due_today'?'출고예정일 기준 · 검사 완료 수주 포함':start!==end&&!weekend?'주말 출고 예정 없음 · 다음 평일 일정':'출고가 있는 날짜만 묶어서 표시합니다';
  }else{title.textContent='출고 일정을 확인하세요';hint.textContent=panel.querySelector('.delivery-calendar-open')?'날짜별 출고 예정 수주를 캘린더로 확인하세요':'오늘 납기 또는 다음 출고를 선택하세요'}
 }
 function group(container,items,stage){
  container.querySelectorAll('.due-day-heading').forEach(n=>n.remove());
  items.forEach(n=>{n.classList.toggle('due-weekend',[0,6].includes(weekday(n.dataset.due||'2000-01-03')));});
  if(!['due_today','due_tomorrow'].includes(stage))return;
  const focused=document.activeElement;const selection=focused&&typeof focused.selectionStart==='number'?[focused.selectionStart,focused.selectionEnd]:null;
  const visible=items.filter(n=>!n.hidden).sort(compare);
  // Move only out-of-order nodes; retain focused inputs and open inspection forms.
  let cursor=items.find(n=>!n.hidden)||null;
  for(const item of visible){if(item===cursor){cursor=cursor.nextElementSibling;while(cursor&&!items.includes(cursor))cursor=cursor.nextElementSibling}else container.insertBefore(item,cursor)}
  let last='';
  for(const item of visible){const day=item.dataset.due;if(day===last)continue;last=day;const h=document.createElement('div');h.className='due-day-heading'+([0,6].includes(weekday(day))?' weekend':'');h.textContent=`${label(day)} · ${visible.filter(n=>n.dataset.due===day).length}건`;container.insertBefore(h,item)}
  if(focused?.isConnected&&document.activeElement!==focused){focused.focus({preventScroll:true});if(selection)focused.setSelectionRange(...selection)}
 }
 window.AlphaDue={add,range,label,isDue,paint,group,compare,order};
})();
