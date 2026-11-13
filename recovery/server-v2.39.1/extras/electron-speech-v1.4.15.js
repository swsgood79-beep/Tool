(()=>{
 'use strict';
 const root=document.getElementById('electron-speech');if(!root)return;
 const toggle=document.getElementById('electron-speech-toggle'),preview=document.getElementById('electron-speech-preview'),status=document.getElementById('electron-speech-status'),synth=window.speechSynthesis;
 if(!synth||!window.SpeechSynthesisUtterance){toggle.disabled=true;status.textContent='이 브라우저는 음성 읽기를 지원하지 않습니다.';return}
 const shared=root.dataset.mode==='shared',endpoint=root.dataset.endpoint||'/admin-speech/poll',label=shared?'공용 PC 신청 알림':'전자녀 알람',topics=shared?'검사 신청':'검사 신청·비고';
 const random=()=>{const b=new Uint8Array(24);crypto.getRandomValues(b);return [...b].map(n=>n.toString(16).padStart(2,'0')).join('')};
 let device=random();try{const saved=localStorage.getItem('alpha-speech-device');if(/^[A-Za-z0-9_-]{16,80}$/.test(saved||''))device=saved;else localStorage.setItem('alpha-speech-device',device)}catch(_){}
 let enabled=false,active=false,established=false,owner='',cursor=0,generation=0,timer=null,controller=null,audio=null,busy=false,watchdog=null,chimeTimer=null,current=null,failed=false,selectBrowser=false;
 const queue=[],seen=new Set(),nodes=new Set();
 function voice(){const voices=synth.getVoices();for(const name of ['Google 한국의','Microsoft SunHi Online (Natural)','Microsoft Heami']){const found=voices.find(v=>v.name===name||v.name.startsWith(name+' ')||v.name.startsWith(name+' -'));if(found)return found}return voices.find(v=>/^ko(?:-|$)/i.test(v.lang))||null}
 function tell(text,error=false){status.textContent=text;root.dataset.error=String(error)}
 function ready(){const v=voice();return `${topics} 읽기 · ${v?v.name:'한국어 기본 음성'} · 속도 1.1`}
 function stopSound(){queue.length=0;clearTimeout(watchdog);clearTimeout(chimeTimer);watchdog=chimeTimer=null;current=null;busy=false;synth.cancel();for(const n of nodes){try{n.stop()}catch(_){}}nodes.clear()}
 function bell(){if(!audio||audio.state!=='running')return 0;const start=audio.currentTime;for(const [offset,freq] of [[0,880],[.36,659.25]]){const o=audio.createOscillator(),g=audio.createGain();o.frequency.value=freq;o.type='sine';g.gain.setValueAtTime(0,start+offset);g.gain.linearRampToValueAtTime(.16,start+offset+.012);g.gain.exponentialRampToValueAtTime(.001,start+offset+.4);o.connect(g);g.connect(audio.destination);nodes.add(o);o.onended=()=>{nodes.delete(o);o.disconnect();g.disconnect()};o.start(start+offset);o.stop(start+offset+.42)}return 850}
 function next(){if(!enabled||busy||!queue.length)return;busy=true;const job=queue.shift(),gen=generation;const play=()=>{if(!enabled||gen!==generation)return;const u=new SpeechSynthesisUtterance(job.text);u.lang='ko-KR';u.rate=1.1;u.pitch=1.0;u.volume=1;const selected=voice();if(selected)u.voice=selected;current=u;
   let done=false;const finish=error=>{if(done||gen!==generation||current!==u)return;done=true;clearTimeout(watchdog);current=null;busy=false;if(error){failed=true;tell('음성을 재생하지 못했습니다. 음성·스피커 설정을 확인하거나 다시 켜 주세요.',true)}else if(!failed)tell(ready());next()};
   u.onend=()=>finish(false);u.onerror=()=>finish(true);watchdog=setTimeout(()=>{synth.cancel();finish(true)},Math.min(180000,Math.max(20000,job.text.length*220)));try{synth.speak(u)}catch(_){finish(true)}};
   const delay=job.chime?bell():0;if(delay)chimeTimer=setTimeout(play,delay);else play()
 }
 function enqueue(text,chime=true){if(typeof text!=='string'||!text.trim())return;if(queue.length>=256){tell(`음성 대기가 많습니다. ${topics} 목록을 확인해 주세요.`,true);return}queue.push({text:text.slice(0,1600),chime});next()}
 function form(on,baseline=false,identity=owner){return new URLSearchParams({csrf:root.dataset.csrf,owner:identity,device,enabled:on?'1':'0',after:String(cursor),baseline:baseline?'1':'0',select_browser:!shared&&on&&selectBrowser?'1':'0'})}
 function release(identity){if(!identity)return;fetch(endpoint,{method:'POST',credentials:'same-origin',cache:'no-store',body:form(false,false,identity),keepalive:true}).catch(()=>{})}
 async function poll(gen){if(!enabled||gen!==generation)return;controller=new AbortController();const c=controller,identity=owner,timeout=setTimeout(()=>c.abort(),5000);try{const r=await fetch(endpoint,{method:'POST',credentials:'same-origin',cache:'no-store',body:form(true,!established),signal:c.signal});if(!r.ok)throw Error();const data=await r.json();if(gen!==generation||!enabled){release(identity);return}if(data.ok!==true||!Array.isArray(data.events)||!Number.isSafeInteger(data.cursor))throw Error();active=data.active===true;if(active)selectBrowser=false;preview.disabled=!active;if(!active){stopSound();established=false;tell(shared?'이 컴퓨터의 다른 창에서 음성을 읽고 있습니다. 그 창에서 끄면 이 창이 이어받습니다.':'다른 관리자 창에서 음성을 읽고 있습니다. 그 창에서 끄면 이 창이 이어받습니다.');return}established=true;cursor=data.cursor;for(const event of data.events.sort((a,b)=>a.id-b.id)){if(!Number.isSafeInteger(event.id)||seen.has(event.id))continue;seen.add(event.id);enqueue(event.text)}if(!busy&&!failed)tell(ready())}catch(_){if(gen===generation&&enabled){active=false;preview.disabled=true;established=false;stopSound();tell(`음성 알림 연결 확인 중 · 현재 ${topics} 목록을 확인해 주세요.`,true)}}finally{clearTimeout(timeout);if(controller===c)controller=null;if(enabled&&gen===generation)timer=setTimeout(()=>poll(gen),3000)}}
 function off(){const identity=owner;enabled=active=false;generation++;clearTimeout(timer);if(controller)controller.abort();stopSound();toggle.textContent=`🔊 ${label} 켜기`;toggle.setAttribute('aria-pressed','false');preview.disabled=true;tell(`${label} 꺼짐 · 신청 목록은 화면에서 확인할 수 있습니다.`);release(identity);if(audio){audio.close().catch(()=>{});audio=null}}
 toggle.addEventListener('click',()=>{if(enabled){off();return}enabled=true;active=false;established=false;failed=false;selectBrowser=true;owner=random();cursor=0;seen.clear();generation++;toggle.textContent=`🔊 ${label} 끄기`;toggle.setAttribute('aria-pressed','true');tell('음성 연결 중…');
   // Invoke both audio APIs directly in the user's click; subsequent jobs use one FIFO.
   try{const A=window.AudioContext||window.webkitAudioContext;if(A){audio=new A();audio.resume().catch(()=>{tell('효과음을 재생하지 못했습니다. 브라우저 소리 권한을 확인해 주세요.',true)})}}catch(_){}
   synth.resume();enqueue(shared?'공용 PC 검사 신청 알림을 켭니다.':'전자녀 알람을 켭니다.',false);poll(generation)
 });
 preview.addEventListener('click',()=>{if(enabled&&active){failed=false;enqueue('홍길동님이 알파 작업 확인의 세팅 및 출하 통합검사를 신청했습니다.')}});
 window.addEventListener('pagehide',()=>{if(enabled)off()});
})();
