(()=>{'use strict';const config=document.getElementById('worker-reconnect-config');if(!config||location.origin!==config.dataset.origin)return;const board=config.dataset.board;if(!/^[A-Za-z0-9_-]{43}$/.test(board))return;const root='/work/'+board,relay='https://swsgood79-beep.github.io/Tool/#k='+encodeURIComponent(board);let busy=false,failures=0,moving=false;
const bar=document.createElement('aside');bar.className='worker-reconnect-bar';bar.hidden=true;bar.setAttribute('role','status');const text=document.createElement('span'),link=document.createElement('a');link.href=relay;link.textContent='새 주소로 연결';link.rel='noopener noreferrer';bar.append(text,link);document.body.append(bar);
const dirty=new Set();document.addEventListener('input',e=>{const form=e.target.closest('form');if(form&&e.isTrusted)dirty.add(form)},true);document.addEventListener('reset',e=>dirty.delete(e.target),true);const working=()=>!!document.querySelector('.worker-inspection-dialog[open],.worker-special-review-dialog[open],form[data-sending]')||[...dirty].some(form=>form.isConnected&&form.getClientRects().length>0);
function message(value,keep){text.textContent=value;bar.hidden=false;link.target=keep?'_blank':'_self';link.textContent=keep?'입력 내용 유지하고 새 탭 연결':'새 주소로 연결'}
async function register(){if(!isSecureContext||!('serviceWorker'in navigator))return;try{await navigator.serviceWorker.register('/worker-reconnect-sw-v1.4.42.js?board='+encodeURIComponent(board),{scope:'/',updateViaCache:'none'});await navigator.serviceWorker.ready}catch(_){/* The ordinary fixed entry remains available when browser policy blocks workers. */}}
function recover(versionChanged){if(moving)return;if(working()){message('서버가 다시 연결되는 중입니다. 입력 중인 화면은 그대로 두었습니다.',true);return}moving=true;message(versionChanged?'새 버전으로 화면을 갱신합니다.':'최신 서버 주소로 다시 연결하고 있습니다.',false);if(versionChanged||navigator.serviceWorker?.controller)location.reload();else location.replace(relay)}
async function check(){if(busy||moving||document.hidden)return;busy=true;const controller=new AbortController(),timeout=setTimeout(()=>controller.abort(),4500);try{const response=await fetch('/ready/'+encodeURIComponent(board),{cache:'no-store',credentials:'same-origin',signal:controller.signal});if(!response.ok)throw Error('unavailable');const data=await response.json();if(data.application!=='wanryotalk')throw Error('wrong application');failures=0;if(typeof data.release==='string'&&data.release!==config.dataset.release){recover(true)}else{bar.hidden=true}}catch(_){failures++;if(failures>=2)recover(false)}finally{clearTimeout(timeout);busy=false}}
// A valid Talk multipart POST intentionally navigates. Keep the draft guard for
// cancelled/invalid submissions and ordinary exits, and keep dirty state during upload.
let talkSubmission=null;
const clearSubmission=()=>{talkSubmission=null};
document.addEventListener('pointerdown',e=>{if(e.isTrusted)clearSubmission()},true);
document.addEventListener('keydown',e=>{if(e.isTrusted)clearSubmission()},true);
document.addEventListener('input',e=>{if(e.isTrusted)clearSubmission()},true);
window.addEventListener('pageshow',clearSubmission);
document.addEventListener('submit',event=>{
 talkSubmission=null;const form=event.target;if(!(form instanceof HTMLFormElement)||form.getAttribute('id')!=='community-form')return;
 const submitter=event.submitter,method=(submitter?.getAttribute('formmethod')||form.getAttribute('method')||'get').toLowerCase(),target=submitter?.getAttribute('formtarget')||form.getAttribute('target');
 const url=new URL(submitter?.getAttribute('formaction')||form.getAttribute('action')||location.href,location.href);
 if(method==='post'&&(!target||target==='_self')&&url.origin===location.origin&&url.pathname==='/community/'+board+'/save')talkSubmission={event,form};
},true);
window.addEventListener('beforeunload',event=>{
 const submission=talkSubmission;talkSubmission=null;
 // Inspect defaultPrevented here, after the form's photo validator has run.
 const posting=submission&&!submission.event.defaultPrevented&&submission.form.isConnected;
 if(!posting&&working()){event.preventDefault();event.returnValue=''}
});document.addEventListener('visibilitychange',()=>{if(!document.hidden)check()});document.addEventListener('close',()=>{if(failures>=2&&!working())recover(false)},true);window.addEventListener('online',check);register();setInterval(check,10000);check();
})();
