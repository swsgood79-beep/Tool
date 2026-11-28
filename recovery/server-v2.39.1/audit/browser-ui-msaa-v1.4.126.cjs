const {chromium}=require('playwright');
const fs=require('fs');
const assert=require('assert');
(async()=>{
 const [origin,raw]=process.argv.slice(2);
 const cookie=JSON.parse(raw);
 const browser=await chromium.launch({executablePath:'/usr/bin/chromium',headless:true,args:['--no-sandbox']});
 const results=[];
 for(const width of [320,390,430,1100]){
  const context=await browser.newContext({viewport:{width,height:1100}});
  await context.addCookies([{...cookie,url:origin}]);
  const page=await context.newPage();
  const errors=[];page.on('pageerror',error=>errors.push(error.message));
  await page.goto(origin+'/machining-admin');
  await page.locator('.erp-db-panel>summary').click();
  await page.getByRole('button',{name:'ERP 화면 인식 확인',exact:true}).click();
  await page.locator('#erp-db-check-status').filter({hasText:'구형 접근성 4개'}).waitFor();
  const response=await context.request.get(origin+'/machining-admin/db-check/download');
  assert.equal(response.status(),200);
  const data=await response.json();
  assert.equal(data.ui_receipt.proxy_status,'registered');
  assert.equal(data.ui_receipt.msaa.elements[2].name,'가상 SIDE COVER');
  assert.equal(data.ui_receipt.msaa.elements[3].state,16);
  assert.equal(data.ui_receipt.legacy_managed_available,false);
  assert.equal(data.ui_receipt.saved,true);
  assert.equal(errors.length,0);
  assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
  await page.locator('.erp-db-panel').screenshot({path:`audit/ui-msaa-${width}-v1.4.126.png`});
  results.push({width,result:'pass',scope:'synthetic MSAA cell/check-state report; actual browser/protected save/download. Native Windows MSAA pending.'});
  await context.close();
 }
 await browser.close();
 fs.writeFileSync('audit/browser-release-v1.4.126.json',JSON.stringify({results},null,2)+'\n');
})().catch(error=>{console.error(error);process.exit(1)});
