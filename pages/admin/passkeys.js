(()=>{
 'use strict';
 const start=document.getElementById('fingerprint-login'),box=document.getElementById('passkey-flow'),confirm=document.getElementById('passkey-confirm'),status=document.getElementById('passkey-status');
 const mode=new URLSearchParams(location.search).get('passkey'),fragment=new URLSearchParams(location.hash.slice(1)),token=fragment.get(mode==='register'?'enroll':'start')||'';
 if(mode||location.hash)history.replaceState(null,'',location.pathname);
 const decode=s=>Uint8Array.from(atob(s.replace(/-/g,'+').replace(/_/g,'/')+'='.repeat((4-s.length%4)%4)),c=>c.charCodeAt(0));
 const encode=b=>btoa(String.fromCharCode(...new Uint8Array(b))).replace(/\+/g,'-').replace(/\//g,'_').replace(/=+$/,'');
 function serial(c){const r=c.response,out={id:c.id,rawId:encode(c.rawId),type:c.type,response:{clientDataJSON:encode(r.clientDataJSON)},clientExtensionResults:c.getClientExtensionResults()};if(c.authenticatorAttachment)out.authenticatorAttachment=c.authenticatorAttachment;if(r.attestationObject){out.response.attestationObject=encode(r.attestationObject);if(r.getTransports)out.response.transports=r.getTransports()}else{out.response.authenticatorData=encode(r.authenticatorData);out.response.signature=encode(r.signature);out.response.userHandle=r.userHandle?encode(r.userHandle):null}return out}
 class RequestFailure extends Error{constructor(code){super();this.code=code}}
 async function api(origin,path,data){const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),15000);try{const r=await fetch(origin+path,{method:'POST',mode:'cors',credentials:'omit',cache:'no-store',headers:{'Content-Type':'application/json'},body:JSON.stringify(data),signal:controller.signal});if(!r.ok)throw new RequestFailure('HTTP'+r.status);try{return await r.json()}catch(_){throw new RequestFailure('RESPONSE')}}finally{clearTimeout(timer)}}
 function failure(stage,e){
  const code=e instanceof RequestFailure?e.code:['NotAllowedError','NotSupportedError','InvalidStateError','SecurityError','NotReadableError','AbortError','TypeError'].includes(e?.name)?e.name:'UNKNOWN';
  let message;
  if(stage==='DEVICE'){
   if(code==='InvalidStateError')message='휴대폰 보관함에 같은 패스키가 이미 있을 수 있습니다. 기존 패스키 로그인을 먼저 확인해 주세요.';
   else if(code==='NotSupportedError')message='휴대폰 또는 선택한 패스키 보관함이 이 등록 방식을 지원하지 않습니다. 다른 지원 브라우저에서 비밀번호로 로그인한 뒤 등록해 주세요.';
   else if(code==='SecurityError')message='브라우저가 패스키 등록 주소를 허용하지 않았습니다. 고정 관리자 주소를 브라우저에서 직접 열어 다시 시작해 주세요.';
   else message='휴대폰에서 패스키 생성 또는 인증을 완료하지 못했습니다. 지문 확인 뒤에도 보관함 저장이 실패할 수 있습니다. 화면 잠금과 패스키 보관함 설정을 확인해 주세요.';
  }else if(stage==='VERIFY'){
   message=code.startsWith('HTTP')?'지문 확인 결과는 받았지만 회사 PC가 등록·로그인 검증을 승인하지 않았습니다. 아래 오류 코드를 알려 주세요.':'지문 확인 결과를 회사 PC로 보내는 과정에서 연결을 마치지 못했습니다. 회사 PC의 인터넷 연결을 확인해 주세요.';
  }else if(stage==='SERIALIZE')message='휴대폰이 반환한 패스키 결과를 읽지 못했습니다. 아래 오류 코드를 알려 주세요.';
  else if(stage==='BEGIN'&&code==='HTTP403')message='등록 대기 시간이 지났거나 회사 PC의 로그인 상태가 변경되었습니다. 관리자 화면에서 등록을 다시 시작해 주세요.';
  else message='패스키 인증 준비 중 회사 PC와 연결하지 못했습니다. PC 프로그램·인터넷 연결을 확인하고 다시 시작해 주세요.';
  return message+' (PK-'+stage+'-'+code+')';
 }
 const supported=!!window.PublicKeyCredential&&!!navigator.credentials;
 if(!supported){start.disabled=true;start.textContent='지문 로그인을 지원하는 브라우저에서 열어 주세요'}
 start.addEventListener('click',async()=>{start.disabled=true;try{const origin=await window.AlphaAdminEntry.resolve();location.assign(origin+'/mobile-admin/passkey/start')}catch(_){document.getElementById('status').textContent='회사 PC의 연결을 확인한 뒤 다시 눌러 주세요.'}finally{start.disabled=!supported}});
 if((mode==='register'||mode==='login')&&/^[A-Za-z0-9_-]{43}$/.test(token)){
  let restart=false;
  box.hidden=false;start.hidden=true;document.getElementById('passkey-title').textContent=mode==='register'?'이 휴대폰 패스키 등록':'지문으로 관리자 로그인';document.getElementById('passkey-description').textContent='휴대폰 설정에 따라 지문·얼굴·화면 잠금 PIN으로 확인합니다.';confirm.textContent=mode==='register'?'지문·패스키 등록하기':'지문으로 로그인';
  if(!supported){confirm.disabled=true;status.textContent='이 브라우저에서는 패스키를 사용할 수 없습니다. 관리자 화면 열기로 비밀번호 로그인하세요.'}
  confirm.addEventListener('click',async()=>{
   if(restart){document.getElementById('open-admin').click();return}
   confirm.disabled=true;status.classList.remove('error');status.textContent='회사 PC에 연결하고 있습니다…';let stage='CONNECT';
   try{
    const origin=await window.AlphaAdminEntry.resolve(),kind=mode==='register'?'register':'auth';stage='BEGIN';
    const begin=await api(origin,`/passkeys/${kind}/begin`,{token}),options=begin.options.publicKey;
    options.challenge=decode(options.challenge);if(options.user)options.user.id=decode(options.user.id);for(const list of ['allowCredentials','excludeCredentials'])if(options[list])options[list]=options[list].map(c=>({...c,id:decode(c.id)}));
    stage='DEVICE';status.textContent='휴대폰에서 지문 또는 잠금 해제를 확인해 주세요.';
    const credential=mode==='register'?await navigator.credentials.create({publicKey:options}):await navigator.credentials.get({publicKey:options});if(!credential)throw new RequestFailure('EMPTY');
    stage='SERIALIZE';const result=serial(credential);
    stage='VERIFY';status.textContent='지문 확인 결과를 회사 PC에서 검증하고 있습니다…';
    const end=await api(origin,`/passkeys/${kind}/finish`,{flow:begin.flow,credential:result});if(!/^[A-Za-z0-9_-]{43}$/.test(end.ticket))throw new RequestFailure('RESPONSE');
    status.textContent='확인되었습니다. 관리자 화면으로 이동합니다…';location.assign(origin+'/mobile-admin/passkey/enter#ticket='+encodeURIComponent(end.ticket));
   }catch(e){status.classList.add('error');status.textContent=failure(stage,e);restart=true;confirm.disabled=false;confirm.textContent='관리자 화면에서 다시 시작'}
  });
 }
})();
