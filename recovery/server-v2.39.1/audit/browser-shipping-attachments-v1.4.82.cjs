const {chromium}=require('playwright'),assert=require('assert/strict'),fs=require('fs');const[origin,raw]=process.argv.slice(2),cfg=JSON.parse(raw),results=[];
(async()=>{const browser=await chromium.launch({executablePath:'/usr/bin/chromium',args:['--no-sandbox']});try{let i=0;for(const mobile of [false,true])for(const theme of ['light','dark']){
 const admin=theme==='dark',ctx=await browser.newContext({viewport:{width:mobile?390:1100,height:1000},isMobile:mobile,colorScheme:theme,...(mobile&&admin?{userAgent:'Mozilla/5.0 (Linux; Android 14) Chrome/131 Mobile Safari/537.36 AlphaAdminAndroid/1.0.11'}:{})});if(admin)await ctx.addCookies([{...cfg.cookie,url:origin}]);
 assert.ok(fs.statSync('/tmp/alpha-shipping-video.mp4').size>20*1024*1024,'fixture must exercise production auto-compression threshold');
 const page=await ctx.newPage(),dialogs=[],errors=[];page.on('dialog',async d=>{dialogs.push(d.type());await d.dismiss()});page.on('pageerror',e=>errors.push(e.message));
 const base=origin+(admin?'/shipping-archive-admin/':'/shipping-records/'+cfg.board+'/')+cfg.ids[i];await page.goto(base);await page.locator('#shipment-attachment-form').waitFor();
 await page.locator('[data-kind=file]').setInputFiles({name:'리니어-파라미터.prm',mimeType:'application/octet-stream',buffer:Buffer.from('linear-axis.parameter=1234\r\n')});
 await page.locator('[data-kind=video]').setInputFiles('/tmp/alpha-shipping-video.mp4');assert.equal(await page.locator('.shipment-attachment-draft-row').count(),2);
 if(admin)await page.locator('#shipment-attachment-save').click();else{
  await page.locator('#shipment-album').setInputFiles({name:'packing.png',mimeType:'image/png',buffer:Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAgAAAAICAIAAABLbSncAAAAFElEQVR4nGPUqDjBgA0wYRUdtBIAHOcBeDC+JtEAAAAASUVORK5CYII=','base64')});
  await page.locator('#shipment-finish').click();
 }
 await page.waitForFunction(()=>document.querySelectorAll('[data-attachment-row]').length===2||!document.getElementById('shipment-attachment-error')?.hidden,null,{timeout:90000});
 const error=await page.locator('#shipment-attachment-error').textContent();assert.equal(error,'','upload/compression error: '+error);
 if(!admin)await page.waitForURL('**saved=1',{timeout:15000});
 await page.goto(base);assert.equal(await page.locator('[data-attachment-row]').count(),2);assert.equal(await page.locator('[data-attachment-delete]').count(),admin?2:0);
 const param=page.locator('[data-attachment-row]').filter({hasText:'리니어-파라미터.prm'});const url=await param.locator('a').getAttribute('href');const response=await ctx.request.get(origin+url);assert.equal(response.status(),200);assert.equal((await response.body()).toString(),'linear-axis.parameter=1234\r\n');assert.match(response.headers()['content-disposition'],/attachment/);
 const video=page.locator('[data-attachment-row] video');await video.scrollIntoViewIfNeeded();await video.evaluate(v=>v.play());await page.waitForFunction(()=>{const v=document.querySelector('[data-attachment-row] video');return v.videoWidth>0&&v.currentTime>.2});
 const dimensions=await video.evaluate(v=>({width:v.videoWidth,height:v.videoHeight,duration:v.duration,src:v.getAttribute('src')}));assert.ok(dimensions.height>dimensions.width);assert.ok(Math.max(dimensions.width,dimensions.height)<=1280);assert.ok(dimensions.duration>4.5&&dimensions.duration<6.5);
 const downloaded=await ctx.request.get(origin+dimensions.src+'?download=1');const compressed=await downloaded.body();assert.ok(compressed.length<fs.statSync('/tmp/alpha-shipping-video.mp4').size);fs.writeFileSync('/tmp/alpha-compressed-'+i+'.webm',compressed);
 const range=await ctx.request.get(origin+dimensions.src,{headers:{Range:'bytes=0-31'}});assert.equal(range.status(),206);assert.equal((await range.body()).length,32);
 await video.evaluate(v=>v.pause());if(admin){await param.locator('[data-attachment-delete]').click();await page.locator('.shipment-attachment-dialog .danger').click();await param.waitFor({state:'detached'});assert.equal(await page.locator('[data-attachment-row] video').count(),1)}
 await page.screenshot({path:'audit/shipping-attachments-'+(mobile?'phone':'PC')+'-'+theme+'-v1.4.82.png',fullPage:true});assert.deepEqual(dialogs,[]);assert.deepEqual(errors,[]);
 results.push({mobile,theme,admin,dimensions,sourceBytes:fs.statSync('/tmp/alpha-shipping-video.mp4').size,compressedBytes:compressed.length,result:'pass'});await ctx.close();i++;
 }fs.writeFileSync('audit/browser-shipping-attachments-v1.4.82.json',JSON.stringify({scope:'real temporary server and Chromium video encoder; 20MB+ portrait MP4 with audio; production compression threshold; PC/mobile viewport and unchanged APK user agent; not physical Android',results},null,2)+'\n');console.log('Shipping attachments and video compression browser passed',results.length)}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
