const {chromium} = require('playwright');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const [origin, raw] = process.argv.slice(2);
const config = JSON.parse(raw);
(async () => {
 const browser = await chromium.launch({executablePath:'/usr/bin/chromium',args:['--no-sandbox']});
 const results = [];
 const path = role => role === 'worker' ? `/work/${config.board}` : '/?design-tab=orders';
 try {
  for (const role of ['worker','admin']) for (const width of [320,360,390,430,1100]) for (const theme of ['light','dark']) {
   const context = await browser.newContext({viewport:{width,height:1000},isMobile:width<500,hasTouch:width<500,colorScheme:theme});
   if (role === 'admin') await context.addCookies([{...config.owner,url:origin}]);
   const page = await context.newPage();
   const errors = [];
   page.on('pageerror', error => errors.push(error.message));
   await page.goto(origin + path(role));
   const card = page.locator(`${role==='worker'?'.work-item':'.admin-project'}[data-project="${config.first}"]`);
   const empty = page.locator(`${role==='worker'?'.work-item':'.admin-project'}[data-project="${config.second}"]`);
   const button = card.locator('.order-outstanding-button');
   await page.waitForFunction(id => {
    const button = document.querySelector(`[data-project="${id}"] .order-outstanding-button`);
    return button?.dataset.state === 'ready' && !button.hidden;
   }, config.first);
   await page.waitForFunction(id => {
    const button = document.querySelector(`[data-project="${id}"] .order-outstanding-button`);
    return button?.dataset.state === 'ready' && button.hidden;
   }, config.second);
   const inspection = card.locator(role==='worker'?'.design-order-footer .expand-hint':'.e-admin-footer .e-action');
   const position = await button.boundingBox();
   const target = await inspection.boundingBox();
   assert.ok(position && target);
   assert.ok(position.x + position.width <= target.x + 1, JSON.stringify({role,width,position,target}));
   assert.ok(Math.abs(position.y-target.y)<2, JSON.stringify({role,width,position,target}));
   assert.ok(Math.abs(position.height-target.height)<2);
   assert.equal(await empty.locator('.order-outstanding-button').isVisible(),false);
   const overflow = await card.evaluate(element => ({page:document.documentElement.scrollWidth>innerWidth,card:element.scrollWidth>element.clientWidth+1}));
   assert.deepEqual(overflow,{page:false,card:false});
   if (role==='worker') {
    const purchase = await card.locator('.purchasing-call-button').boundingBox();
    assert.ok(purchase.y>=position.y+position.height);
   }
   await card.screenshot({path:`audit/basic-card-${role}-${width}-${theme}-v1.4.119.png`});
   if (width===390 && theme==='light') await page.screenshot({path:`audit/basic-page-${role}-390-v1.4.119.png`,fullPage:true});
   await button.focus();
   await page.keyboard.press('Enter');
   const dialog = page.locator('.outstanding-dialog');
   await dialog.getByText('2품목',{exact:true}).waitFor();
   assert.equal(await page.locator('dialog[open]').count(),1);
   assert.equal(await card.getAttribute('open'),null);
   assert.equal(await dialog.locator('h3').innerText(),'TEST-ROBOT-PR2-400(530)-20N-N-EL-V3-LONG-ROBOT-PART-NUMBER-2026');
   assert.equal(await dialog.locator('th').count(),4);
   assert.ok((await dialog.innerText()).includes('가상 업체 A'));
   assert.ok((await dialog.innerText()).includes('1.5'));
   assert.equal(await dialog.evaluate(element=>element.scrollWidth>element.clientWidth+1),false);
   await page.screenshot({path:`audit/basic-popup-${role}-${width}-${theme}-v1.4.119.png`});
   await dialog.getByRole('button',{name:'닫기',exact:true}).click();
   if (role==='worker') {
    await card.locator('.purchasing-call-button').click();
    const purchase = page.locator('.purchasing-worker-dialog');
    await purchase.waitFor({state:'visible'});
    await purchase.getByRole('button',{name:'닫기',exact:true}).click();
    await inspection.click();
    const gate = page.locator('dialog[open]');
    if (await gate.locator('input[type=checkbox]').count()) {
     await gate.locator('input[type=checkbox]').check();
     await gate.locator('.special-review-proceed').click();
    }
    await card.locator('.worker-inspection-dialog[open]').waitFor();
    await card.locator('.worker-dialog-close').click();
   } else {
    await inspection.click();
    await page.waitForURL(`**/projects/${config.first}/inspection`);
    assert.equal(await page.locator('#order-outstanding-config').count(),0);
   }
   assert.deepEqual(errors,[]);
   results.push({role,width,theme,result:'pass',scope:'Actual basic card: adjacent buttons, verified zero hidden, independent keyboard popup, full robot text and four columns, no overflow, existing inspection and purchase-call actions'});
   await context.close();
  }
  {
   const context = await browser.newContext({viewport:{width:390,height:1000}});
   const page = await context.newPage();
   await page.goto(origin+path('worker'));
   const card = page.locator(`.work-item[data-project="${config.first}"]`);
   const button = card.locator('.order-outstanding-button');
   await page.waitForFunction(id=>document.querySelector(`[data-project="${id}"] .order-outstanding-button`)?.dataset.state==='ready',config.first);
   await button.click();
   const dialog = page.locator('.outstanding-dialog');
   await dialog.getByText('2품목',{exact:true}).waitFor();
   await page.request.get(origin+'/synthetic-outstanding-mode?value=error');
   await dialog.getByRole('button',{name:'새로고침',exact:true}).click();
   await dialog.getByText('조회 결과가 한도를 초과해 일부가 잘렸습니다. 전체 미입고 현황을 확인할 수 없습니다',{exact:true}).waitFor();
   assert.equal(await dialog.getByText('미입고 없음',{exact:true}).count(),0);
   assert.equal(await button.isVisible(),true);
   assert.equal(await button.innerText(),'미입고 조회 재시도');
   await page.screenshot({path:'audit/basic-popup-error-390-v1.4.119.png'});
   await page.request.get(origin+'/synthetic-outstanding-mode?value=zero');
   await dialog.getByRole('button',{name:'새로고침',exact:true}).click();
   await dialog.getByText('미입고 없음',{exact:true}).waitFor();
   assert.equal(await button.isVisible(),false);
   await dialog.getByRole('button',{name:'닫기',exact:true}).click();
   await page.request.get(origin+'/synthetic-outstanding-mode?value=ready');
   // Expire the verified zero through the normal read-only refresh endpoint.
   await page.request.get(`${origin}/order-outstanding/${config.board}?projects=${config.first}&refresh=1`);
   await page.reload();
   await page.waitForFunction(id=>document.querySelector(`[data-project="${id}"] .order-outstanding-button`)?.dataset.state==='ready'&&!document.querySelector(`[data-project="${id}"] .order-outstanding-button`).hidden,config.first);
   const fragment = await (await page.request.get(`${origin}/card/${config.board}/${config.first}`)).text();
   await card.evaluate((element,html)=>{element.outerHTML=html},fragment);
   await page.waitForFunction(id=>{
    const card=document.querySelector(`.work-item[data-project="${id}"]`);
    return card?.querySelectorAll('.order-outstanding-button').length===1 && card.querySelector('.order-outstanding-button').dataset.state==='ready';
   },config.first);
   await card.locator('.order-outstanding-button').click();
   await dialog.getByText('2품목',{exact:true}).waitFor();
   results.push({role:'worker',width:390,result:'pass',scope:'Truncated query remains retryable, confirmed zero hides button, normal recovery and polled-card replacement retain one working popup button'});
   await context.close();
  }
  fs.writeFileSync('audit/browser-release-v1.4.119.json',JSON.stringify({scope:'Actual Chromium implementation with synthetic order/ERP data, worker public-board and authenticated administrator entry points. Company ERP/physical Windows/Android remain field checks.',results},null,2)+'\n');
  console.log('Basic order card browser checks passed',results.length);
 } finally { await browser.close(); }
})().catch(error=>{console.error(error);process.exit(1)});
