(()=>{'use strict';for(const form of document.querySelectorAll('[data-shipping-export]')){
 const button=form.querySelector('button'),message=form.querySelector('[role=status]'),download=form.querySelector('[data-export-download]'),scope=form.querySelector('[name=record]')?.value||'',storeKey='alpha-shipping-export-v1:'+scope;let key='',job='',busy=false,timer;
 const nonce=()=>{const bytes=new Uint8Array(32);crypto.getRandomValues(bytes);return btoa(String.fromCharCode(...bytes)).replace(/\+/g,'-').replace(/\//g,'_').replace(/=+$/,'')};
 const remember=()=>{try{sessionStorage.setItem(storeKey,JSON.stringify({key,job}))}catch(_){}};
 function fail(text){busy=false;button.disabled=false;message.textContent=text;message.classList.add('error');try{sessionStorage.removeItem(storeKey)}catch(_){}key=job=''}
 function render(data,automatic){
  if(!data||!['building','ready','failed'].includes(data.state)||data.id!==job){fail('내보내기 결과를 확인하지 못했습니다. 다시 시도해 주세요.');return}
  if(data.state==='failed'){fail(data.message||'엑셀을 만들지 못했습니다. 다시 시도해 주세요.');return}
  if(data.state==='building'){message.textContent='엑셀 준비 중 · '+data.processed+' / '+data.total+'건';timer=setTimeout(()=>poll(automatic),1500);return}
  busy=false;button.disabled=false;const url=new URL(form.action);url.searchParams.set('job',job);url.searchParams.set('download','1');download.href=url.href;download.hidden=false;download.textContent='준비된 파일 다운로드';
  message.textContent='준비 완료 · '+data.total+'건 · '+(data.bytes/(1024*1024)).toFixed(1)+' MB'+(data.warnings?' · 사진 확인 필요 '+data.warnings+'개':'');
  if(automatic){const frame=document.createElement('iframe');frame.hidden=true;frame.title='엑셀 다운로드';frame.src=url.href;document.body.append(frame);setTimeout(()=>frame.remove(),60000);message.textContent+=' · 다운로드를 요청했습니다. 시작되지 않으면 아래 링크를 눌러 주세요.'}
 }
 async function poll(automatic){try{const url=new URL(form.action);url.searchParams.set('job',job);const response=await fetch(url,{cache:'no-store'});if(!response.ok){fail(response.status===404?'이전 내보내기 파일이 만료되었습니다. 다시 저장해 주세요.':'로그인·연결 상태를 확인한 뒤 다시 시도해 주세요.');return}render(await response.json(),automatic)}catch(_){message.textContent='연결을 다시 확인하고 있습니다…';timer=setTimeout(()=>poll(automatic),3000)}}
 form.addEventListener('submit',async e=>{e.preventDefault();if(busy)return;clearTimeout(timer);busy=true;button.disabled=true;download.hidden=true;message.classList.remove('error');message.textContent='보관 자료를 확인하고 있습니다…';key=nonce();job='';form.dataset.sending='1';
  try{const body=new FormData(form);body.set('export_key',key);const response=await fetch(form.action,{method:'POST',body:new URLSearchParams(body),headers:{Accept:'application/json'}});if(!response.ok){const text=await response.text();throw Error(text.length<500?text.trim():'내보내기 요청에 실패했습니다.')};const data=await response.json();job=data.id;remember();form.reset();delete form.dataset.sending;render(data,true)}catch(err){fail(err.message||'내보내기를 시작하지 못했습니다. 다시 시도해 주세요.')}finally{delete form.dataset.sending}
 });
 try{const saved=JSON.parse(sessionStorage.getItem(storeKey)||'null');if(saved&&/^[A-Za-z0-9_-]{43}$/.test(saved.job)){job=saved.job;key=saved.key;busy=true;button.disabled=true;poll(false)}}catch(_){}
 }} )();
