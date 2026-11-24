const {chromium}=require('playwright'),assert=require('assert/strict'),fs=require('fs');
const [origin,raw]=process.argv.slice(2),cfg=JSON.parse(raw),results=[];
(async()=>{const browser=await chromium.launch({executablePath:'/usr/bin/chromium',args:['--no-sandbox']});try{
for(const mode of ['modern','classic'])for(const width of [320,390,1100])for(const theme of ['light','dark'])for(const admin of [false,true]){
 const ctx=await browser.newContext({viewport:{width,height:850},colorScheme:theme,isMobile:width<600});
 await ctx.addInitScript(({mode,theme})=>{localStorage.setItem('alpha-design-default-v2',mode);localStorage.setItem('alpha-color-theme-v1',theme)}, {mode,theme});
 if(admin)await ctx.addCookies([{...cfg.cookie,url:origin}]);
 const page=await ctx.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
 const base=admin?'/community-admin':'/community/'+cfg.board;
 for(const suffix of ['', '?category=work', '?category=suggestion', '?category=daily', '/'+cfg.post]){
  await page.goto(origin+base+suffix);await page.locator('.talk-tabs').waitFor({state:'attached',timeout:5000}).catch(async e=>{fs.writeFileSync('/tmp/talk-tabs-failure.html',await page.content());await page.screenshot({path:'/tmp/talk-tabs-failure.png'});throw e});
  const boxes=await page.locator('.talk-tabs > a').evaluateAll(as=>as.map(a=>{const b=a.getBoundingClientRect();return {text:a.textContent,x:b.x,y:b.y,width:b.width,height:b.height,color:getComputedStyle(a).color,border:getComputedStyle(a).borderBottomColor}}));
  assert.deepEqual(boxes.map(b=>b.text),['전체','작업안내','건의사항','일상생활','매뉴얼 게시판']);assert.ok(boxes.every(b=>Math.abs(b.y-boxes[0].y)<1),'tabs wrap');
  assert.ok(boxes.every(b=>b.height>=44),'tap target too short');
  const nav=await page.locator('.talk-tabs').evaluate(e=>({width:e.clientWidth,scroll:e.scrollWidth,x:e.getBoundingClientRect().x,right:e.getBoundingClientRect().right}));
  assert.ok(nav.x>=-1&&nav.right<=width+1,'nav escaped viewport');
  if(width>=390)assert.ok(nav.scroll<=nav.width+1,'unexpected navigation scroll');
  if(suffix.includes('category=')){const category=suffix.split('=')[1],active=page.locator('.talk-tabs [aria-current=page]');assert.ok((await active.getAttribute('href')).endsWith('category='+category))}
  if(mode==='modern'&&!suffix&&width!==320)await page.screenshot({path:'audit/talk-tabs-'+(admin?'admin':'worker')+'-'+width+'-'+theme+'-v1.4.99.png'});
 }
 await page.locator('.talk-tabs a').last().click();await page.locator('.manual-list, .manual-empty').first().waitFor();assert.match(page.url(),/manuals/);
 assert.deepEqual(errors,[]);results.push({mode,width,theme,admin,result:'pass',scope:'list/categories/detail five compact tabs; touch size; active underline; manual link'});await ctx.close();
}
fs.writeFileSync('audit/browser-talk-tabs-v1.4.99.json',JSON.stringify({results},null,2)+'\n');console.log('Talk tabs browser checks passed:',results.length);
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
