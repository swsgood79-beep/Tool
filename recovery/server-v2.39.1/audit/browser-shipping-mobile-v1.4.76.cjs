const {chromium}=require('playwright'),fs=require('fs'),assert=require('assert/strict');
const [origin,board,raw]=process.argv.slice(2),cfg=JSON.parse(raw),results=[];
const buffer=Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAgAAAAICAIAAABLbSncAAAAFElEQVR4nGPUqDjBgA0wYRUdtBIAHOcBeDC+JtEAAAAASUVORK5CYII=','base64');
const photo=(mimeType='image/png',name='photo.png')=>({name,mimeType,buffer});
(async()=>{const b=await chromium.launch({headless:true,executablePath:'/usr/bin/chromium',args:['--no-sandbox']});try{
 for(let i=0;i<4;i++){
  const mobile=i!==0,app=i>=2,broken=i%2===1;
  const ctx=await b.newContext({viewport:{width:mobile?390:1100,height:950},isMobile:mobile,...(app?{userAgent:'Mozilla/5.0 (Linux; Android 14) Chrome/131 Mobile Safari/537.36 AlphaAdminAndroid/1.0.9'}:{})});
  await ctx.addCookies([{...cfg.cookie,url:origin}]);
  await ctx.addInitScript(broken=>{localStorage.setItem('alpha-speech-browser-enabled-v1','off');if(broken)window.DataTransfer=class{constructor(){throw Error('DataTransfer unavailable')}}},broken);
  const p=await ctx.newPage(),errors=[],dialogs=[];p.on('pageerror',e=>errors.push(e.message));p.on('dialog',async d=>{dialogs.push(d.type());await d.dismiss()});
  for(const addition of [false,true]){
   const base=app&&addition?origin+'/shipping-archive-admin':origin+'/shipping-records/'+board;
   await p.goto(base+'/'+(addition?cfg.archived[i]:cfg.pending[i]));
   if(addition)await p.locator('#shipment-add-open').click();
   await p.locator('#shipment-album').setInputFiles(photo('image/heic','bad.heic'));assert.equal(await p.locator('.shipment-preview-item').count(),0);assert.match(await p.locator('#shipment-photo-error').innerText(),/HEIC/);
   // A generic MIME from a file/cloud provider remains a valid image by bytes.
   await p.locator('#shipment-album').setInputFiles([photo('application/octet-stream'),photo(),photo('image/jpeg','same.jpg')]);
   assert.equal(await p.locator('.shipment-preview-item').count(),3);
   await p.locator('.shipment-preview-item button').nth(2).click();assert.equal(await p.locator('.shipment-preview-item').count(),2);
   await p.waitForFunction(()=>Array.from(document.querySelectorAll('.shipment-preview-item img')).every(x=>x.complete&&x.naturalWidth>0),null,{timeout:10000});
   const action=await p.locator('#shipment-photo-form').getAttribute('action');let first=true;
   await p.route('**'+action,async route=>{
    if(!first)return route.continue();first=false;
    if(addition){await route.fetch();await route.abort('failed')}else await route.fulfill({status:400,contentType:'text/plain',body:'시험 저장 실패'});
   });
   await p.locator('#shipment-finish').click();await p.waitForFunction(()=>!document.querySelector('#shipment-photo-error').hidden&&!document.querySelector('#shipment-finish').disabled);
   assert.equal(await p.locator('.shipment-preview-item').count(),2);assert.equal(await p.locator('#shipment-photo-form[data-sending]').count(),0);
   await p.locator('#shipment-finish').click();await p.waitForURL(addition?'**photos_added=1':'**saved=1');
   assert.equal(await p.locator('.shipment-gallery figure').count(),addition?3:2);assert.deepEqual(dialogs,[]);assert.deepEqual(errors,[]);
   results.push({mobile,app,brokenDataTransfer:broken,addition,case:'generic MIME and normal photos preview, remove, server/network error preserves files, retry exactly once and no leave dialog',result:'pass'});
  }
  await ctx.close();
 }
 fs.writeFileSync('audit/browser-shipping-mobile-v1.4.76.json',JSON.stringify({scope:'Go server + Chromium PC/mobile and APK user agent; DataTransfer failure simulated; actual Samsung/Photos not executed',results},null,2)+'\n');console.log('Mobile photo browser checks passed',results.length);
}finally{await b.close()}})().catch(e=>{console.error(e);process.exit(1)});
