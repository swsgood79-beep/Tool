const {chromium} = require('playwright');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const [origin, raw] = process.argv.slice(2);
const config = JSON.parse(raw);
(async () => {
 const browser=await chromium.launch({executablePath:'/usr/bin/chromium',args:['--no-sandbox']});
 const results=[];
 try {
  for(const width of [320,360,390,430,1100]) {
   const context=await browser.newContext({viewport:{width,height:900},isMobile:width<500,hasTouch:width<500});
   await context.addCookies([{...config.worker,url:origin}]);
   const page=await context.newPage();const errors=[];
   page.on('pageerror',error=>errors.push(error.message));
   await page.goto(origin+'/machining');
   await page.getByRole('button',{name:'미입고 현황 보기',exact:true}).click();
   const dialog=page.locator('.outstanding-dialog');
   await dialog.getByText('3품목',{exact:true}).waitFor();
   const rows=dialog.locator('tbody tr');
   assert.equal(await rows.count(),3);
   assert.equal(await rows.nth(0).locator('td').nth(2).innerText(),'가상 공통 거래처');
   assert.equal(await rows.nth(1).locator('td').nth(2).innerText(),'가상 가공 거래처');
   assert.equal(await rows.nth(2).locator('td').nth(2).innerText(),'확인 불가');
   assert.equal(await rows.nth(1).locator('td').nth(1).innerText(),'1.5');
   assert.equal(await dialog.getByText('가상 다른 사업장 업체',{exact:true}).count(),0);
   assert.equal(await dialog.getByText('가상 후보 업체 A',{exact:true}).count(),0);
   assert.equal(await dialog.evaluate(element=>document.documentElement.scrollWidth>innerWidth || element.scrollWidth>element.clientWidth+1),false);
   await page.screenshot({path:`audit/supplier-${width}-v1.4.117.png`,fullPage:true});
   await dialog.getByRole('button',{name:'새로고침',exact:true}).click();
   await dialog.getByText('3품목',{exact:true}).waitFor();
   assert.equal(await rows.count(),3);
   assert.deepEqual(errors,[]);
   results.push({width,result:'pass',scope:'Synthetic probe JSON decoded and resolved through the actual endpoint. Shared-unit supplier, exact-unit priority, conflicting names left unknown, quantities and row counts preserved, no overflow, refresh.'});
   await context.close();
  }
  fs.writeFileSync('audit/browser-release-v1.4.117.json',JSON.stringify({scope:'Actual Chromium implementation using synthetic ERP probe records. Actual company ERP not accessed.',results},null,2)+'\n');
  console.log('Supplier browser checks passed',results.length);
 } finally {await browser.close()}
})().catch(error=>{console.error(error);process.exit(1)});
