const {chromium}=require('playwright');
const fs=require('fs'),path=require('path'),assert=require('assert');
const moduleRoot=path.join(__dirname,'..');
const cases=[
 {name:'google',voices:[{name:'Microsoft Heami - Korean (Korea)',lang:'ko-KR'},{name:'Microsoft SunHi Online (Natural) - Korean (Korea)',lang:'ko-KR'},{name:'Google 한국의',lang:'ko-KR'}],expected:'Google 한국의'},
 {name:'sunhi',voices:[{name:'Microsoft Heami - Korean (Korea)',lang:'ko-KR'},{name:'Microsoft SunHi Online (Natural) - Korean (Korea)',lang:'ko-KR'}],expected:'Microsoft SunHi Online (Natural) - Korean (Korea)'},
 {name:'heami',voices:[{name:'Microsoft Heami - Korean (Korea)',lang:'ko-KR'}],expected:'Microsoft Heami - Korean (Korea)'},
 {name:'korean-fallback',voices:[{name:'Other Korean',lang:'ko-KR'}],expected:'Other Korean'}
];
async function setup(browser,voices,width){
 const context=await browser.newContext({viewport:{width,height:950}}),page=await context.newPage();
 await page.clock.install();
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.addInitScript(voices=>{
  window.__spoken=[];window.__cancel=0;window.__tones=[];window.__resume=[];window.__voices=voices;window.__live=null;
  Object.defineProperty(window,'speechSynthesis',{value:{getVoices:()=>window.__voices,resume(){window.__resume.push('speech')},cancel(){window.__cancel++;window.__live=null},speak(u){if(window.__live)throw Error('overlapping utterances');window.__live=u;window.__spoken.push({text:u.text,voice:u.voice?.name,rate:u.rate,pitch:u.pitch,lang:u.lang,userActivation:navigator.userActivation.isActive})}}});
  window.SpeechSynthesisUtterance=class{constructor(text){this.text=text}};
  window.AudioContext=class{state='running';currentTime=0;destination={};resume(){window.__resume.push('audio');return Promise.resolve()}close(){return Promise.resolve()}createOscillator(){return{frequency:{value:0},connect(){},disconnect(){},start(){window.__tones.push(this.frequency.value)},stop(){}}}createGain(){return{gain:{setValueAtTime(){},linearRampToValueAtTime(){},exponentialRampToValueAtTime(){}},connect(){},disconnect(){}}}};
  window.__finish=()=>{const u=window.__live;window.__live=null;u?.onend?.()};
 },voices);
 let cursor=1,events=[],active=true,fail=false,polls=0,releases=0;
 await page.route('**/*',async route=>{
  const req=route.request(),u=new URL(req.url());
  if(u.pathname==='/admin-speech/poll'){
   const form=new URLSearchParams(req.postData());assert.equal(form.get('csrf'),'admin-csrf');
   if(form.get('enabled')==='0'){releases++;return route.fulfill({json:{ok:true,active:false,cursor,events:[]}})}
   polls++;if(fail)return route.fulfill({status:401,body:'login required'});
   const batch=form.get('baseline')==='1'||!active?[]:events;
   events=[];return route.fulfill({json:{ok:true,active,cursor,events:batch,native_suppressed:true}})
  }
  if(u.pathname==='/')return route.fulfill({contentType:'text/html',body:fs.readFileSync('/tmp/electron-admin-v1414.html')});
  const name=path.basename(u.pathname);
  for(const dir of ['extras','web']){const p=path.join(moduleRoot,dir,name);if(name&&fs.existsSync(p)&&fs.statSync(p).isFile())return route.fulfill({contentType:name.endsWith('.js')?'application/javascript':name.endsWith('.css')?'text/css':'text/html',body:fs.readFileSync(p)})}
  return route.fulfill({status:404,body:''});
 });
 await page.goto('http://127.0.0.1:18791/');
 return{context,page,errors,get polls(){return polls},get releases(){return releases},setEvents(e){events=e;cursor=e.length?Math.max(cursor,...e.map(x=>x.id)):cursor},setActive(a){active=a},setFail(f){fail=f}}
}
async function waitPoll(page,condition){await page.waitForFunction(condition);}
(async()=>{const browser=await chromium.launch({headless:true,executablePath:'/usr/bin/chromium',args:['--no-sandbox']});try{
 const results=[];
 for(const item of cases)for(const width of [1100,390]){
  const t=await setup(browser,item.voices,width),p=t.page;
  await p.locator('#electron-speech-toggle').click();
  await p.waitForFunction(()=>document.getElementById('electron-speech-preview').disabled===false);
  assert.equal(t.polls,1);const unlock=await p.evaluate(()=>window.__spoken[0]);assert.equal(unlock.userActivation,true);assert.equal(unlock.voice,item.expected);
  await p.evaluate(()=>window.__finish());
  t.setEvents([{id:3,text:'김작업님이 샘플 비고 메시지를 남겼습니다. 센서를 확인해 주세요.'},{id:2,text:'홍길동님이 샘플의 세팅 및 출하 통합검사를 신청했습니다.'},{id:2,text:'중복'}]);
  await p.clock.runFor(3000);await p.waitForFunction(()=>document.querySelector('#electron-speech-status').textContent.includes('속도'));
  await p.clock.runFor(900);await p.waitForFunction(()=>window.__spoken.length===2);
  let spoken=await p.evaluate(()=>window.__spoken);assert(spoken[1].text.startsWith('홍길동님이'));assert.equal(spoken[1].voice,item.expected);assert.equal(spoken[1].rate,1.1);assert.equal(spoken[1].pitch,1);assert.equal(spoken[1].lang,'ko-KR');
  assert.deepEqual(await p.evaluate(()=>window.__tones.slice(0,2)),[880,659.25]);
  await p.clock.runFor(1000);assert.equal(await p.evaluate(()=>window.__spoken.length),2,'second alert interrupted first');
  await p.evaluate(()=>window.__finish());await p.clock.runFor(900);await p.waitForFunction(()=>window.__spoken.length===3);
  assert((await p.evaluate(()=>window.__spoken[2].text)).startsWith('김작업님이'));
  await p.evaluate(()=>window.__finish());await p.clock.runFor(900);assert.equal(await p.evaluate(()=>window.__spoken.length),3,'duplicate ID spoken');
  await p.locator('#electron-speech-preview').click();await p.clock.runFor(900);await p.waitForFunction(()=>window.__spoken.length===4);
  if(item.name==='google')await p.screenshot({path:path.join(__dirname,`electron-speech-${width}-v1.4.14.png`),fullPage:false});
  await p.locator('#electron-speech-toggle').click();await p.waitForFunction(()=>document.getElementById('electron-speech-toggle').getAttribute('aria-pressed')==='false');
  assert(await p.evaluate(()=>window.__cancel>0));assert.equal(await p.evaluate(()=>window.__live),null);await p.clock.runFor(4000);assert.equal(await p.evaluate(()=>window.__spoken.length),4);
  assert.equal(await p.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false,'mobile overflow');assert.deepEqual(t.errors,[]);
  results.push({voice:item.name,width,selectedVoice:item.expected,rate:1.1,pitch:1,clickUnlock:'pass',fifoAndNoOverlap:'pass',duplicateSuppression:'pass',chimeBeforeMessage:'pass',offCancellation:'pass',noOverflow:'pass'});
  await t.context.close();
 }
 const t=await setup(browser,[],390),p=t.page;await p.locator('#electron-speech-toggle').click();await p.waitForFunction(()=>document.getElementById('electron-speech-preview').disabled===false);await p.evaluate(()=>window.__finish());
 await p.evaluate(()=>window.__voices=[{name:'Microsoft Heami - Korean (Korea)',lang:'ko-KR'}]);await p.locator('#electron-speech-preview').click();await p.clock.runFor(900);assert.equal(await p.evaluate(()=>window.__spoken.at(-1).voice),'Microsoft Heami - Korean (Korea)');await p.evaluate(()=>window.__finish());
 t.setActive(false);await p.clock.runFor(3100);await p.waitForFunction(()=>document.getElementById('electron-speech-status').textContent.includes('다른 관리자 창'));assert(await p.locator('#electron-speech-preview').isDisabled());
 t.setActive(true);await p.clock.runFor(3100);await p.waitForFunction(()=>document.getElementById('electron-speech-preview').disabled===false);
 t.setFail(true);await p.clock.runFor(3100);await p.waitForFunction(()=>document.getElementById('electron-speech').dataset.error==='true');assert(await p.locator('#electron-speech-preview').isDisabled());assert.deepEqual(t.errors,[]);await t.context.close();
 fs.writeFileSync(path.join(__dirname,'browser-electron-speech-v1.4.14.json'),JSON.stringify({scope:'Go-rendered synthetic admin HTML and mocked authenticated API, SpeechSynthesis and AudioContext; no actual installed voice/audio output claim',results,lateVoices:'pass',rivalTabAndTakeover:'pass',authenticationFailureStopsPlayback:'pass'},null,2)+'\n');
 console.log('Electron speech browser checks passed');
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
