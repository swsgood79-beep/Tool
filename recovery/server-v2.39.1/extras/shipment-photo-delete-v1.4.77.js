(()=>{'use strict';
const forms=[...document.querySelectorAll('form.shipment-photo-delete')];if(!forms.length)return;
const dialog=document.createElement('dialog');dialog.className='shipment-delete-dialog';dialog.setAttribute('aria-labelledby','shipment-delete-title');
const title=document.createElement('h2');title.id='shipment-delete-title';title.textContent='이 사진을 삭제할까요?';
const image=document.createElement('img');image.alt='삭제할 출하사진';
const text=document.createElement('p'),error=document.createElement('p');error.className='error';error.setAttribute('role','alert');
const actions=document.createElement('div');actions.className='shipment-delete-actions';
const cancel=document.createElement('button');cancel.type='button';cancel.className='quiet';cancel.textContent='취소';
const remove=document.createElement('button');remove.type='button';remove.className='danger';remove.textContent='이 사진 삭제';
actions.append(cancel,remove);dialog.append(title,image,text,error,actions);document.body.append(dialog);
let selected=null,sending=false,opener=null;
function close(){if(!sending)dialog.close()}
cancel.addEventListener('click',close);dialog.addEventListener('cancel',e=>{if(sending)e.preventDefault()});
dialog.addEventListener('close',()=>{opener?.focus({preventScroll:true});selected=null});
for(const form of forms){const button=form.querySelector('button');button.addEventListener('click',()=>{
 const upload=document.getElementById('shipment-photo-form');
 if(upload?.dataset.sending||document.querySelector('#shipment-photo-preview figure')){
  const notice=document.getElementById('shipment-photo-error');if(notice){notice.textContent='추가할 사진을 먼저 저장하거나 사진 추가를 취소한 뒤 삭제해 주세요.';notice.hidden=false;notice.scrollIntoView({block:'center'})}return
 }
 if(sending)return;selected=form;opener=button;error.textContent='';remove.disabled=cancel.disabled=false;
 image.src=form.closest('figure').querySelector('img').src;
 text.textContent=button.dataset.last==='true'?(button.dataset.reset==='true'?'마지막 사진입니다. 삭제하면 출하사진 필요 목록으로 돌아갑니다. 검사 완료 상태는 유지됩니다.':'마지막 사진입니다. 현재 검사 완료 수주와 연결되지 않은 보관 자료이므로 기록도 함께 삭제됩니다.'):'선택한 사진만 삭제합니다. 나머지 사진과 보관 날짜는 유지됩니다.';
 dialog.showModal();cancel.focus()
})}
remove.addEventListener('click',async()=>{
 if(sending||!selected)return;const form=selected;sending=true;form.dataset.sending='1';remove.disabled=cancel.disabled=true;remove.textContent='삭제 중…';error.textContent='';
 try{
  const response=await fetch(form.action,{method:'POST',body:new URLSearchParams(new FormData(form)),credentials:'same-origin',headers:{Accept:'application/json'}});
  if(!response.ok){const message=await response.text();throw Error(response.status===400&&message.length<600?message.trim():'사진을 삭제하지 못했습니다. 로그인·연결 상태를 확인해 주세요.')}
  const result=await response.json(),next=new URL(result.next,location.href);
  if(result.ok!==true||next.origin!==location.origin||!/^\/shipping-archive-admin(?:\/[A-Za-z0-9_-]+)?$/.test(next.pathname)||next.search!=='?photo_deleted=1')throw Error('삭제 결과를 확인하지 못했습니다. 새로고침해서 사진 목록을 확인해 주세요.');
  delete form.dataset.sending;sending=false;form.reset();dialog.close();location.assign(next.href)
 }catch(e){delete form.dataset.sending;sending=false;remove.disabled=cancel.disabled=false;remove.textContent='이 사진 삭제';error.textContent=e.message||'연결이 끊겼습니다. 다시 시도해 주세요.'}
});
})();
