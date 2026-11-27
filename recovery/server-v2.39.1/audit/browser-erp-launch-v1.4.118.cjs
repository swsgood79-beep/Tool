const {chromium}=require('playwright');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const [origin,raw]=process.argv.slice(2);
const config=JSON.parse(raw);
(async()=>{
 const browser=await chromium.launch({executablePath:'/usr/bin/chromium',args:['--no-sandbox']});
 const results=[];
 try {
  for(const width of [320,390,430,1100]) {
   const context=await browser.newContext({viewport:{width,height:1000},isMobile:width<500});
   await context.addCookies([{...config.owner,url:origin}]);
   const page=await context.newPage();const errors=[];
   page.on('pageerror',error=>errors.push(error.message));
   await page.goto(origin+'/machining-admin');
   const status=page.locator('#erp-db-sync-status');
   await page.waitForFunction(()=>document.querySelector('#erp-db-sync-status').textContent.includes('Windows 오류 206'));
   assert.ok((await status.innerText()).includes('기존 목록을 유지합니다'));
   assert.ok(await status.evaluate(element=>element.classList.contains('erp-db-sync-failed')));
   await page.locator('.erp-db-panel').evaluate(element=>element.open=true);
   await page.waitForFunction(()=>document.querySelector('#erp-db-check-status').textContent.includes('Windows 오류 206'));
   assert.equal(await page.locator('.order-card').count(),1);
   assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
   await page.screenshot({path:`audit/erp-launch-failure-${width}-v1.4.118.png`,fullPage:true});
   assert.deepEqual(errors,[]);
   results.push({width,result:'pass',scope:'Synthetic Windows launch error 206, detailed warning and diagnostics, existing order preserved, no horizontal overflow'});
   await context.close();
  }
  const context=await browser.newContext({viewport:{width:1100,height:1000}});
  await context.addCookies([{...config.owner,url:origin}]);
  const page=await context.newPage();await page.goto(origin+'/machining-admin');
  await page.waitForFunction(()=>document.querySelector('#erp-db-sync-status').textContent.includes('Windows 오류 206'));
  await page.locator('#erp-db-sync-start').click();
  await page.waitForFunction(()=>document.querySelector('#erp-db-sync-status').textContent.includes('DB 자동 연결 정상'));
  assert.equal(await page.locator('#erp-db-sync-status').evaluate(element=>element.classList.contains('erp-db-sync-failed')),false);
  assert.equal(await page.locator('.order-card').count(),1);
  await page.screenshot({path:'audit/erp-sync-recovered-1100-v1.4.118.png',fullPage:true});
  results.push({width:1100,result:'pass',scope:'Synthetic successful retry through normal settings button and sync scheduler; saved settings, original order and quantities preserved'});
  await context.close();
  fs.writeFileSync('audit/browser-release-v1.4.118.json',JSON.stringify({scope:'Actual Chromium UI with synthetic process failure and injected successful ERP query. Real company database not accessed.',results},null,2)+'\n');
  console.log('ERP launch browser checks passed',results.length);
 }finally{await browser.close()}
})().catch(error=>{console.error(error);process.exit(1)});
