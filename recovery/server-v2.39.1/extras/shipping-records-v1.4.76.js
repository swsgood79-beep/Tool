(()=>{'use strict';
const pendingList=document.querySelector('[data-shipping-pending-list]');let queryEditing=false;document.querySelector('.shipment-query-form')?.addEventListener('input',e=>{if(e.isTrusted)queryEditing=true});
const board=document.querySelector('meta[name=shipping-records-board]')?.content||document.querySelector('main[data-shipping-records-page]')?.dataset.board;
async function refresh(){if(!board||document.hidden)return;try{const response=await fetch('/shipping-records/'+encodeURIComponent(board)+'/status',{cache:'no-store'});if(!response.ok)return;const data=await response.json();if(!Array.isArray(data.pending))return;if(pendingList&&!queryEditing&&typeof data.pending_signature==='string'&&data.pending_signature!==pendingList.dataset.pendingSignature){location.reload();return}document.querySelectorAll('[data-shipping-pending-count]').forEach(n=>{n.textContent=String(data.pending.length)})}catch(_){}}
if(board){refresh();setInterval(refresh,10000);document.addEventListener('visibilitychange',refresh)}
for(const node of document.querySelectorAll('[data-shipment-material-color]')){const color=node.dataset.shipmentMaterialColor;if(/^#[0-9a-f]{6}$/i.test(color)){node.style.backgroundColor=color;const values=color.slice(1).match(/../g).map(v=>parseInt(v,16));node.style.color=values[0]*.299+values[1]*.587+values[2]*.114>155?'#111':'#fff'}}
const custom=document.getElementById('shipment-period-custom'),range=document.getElementById('shipment-date-range'),period=document.getElementById('shipment-period');if(custom&&range&&period){custom.addEventListener('click',()=>{range.hidden=false;period.value='custom';custom.setAttribute('aria-expanded','true');custom.setAttribute('aria-pressed','true');document.querySelector('.shipment-period-tabs [aria-current]')?.removeAttribute('aria-current');for(const date of range.querySelectorAll('input'))date.required=true;range.querySelector('input')?.focus()})}document.getElementById('shipment-sort')?.addEventListener('change',e=>e.target.form.requestSubmit());
const form=document.getElementById('shipment-photo-form');if(!form)return;
const main=document.getElementById('shipment-files'),camera=document.getElementById('shipment-camera'),album=document.getElementById('shipment-album'),preview=document.getElementById('shipment-photo-preview'),count=document.getElementById('shipment-photo-count'),error=document.getElementById('shipment-photo-error'),finish=document.getElementById('shipment-finish');const existing=Number(form.dataset.existing||0),addition=form.hasAttribute('data-existing');let files=[];
// Keep actual File objects instead of assigning a synthetic FileList. Some mobile
// WebViews cannot construct DataTransfer or assign input.files.
main.required=false;main.disabled=true;
let urls=[];
const status=document.getElementById('shipment-upload-status');
function showError(message){error.textContent=message;error.hidden=false}
function render(){
 for(const url of urls)URL.revokeObjectURL(url);urls=[];preview.replaceChildren();
 count.textContent=addition?'기존 '+existing+'장 + 추가 '+files.length+'장 = 총 '+(existing+files.length)+' / 30장':'사진 '+files.length+' / 30장 · 한 장당 8MB 이하';
 files.forEach((file,index)=>{
  const figure=document.createElement('figure');figure.className='shipment-preview-item';
  const image=document.createElement('img');image.alt='출하사진 '+(index+1);image.decoding='async';
  try{const url=URL.createObjectURL(file);urls.push(url);image.src=url}catch(_){image.alt='미리보기 불가 · 사진 '+(index+1)}
  const remove=document.createElement('button');remove.type='button';remove.className='quiet';remove.textContent='사진 '+(index+1)+' 삭제';
  remove.addEventListener('click',()=>{if(form.dataset.sending)return;files.splice(index,1);render();main.dispatchEvent(new Event('input',{bubbles:true}))});
  figure.append(image,remove);preview.append(figure)
 })
}
function validImage(file){
 const type=(file.type||'').toLowerCase();
 if(['image/jpeg','image/jpg','image/png','image/gif','image/webp'].includes(type))return true;
 // Android file providers sometimes omit image MIME information. Actual bytes
 // are decoded and validated by the server before any photo is persisted.
 return ['', 'application/octet-stream','binary/octet-stream'].includes(type)&&/\.(jpe?g|png|gif|webp)$/i.test(file.name)
}
function add(input){
 if(form.dataset.sending)return;
 const incoming=Array.from(input.files||[]);if(!incoming.length)return;error.hidden=true;
 if(existing+files.length+incoming.length>30){showError('기존 사진을 포함해 최대 30장입니다.');input.value='';return}
 if(incoming.some(f=>f.size>8*1024*1024||!validImage(f))){showError('한 장당 8MB 이하의 JPG·PNG·GIF·WebP 사진을 선택해 주세요. HEIC 사진은 JPG로 변환해 주세요.');input.value='';return}
 files.push(...incoming);input.value='';render()
}
const panel=document.getElementById('shipment-add-panel'),open=document.getElementById('shipment-add-open');
open?.addEventListener('click',()=>{panel.hidden=false;open.setAttribute('aria-expanded','true');panel.scrollIntoView({block:'center',behavior:'smooth'})});
const cancel=document.getElementById('shipment-add-cancel');
cancel?.addEventListener('click',()=>{if(form.dataset.sending)return;form.reset();files=[];render();error.hidden=true;status.textContent='';panel.hidden=true;open.setAttribute('aria-expanded','false');open.focus()});
camera.addEventListener('change',()=>add(camera));album.addEventListener('change',()=>add(album));
const idleText=finish.textContent;
function busy(value){
 if(value)form.dataset.sending='1';else delete form.dataset.sending;
 for(const control of [finish,camera,album,cancel,...preview.querySelectorAll('button')])if(control)control.disabled=value;
 finish.textContent=value?'사진 저장 중…':idleText
}
form.addEventListener('submit',async event=>{
 event.preventDefault();if(form.dataset.sending)return;
 if(!files.length){showError('출하사진을 1장 이상 등록해 주세요.');return}
 const body=new FormData(form);body.delete('photos');for(const file of files)body.append('photos',file,file.name);
 error.hidden=true;busy(true);status.textContent='사진을 저장하고 있습니다. 잠시 기다려 주세요.';
 try{
  const response=await fetch(form.action,{method:'POST',body,credentials:'same-origin',headers:{Accept:'application/json'}});
  if(!response.ok){const message=await response.text();throw Error(response.status===400&&message.length<600?message.trim():'사진을 저장하지 못했습니다. 로그인·연결 상태를 확인한 후 다시 시도해 주세요.')}
  const result=await response.json(),next=new URL(result.next,location.href),action=new URL(form.action);
  const expected=action.pathname.replace(/\/(save|add-photos)$/,'');
  if(result.ok!==true||next.origin!==location.origin||next.pathname!==expected||next.search!==(addition?'?photos_added=1':'?saved=1'))throw Error('저장 결과를 확인하지 못했습니다. 다시 저장을 눌러 주세요.');
  // Reset only AFTER confirmed persistence. All app/worker dirty guards observe
  // reset and permit the success navigation without a leave-site warning.
  form.reset();files=[];render();busy(false);status.textContent='사진이 저장되었습니다.';location.assign(next.href)
 }catch(e){busy(false);status.textContent='';showError(e.message||'사진 저장 중 연결이 끊겼습니다. 다시 시도해 주세요.')}
});
})();
