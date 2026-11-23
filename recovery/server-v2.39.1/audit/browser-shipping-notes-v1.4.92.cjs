const {chromium}=require('playwright'),assert=require('assert/strict'),fs=require('fs');
const [origin,raw]=process.argv.slice(2),cfg=JSON.parse(raw),results=[];
(async()=>{const browser=await chromium.launch({executablePath:'/usr/bin/chromium',args:['--no-sandbox']});try {
 for(const mobile of [false,true])for(const theme of ['light','dark']) {
  const ctx=await browser.newContext({viewport:{width:mobile?390:1100,height:1000},isMobile:mobile,colorScheme:theme,...(mobile?{userAgent:'Mozilla/5.0 (Linux; Android 14) Chrome/131 Mobile Safari/537.36 AlphaAdminAndroid/1.0.11'}:{})});await ctx.addCookies([{...cfg.cookie,url:origin}]);
  const page=await ctx.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
  for(const admin of [true,false]) {
   await page.goto(origin+(admin?'/shipping-archive-admin/':'/shipping-records/'+cfg.board+'/')+cfg.record);
   const pics=page.locator('[data-shipment-photo-group="special"] img');assert.equal(await pics.count(),2);
   await pics.evaluateAll(async images=>{await Promise.all(images.map(img=>{img.loading='eager';return img.decode()}))});
   assert.deepEqual(await pics.evaluateAll(images=>images.map(img=>img.naturalWidth>0)),[true,true]);
   const info=await page.locator('.shipment-info').innerText();assert.match(info,/출고일자/);assert.doesNotMatch(info,/출고 예정/);
   assert.equal(await page.locator('[data-shipment-photo-group="shipping"] img').count(),1);
   await page.screenshot({path:'audit/notes-'+(admin?'admin':'worker')+'-'+(mobile?'phone':'PC')+'-'+theme+'-v1.4.92.png',fullPage:true});
   results.push({admin,mobile,theme,result:'pass',specialPhotos:2,shippingPhotos:1,dateLabel:'출고일자'});
  }
  assert.deepEqual(errors,[]);await ctx.close();
 }
 fs.writeFileSync('audit/browser-shipping-notes-v1.4.92.json',JSON.stringify({scope:'synthetic copied/source photos; Chromium PC and mobile with APK user agent, light/dark; actual Windows/Android not executed',results},null,2)+'\n');console.log('Shipping notes duplicate/date browser checks passed',results.length);
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
