(()=>{
 'use strict';
 const status=document.getElementById('status'),button=document.getElementById('connect');
 const params=new URLSearchParams(location.hash.slice(1));
 const endpoint=params.get('endpoint'),token=params.get('token'),key=params.get('key');
 const validKey=s=>/^[A-Za-z0-9_-]{43}$/.test(s||'');
 function validOrigin(raw){try{const u=new URL(raw);return u.protocol==='https:'&&!u.username&&!u.password&&!u.port&&u.pathname==='/'&&!u.search&&!u.hash&&/^[a-z0-9]+(?:-[a-z0-9]+)*\.trycloudflare\.com$/.test(u.hostname)}catch(_){return false}}
 const pairing=validOrigin(endpoint)&&validKey(token)&&/^[A-Za-z0-9_-]{87}$/.test(key||'');
 let enrolling=pairing;
 const supported=isSecureContext&&'serviceWorker'in navigator&&'PushManager'in window&&'Notification'in window;
 const bytes=s=>Uint8Array.from(atob(s.replace(/-/g,'+').replace(/_/g,'/')+'='.repeat((4-s.length%4)%4)),c=>c.charCodeAt(0));
 const same=(a,b)=>a&&a.length===b.length&&a.every((v,i)=>v===b[i]);
 if(!supported)status.textContent='이 브라우저는 웹 알림을 지원하지 않습니다. 최신 삼성 인터넷 또는 Chrome의 일반 모드에서 열어 주세요.';
 else if(pairing){status.textContent='아래 버튼을 누르고 알림을 허용하면 이 휴대폰을 회사 PC에 연결합니다. 연결 QR은 10분 동안 한 번 사용할 수 있습니다.';button.disabled=false}
 else{status.textContent='이미 연결했다면 이 화면을 닫아도 알림을 받을 수 있습니다. 처음 연결하거나 다시 연결하려면 PC의 관리자 화면 → 내 휴대폰 알림에서 연결용 QR을 스캔하세요.';button.hidden=true}
 button.addEventListener('click',async()=>{
  if(!pairing||!supported)return;button.disabled=true;
  try{
   const permission=await Notification.requestPermission();
   if(permission!=='granted')throw new Error('알림이 허용되지 않았습니다. 삼성 인터넷 설정 → 사이트 및 다운로드 → 알림과 휴대폰 앱 알림 설정을 확인하세요.');
   status.textContent='알림 등록 중입니다. 잠시 기다려 주세요.';
   await navigator.serviceWorker.register('sw.js',{scope:'./',updateViaCache:'none'});
   const registration=await navigator.serviceWorker.ready;
   const publicKey=bytes(key);let subscription=await registration.pushManager.getSubscription();
   if(subscription&&!same(subscription.options.applicationServerKey&&new Uint8Array(subscription.options.applicationServerKey),publicKey)){await subscription.unsubscribe();subscription=null}
   if(!subscription)subscription=await registration.pushManager.subscribe({userVisibleOnly:true,applicationServerKey:publicKey});
   const controller=new AbortController(),timeout=setTimeout(()=>controller.abort(),20000);
   let response;try{response=await fetch(`${endpoint}/phone/register`,{method:'POST',credentials:'omit',headers:{'Content-Type':'application/json','Authorization':`Bearer ${token}`},body:JSON.stringify(subscription.toJSON()),signal:controller.signal})}finally{clearTimeout(timeout)}
   if(!response.ok){const text=await response.text();throw new Error(text.slice(0,250)||'PC에 연결하지 못했습니다.')}
   history.replaceState(null,'',location.pathname);enrolling=false;
   status.textContent='휴대폰 연결 완료!\nPC 화면을 새로고침하고 ‘휴대폰 테스트 알림 보내기’를 눌러 주세요. 브라우저 화면을 닫은 상태에서도 수신되는지 확인하세요.';
   button.hidden=true;messages();
  }catch(err){status.textContent=(err instanceof TypeError||err.name==='AbortError')?'PC에 연결하지 못했습니다. PC 인터넷 연결을 확인하고 새 연결용 QR을 만든 뒤 다시 시도하세요.':err.message;button.disabled=false}
 });
 function openDB(){return new Promise((resolve,reject)=>{const req=indexedDB.open('alpha-phone-messages',1);req.onupgradeneeded=()=>req.result.createObjectStore('messages',{keyPath:'tag'});req.onsuccess=()=>resolve(req.result);req.onerror=()=>reject(req.error)})}
 const setup=document.getElementById('setup'),detail=document.getElementById('detail');
 const transient=new Map();let newestRequest=0;
 function selectedTag(){return new URLSearchParams(location.hash.slice(1)).get('message')||''}
 function renderDetail(message){
  setup.hidden=true;detail.hidden=false;document.getElementById('received').hidden=true;
  document.getElementById('detail-title').textContent=message.title||'작업 소식';
  document.getElementById('detail-body').textContent=message.body||'내용이 없는 알림입니다.';
  const review=document.getElementById('review-order'),request=Number.isSafeInteger(message.requestId)&&message.requestId>0?message.requestId:Number((/^alpha-([1-9][0-9]*)$/.exec(message.tag||'')||[])[1]);review.hidden=!((message.pending===true||/(?:세팅검사|출하검사) 신청\s*$/.test((message.body||'').split('\n')[0]))&&Number.isSafeInteger(request)&&request>0);if(!review.hidden){const target=new URL('../admin/',location.href);target.hash='';const params=new URLSearchParams({request:String(request),open:'inspection',v:'2.37.1'});if(Number.isSafeInteger(message.projectId)&&message.projectId>0)params.set('project',String(message.projectId));target.search=params;review.href=target.href}else review.removeAttribute('href');
  document.getElementById('detail-time').textContent=message.received?new Date(message.received).toLocaleString('ko-KR'):'';
 }
 async function messages(){
  const request=++newestRequest,box=document.getElementById('messages');let entries=[];
  try{const db=await openDB();entries=await new Promise((resolve,reject)=>{const req=db.transaction('messages').objectStore('messages').getAll();req.onsuccess=()=>resolve(req.result);req.onerror=()=>reject(req.error)});db.close()}
  catch(_){box.textContent='이 브라우저에서 받은 메시지 보관함을 열지 못했습니다.'}
  if(request!==newestRequest)return;
  const byTag=new Map(entries.map(message=>[message.tag,message]));for(const [tag,message]of transient)byTag.set(tag,message);
  entries=[...byTag.values()].sort((a,b)=>b.received-a.received);box.replaceChildren();document.getElementById('received').hidden=false;
  for(const message of entries){
   const article=document.createElement('article');article.className='message';
   const open=document.createElement('button');open.type='button';open.className='message-open';
   const title=document.createElement('span');title.className='message-title';title.textContent=message.body.split('\n')[0]||message.title;
   const preview=document.createElement('span');preview.className='message-preview';preview.textContent=[...message.body].slice(0,90).join('');
   const time=document.createElement('time');time.textContent=new Date(message.received).toLocaleString('ko-KR');
   open.append(title,preview,time);open.addEventListener('click',()=>{renderDetail(message);location.hash=new URLSearchParams({message:message.tag}).toString()});article.append(open);box.append(article);
  }
  const tag=selectedTag();
  if(tag){
   const chosen=byTag.get(tag);
   if(chosen)renderDetail(chosen);
   else renderDetail({title:'알림 내용을 찾지 못했습니다.',body:'이 휴대폰에 저장된 알림이 없거나 최근 100건 보관 범위를 지났습니다. 다른 메시지는 아래 목록에서 확인해 주세요.'});
  }else if(!enrolling&&entries.length){setup.hidden=true;detail.hidden=true}
  else{setup.hidden=false;detail.hidden=true}
  if(!entries.length){const p=document.createElement('p');p.textContent='아직 받은 알림이 없습니다.';box.append(p)}
 }
 document.getElementById('all-messages').addEventListener('click',async()=>{history.replaceState(null,'',location.pathname);await messages();document.getElementById('received').scrollIntoView({block:'start'})});
 document.getElementById('refresh').addEventListener('click',messages);
 window.addEventListener('hashchange',()=>{messages();window.scrollTo(0,0)});
 if('serviceWorker'in navigator){
  navigator.serviceWorker.addEventListener('message',event=>{
   const data=event.data;
   if(data&&data.type==='open-message'&&data.message&&typeof data.message.tag==='string'&&typeof data.message.body==='string'){
    transient.set(data.message.tag,data.message);enrolling=false;
    history.replaceState(null,'',location.pathname+'#'+new URLSearchParams({message:data.message.tag}));window.scrollTo(0,0);
   }
   messages();
  });
  // Refresh the worker even on an already-paired phone, without prompting or replacing its subscription.
  if(supported)navigator.serviceWorker.register('sw.js',{scope:'./',updateViaCache:'none'}).then(registration=>registration.update()).catch(()=>{});
 }
 document.addEventListener('visibilitychange',()=>{if(document.visibilityState==='visible')messages()});messages();
})();
