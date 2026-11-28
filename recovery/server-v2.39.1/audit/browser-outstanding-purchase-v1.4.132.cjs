const {chromium}=require('playwright');const assert=require('assert');const fs=require('fs');
(async()=>{
 const [origin,raw]=process.argv.slice(2),config=JSON.parse(raw);const browser=await chromium.launch({executablePath:'/usr/bin/chromium',headless:true,args:['--no-sandbox']});const results=[];
 for(const role of ['worker','admin'])for(const width of [320,390,430,1100])for(const theme of ['light','dark']){
  const context=await browser.newContext({viewport:{width,height:1100},colorScheme:theme});if(role==='admin')await context.addCookies([{...config.owner,url:origin}]);const page=await context.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto(origin+(role==='admin'?'/?design-tab=orders':'/work/'+config.board));
  const card=page.locator(`${role==='admin'?'.admin-project':'.work-item'}[data-project="${config.id}"]`),button=card.locator('.order-outstanding-button');
  await page.waitForFunction(id=>document.querySelector(`[data-project="${id}"] .order-outstanding-button`)?.dataset.state==='ready',config.id);
  await button.click();const dialog=page.locator('.outstanding-dialog');await dialog.getByText('3품목',{exact:true}).waitFor();
  assert.equal(await dialog.locator('input,.receipt-helper,.receipt-choice').count(),0);assert.equal(await dialog.locator('button').count(),2);
  assert.equal(await dialog.locator('th').count(),4);assert.equal(await dialog.locator('tr.outstanding-purchase').count(),3);assert.equal(await dialog.locator('tr.outstanding-item').count(),3);
  const orders=await dialog.locator('tr.outstanding-purchase').allTextContents();assert(orders[0].includes('TEST-PO-0001-LONG-123456789012345678901234567890'));assert(orders[1].includes('TEST-PO-0002'));assert.equal(orders[2],'발주번호 확인 불가');
  const items=await dialog.locator('tr.outstanding-item').allTextContents();assert(items[0].includes('가상 BASE')&&items[0].includes('4'));assert(items[1].includes('가상 BASE')&&items[1].includes('2'));
  for(const row of await dialog.locator('tr.outstanding-purchase').all()){assert.equal(await row.locator('td').getAttribute('colspan'),'4');assert(await row.evaluate(e=>e.scrollWidth<=e.clientWidth+1));}
  assert(await dialog.evaluate(e=>e.scrollWidth<=e.clientWidth+1));assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
  await dialog.screenshot({path:`audit/outstanding-purchase-${role}-${width}-${theme}-v1.4.132.png`});
  await dialog.getByRole('button',{name:'새로고침',exact:true}).click();await dialog.getByText('3품목',{exact:true}).waitFor();assert.equal(await dialog.locator('tr.outstanding-purchase').count(),3);
  await dialog.getByRole('button',{name:'닫기',exact:true}).click();if(role==='admin'){const edit=await card.locator(':scope>summary>.order-edit').boundingBox(),workers=await card.locator('.e-admin-workers').boundingBox();assert(edit&&workers&&workers.y+workers.height<edit.y);}
  assert.deepEqual(errors,[]);results.push({role,width,theme,result:'pass',scope:'Actual read-only popup, exact purchase numbers per duplicate-named item, missing PO not guessed; no input/helper, full-width mobile lines, refresh/close'});await context.close();
 }
 await browser.close();fs.writeFileSync('audit/browser-release-v1.4.132.json',JSON.stringify({results},null,2)+'\n');
})().catch(e=>{console.error(e);process.exit(1)});
