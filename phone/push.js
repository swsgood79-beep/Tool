(()=>{
 'use strict';
 const status=document.getElementById('status'),button=document.getElementById('connect');
 const params=new URLSearchParams(location.hash.slice(1));
 const endpoint=params.get('endpoint'),token=params.get('token'),key=params.get('key');
 const validKey=s=>/^[A-Za-z0-9_-]{43}$/.test(s||'');
 function validOrigin(raw){try{const u=new URL(raw);return u.protocol==='https:'&&!u.username&&!u.password&&!u.port&&u.pathname==='/'&&!u.search&&!u.hash&&/^[a-z0-9]+(?:-[a-z0-9]+)*\.trycloudflare\.com$/.test(u.hostname)}catch(_){return false}}
 const pairing=validOrigin(endpoint)&&validKey(token)&&/^[A-Za-z0-9_-]{87}$/.test(key||'');
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
   history.replaceState(null,'',location.pathname);
   status.textContent='휴대폰 연결 완료!\nPC 화면을 새로고침하고 ‘휴대폰 테스트 알림 보내기’를 눌러 주세요. 브라우저 화면을 닫은 상태에서도 수신되는지 확인하세요.';
   button.hidden=true;
  }catch(err){status.textContent=(err instanceof TypeError||err.name==='AbortError')?'PC에 연결하지 못했습니다. PC 인터넷 연결을 확인하고 새 연결용 QR을 만든 뒤 다시 시도하세요.':err.message;button.disabled=false}
 });
 function openDB(){return new Promise((resolve,reject)=>{const req=indexedDB.open('alpha-phone-messages',1);req.onupgradeneeded=()=>req.result.createObjectStore('messages',{keyPath:'tag'});req.onsuccess=()=>resolve(req.result);req.onerror=()=>reject(req.error)})}
 async function messages(){
  const box=document.getElementById('messages');
  try{const db=await openDB();const entries=await new Promise((resolve,reject)=>{const req=db.transaction('messages').objectStore('messages').getAll();req.onsuccess=()=>resolve(req.result);req.onerror=()=>reject(req.error)});db.close();box.replaceChildren();entries.sort((a,b)=>b.received-a.received);
   for(const message of entries){const article=document.createElement('article');article.className='message';const title=document.createElement('h3');title.textContent=message.title;const body=document.createElement('p');body.textContent=message.body;const time=document.createElement('time');time.textContent=new Date(message.received).toLocaleString('ko-KR');article.append(title,body,time);box.append(article)}
   if(!entries.length){const p=document.createElement('p');p.textContent='아직 받은 알림이 없습니다.';box.append(p)}
  }catch(_){box.textContent='이 브라우저에서 받은 메시지 보관함을 열지 못했습니다.'}
 }
 document.getElementById('refresh').addEventListener('click',messages);
 if('serviceWorker'in navigator)navigator.serviceWorker.addEventListener('message',messages);
 document.addEventListener('visibilitychange',()=>{if(document.visibilityState==='visible')messages()});messages();
})();
