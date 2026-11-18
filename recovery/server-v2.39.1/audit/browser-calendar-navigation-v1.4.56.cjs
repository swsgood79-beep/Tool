const {chromium}=require('playwright'),fs=require('fs'),assert=require('assert/strict');
const [origin,board]=process.argv.slice(2),results=[];
(async()=>{const browser=await chromium.launch({headless:true,executablePath:'/usr/bin/chromium',args:['--no-sandbox']});try{
for(const mobile of [true,false])for(const scheme of ['light','dark']){
 const ctx=await browser.newContext({viewport:{width:mobile?390:1100,height:950},isMobile:mobile,colorScheme:scheme});
 const p=await ctx.newPage();await p.addInitScript(()=>localStorage.setItem('alpha-speech-browser-enabled-v1','off'));
 for(const source of ['upper','bottom','menu']){
  await p.goto(origin+'/shipping-records/'+board+'?pending=1');await p.evaluate(()=>sessionStorage.clear());
  let link=p.locator('.shipment-tabs a').filter({hasText:'출고 일정'});
  if(source==='bottom')link=p.locator('.design-bottom-nav [data-tab=calendar]');
  if(source==='menu'){await p.locator('.design-bottom-nav [data-tab=menu]').click();link=p.locator('.design-menu-link').filter({hasText:'출고 일정'});}
  const [response]=await Promise.all([p.waitForNavigation(),link.click()]);
  assert.equal(new URL(p.url()).searchParams.get('entry'),'skip');
  assert(!(await response.text()).includes('data-entry-gate'));
  await p.waitForFunction(()=>document.body.dataset.designTab==='calendar'&&document.querySelector('#shipping-calendar')?.open);
  assert(await p.locator('#shipping-calendar').isVisible());
  results.push({mobile,scheme,source,result:'pass'});
 }
 const response=await p.goto(origin+'/work/'+board+'?entry=gate');assert((await response.text()).includes('data-entry-gate'));results.push({mobile,scheme,source:'initial entry retained',result:'pass'});
 await ctx.close();
}
fs.writeFileSync('audit/browser-calendar-navigation-v1.4.56.json',JSON.stringify({scope:'actual temporary Go server; fresh session storage; phone/PC light/dark; no company data',results},null,2)+'\n');console.log('Calendar navigation checks passed',results.length);
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
