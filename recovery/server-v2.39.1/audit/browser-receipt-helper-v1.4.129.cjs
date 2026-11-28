const {chromium}=require('playwright');const fs=require('fs');const assert=require('assert');
(async()=>{
 const [origin,raw]=process.argv.slice(2),config=JSON.parse(raw);const browser=await chromium.launch({executablePath:'/usr/bin/chromium',headless:true,args:['--no-sandbox']});const results=[];
 for(const width of [320,390,430,1100]){
  const context=await browser.newContext({viewport:{width,height:1100}});await context.addCookies([{...config.owner,url:origin}]);const page=await context.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto(origin+'/?design-tab=orders');
  const card=page.locator(`.admin-project[data-project="${config.id}"]`),button=card.locator('.order-outstanding-button');await page.waitForFunction(id=>document.querySelector(`[data-project="${id}"] .order-outstanding-button`)?.dataset.state==='ready',config.id);await button.click();
  const dialog=page.locator('.outstanding-dialog');await dialog.getByText('4품목',{exact:true}).waitFor();const checks=dialog.locator('.receipt-choice input[type=checkbox]:not(:disabled)');assert.equal(await checks.count(),3);assert.equal(await dialog.locator('.receipt-choice input[type=checkbox]:disabled').count(),1);assert((await dialog.innerText()).includes('사업장·발주번호·발주 행 번호'));assert((await dialog.innerText()).includes('ERP 품목 코드: TEST-ERP-ITEM-CODE-NO-DISPLAY-PART'));
  await checks.nth(0).check();await checks.nth(1).check();await checks.nth(2).check();await dialog.locator('.receipt-choice input[type=number]').nth(0).fill('5');await dialog.evaluate(e=>{e.scrollTop=0;});await dialog.screenshot({path:`audit/receipt-selection-${width}-v1.4.129.png`});await dialog.getByRole('button',{name:'선택 품목 ERP 입력 준비',exact:true}).click();await dialog.getByText('3품목 준비 · 내용을 확인한 뒤 회사 PC 도우미를 열어 주세요.',{exact:true}).waitFor();
  const helper=dialog.locator('.receipt-helper');assert.equal(await helper.locator('h4').count(),2);assert((await helper.innerText()).includes('발주 행 0'));assert((await helper.innerText()).includes('5개'));assert(await dialog.evaluate(e=>e.scrollWidth<=e.clientWidth+1));assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
  await dialog.evaluate(e=>{e.scrollTop=0;});
  await dialog.screenshot({path:`audit/receipt-helper-${width}-v1.4.129.png`});
  if(width===1100){await dialog.getByRole('button',{name:'회사 PC 도우미 열기',exact:true}).click();await dialog.getByText('회사 PC 도우미가 열렸습니다.',{exact:false}).waitFor();}
  assert.deepEqual(errors,[]);results.push({width,result:'pass',scope:'Actual Alpha optional display-part blank remains selectable; explicit zero business sequence accepted; genuinely missing keys blocked with reasons; native launch mocked'});await context.close();
 }
 const context=await browser.newContext({viewport:{width:390,height:1000}}),page=await context.newPage();await page.goto(origin+'/work/'+config.board);assert.equal(await page.locator('#order-outstanding-config').getAttribute('data-helper-csrf'),null);await context.close();await browser.close();fs.writeFileSync('audit/browser-release-v1.4.129.json',JSON.stringify({results},null,2)+'\n');
})().catch(e=>{console.error(e);process.exit(1)});
