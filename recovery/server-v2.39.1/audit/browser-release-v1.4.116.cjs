const {chromium} = require('playwright');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const [origin, raw] = process.argv.slice(2);
const cfg = JSON.parse(raw);
(async () => {
 const browser = await chromium.launch({executablePath:'/usr/bin/chromium',args:['--no-sandbox']});
 const results=[];
 try {
  for (const width of [320,360,390,430,1100]) {
   const context=await browser.newContext({viewport:{width,height:900},isMobile:width<500,hasTouch:width<500});
   await context.addCookies([{...cfg.worker,url:origin}]);
   const page=await context.newPage();const errors=[];
   page.on('pageerror',e=>errors.push(e.message));
   await page.goto(origin+'/machining');
   await page.locator('.outstanding-open').first().click();
   const dialog=page.locator('.outstanding-dialog');
   await dialog.getByText('3품목',{exact:true}).waitFor();
   assert.equal(await dialog.locator('h3').count(),2);
   assert.ok((await dialog.innerText()).includes('로봇품번 확인 불가'));
   assert.equal(await dialog.locator('table').first().locator('th').count(),4);
   assert.ok((await dialog.innerText()).includes('1.5'));
   const layout=await dialog.evaluate(d=>({page:document.documentElement.scrollWidth>innerWidth,dialog:d.scrollWidth>d.clientWidth+1,heading:d.querySelector('h3').scrollWidth>d.querySelector('h3').clientWidth+1}));
   assert.deepEqual(layout,{page:false,dialog:false,heading:false});
   await page.screenshot({path:`audit/outstanding-${width}-v1.4.116.png`,fullPage:true});
   await page.route('**/machining/outstanding?*',route=>route.fulfill({status:502,contentType:'application/json',body:JSON.stringify({ok:false,error:'조회 결과가 한도를 초과해 일부가 잘렸습니다'})}));
   await dialog.getByRole('button',{name:'새로고침',exact:true}).click();
   await dialog.getByText('조회 결과가 한도를 초과해 일부가 잘렸습니다',{exact:true}).waitFor();
   assert.equal(await dialog.getByText('미입고 없음',{exact:true}).count(),0);
   assert.equal(await dialog.locator('table').count(),0);
   await page.unroute('**/machining/outstanding?*');
   await dialog.getByRole('button',{name:'새로고침',exact:true}).click();
   await dialog.getByText('3품목',{exact:true}).waitFor();
   assert.deepEqual(errors,[]);
   results.push({width,result:'pass',scope:'real synthetic ERP response, full robot heading, four columns, no overflow, truncated error and refresh recovery'});
   await context.close();
  }
  for(const width of [390,1100]) {
   const context=await browser.newContext({viewport:{width,height:900}});
   await context.addCookies([{...cfg.owner,url:origin}]);const page=await context.newPage();
   await page.goto(origin+'/machining-admin');
   await page.getByRole('link',{name:'가공 테스트 기록 초기화',exact:true}).click();
   await page.locator('select[name=account]').selectOption('target');
   await page.getByRole('button',{name:'기록 미리보기',exact:true}).click();
   await page.getByText('초기화 가능 3건 · 보류 2건 · 연결 완료 알림 1건',{exact:true}).waitFor();
   assert.equal(await page.locator('input[name=confirmed]').isChecked(),false);
   assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
   await page.screenshot({path:`audit/reset-preview-${width}-v1.4.116.png`,fullPage:true});
   if(width===1100) {
    await page.locator('input[name=confirmed]').check();
    await page.getByRole('button',{name:'백업 후 초기화',exact:true}).click();
    await page.getByText('백업 후 초기화했습니다',{exact:true}).waitFor();
    await page.getByText('초기화 가능 0건 · 보류 2건 · 연결 완료 알림 0건',{exact:true}).waitFor();
    await page.screenshot({path:'audit/reset-complete-1100-v1.4.116.png',fullPage:true});
   }
   results.push({width,result:'pass',scope:'owner reset preview with held dependencies and receipt, explicit unchecked confirmation'+(width===1100?', synthetic backup/reset/audit':'' )});
   await context.close();
  }
  fs.writeFileSync('audit/browser-release-v1.4.116.json',JSON.stringify({scope:'Actual implementation in Chromium with synthetic records and injected synthetic ERP response. Company ERP and physical Windows/Android remain untested.',results},null,2)+'\n');
  console.log('Release 116 browser checks passed',results.length);
 }finally{await browser.close()}
})().catch(error=>{console.error(error);process.exit(1)});
