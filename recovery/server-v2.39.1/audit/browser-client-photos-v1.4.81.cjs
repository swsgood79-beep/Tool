const {chromium}=require('playwright'),fs=require('fs'),assert=require('assert/strict');
const [origin,raw]=process.argv.slice(2),cfg=JSON.parse(raw),results=[];
(async()=>{const browser=await chromium.launch({executablePath:'/usr/bin/chromium',args:['--no-sandbox']});try{let i=0;for(const mobile of [false,true])for(const theme of ['light','dark']){
 const ctx=await browser.newContext({viewport:{width:mobile?390:1100,height:950},isMobile:mobile,colorScheme:theme,...(mobile?{userAgent:'Mozilla/5.0 (Linux; Android 14) Chrome/131 Mobile Safari/537.36 AlphaAdminAndroid/1.0.11'}:{})});
 await ctx.addCookies([{...cfg.cookie,url:origin}]);const page=await ctx.newPage(),dialogs=[],errors=[];
 page.on('dialog',async d=>{dialogs.push(d.type());await d.dismiss()});page.on('pageerror',e=>errors.push(e.message));
 await page.goto(origin+'/projects/'+cfg.ids[i]+'/edit');
 assert.equal(await page.locator('.project-photo-grid .project-photo').count(),3);
 assert.equal(await page.locator('[name=sync_client]').isChecked(),true);
 assert.match(await page.locator('body').innerText(),/추가 사진은 이 수주에만 저장/);
 const original=await page.locator('.project-photo-grid .project-photo img').evaluateAll(xs=>xs.map(x=>x.getAttribute('src')));
 await page.locator('[name=additional_instructions]').fill('MOT 수주에만 필요한 별도 작업 주의사항');
 await page.locator('[name=photos]').setInputFiles({name:'MOT-order-only.png',mimeType:'image/png',buffer:Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAgAAAAICAIAAABLbSncAAAAFElEQVR4nGPUqDjBgA0wYRUdtBIAHOcBeDC+JtEAAAAASUVORK5CYII=','base64')});
 const saved=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/projects/'+cfg.ids[i]+'/edit'));
 await page.locator('button[type=submit]').filter({hasText:'수주 수정 저장'}).click();const response=await saved;assert.equal(response.status(),303);
 await page.waitForURL(origin+'/');
 await page.goto(origin+'/projects/'+cfg.ids[i]+'/edit');assert.equal(await page.locator('.project-photo-grid .project-photo').count(),4);
 assert.equal(await page.locator('[name=additional_instructions]').inputValue(),'MOT 수주에만 필요한 별도 작업 주의사항');
 const current=await page.locator('.project-photo-grid .project-photo img').evaluateAll(xs=>xs.map(x=>x.getAttribute('src')));assert.deepEqual(current.slice(0,3),original);
 for(const img of await page.locator('.project-photo-grid img').all()){await img.scrollIntoViewIfNeeded();await img.evaluate(n=>n.complete&&n.naturalWidth>0?Promise.resolve():new Promise((resolve,reject)=>{n.onload=resolve;n.onerror=()=>reject(Error('saved image failed to load'))}))}
 await page.screenshot({path:'audit/client-photos-'+(mobile?'phone':'PC')+'-'+theme+'-v1.4.81.png',fullPage:true});
 await page.goto(origin+'/project-info/'+cfg.board+'/'+cfg.ids[i]);assert.equal(await page.locator('img').count(),4);assert.match(await page.locator('body').innerText(),/MOT 수주에만 필요한 별도 작업 주의사항/);
 assert.deepEqual(dialogs,[]);assert.deepEqual(errors,[]);
 results.push({mobile,theme,case:'five company source photos, only three selected; append one order-only photo without reloading source or disabling sync; preserves original three, extra memo and source; admin and worker view; no leave prompt',result:'pass'});
 await ctx.close();i++;
}fs.writeFileSync('audit/browser-client-photos-v1.4.81.json',JSON.stringify({scope:'temporary Go server + PC/mobile Chromium and APK user agent; real company data, Windows and native Android not executed',results},null,2)+'\n');console.log('Client photo append browser checks passed',results.length)}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
