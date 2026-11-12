(()=>{
 const state=document.querySelector('[data-read-id]');if(!state)return;
 const button=state.querySelector('button'),result=state.querySelector('[role=status]');
 let busy=false,done=false;
 async function read(){
  if(busy||done||document.visibilityState!=='visible')return;
  busy=true;button.disabled=true;
  try{
   const response=await fetch(`/records/${encodeURIComponent(state.dataset.readId)}/read`,{method:'POST',credentials:'same-origin',headers:{'Content-Type':'application/x-www-form-urlencoded'},body:new URLSearchParams({csrf:state.dataset.csrf})});
   if(!response.ok)throw new Error();done=true;result.textContent='읽음으로 저장했습니다.';button.hidden=true;state.querySelector('.unread-label').textContent='읽음';
  }catch(_){result.textContent='읽음 저장 실패 · 다시 눌러 주세요.';button.disabled=false}finally{busy=false}
 }
 button.addEventListener('click',read);document.addEventListener('visibilitychange',read);read();
})();
