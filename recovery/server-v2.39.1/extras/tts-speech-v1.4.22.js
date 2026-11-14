(()=>{
 'use strict';
 const root=document.getElementById('electron-speech');if(!root)return;
 const toggle=document.getElementById('electron-speech-toggle'),preview=document.getElementById('electron-speech-preview'),status=document.getElementById('electron-speech-status');
 const shared=root.dataset.mode==='shared',endpoint=root.dataset.endpoint||'/admin-speech/poll',audioEndpoint=shared?endpoint+'/audio':'/admin-speech/audio',label=shared?'공용 PC 음성 알림':'음성 알림',topics=shared?'검사 신청':'검사 신청·비고';
 const random=()=>{const b=new Uint8Array(24);crypto.getRandomValues(b);return [...b].map(n=>n.toString(16).padStart(2,'0')).join('')};
 let device=random();try{const saved=localStorage.getItem('alpha-speech-device');if(/^[A-Za-z0-9_-]{16,80}$/.test(saved||''))device=saved;else localStorage.setItem('alpha-speech-device',device)}catch(_){}
 const player=new Audio();player.preload='auto';
 let voiceName="선택한 음성",displaySpeed=1.1;
 let enabled=false,active=false,established=false,owner='',cursor=0,generation=0,timer=null,controller=null,context=null,busy=false,watchdog=null,chimeTimer=null,audioRequest=null,objectURL=null,sequence=0,failed=false,selectBrowser=false,unlock=Promise.resolve(),lastBell=-Infinity,bellUntil=0;
 const queue=[],seen=new Set(),nodes=new Set();
 const preferenceKey='alpha-speech-browser-enabled-v1';let needsGesture=false,unlocking=false,automaticStart=false;
 function wanted(){try{return localStorage.getItem(preferenceKey)!=='off'}catch(_){return true}}
 function remember(on){try{localStorage.setItem(preferenceKey,on?'on':'off')}catch(_){}}

 function tell(text,error=false){status.textContent=text;root.dataset.error=String(error)}
 function ready(){return `${topics} 읽기 · ${voiceName} · ${displaySpeed.toFixed(2)}배속`}
 function discardURL(){if(objectURL){URL.revokeObjectURL(objectURL);objectURL=null}}
 function stopSound(){bellUntil=0;lastBell=-Infinity;sequence++;queue.length=0;clearTimeout(watchdog);clearTimeout(chimeTimer);watchdog=chimeTimer=null;if(audioRequest)audioRequest.abort();audioRequest=null;player.onended=player.onerror=null;player.pause();player.removeAttribute('src');player.load();discardURL();busy=false;for(const n of nodes){try{n.stop()}catch(_){}}nodes.clear()}
 function bell(){if(!context||context.state!=='running')return 0;const start=context.currentTime;for(const [offset,freq] of [[0,880],[.36,659.25]]){const o=context.createOscillator(),g=context.createGain();o.frequency.value=freq;o.type='sine';g.gain.setValueAtTime(0,start+offset);g.gain.linearRampToValueAtTime(.16,start+offset+.012);g.gain.exponentialRampToValueAtTime(.001,start+offset+.4);o.connect(g);g.connect(context.destination);nodes.add(o);o.onended=()=>{nodes.delete(o);o.disconnect();g.disconnect()};o.start(start+offset);o.stop(start+offset+.42)}return 850}
 async function next(){
  if(!enabled||busy||!queue.length)return;busy=true;const job=queue.shift(),gen=generation,seq=++sequence;
  const current=()=>enabled&&gen===generation&&seq===sequence;
  const finish=error=>{if(!current())return;clearTimeout(watchdog);clearTimeout(chimeTimer);player.onended=player.onerror=null;player.pause();player.removeAttribute('src');player.load();discardURL();busy=false;failed=error;if(error)tell(typeof error==='string'?error:'선택한 음성 재생 실패 · 스피커를 확인해 주세요.',true);else tell(ready());next()};
  try{
   await unlock;if(!current())return;
   tell('선택한 음성 생성 중…');audioRequest=new AbortController();
   const request=audioRequest;watchdog=setTimeout(()=>{request.abort();finish(true)},90000);
   const body=new URLSearchParams({csrf:root.dataset.csrf,event:String(job.id||0),preview:job.preview?'1':'0',welcome:job.welcome?'1':'0'});
   const r=await fetch(audioEndpoint,{method:'POST',credentials:'same-origin',cache:'no-store',body,signal:request.signal});if(r.status===409){finish(false);return}if(!r.ok)throw Error((await r.text()).trim());
   const mediaRate=Number(r.headers.get('X-TTS-Playback-Rate')||'1.1');displaySpeed=Number(r.headers.get('X-TTS-Requested-Speed')||'1.1');voiceName=({gtts:'구글 전자녀',sunhi:'MS 선희',injoon:'MS 인준',sapi:'윈도우 기본 내장'})[r.headers.get('X-TTS-Voice')]||'선택한 음성';
   const blob=await r.blob();if(!current())return;if(blob.size<4||blob.size>8*1024*1024)throw Error();
   audioRequest=null;objectURL=URL.createObjectURL(blob);player.src=objectURL;player.playbackRate=mediaRate;player.preservesPitch=true;player.webkitPreservesPitch=true;
   let remaining=Number(r.headers.get('X-TTS-Repeat')||'1');if(remaining!==1&&remaining!==2)throw Error('반복 횟수 오류');clearTimeout(watchdog);watchdog=setTimeout(()=>finish(true),90000*remaining);player.onended=()=>{if(!current())return;remaining--;if(remaining>0){player.currentTime=0;player.play().catch(()=>finish(true))}else finish(false)};player.onerror=()=>finish(true);
   const play=()=>{if(current()){tell(`${voiceName} 읽는 중 · ${displaySpeed.toFixed(2)}배속`);player.play().catch(()=>finish(true))}};const delay=Math.max(0,(job.notBefore||0)-performance.now());if(delay)chimeTimer=setTimeout(play,delay);else play();
  }catch(e){finish(e.message||true)}
 }
 function enqueue(job){if(queue.length>=256){tell(`음성 대기가 많습니다. ${topics} 목록을 확인해 주세요.`,true);return}if(!job.welcome){const now=performance.now();if(now-lastBell>=1000){lastBell=now;bellUntil=now+bell()}job.notBefore=bellUntil}queue.push(job);next()}
 function form(on,baseline=false,identity=owner){return new URLSearchParams({csrf:root.dataset.csrf,owner:identity,device,enabled:on?'1':'0',after:String(cursor),baseline:baseline?'1':'0',select_browser:on&&selectBrowser?'1':'0'})}
 function release(identity){if(identity)fetch(endpoint,{method:'POST',credentials:'same-origin',cache:'no-store',body:form(false,false,identity),keepalive:true}).catch(()=>{})}
 async function poll(gen){
  if(!enabled||gen!==generation)return;controller=new AbortController();const c=controller,identity=owner,timeout=setTimeout(()=>c.abort(),5000);
  try{const r=await fetch(endpoint,{method:'POST',credentials:'same-origin',cache:'no-store',body:form(true,!established),signal:c.signal});if(!r.ok)throw Error();const data=await r.json();
   if(gen!==generation||!enabled){release(identity);return}if(data.ok!==true||!Array.isArray(data.events)||!Number.isSafeInteger(data.cursor))throw Error();active=data.active===true;selectBrowser=false;preview.disabled=!active;
   if(data.native_selected===true){off(false);toggle.textContent='🔊 이 화면에서 재생';root.dataset.state='native';tell('음성 알림 켜짐 · PC 프로그램에서 재생합니다. 브라우저 재생으로 바꾸려면 위 버튼을 누르세요.');return}
   if(!active){stopSound();established=false;tell('이 브라우저의 다른 창에서 음성을 읽고 있습니다. 그 창에서 끄면 이 창이 이어받습니다.');return}
   established=true;cursor=data.cursor;for(const event of data.events.sort((a,b)=>a.id-b.id)){if(!Number.isSafeInteger(event.id)||seen.has(event.id))continue;seen.add(event.id);enqueue({id:event.id})}if(!busy&&!failed)tell(ready());
  }catch(_){if(gen===generation&&enabled){active=false;preview.disabled=true;established=false;stopSound();tell(`음성 알림 연결 확인 중 · 현재 ${topics} 목록을 확인해 주세요.`,true)}}
  finally{clearTimeout(timeout);if(controller===c)controller=null;if(enabled&&gen===generation)timer=setTimeout(()=>poll(gen),500)}
 }
 function off(persist=true){const identity=owner;enabled=active=false;needsGesture=unlocking=false;generation++;clearTimeout(timer);if(controller)controller.abort();stopSound();toggle.textContent=`🔊 ${label} 켜기`;toggle.setAttribute('aria-pressed','false');preview.disabled=true;root.dataset.state='off';tell(`${label} 꺼짐`);release(identity);if(context){context.close().catch(()=>{});context=null}if(persist)remember(false)}
 function activate(gen){
  if(!enabled||gen!==generation||unlocking)return;
  needsGesture=false;unlocking=true;root.dataset.state='connecting';tell('음성 연결 중…');
  // Attempt autoplay first. When the browser requires user activation, keep the
  // ON preference and retry synchronously on a real tap/key, without taking a
  // speech lease or stopping the PC's native voice before media is unlocked.
  try{const A=window.AudioContext||window.webkitAudioContext;if(A){if(!context)context=new A();context.resume().catch(()=>{})}}catch(_){}
  const wav=new Uint8Array(44+1600),v=new DataView(wav.buffer);const put=(at,s)=>[...s].forEach((c,i)=>wav[at+i]=c.charCodeAt(0));put(0,'RIFF');v.setUint32(4,wav.length-8,true);put(8,'WAVEfmt ');v.setUint32(16,16,true);v.setUint16(20,1,true);v.setUint16(22,1,true);v.setUint32(24,8000,true);v.setUint32(28,16000,true);v.setUint16(32,2,true);v.setUint16(34,16,true);put(36,'data');v.setUint32(40,1600,true);
  const silent=URL.createObjectURL(new Blob([wav],{type:'audio/wav'}));player.src=silent;
  unlock=Promise.resolve(player.play()).then(()=>{if(enabled&&gen===generation){player.pause();player.removeAttribute('src');player.load()}}).finally(()=>URL.revokeObjectURL(silent));
  unlock.then(()=>{
   if(!enabled||gen!==generation)return;unlocking=false;
   if(context&&context.state==='suspended'){needsGesture=true;root.dataset.state='gesture';tell('음성 알림 켜짐 · 소리 활성화를 위해 화면을 한 번 눌러 주세요.');return}
   root.dataset.state='on';if(!automaticStart)enqueue({preview:true,welcome:true});poll(gen);
  }).catch(e=>{if(enabled&&gen===generation){unlocking=false;needsGesture=true;root.dataset.state='gesture';tell('음성 알림 켜짐 · 브라우저에서 소리 재생을 허용하려면 화면을 한 번 눌러 주세요.',e.name!=='NotAllowedError')}});
 }
 function start(automatic=false){
  if(enabled)return;enabled=true;active=false;established=false;failed=false;automaticStart=automatic;selectBrowser=!automatic;owner=random();cursor=0;seen.clear();generation++;
  toggle.textContent=`🔊 ${label} 끄기`;toggle.setAttribute('aria-pressed','true');activate(generation);
 }
 toggle.addEventListener('click',()=>{if(enabled){off();return}remember(true);start(false)});
 function gesture(event){if(!event.isTrusted||!needsGesture||!enabled||event.target.closest?.('#electron-speech-toggle')||event.type==='keydown'&&(event.repeat||event.key==='Escape'))return;activate(generation)}
 document.addEventListener('pointerdown',gesture,true);document.addEventListener('keydown',gesture,true);
 window.addEventListener('storage',e=>{if(e.key!==preferenceKey)return;if(wanted())start(true);else off(false)});
 preview.addEventListener('click',()=>{if(enabled&&active){failed=false;enqueue({preview:true})}});
 window.addEventListener('pagehide',()=>{if(enabled)off(false)});
 window.addEventListener('pageshow',e=>{if(e.persisted&&wanted())start(true)});
 if(wanted())start(true);else off(false);
})();
