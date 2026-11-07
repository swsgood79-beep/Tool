'use strict';
const CACHE='alpha-phone-v4';
const HOME=new URL('./',self.location).href;
const ASSETS=['./','index.html','push.js','phone.css','push.js?v=20261108-queue','phone.css?v=20261108-queue','icon.svg','../style.css'].map(p=>new URL(p,self.location).href);
self.addEventListener('install',event=>event.waitUntil(caches.open(CACHE).then(cache=>cache.addAll(ASSETS.map(url=>new Request(url,{cache:'reload'})))).then(()=>self.skipWaiting())));
self.addEventListener('activate',event=>event.waitUntil(Promise.all([self.clients.claim(),caches.delete('alpha-phone-v1'),caches.delete('alpha-phone-v2'),caches.delete('alpha-phone-v3')])));
// This worker controls /Tool/phone/ only; never caches or intercepts the printed work QR relay.
self.addEventListener('fetch',event=>{
 if(event.request.method!=='GET')return;
 const url=new URL(event.request.url);url.hash='';
 if(!ASSETS.includes(url.href))return;
 event.respondWith(fetch(event.request).then(response=>{if(response.ok){const copy=response.clone();event.waitUntil(caches.open(CACHE).then(c=>c.put(event.request,copy)))}return response}).catch(()=>caches.match(event.request).then(r=>r||Response.error())));
});
function saveMessage(message){return new Promise((resolve,reject)=>{
 const request=indexedDB.open('alpha-phone-messages',1);request.onupgradeneeded=()=>request.result.createObjectStore('messages',{keyPath:'tag'});request.onerror=()=>reject(request.error);
 request.onsuccess=()=>{const db=request.result,tx=db.transaction('messages','readwrite'),store=tx.objectStore('messages');store.put(message);const all=store.getAll();all.onsuccess=()=>{const rows=all.result.sort((a,b)=>b.received-a.received);for(const old of rows.slice(100))store.delete(old.tag)};tx.oncomplete=()=>{db.close();resolve()};tx.onerror=()=>{db.close();reject(tx.error)}};
})}
self.addEventListener('push',event=>event.waitUntil((async()=>{
 let message;try{message=event.data.json()}catch(_){message={}}
 const safe={title:typeof message.title==='string'?message.title.slice(0,100):'알파작업완료확인 프로그램',body:typeof message.body==='string'?message.body.slice(0,4000):'새 작업 소식이 도착했습니다.',tag:typeof message.tag==='string'?message.tag.slice(0,80):'alpha-message',received:Date.now()};
 safe.pending=message.pending_approval===true;safe.projectId=Number.isSafeInteger(message.project_id)&&message.project_id>0?message.project_id:0;safe.requestId=Number.isSafeInteger(message.request_id)&&message.request_id>0?message.request_id:0;
 // Always show a notification, even if the optional local inbox cannot be saved.
 await Promise.all([saveMessage(safe).catch(()=>{}),self.registration.showNotification(safe.title,{actions:safe.pending&&safe.projectId&&safe.requestId?[{action:'review',title:'확인'}]:[],body:safe.body,tag:safe.tag,renotify:true,icon:new URL('icon.svg',HOME).href,badge:new URL('icon.svg',HOME).href,data:{url:HOME+'#'+new URLSearchParams({message:safe.tag}),message:safe}})]);
 const pages=await self.clients.matchAll({type:'window'});for(const page of pages)page.postMessage({type:'message-received'});
})()));
function inspectionTarget(message){
 const id=Number.isSafeInteger(message.requestId)&&message.requestId>0?message.requestId:Number((/^alpha-([1-9][0-9]*)$/.exec(message.tag||'')||[])[1]);
 const pending=message.pending===true||/(?:세팅검사|출하검사) 신청\s*$/.test((message.body||'').split('\n')[0]);
 if(!pending||!Number.isSafeInteger(id)||id<1)return null;
 const review=new URL('../admin/',HOME),params=new URLSearchParams({request:String(id),open:'inspection',v:'2.37.1'});
 if(Number.isSafeInteger(message.projectId)&&message.projectId>0)params.set('project',String(message.projectId));
 review.search=params;return review.href;
}
self.addEventListener('notificationclick',event=>{
 event.notification.close();
 const notice=event.notification,data=notice.data||{},stored=data.message||data;
 const message={title:typeof stored.title==='string'?stored.title:notice.title,body:typeof stored.body==='string'?stored.body:notice.body,tag:notice.tag||(typeof stored.tag==='string'?stored.tag:'alpha-message'),received:stored.received||Date.now(),pending:stored.pending===true,projectId:stored.projectId,requestId:stored.requestId};
 const review=inspectionTarget(message);
 if(review){
  // Open while the notification gesture is active, before waiting on storage or
  // attempting a cross-scope WindowClient.navigate in Samsung Internet.
  let opening;try{opening=self.clients.openWindow(review)}catch(e){opening=Promise.reject(e)}
  event.waitUntil(Promise.all([saveMessage(message).catch(()=>{}),Promise.resolve(opening).then(async page=>{if(page){try{await page.focus()}catch(_){}}}).catch(async()=>{
   const pages=await self.clients.matchAll({type:'window',includeUncontrolled:true});for(const page of pages){if(new URL(page.url).origin===self.location.origin){try{const opened=await page.navigate(review);if(opened){await opened.focus();return}}catch(_){}}}
  })]));return;
 }
 event.waitUntil((async()=>{
  await saveMessage(message).catch(()=>{});
  const target=HOME+'#'+new URLSearchParams({message:message.tag});
  const pages=await self.clients.matchAll({type:'window',includeUncontrolled:true});
  for(const page of pages){const url=new URL(page.url);if(url.origin===self.location.origin&&url.pathname.startsWith(new URL(HOME).pathname)){try{const opened=await page.navigate(target);if(opened){await opened.focus();opened.postMessage({type:'open-message',message});return}}catch(_){}}}
  const opened=await self.clients.openWindow(target);if(opened)opened.postMessage({type:'open-message',message});
 })());
});
