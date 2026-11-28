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
  const errors=[];
  page.on('pageerror',error=>errors.push(error.message));
  await page.goto(origin+'/machining-admin');
  await page.locator('.erp-db-panel>summary').click();
  await page.getByRole('button',{name:'ERP 화면 인식 확인',exact:true}).click();
  await page.locator('#erp-db-check-status').filter({hasText:'ERP 화면 인식 확인 완료'}).waitFor();
  await page.locator('#erp-db-check-download').waitFor({state:'visible'});
  const response=await context.request.get(origin+'/machining-admin/db-check/download');
  assert.equal(response.status(),200);
  assert(response.headers()['content-disposition'].includes('ERP-ui-receipt-check.json'));
  const data=await response.json();
  assert.equal(data.ui_receipt.saved,true);
  assert.equal(data.ui_receipt.elements[1].name,'TEST-PO / 12행 / TEST-PART');
  assert.equal(data.ui_receipt.elements[2].toggle,'Off');
  assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
  assert.equal(errors.length,0);
  await page.locator('#erp-ui-receipt-panel').screenshot({path:`audit/ui-receipt-${width}-v1.4.124.png`});
  results.push({width,result:'pass',scope:'actual implementation with synthetic UIA report; physical ERP Windows capture pending'});
  await context.close();
 }
 await browser.close();
 fs.writeFileSync('audit/browser-release-v1.4.124.json',JSON.stringify({results},null,2)+'\n');
})().catch(error=>{console.error(error);process.exit(1)});
