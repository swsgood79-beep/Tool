const{chromium}=require('playwright');const fs=require('fs'),path=require('path'),assert=require('assert');
const root=path.join(__dirname,'..');
async function setup(browser,shared,width,engine="gtts"){
 const html=fs.readFileSync(shared?'/tmp/tts-worker-v18.html':'/tmp/tts-admin-v18.html','utf8').replaceAll('/tts-speech-v1.4.18.js','/tts-speech-v1.4.20.js'),csrf=html.match(/id="electron-speech"[^>]*data-csrf="([^"]+)"/)[1],poll=shared?'/worker-speech/'+csrf:'/admin-speech/poll',audio=shared?poll+'/audio':'/admin-speech/audio';
 const ctx=await browser.newContext({viewport:{width,height:950}}),p=await ctx.newPage(),errors=[];p.on('pageerror',e=>errors.push(e.message));await p.clock.install();
 await p.addInitScript(()=>{
  window.__played=[];window.__tones=[];window.__unlocked=[];window.__urls=new Map();window.__sizes=new Map();window.__revoked=[];
  const create=URL.createObjectURL.bind(URL),revoke=URL.revokeObjectURL.bind(URL);URL.createObjectURL=b=>{const u=create(b);window.__urls.set(u,b.type);window.__sizes.set(u,b.size);return u};URL.revokeObjectURL=u=>{window.__revoked.push(u);revoke(u)};
  Object.defineProperty(window,'speechSynthesis',{get(){throw Error('old browser TTS used')}});
  window.Audio=class{constructor(){window.__player=this}play(){const type=window.__urls.get(this.src);if(type==='audio/wav'&&window.__sizes.get(this.src)===1644){window.__unlocked.push(navigator.userActivation.isActive)}else{if(this.busy)throw Error('overlapping audio');this.busy=true;window.__played.push({rate:this.playbackRate,pitch:this.preservesPitch,type});}return Promise.resolve()}pause(){this.busy=false}load(){}removeAttribute(name){delete this[name]}};
  window.__finish=()=>{const p=window.__player;p.busy=false;p.onended?.()};
  window.AudioContext=class{state='running';currentTime=0;destination={};resume(){return Promise.resolve()}close(){return Promise.resolve()}createOscillator(){return{frequency:{value:0},connect(){},disconnect(){},start(){window.__tones.push(this.frequency.value)},stop(){}}}createGain(){return{gain:{setValueAtTime(){},linearRampToValueAtTime(){},exponentialRampToValueAtTime(){}},connect(){},disconnect(){}}}};
 });
 let events=[],cursor=1,fail=false,held=false,finishHeld=null,repeat=1,nativeSelected=false,waiting=false;const requests=[],selections=[];
 await p.route('**/*',async route=>{
  const req=route.request(),u=new URL(req.url());
  if(u.pathname===poll){const f=new URLSearchParams(req.postData());assert.equal(f.get('csrf'),csrf);selections.push(f.get('select_browser'));const out=f.get('baseline')==='1'?[]:events;events=[];return route.fulfill({json:{ok:true,active:f.get('enabled')==='1'&&!nativeSelected&&!waiting,native_selected:nativeSelected,cursor,events:out}})}
  if(u.pathname===audio){const f=new URLSearchParams(req.postData());assert.equal(f.get('csrf'),csrf);requests.push({id:Number(f.get('event')),preview:f.get('preview'),welcome:f.get('welcome')});
   if(held)await new Promise(resolve=>finishHeld=resolve);
   return route.fulfill(fail?{status:502,body:'network failed'}:{contentType:engine==='sapi'?'audio/wav':'audio/mpeg',headers:{'X-TTS-Voice':engine,'X-TTS-Playback-Rate':['sunhi','injoon'].includes(engine)?'1.00':'1.25','X-TTS-Requested-Speed':'1.25','X-TTS-Repeat':String(repeat)},body:Buffer.from(engine==='sapi'?'RIFFsynthetic-wave':'ID3synthetic-audio')}).catch(()=>{});
  }
  if(u.pathname==='/')return route.fulfill({contentType:'text/html',body:html});
  const name=path.basename(u.pathname);for(const dir of ['extras','web']){const file=path.join(root,dir,name);if(name&&fs.existsSync(file)&&fs.statSync(file).isFile())return route.fulfill({contentType:name.endsWith('.js')?'application/javascript':'text/css',body:fs.readFileSync(file)})}
  return route.fulfill({status:404,body:''});
 });
 await p.goto('http://127.0.0.1:18791/');
 return{ctx,p,errors,requests,selections,setWaiting(v){waiting=v},setNative(v){nativeSelected=v},setRepeat(v){repeat=v},setEvents(e){events=e;cursor=Math.max(cursor,...e.map(x=>x.id))},setFail(v){fail=v},setHeld(v){held=v;if(!v)finishHeld?.()}};
}
(async()=>{const browser=await chromium.launch({headless:true,executablePath:'/usr/bin/chromium',args:['--no-sandbox']});try{
 const results=[];
 for(const engine of ["gtts","sunhi","injoon","sapi"])for(const shared of [false,true])for(const width of [1100,390]){
  const t=await setup(browser,shared,width,engine),p=t.p;
  await p.locator('#electron-speech-toggle').click();await p.waitForFunction(()=>window.__played.length===1);assert.deepEqual(await p.evaluate(()=>window.__unlocked),[true]);
  await p.evaluate(()=>window.__finish());await p.waitForFunction(()=>!document.getElementById('electron-speech-preview').disabled);
  t.setEvents([{id:3,text:'not sent to synthesizer from client'},{id:2,text:'request'},{id:2,text:'duplicate'}]);
  await p.clock.runFor(3100);
  await p.waitForFunction(()=>window.__urls.size>=3);await p.clock.runFor(900);await p.waitForFunction(()=>window.__played.length===2);
  assert.deepEqual(t.requests.filter(x=>!Number(x.preview)).map(x=>x.id),[2]);assert.deepEqual(await p.evaluate(()=>window.__tones.slice(0,2)),[880,659.25]);
  await p.clock.runFor(1000);assert.equal(await p.evaluate(()=>window.__played.length),2);
  await p.evaluate(()=>window.__finish());await p.waitForFunction(()=>window.__urls.size>=4);await p.clock.runFor(900);await p.waitForFunction(()=>window.__played.length===3);
  assert.deepEqual(t.requests.filter(x=>!Number(x.preview)).map(x=>x.id),[2,3]);
  for(const x of await p.evaluate(()=>window.__played)){assert.equal(x.rate,["sunhi","injoon"].includes(engine)?1:1.25);assert.equal(x.pitch,true);assert.equal(x.type,engine==='sapi'?'audio/wav':'audio/mpeg')}
  await p.evaluate(()=>window.__finish());
  t.setFail(true);await p.locator('#electron-speech-preview').click();await p.waitForFunction(()=>document.getElementById('electron-speech').dataset.error==='true');
  t.setFail(false);await p.locator('#electron-speech-preview').click();await p.waitForFunction(()=>window.__urls.size>=5);await p.clock.runFor(900);await p.waitForFunction(()=>window.__played.length===4);
  await p.screenshot({path:path.join(__dirname,`tts-${engine}-${shared?'shared':'admin'}-${width}-v1.4.20.png`),fullPage:false});
  await p.locator('#electron-speech-toggle').click();assert.equal(await p.evaluate(()=>window.__player.busy),false);assert.equal(await p.evaluate(()=>window.__player.src),undefined);
  assert.equal(await p.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false);assert.deepEqual(t.errors,[]);
  results.push({engine,mode:shared?'shared':'admin',width,clickUnlock:'pass',gttsMP3API:'pass',playbackRate:["sunhi","injoon"].includes(engine)?1:1.25,preservesPitch:true,fifoAndNoOverlap:'pass',duplicateSuppression:'pass',networkFailureAndRecovery:'pass',offAndURLCleanup:'pass',noOldSpeechSynthesis:'pass'});await t.ctx.close();
 }
 const t=await setup(browser,false,390),p=t.p;await p.locator('#electron-speech-toggle').click();await p.waitForFunction(()=>window.__played.length===1);await p.evaluate(()=>window.__finish());await p.waitForFunction(()=>!document.getElementById('electron-speech-preview').disabled);
 t.setHeld(true);await p.locator('#electron-speech-preview').click();await p.waitForFunction(()=>document.getElementById('electron-speech-status').textContent.includes('생성 중'));assert((await p.evaluate(()=>window.__tones.length))>=2,'doorbell waited for held synthesis');assert.equal(await p.evaluate(()=>window.__played.length),1);await p.locator('#electron-speech-toggle').click();t.setHeld(false);await p.clock.runFor(2000);assert.equal(await p.evaluate(()=>window.__played.length),1);assert.deepEqual(t.errors,[]);await t.ctx.close();
 const repeatResults=[];
 for(const shared of [false,true]){
  const t=await setup(browser,shared,390,'sunhi'),p=t.p;await p.locator('#electron-speech-toggle').click();await p.waitForFunction(()=>window.__played.length===1);await p.evaluate(()=>window.__finish());await p.waitForFunction(()=>!document.getElementById('electron-speech-preview').disabled);
  t.setRepeat(2);t.setEvents([{id:2,text:'repeat request'},{id:3,text:'next request'}]);await p.clock.runFor(3100);await p.waitForFunction(()=>window.__urls.size>=3);await p.clock.runFor(900);await p.waitForFunction(()=>window.__played.length===2);
  await p.evaluate(()=>window.__finish());await p.waitForFunction(()=>window.__played.length===3);assert.deepEqual(t.requests.filter(x=>!Number(x.preview)).map(x=>x.id),[2]);
  await p.evaluate(()=>window.__finish());await p.waitForFunction(()=>window.__urls.size>=4);await p.clock.runFor(10);await p.waitForFunction(()=>window.__played.length===4);assert.deepEqual(t.requests.filter(x=>!Number(x.preview)).map(x=>x.id),[2,3]);
  await p.locator('#electron-speech-toggle').click();await p.clock.runFor(1500);assert.equal(await p.evaluate(()=>window.__played.length),4);assert.equal(await p.evaluate(()=>window.__player.busy),false);assert.deepEqual(t.errors,[]);repeatResults.push({mode:shared?'shared':'admin',twoReadsBeforeNextMessage:'pass',oneSynthesisPerMessage:'pass',offStopsRepetition:'pass'});await t.ctx.close();
 }
 const waiting=await setup(browser,false,390,'sunhi');waiting.setWaiting(true);await waiting.p.locator('#electron-speech-toggle').click();await waiting.p.waitForFunction(()=>document.getElementById('electron-speech-status').textContent.includes('다른 창'));
 await waiting.p.clock.runFor(1200);assert.equal(waiting.selections[0],'1');assert(waiting.selections.slice(1).length>0&&waiting.selections.slice(1).every(x=>x==='0'));
 waiting.setNative(true);await waiting.p.clock.runFor(1200);await waiting.p.waitForFunction(()=>document.getElementById('electron-speech-toggle').getAttribute('aria-pressed')==='false');assert.deepEqual(waiting.errors,[]);await waiting.ctx.close();
 const nativeResults=[];
 for(const shared of [false,true]){
 const t=await setup(browser,shared,390,'sunhi'),p=t.p;await p.locator('#electron-speech-toggle').click();await p.waitForFunction(()=>window.__played.length===1);
 t.setNative(true);await p.clock.runFor(1000);await p.waitForFunction(()=>document.getElementById('electron-speech-toggle').getAttribute('aria-pressed')==='false');
 assert((await p.locator('#electron-speech-status').innerText()).includes('PC 프로그램에서'));assert.equal(await p.evaluate(()=>window.__player.busy),false);assert.deepEqual(t.errors,[]);nativeResults.push({shared,nativeHandoffStopsBrowser:'pass'});await t.ctx.close();
 }
 fs.writeFileSync(path.join(__dirname,'browser-tts-v1.4.20.json'),JSON.stringify({scope:'Go-rendered synthetic pages; mocked MP3 API and media output, no live Google synthesis/real speakers',results,repeatResults,nativeResults,waitingTabDoesNotReselect:'pass',immediateBellBeforeSynthesis:'pass',cancelWhileGenerating:'pass'},null,2)+'\n');console.log('Four-engine speech browser checks passed');
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
