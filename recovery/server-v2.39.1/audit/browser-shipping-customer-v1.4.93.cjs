const {chromium}=require('playwright'),assert=require('assert/strict'),fs=require('fs');
const [origin,raw]=process.argv.slice(2),cfg=JSON.parse(raw),results=[];
(async()=>{const browser=await chromium.launch({executablePath:'/usr/bin/chromium',args:['--no-sandbox']});try {
 for(const mobile of [false,true])for(const theme of ['light','dark']) {
  const ctx=await browser.newContext({viewport:{width:mobile?390:1100,height:1000},isMobile:mobile,colorScheme:theme,...(mobile?{userAgent:'Mozilla/5.0 (Linux; Android 14) Chrome/131 Mobile Safari/537.36 AlphaAdminAndroid/1.0.11'}:{})});await ctx.addCookies([{...cfg.cookie,url:origin}]);
  const page=await ctx.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
  for(const admin of [true,false]) {
   const base=admin?'/shipping-archive-admin':'/shipping-records/'+cfg.board;
   for(const [index,record] of cfg.records.entries()) {
    await page.goto(origin+base+'/'+record.id);
    const info=await page.locator('.shipment-info').innerText();assert.match(info,new RegExp('고객사\\s+'+record.customer));assert.match(info,/출고일자/);assert.doesNotMatch(info,/미등록|출고 예정/);
    if(admin)await page.screenshot({path:'audit/customer-'+index+'-'+(mobile?'phone':'PC')+'-'+theme+'-v1.4.93.png',fullPage:true});
    results.push({admin,mobile,theme,customer:record.customer,result:'pass'});
   }
   await page.goto(origin+base+'?q='+encodeURIComponent('은일'));
   assert.equal(await page.locator('.shipment-row').count(),1);assert.match(await page.locator('.shipment-row small').innerText(),/은일/);
  }
  assert.deepEqual(errors,[]);await ctx.close();
 }
 fs.writeFileSync('audit/browser-shipping-customer-v1.4.93.json',JSON.stringify({scope:'synthetic missing/present archived customers; Chromium admin/worker desktop/mobile APK user agent light/dark; actual Windows/Android not executed',results},null,2)+'\n');console.log('Shipping customer browser checks passed',results.length);
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
