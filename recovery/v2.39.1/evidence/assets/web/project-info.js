(()=>{'use strict';let pending=false;
 async function refresh(){if(pending||document.hidden)return;const current=document.querySelector('.special-info-card[data-info-id]');if(!current)return;pending=true;const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),8000);
 try{const response=await fetch(location.pathname,{cache:'no-store',credentials:'same-origin',signal:controller.signal});if(!response.ok)return;const html=await response.text(),next=new DOMParser().parseFromString(html,'text/html').querySelector('.special-info-card[data-info-id]');if(next&&next.dataset.infoId===current.dataset.infoId&&next.dataset.infoRevision!==current.dataset.infoRevision)current.replaceWith(document.importNode(next,true))}catch(_){}finally{clearTimeout(timer);pending=false}}
 setInterval(refresh,3000);document.addEventListener('visibilitychange',()=>{if(!document.hidden)refresh()});
})();
