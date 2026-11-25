/* Complete only order searches; typing and other forms keep their normal behavior. */
(()=>{'use strict';
const selector='#project-search,#admin-order-search',composing=new WeakSet();let cancelScroll=()=>{};
const mobile=()=>matchMedia('(pointer: coarse)').matches||/Android|iPhone|iPad|iPod/i.test(navigator.userAgent);
const visible=e=>!!e&&!e.closest('[hidden]')&&e.getClientRects().length>0&&getComputedStyle(e).visibility!=='hidden';
function result(field){
 const selectors=field.id==='admin-order-search'?['#order-items .admin-project','#order-items .admin-no-results','#order-items']:['#today-panel .today-item','.work-list .work-item','.work-list .delivery-completed','#search-empty','#today-empty','.work-list'];
 for(const s of selectors){const found=[...document.querySelectorAll(s)].find(visible);if(found)return found}
}
function finish(field){
 cancelScroll();field.dispatchEvent(new Event('input',{bubbles:true}));
 if(!mobile())return;
 field.blur();let timer=0,cleanup=0,stopped=false;const viewport=window.visualViewport;
 const stop=()=>{stopped=true;clearTimeout(timer);clearTimeout(cleanup);viewport?.removeEventListener('resize',schedule);cancelScroll=()=>{}};
 function scroll(){if(stopped||document.activeElement===field)return;const target=result(field);if(!target)return;target.scrollIntoView({block:'start',behavior:'auto'});
  let inset=12;for(const header of document.querySelectorAll('.design-header,header')){if(!visible(header))continue;const style=getComputedStyle(header),r=header.getBoundingClientRect();if(['fixed','sticky'].includes(style.position)&&r.top<=1)inset=Math.max(inset,r.bottom+12)}
  const top=target.getBoundingClientRect().top;if(top<inset)window.scrollBy(0,top-inset);
 }
 function schedule(){clearTimeout(timer);timer=setTimeout(scroll,180)}
 cancelScroll=stop;viewport?.addEventListener('resize',schedule);schedule();cleanup=setTimeout(stop,1200);
}
for(const field of document.querySelectorAll(selector))field.setAttribute('enterkeyhint','search');
document.addEventListener('compositionstart',e=>{if(e.target.matches(selector))composing.add(e.target)});
document.addEventListener('compositionend',e=>composing.delete(e.target));
document.addEventListener('keydown',e=>{const field=e.target;if(!field.matches(selector)||e.defaultPrevented||e.key!=='Enter'&&e.keyCode!==13||e.isComposing||e.keyCode===229||composing.has(field))return;e.preventDefault();finish(field)});
// Android Chrome/Samsung Internet also emit this for the keyboard search action.
document.addEventListener('search',e=>{const field=e.target;if(field.matches(selector)&&field.value.trim()&&!composing.has(field))finish(field)},true);
// A person starting another input must not be pulled back to the previous result.
document.addEventListener('focusin',()=>cancelScroll());
})();
