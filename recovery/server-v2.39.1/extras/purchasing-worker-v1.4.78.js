(()=>{'use strict';const cfg=document.getElementById('purchasing-worker-config');if(!cfg)return;
const el=(tag,text)=>{const n=document.createElement(tag);if(text)n.textContent=text;return n};
const dialog=el('dialog');dialog.className='purchasing-worker-dialog';dialog.setAttribute('aria-labelledby','purchasing-worker-title');
const head=el('div');head.className='purchasing-worker-head';const title=el('h2','구매팀 호출');title.id='purchasing-worker-title';const close=el('button','✕');close.type='button';close.className='quiet';close.setAttribute('aria-label','닫기');head.append(title,close);
const hint=el('p','구매 담당자에게만 알림이 전달됩니다.');hint.className='muted';
const info=el('div');info.className='purchasing-worker-info';const product=el('strong'),order=el('small'),recipient=el('small','받는 사람  구매팀 · '+cfg.dataset.recipient);info.append(product,order,recipient);
const form=el('form');form.method='post';const label=el('label','작업자 이름'),name=el('input');name.name='worker';name.required=true;name.maxLength=40;name.autocomplete='name';name.placeholder='이름을 입력해 주세요';label.append(name);
const error=el('p');error.className='error';error.setAttribute('role','alert');const send=el('button','구매팀에 호출 보내기');send.type='submit';send.className='primary full';form.append(label,error,send);dialog.append(head,hint,info,form);document.body.append(dialog);
const notice=el('p');notice.className='purchasing-worker-notice';notice.setAttribute('role','status');notice.hidden=true;document.body.append(notice);
let project='',key='',sending=false,opener=null,timer;
function nonce(){const a=new Uint8Array(32);crypto.getRandomValues(a);return btoa(String.fromCharCode(...a)).replace(/\+/g,'-').replace(/\//g,'_').replace(/=+$/,'')}
function cleanup(){form.reset();delete form.dataset.sending;opener?.focus({preventScroll:true})}
close.addEventListener('click',()=>{if(!sending)dialog.close()});dialog.addEventListener('cancel',e=>{if(sending)e.preventDefault()});dialog.addEventListener('close',cleanup);
function install(){for(const card of document.querySelectorAll('.work-item[data-project]:not([data-purchasing-ready])')){
 card.dataset.purchasingReady='true';const button=el('button','구매팀 호출');button.type='button';button.className='purchasing-call-button';button.addEventListener('click',e=>{
  e.stopPropagation();if(sending)return;project=card.dataset.project;key=nonce();product.textContent=card.dataset.name;order.textContent='수주 '+card.dataset.order;error.textContent='';
  try{name.value=localStorage.getItem('alpha-purchasing-worker-name-v1')||''}catch(_){}opener=button;dialog.showModal();name.focus()
 });card.append(button)
}}
if(document.readyState==='loading'||document.readyState==='interactive')document.addEventListener('DOMContentLoaded',install,{once:true});else install();
new MutationObserver(install).observe(document.querySelector('.work-list')||document.body,{childList:true,subtree:true});
form.addEventListener('submit',async e=>{e.preventDefault();if(sending)return;if(!name.value.trim()){error.textContent='작업자 이름을 입력해 주세요.';return}
 sending=true;form.dataset.sending='1';send.disabled=close.disabled=true;send.textContent='호출 보내는 중…';error.textContent='';
 try{const body=new URLSearchParams({csrf:cfg.dataset.board,worker:name.value.trim(),key});const r=await fetch('/purchasing-call/'+encodeURIComponent(cfg.dataset.board)+'/'+encodeURIComponent(project),{method:'POST',body,headers:{Accept:'application/json'}});
 if(!r.ok){const text=await r.text();throw Error(r.status===400&&text.length<600?text:'호출을 보내지 못했습니다. 연결을 확인하고 다시 시도해 주세요.')}
 const result=await r.json();if(result.ok!==true)throw Error('호출 결과를 확인하지 못했습니다. 다시 시도해 주세요.');
 try{localStorage.setItem('alpha-purchasing-worker-name-v1',name.value.trim())}catch(_){}
 sending=false;form.reset();delete form.dataset.sending;dialog.close();notice.textContent=result.duplicate?'이미 접수된 호출이 있습니다. 중복으로 보내지 않았습니다.':'구매팀에 호출을 보냈습니다.';notice.hidden=false;clearTimeout(timer);timer=setTimeout(()=>notice.hidden=true,4500)
 }catch(err){error.textContent=err.message||'연결이 끊겼습니다. 다시 시도해 주세요.'}
 finally{sending=false;delete form.dataset.sending;send.disabled=close.disabled=false;send.textContent='구매팀에 호출 보내기'}
});
})();
