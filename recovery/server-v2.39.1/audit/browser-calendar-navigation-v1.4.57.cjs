const {chromium}=require('playwright'),fs=require('fs'),assert=require('assert/strict');
const [origin,board]=process.argv.slice(2),results=[];
(async()=>{const browser=await chromium.launch({headless:true,executablePath:'/usr/bin/chromium',args:['--no-sandbox']});try{
for(const mobile of [true,false])for(const scheme of ['light','dark']){
 const ctx=await browser.newContext({viewport:{width:mobile?390:1100,height:950},isMobile:mobile,colorScheme:scheme});
 const p=await ctx.newPage();await p.addInitScript(()=>localStorage.setItem('alpha-speech-browser-enabled-v1','off'));
 for(const source of ['upper','bottom','menu']){
  let releaseDesign,finishRoute;const designHeld=new Promise(resolve=>releaseDesign=resolve),routeDone=new Promise(resolve=>finishRoute=resolve);
  await p.goto(origin+'/shipping-records/'+board+'?pending=1');await p.evaluate(()=>sessionStorage.clear());
  await p.route('**/mobile-design-v1.4.56.js',async route=>{await designHeld;await route.continue();finishRoute()});
  let link=p.locator('.shipment-tabs a').filter({hasText:'출고 일정'});
  if(source==='bottom')link=p.locator('.design-bottom-nav [data-tab=calendar]');
  if(source==='menu'){await p.locator('.design-bottom-nav [data-tab=menu]').click();link=p.locator('.design-menu-link').filter({hasText:'출고 일정'});}
  const [response]=await Promise.all([p.waitForNavigation({waitUntil:'commit'}),link.evaluate(el=>el.click())]);
  await p.locator('.work-list').waitFor({state:'attached'});
  // Force two paint opportunities while the deferred design script cannot run.
  await p.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
  assert.equal(await p.locator('body').getAttribute('data-design-tab'),'calendar');
  assert.equal(await p.locator('.work-list').isVisible(),false);
  assert.equal(await p.locator('.worker-tools').isVisible(),false);
  assert.equal(await p.locator('#shipping-calendar').evaluate(el=>el.open),false);
  results.push({mobile,scheme,source,phase:'before deferred design script; work list never exposed',result:'pass'});
  releaseDesign();await routeDone;await p.unroute('**/mobile-design-v1.4.56.js');
  assert.equal(new URL(p.url()).searchParams.get('entry'),'skip');
  assert(!(await response.text()).includes('data-entry-gate'));
  await p.waitForFunction(()=>document.body.dataset.designTab==='calendar'&&document.querySelector('#shipping-calendar')?.open);
  assert(await p.locator('#shipping-calendar').isVisible());
  await p.locator('.design-bottom-nav [data-tab=work]').click();assert(await p.locator('.work-list').isVisible());
  await p.locator('.design-bottom-nav [data-tab=calendar]').click();assert(await p.locator('#shipping-calendar').isVisible());
  results.push({mobile,scheme,source,result:'pass'});
 }
 const response=await p.goto(origin+'/work/'+board+'?entry=gate');assert((await response.text()).includes('data-entry-gate'));results.push({mobile,scheme,source:'initial entry retained',result:'pass'});
 await ctx.close();
}
fs.writeFileSync('audit/browser-calendar-navigation-v1.4.57.json',JSON.stringify({scope:'actual temporary Go server; design script held until after two animation frames; initial worker list hidden before script; fresh session storage; phone/PC light/dark; no company data',results},null,2)+'\n');console.log('Calendar navigation checks passed',results.length);
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
