const {chromium}=require('playwright');const fs=require('fs'),path=require('path'),assert=require('assert/strict');
(async()=>{const browser=await chromium.launch({headless:true,executablePath:'/usr/bin/chromium',args:['--no-sandbox']});try{const results=[];
for(const width of [1100,390]){
 const ctx=await browser.newContext({viewport:{width,height:860}}),page=await ctx.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));let submitted=0;
 const read=name=>fs.readFileSync('/tmp/review-v19-html/'+name,'utf8');let card=read('card.html');
 await page.route('**/*',async route=>{const url=new URL(route.request().url()),n=path.basename(url.pathname);
 if(url.pathname==='/work-fixture')return route.fulfill({contentType:'text/html',body:read('work.html')});
 if(url.pathname.startsWith('/card/'))return route.fulfill({contentType:'text/html',body:card});
 if(url.pathname.startsWith('/complete/')||url.pathname.startsWith('/combined/')){submitted++;card=read('pending-card.html');return route.fulfill({status:200,body:'accepted'})}
 if(url.pathname.startsWith('/project-photo/'))return route.fulfill({contentType:'image/svg+xml',body:'<svg xmlns="http://www.w3.org/2000/svg" width="180" height="100"><rect width="180" height="100" fill="#dae8fd"/><text x="10" y="55">TEST PHOTO</text></svg>'});
 for(const dir of ['extras','web']){const p=path.join(__dirname,'..',dir,n);if(n&&fs.existsSync(p)&&fs.statSync(p).isFile())return route.fulfill({contentType:n.endsWith('.js')?'application/javascript':'text/css',body:fs.readFileSync(p)})}
 return route.fulfill({status:404,body:''});
 });
 await page.goto('http://review.test/work-fixture');
 const summary=page.locator('summary.project-summary'),gate=page.locator('.worker-special-review-dialog[open]'),inspection=page.locator('.worker-inspection-dialog[open]');
 await summary.click();await gate.waitFor();assert.equal(await inspection.count(),0);assert.equal(submitted,0);
 assert.match(await gate.innerText(),/치수 확인/);assert.match(await gate.innerText(),/추가 메모 확인/);assert.equal(await gate.locator('img').count(),1);assert.equal(await gate.locator('script').count(),0);assert.equal(await gate.locator('.special-review-proceed').isDisabled(),true);
 await gate.locator('.special-review-cancel').click();assert.equal(await inspection.count(),0);
 await summary.click();await page.keyboard.press('Escape');await page.waitForFunction(()=>!document.querySelector('.worker-special-review-dialog[open]'));assert.equal(await inspection.count(),0);
 // Calendar uses this same custom event, after closing the calendar dialog.
 await page.locator('.work-item').evaluate(e=>e.dispatchEvent(new CustomEvent('alpha-open-inspection',{bubbles:true})));await gate.waitFor();
 assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false);assert.equal(await gate.evaluate(e=>e.scrollWidth>e.clientWidth+1),false);
 await page.screenshot({path:path.join(__dirname,`worker-special-review-${width}-v1.4.19.png`)});
 await gate.locator('input').check();await gate.locator('.special-review-proceed').click();await inspection.waitFor();assert.equal(await gate.count(),0);
 assert.equal(await inspection.locator('.note-panel textarea[name="note"]').count(),1);
 await inspection.locator('.worker-dialog-close').click();await summary.click();await gate.waitFor();assert.equal(await gate.locator('input').isChecked(),false);
 // A poll may replace the card while acknowledgement is visible. Never open a
 // detached/stale form, and require re-checking if information changed.
 card=read('card.html').replace('치수 확인','변경된 치수 확인').replace('data-info-revision="1"','data-info-revision="2"').replace('data-revision="1"','data-revision="2"');
 await page.evaluate(()=>window.AlphaWorkerView.refreshCard({id:1,revision:-1}));await gate.locator('input').check();await gate.locator('.special-review-proceed').click();await page.waitForFunction(()=>document.querySelector('.special-review-status')?.textContent.includes('변경되었습니다'));
 assert.match(await gate.innerText(),/변경된 치수 확인/);assert.equal(await gate.locator('input').isChecked(),false);await gate.locator('input').check();await gate.locator('.special-review-proceed').click();await inspection.waitFor();
 // Sending one request refreshes the pending UI without a second acknowledgement.
 await inspection.locator('.stage-setting input[name="worker"]').fill('테스트 신청자');await inspection.locator('.stage-setting button.primary').click();await page.waitForFunction(()=>document.querySelector('.worker-submit-status')?.textContent.includes('접수되었습니다'));
 assert.equal(submitted,1);assert.equal(await gate.count(),0);assert.equal(await inspection.count(),1);assert.match(await inspection.innerText(),/관리자 확인 대기/);
 // Empty instructions: no extra step.
 await inspection.locator('.worker-dialog-close').click();card=read('empty-card.html');await page.evaluate(()=>window.AlphaWorkerView.refreshCard({id:1,revision:-1}));await summary.click();await inspection.waitFor();assert.equal(await gate.count(),0);
 assert.deepEqual(errors,[]);results.push({width,textMemoPhotos:'pass',escapedText:'pass',checkboxRequired:'pass',cancelEscape:'pass',calendarEntry:'pass',everyManualOpening:'pass',changedDuringReview:'pass',requestRefreshNoExtraGate:'pass',notesRetained:'pass',emptyDirect:'pass',noOverflow:'pass'});await ctx.close();
}
fs.writeFileSync(path.join(__dirname,'browser-worker-special-review-v1.4.19.json'),JSON.stringify({scope:'Go-rendered synthetic worker cards; browser protocol simulated; no company data',results},null,2)+'\n');console.log('Worker special review browser checks passed');}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
