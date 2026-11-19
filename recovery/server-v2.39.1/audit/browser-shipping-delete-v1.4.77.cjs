const {chromium}=require('playwright'),fs=require('fs'),assert=require('assert/strict');
const [origin,board,raw]=process.argv.slice(2),cfg=JSON.parse(raw),results=[];
(async()=>{const b=await chromium.launch({executablePath:'/usr/bin/chromium',args:['--no-sandbox']});try{let i=0;for(const mobile of [false,true])for(const theme of ['light','dark']){
 const ctx=await b.newContext({viewport:{width:mobile?390:1100,height:950},isMobile:mobile,colorScheme:theme,...(mobile?{userAgent:'Mozilla/5.0 (Linux; Android 14) Chrome/131 Mobile Safari/537.36 AlphaAdminAndroid/1.0.9'}:{})});
 await ctx.addCookies([{...cfg.cookie,url:origin}]);await ctx.addInitScript(()=>localStorage.setItem('alpha-speech-browser-enabled-v1','off'));
 const p=await ctx.newPage(),dialogs=[],errors=[];p.on('dialog',async d=>{dialogs.push(d.type());await d.dismiss()});p.on('pageerror',e=>errors.push(e.message));
 await p.goto(origin+'/shipping-records/'+board+'/'+cfg.ids[i]);assert.equal(await p.locator('[data-delete-shipping-photo]').count(),0);
 await p.goto(origin+'/shipping-archive-admin/'+cfg.ids[i]);
 const gallery=p.locator('[data-shipment-photo-group=shipping]'),original=await gallery.locator('img').evaluateAll(xs=>xs.map(x=>x.getAttribute('src'))),info=await p.locator('.shipment-info').textContent();
 await p.locator('[data-delete-shipping-photo]').nth(1).click();await p.locator('.shipment-delete-dialog').waitFor({state:'visible'});
 assert.equal(await p.locator('.shipment-delete-dialog>img').getAttribute('src'),origin+original[1]);await p.locator('.shipment-delete-dialog button').first().click();assert.equal(await gallery.locator('figure').count(),3);
 await p.locator('[data-delete-shipping-photo]').nth(1).click();let first=true;
 await p.route('**/delete-photo',async route=>{if(first){first=false;await route.fulfill({status:400,body:'시험 삭제 실패'});}else await route.continue()});
 await p.locator('.shipment-delete-dialog button').last().click();await p.waitForFunction(()=>document.querySelector('.shipment-delete-dialog .error').textContent==='시험 삭제 실패',null,{timeout:10000});assert.equal(await gallery.locator('figure').count(),3);
 await p.locator('.shipment-delete-dialog button').last().click();await p.waitForURL('**photo_deleted=1');
 assert.deepEqual(await gallery.locator('img').evaluateAll(xs=>xs.map(x=>x.getAttribute('src'))),[original[0],original[2]]);assert.equal(await p.locator('.shipment-info').textContent(),info);
 // Upload draft is retained and blocks destructive actions until explicitly canceled.
 await p.locator('#shipment-add-open').click();await p.locator('#shipment-album').setInputFiles({name:'packing.png',mimeType:'image/png',buffer:Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAgAAAAICAIAAABLbSncAAAAFElEQVR4nGPUqDjBgA0wYRUdtBIAHOcBeDC+JtEAAAAASUVORK5CYII=','base64')});
 await p.locator('[data-delete-shipping-photo]').first().click();assert.equal(await p.locator('.shipment-delete-dialog[open]').count(),0);assert.equal(await p.locator('.shipment-preview-item').count(),1);await p.locator('#shipment-add-cancel').click();
 for(const remaining of [1,0]){
  await p.locator('[data-delete-shipping-photo]').first().click();if(remaining===0)assert.match(await p.locator('.shipment-delete-dialog').innerText(),/출하사진 필요/);
  const navigation=p.waitForNavigation({waitUntil:'load',timeout:10000});await p.locator('.shipment-delete-dialog button').last().click();await navigation;
  if(remaining)await p.locator('[data-delete-shipping-photo]').waitFor();
 }
 assert.equal(await p.locator('[data-shipping-pending-page]').count(),1);assert.equal(await gallery.locator('figure').count(),0);assert.deepEqual(dialogs,[]);assert.deepEqual(errors,[]);
 results.push({mobile,theme,case:'worker cannot delete; cancel and failure keep photos; delete middle only; original archive date preserved; upload draft protected; final deletion requeues without leave prompt',result:'pass'});await ctx.close();i++;
}fs.writeFileSync('audit/browser-shipping-delete-v1.4.77.json',JSON.stringify({scope:'actual Go server + PC/mobile Chromium and APK user agent; native Android not executed',results},null,2)+'\n');console.log('Photo deletion browser checks passed',results.length)}finally{await b.close()}})().catch(e=>{console.error(e);process.exit(1)});
