(()=>{'use strict';const root=document.getElementById('tts-settings');if(!root)return;
 const enabled=document.getElementById('tts-enabled'),voice=document.getElementById('tts-voice'),repeat=document.getElementById('tts-repeat'),speed=document.getElementById('tts-speed'),value=document.getElementById('tts-speed-value'),saveStatus=document.getElementById('tts-save-status'),playStatus=document.getElementById('tts-play-status'),test=document.getElementById('tts-test');
 let debounce=null,chain=Promise.resolve(),actionError='',saving=0,testing=false,edit=0;
 const draw=()=>value.textContent=Number(speed.value).toFixed(2)+'x';draw();
 const tell=(text,error=false)=>{playStatus.textContent=text;playStatus.classList.toggle('tts-error',error)};
 const body=()=>new URLSearchParams({csrf:root.dataset.csrf,enabled:enabled.checked?'1':'0',voice:voice.value,speed:speed.value,repeat:repeat.value});
 async function post(path,data){const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),20000);try{const r=await fetch(path,{method:'POST',credentials:'same-origin',cache:'no-store',body:data,signal:controller.signal});if(!r.ok)throw Error((await r.text()).trim()||`요청 실패 (HTTP ${r.status})`);return await r.json()}catch(e){if(e.name==='AbortError')throw Error('요청 시간이 초과되었습니다. PC 프로그램 연결을 확인해 주세요.');throw e}finally{clearTimeout(timer)}}
 function save(){clearTimeout(debounce);debounce=null;const data=body();saving++;saveStatus.classList.remove('tts-error');saveStatus.textContent='설정 저장 중…';chain=chain.catch(()=>{}).then(()=>post('/tts-settings',data)).then(()=>{saveStatus.textContent='저장됨 · tts_config.json';saveStatus.classList.remove('tts-error')}).catch(e=>{saveStatus.textContent=e.message;saveStatus.classList.add('tts-error');throw e}).finally(()=>saving--);return chain}
 const changed=()=>{edit++;save().catch(()=>{})};
 enabled.addEventListener('change',changed);voice.addEventListener('change',changed);repeat.addEventListener('change',changed);
 speed.addEventListener('input',()=>{edit++;draw();clearTimeout(debounce);debounce=setTimeout(()=>save().catch(()=>{}),200)});speed.addEventListener('change',changed);
 test.addEventListener('click',async()=>{test.disabled=true;testing=true;actionError='';tell('선택한 목소리로 테스트를 요청하고 있습니다…');try{await save();await post('/tts-settings/test',new URLSearchParams({csrf:root.dataset.csrf}));tell('테스트를 재생 대기열에 추가했습니다. 음성을 준비하고 있습니다.')}catch(e){actionError=e.message;tell(actionError,true)}finally{testing=false;test.disabled=false}});
 async function status(){const version=edit,controller=new AbortController(),timer=setTimeout(()=>controller.abort(),5000);
 try{const r=await fetch('/tts-settings/status',{credentials:'same-origin',cache:'no-store',signal:controller.signal});if(!r.ok)throw Error(`음성 상태 확인 실패 (HTTP ${r.status})`);const data=await r.json();
 if(!saving&&!debounce&&!testing&&version===edit&&typeof data.config?.enabled==='boolean')enabled.checked=data.config.enabled;
 // POST errors are actionable. An unrelated idle/OFF poll must never erase them.
 if(!actionError&&!testing&&data.status)tell(data.status,/실패|오류|Error|Exception/.test(data.status));
 }catch(e){if(!actionError&&!testing)tell(e.name==='AbortError'?'음성 상태 확인 시간 초과 · PC 프로그램 연결을 확인해 주세요.':e.message,true)}finally{clearTimeout(timer);setTimeout(status,1000)}}status();
})();
