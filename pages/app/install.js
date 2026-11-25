/* Installation controls must work even when the connection script has no board. */
(()=>{'use strict';
const button=document.getElementById('install'),help=document.getElementById('install-help'),context=document.getElementById('install-context'),feedback=document.getElementById('install-feedback'),copy=document.getElementById('copy-install-link'),field=document.getElementById('install-link'),label=document.getElementById('install-link-label'),chrome=document.getElementById('open-chrome');
if(!button||!help)return;
const valid=s=>typeof s==='string'&&/^[A-Za-z0-9_-]{43}$/.test(s);
function board(){const params=new URLSearchParams(location.hash.slice(1));if(params.has('k'))return params.get('k');try{return localStorage.getItem('alpha-app-board-v1')}catch(_){return ''}}
function link(){const b=board();if(!valid(b))return '';const u=new URL(location.pathname,location.origin);u.hash='k='+b;return u.href}
const ua=navigator.userAgent,inApp=/NAVER|KAKAOTALK|Instagram|FBAN|FBAV|; wv\)/i.test(ua),android=/Android/i.test(ua);
let deferred=null,preparationError=false;
function show(text=''){
 const url=link();feedback.textContent=text||(preparationError?'설치 준비가 완료되지 않았습니다. 인터넷 연결을 확인한 뒤 새로고침하거나 아래 메뉴 안내를 이용해 주세요.':'');
 context.textContent=!url?'작업장 연결 정보가 없습니다. 원래 작업자 QR 주소를 Chrome에서 다시 열고, 전체 메뉴의 앱 설치를 눌러 주세요.':inApp?'네이버 등 앱 안의 브라우저에서는 설치가 제한될 수 있습니다. 아래 설치 주소를 Chrome에서 열어 주세요.':'설치 창이 표시되지 않으면 아래 브라우저 메뉴에서 설치할 수 있습니다.';
 copy.hidden=!url;field.value=url;label.hidden=!url;chrome.hidden=!(url&&android);
 if(url&&android){const u=new URL(url);chrome.href='intent://'+u.host+u.pathname+u.hash+'#Intent;scheme=https;package=com.android.chrome;S.browser_fallback_url='+encodeURIComponent(url)+';end'}
 if(!help.open){try{if(typeof help.showModal==='function')help.showModal();else help.setAttribute('open','')}catch(_){help.setAttribute('open','')}}
}
addEventListener('beforeinstallprompt',event=>{event.preventDefault();deferred=event;button.textContent='알파 앱 설치'});
button.addEventListener('click',async()=>{
 if(!deferred||!link()||inApp){show();return}
 const prompt=deferred;deferred=null;
 try{await prompt.prompt();const choice=await prompt.userChoice;if(choice?.outcome!=='accepted')show('설치가 완료되지 않았습니다. 아래 메뉴 안내로 다시 설치할 수 있습니다.')}catch(_){show('설치 창을 열지 못했습니다. 아래 메뉴 안내로 설치해 주세요.')}
});
addEventListener('appinstalled',()=>{document.getElementById('install-bar').hidden=true;if(help.open&&typeof help.close==='function')help.close()});
copy.addEventListener('click',async()=>{const url=link();if(!url){show();return}try{if(!navigator.clipboard?.writeText)throw Error();await navigator.clipboard.writeText(url);feedback.textContent='설치 주소를 복사했습니다. Chrome 주소창에 붙여넣어 열어 주세요.'}catch(_){field.focus();field.select();feedback.textContent='아래 주소를 길게 눌러 복사한 뒤 Chrome 주소창에 붙여넣어 주세요.'}});
document.getElementById('help-close').addEventListener('click',()=>{if(typeof help.close==='function')help.close();else help.removeAttribute('open')});
addEventListener('alpha-install-error',()=>{preparationError=true;button.textContent='설치 방법 보기'});
})();
