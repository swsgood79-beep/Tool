const {chromium}=require('playwright'),assert=require('assert/strict'),fs=require('fs');const [origin,raw]=process.argv.slice(2),cfg=JSON.parse(raw),results=[];
(async()=>{const browser=await chromium.launch({executablePath:'/usr/bin/chromium',args:['--no-sandbox']});try {
 for(const mobile of [false,true])for(const theme of ['light','dark']) {
  const ctx=await browser.newContext({viewport:{width:mobile?390:1100,height:1000},isMobile:mobile,colorScheme:theme,...(mobile?{userAgent:'Mozilla/5.0 (Linux; Android 14) Chrome/131 Mobile Safari/537.36 AlphaAdminAndroid/1.0.11'}:{})});await ctx.addCookies([{...cfg.cookie,url:origin}]);
  const page=await ctx.newPage(),errors=[],dialogs=[];page.on('pageerror',e=>errors.push(e.message));page.on('dialog',async d=>{dialogs.push(d.type());await d.dismiss()});
  await page.goto(origin+'/shipping-archive-admin/'+cfg.record);
  const info=await page.locator('.shipment-info').innerText(),inspection=await page.locator('.inspector-grid').innerText(),dates=await page.locator('.shipment-facts small').allInnerTexts();
  await page.locator('.shipment-worker-editor summary').click();
  const setting='세팅 수정 '+(mobile?'휴대폰':'PC')+' '+theme,shipping='출하 수정 '+(mobile?'휴대폰':'PC')+' '+theme;
  await page.locator('input[name=setting_worker]').fill(setting);await page.locator('input[name=shipping_worker]').fill(shipping);
  await page.screenshot({path:'audit/workers-edit-'+(mobile?'phone':'PC')+'-'+theme+'-v1.4.94.png',fullPage:true});
  await Promise.all([page.waitForNavigation(),page.getByRole('button',{name:'작업자 이름 저장',exact:true}).click()]);
  assert.match(await page.locator('main').innerText(),/작업자 이름을 수정했습니다/);const names=await page.locator('.shipment-facts').innerText();assert.ok(names.includes(setting)&&names.includes(shipping));
  assert.equal(await page.locator('.shipment-info').innerText(),info);assert.equal(await page.locator('.inspector-grid').innerText(),inspection);assert.deepEqual(await page.locator('.shipment-facts small').allInnerTexts(),dates);
  await page.goto(origin+'/shipping-records/'+cfg.board+'/'+cfg.record);
  assert.equal(await page.locator('.shipment-worker-editor').count(),0);assert.ok((await page.locator('.shipment-facts').innerText()).includes(setting));
  assert.deepEqual(errors,[]);assert.deepEqual(dialogs,[]);results.push({mobile,theme,result:'pass',case:'admin edit/save; worker read-only; dates/inspectors unchanged; no leave prompts'});await ctx.close();
 }
 fs.writeFileSync('audit/browser-shipping-workers-v1.4.94.json',JSON.stringify({scope:'synthetic archive; Chromium PC/mobile APK agent light/dark; actual Windows/Android not executed',results},null,2)+'\n');console.log('Shipping worker edit browser checks passed',results.length);
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
