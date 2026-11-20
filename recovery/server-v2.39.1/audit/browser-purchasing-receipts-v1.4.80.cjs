const {chromium}=require('playwright'),assert=require('assert/strict'),fs=require('fs');const [origin,raw]=process.argv.slice(2),cfg=JSON.parse(raw),results=[];
(async()=>{const browser=await chromium.launch({executablePath:'/usr/bin/chromium',args:['--no-sandbox']});try{let i=0;for(const mobile of [false,true])for(const theme of ['light','dark']){
 const viewport={width:mobile?390:1100,height:950},ctx=await browser.newContext({viewport,isMobile:mobile,colorScheme:theme}),worker=await ctx.newPage(),dialogs=[],errors=[];
 worker.on('dialog',async d=>{dialogs.push(d.type());await d.dismiss()});worker.on('pageerror',e=>errors.push(e.message));await ctx.addInitScript(()=>localStorage.setItem('alpha-speech-browser-enabled-v1','off'));
 await worker.clock.install();await worker.goto(origin+'/work/'+cfg.board+'?entry=skip');await worker.locator('.purchasing-call-button').first().waitFor();
 for(const [index,name]of [i+' 김승준',i+' 강준필'].entries()){
  await worker.locator('.purchasing-call-button').first().click();assert.equal(await worker.locator('.purchasing-worker-dialog select,.purchasing-worker-dialog textarea').count(),0);
  await worker.locator('.purchasing-worker-dialog [name=worker]').fill(name);
  if(index===0){let first=true;await worker.route('**/purchasing-call/**',async route=>{if(first){first=false;return route.fulfill({status:400,body:'시험 호출 실패'})}return route.continue()});
   await worker.locator('.purchasing-worker-dialog button[type=submit]').click();await worker.waitForFunction(()=>document.querySelector('.purchasing-worker-dialog .error').textContent==='시험 호출 실패',null,{timeout:10000});assert.equal(await worker.locator('.purchasing-worker-dialog [name=worker]').inputValue(),name);
  }
  await worker.locator('.purchasing-worker-dialog button[type=submit]').click();await worker.locator('.purchasing-worker-dialog').waitFor({state:'hidden'});assert.equal(worker.url(),origin+'/work/'+cfg.board+'?entry=skip');assert.equal(await worker.locator('[data-purchasing-history]').count(),0);
 }
 await worker.waitForFunction(()=>document.querySelectorAll('.purchasing-receipt').length===2);
 assert.match(await worker.locator('.purchasing-receipts').first().innerText(),/접수 완료/);
 await worker.reload();await worker.waitForFunction(()=>document.querySelectorAll('.purchasing-receipt').length===2);
 assert.equal(await worker.locator('.purchasing-call-button').first().isEnabled(),true);
 const stranger=await browser.newContext({viewport});const strangerPage=await stranger.newPage();await strangerPage.goto(origin+'/work/'+cfg.board+'?entry=skip');await strangerPage.waitForTimeout(600);assert.equal(await strangerPage.locator('.purchasing-receipt').count(),0);await stranger.close();
 await worker.screenshot({path:'audit/receipt-'+(mobile?'phone':'PC')+'-'+theme+'-v1.4.80.png',fullPage:true});
 const buyerCtx=await browser.newContext({viewport,isMobile:mobile,colorScheme:theme,...(mobile?{userAgent:'Mozilla/5.0 (Linux; Android 14) Chrome/131 Mobile Safari/537.36 AlphaAdminAndroid/1.0.10'}:{})});await buyerCtx.addCookies([{...cfg.cookie,url:origin}]);const buyer=await buyerCtx.newPage();buyer.on('dialog',async d=>{dialogs.push(d.type());await d.dismiss()});buyer.on('pageerror',e=>errors.push(e.message));
 await buyer.goto(origin+'/purchasing');await buyer.locator('.purchase-row').first().waitFor();assert.equal(await buyer.locator('.purchase-row').count(),2);assert.match(await buyer.locator('.purchase-row').first().innerText(),new RegExp(i+' 김승준'));assert.equal(await buyer.locator('.purchase-nav button').count(),3);
 assert.equal(await buyer.locator('a[href="/inspection-requests"],a[href="/projects/new"],a[href="/shipping-archive-admin"]').count(),0);
 await buyer.screenshot({path:'audit/purchasing-queue-'+(mobile?'phone':'PC')+'-'+theme+'-v1.4.80.png',fullPage:true});
 await buyer.locator('#purchase-first').click();await buyer.waitForFunction(()=>!document.getElementById('purchase-detail').hidden&&document.getElementById('purchase-caller').textContent.includes('김승준'),null,{timeout:10000});
 assert.match(await buyer.locator('#purchase-next-worker').innerText(),new RegExp(i+' 강준필'));await buyer.screenshot({path:'audit/purchasing-detail-'+(mobile?'phone':'PC')+'-'+theme+'-v1.4.80.png',fullPage:true});
 let firstFinish=true;await buyer.route('**/purchasing/complete',async route=>{if(firstFinish){firstFinish=false;return route.fulfill({status:400,body:'시험 완료 실패'})}return route.continue()});
 await buyer.locator('#purchase-finish').click();await buyer.waitForFunction(()=>!document.getElementById('purchase-finish').disabled&&!document.getElementById('purchase-message').hidden,null,{timeout:10000});assert.match(await buyer.locator('#purchase-caller').innerText(),/김승준/);
 await buyer.locator('#purchase-finish').click();await buyer.waitForFunction(()=>document.getElementById('purchase-caller').textContent.includes('강준필'),null,{timeout:10000});
 await worker.waitForFunction(()=>document.querySelectorAll('.purchasing-receipt.done').length===1,null,{timeout:12000});
 assert.equal(await worker.locator('.purchasing-receipt:not(.done)').count(),1);
 await worker.route('**/purchasing-receipts/**',route=>route.abort());await worker.clock.fastForward(61000);
 assert.equal(await worker.locator('.purchasing-receipt.done').count(),0);
 assert.equal(await worker.locator('.purchasing-receipt:not(.done)').count(),1);
 await worker.clock.resume();await worker.unroute('**/purchasing-receipts/**');
 await buyer.locator('#purchase-finish').click();await buyer.waitForFunction(()=>!document.getElementById('purchase-list').hidden&&document.getElementById('purchase-pending-count').textContent==='0',null,{timeout:10000});
 await buyer.locator('.purchase-nav [data-tab=done]').click();await buyer.locator('.purchase-row').first().waitFor();assert.equal(await buyer.locator('.purchase-row').count(),(i+1)*2);assert.match(await buyer.locator('.purchase-row').first().innerText(),new RegExp(i+' 강준필'));
 await buyer.locator('.purchase-nav [data-tab=menu]').click();await buyer.locator('#purchase-menu').waitFor({state:'visible'});assert.equal(await buyer.locator('#purchase-menu a').count(),4);assert.match(await buyer.locator('#purchase-menu').innerText(),/buyerbrowser/);
 await buyer.locator('#purchase-theme').selectOption(theme==='dark'?'light':'dark');assert.equal(await buyer.locator('body').getAttribute('data-theme'),theme==='dark'?'light':'dark');assert.deepEqual(dialogs,[]);assert.deepEqual(errors,[]);
 results.push({mobile,theme,case:'name-only worker call with retry/no navigation/history; own chronological queue; restricted menus; failed completion retains current call; complete advances and archives; themes; no leave prompt',result:'pass'});
 await buyerCtx.close();await ctx.close();i++;
 }
 fs.writeFileSync('audit/browser-purchasing-receipts-v1.4.80.json',JSON.stringify({scope:'actual temporary Go server + PC/mobile Chromium and shared APK user agent; native Android/real FCM not executed',results},null,2)+'\n');console.log('Purchasing browser checks passed',results.length);
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
