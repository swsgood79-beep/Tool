(()=>{
 'use strict';
 const list=document.querySelector('.work-list');if(!list)return;
 function showInspection(card){
  const dialog=card?.querySelector('.worker-inspection-dialog');if(!dialog)return;
  for(const old of document.querySelectorAll('.worker-inspection-dialog[open]'))if(old!==dialog)old.close();
  card.querySelector('details').open=true;
  card.hidden=false;
  if(!dialog.open)dialog.showModal();
  dialog.scrollTop=0;
 }
 let review=null;
 function currentCard(id){return [...list.querySelectorAll('.work-item')].find(card=>card.dataset.project===id)}
 function open(card, changed=false){
  if(review)review.close();
  const source=card?.querySelector('template.worker-special-review');
  if(!source){showInspection(card);return}
  // A fresh acknowledgement is required on every deliberate opening, including
  // calendar entries. A status poll resumes an already open inspection separately.
  const gate=source.content.querySelector('dialog').cloneNode(true);
  const id=card.dataset.project,revision=gate.dataset.infoRevision,generation=card.dataset.generation;
  const check=gate.querySelector('input'),proceed=gate.querySelector('.special-review-proceed');
  if(changed)gate.querySelector('.special-review-status').textContent='특이사항이 변경되었습니다. 내용을 다시 확인해 주세요.';
  check.addEventListener('change',()=>{proceed.disabled=!check.checked});
  gate.querySelector('.special-review-cancel').addEventListener('click',()=>gate.close());
  proceed.addEventListener('click',()=>{
   if(!check.checked)return;
   const fresh=currentCard(id);
   if(!fresh||fresh.dataset.generation!==generation){
    gate.querySelector('.special-review-status').textContent='수주 상태가 변경되었습니다. 창을 닫고 목록에서 다시 선택해 주세요.';
    proceed.disabled=true;return;
   }
   const updated=fresh.querySelector('template.worker-special-review');
   if(updated&&updated.content.querySelector('dialog').dataset.infoRevision!==revision){open(fresh,true);return}
   gate.close();showInspection(fresh);
  });
  gate.addEventListener('close',()=>{if(review===gate)review=null;gate.remove();if(!review&&!document.querySelector('.worker-inspection-dialog[open]'))currentCard(id)?.querySelector('summary')?.focus({preventScroll:true})},{once:true});
  document.body.append(gate);review=gate;gate.showModal();
 }
 window.AlphaWorkDialog={open,resume:showInspection};
 list.addEventListener('click',event=>{
  const close=event.target.closest('.worker-dialog-close');
  if(close){close.closest('dialog').close();return}
  const summary=event.target.closest('summary.project-summary');
  if(summary){event.preventDefault();open(summary.closest('.work-item'));return}
  const dialog=event.target;
  if(dialog.matches('.worker-inspection-dialog')){
   const r=dialog.getBoundingClientRect();
   if(event.clientX<r.left||event.clientX>r.right||event.clientY<r.top||event.clientY>r.bottom)dialog.close();
  }
 });
 list.addEventListener('alpha-open-inspection',event=>open(event.target.closest('.work-item')));
 // Inspection requests stay in the dialog. Notes retain their existing,
 // independent receipt flow and are deliberately not intercepted here.
 list.addEventListener('submit',async event=>{
  const form=event.target;
  if(!form.matches('.inspection-row form'))return;
  event.preventDefault();if(form.dataset.sending)return;
  const card=form.closest('.work-item'),dialog=form.closest('dialog'),status=dialog.querySelector('.worker-submit-status');
  const data=new URLSearchParams(new FormData(form)),endpoint=event.submitter?.hasAttribute('formaction')?event.submitter.formAction:form.action;
  const buttons=[...form.querySelectorAll('button')];
  form.dataset.sending='true';buttons.forEach(b=>b.disabled=true);status.textContent='검사 신청을 보내고 있습니다…';
  try{
   const response=await fetch(endpoint,{method:'POST',body:data,credentials:'same-origin'});
   if(!response.ok)throw Error('request not accepted');
   await window.AlphaWorkerView.refreshCard({id:card.dataset.project,revision:-1});
   const fresh=list.querySelector(`[data-project="${card.dataset.project}"] .worker-submit-status`);
   if(fresh)fresh.textContent='신청이 접수되었습니다. 관리자 확인을 기다려 주세요.';
  }catch(_){status.textContent='신청 결과를 확인하지 못했습니다. 잠시 후 상태를 확인하거나 다시 시도해 주세요.'}
  finally{delete form.dataset.sending;buttons.forEach(b=>b.disabled=false)}
 });
 list.addEventListener('close',event=>{
  if(!event.target.matches('.worker-inspection-dialog'))return;
  const card=event.target.closest('.work-item');if(!card?.isConnected)return;
  card.querySelector('details').open=false;
  document.getElementById('project-search').dispatchEvent(new Event('input',{bubbles:true}));
  card.querySelector('summary')?.focus({preventScroll:true});
 },true);
})();
