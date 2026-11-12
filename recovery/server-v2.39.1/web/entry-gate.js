(() => {
 'use strict';
 const gate=document.querySelector('[data-entry-gate]');if(!gate)return;
 const reduced=matchMedia('(prefers-reduced-motion: reduce)').matches;
 let finished=false;
 const content=[...document.querySelectorAll('body>header,body>main')].filter(n=>!n.inert);
 content.forEach(n=>n.inert=true);
 const wait=ms=>new Promise(resolve=>setTimeout(resolve,ms));
 function finish(){if(finished)return;finished=true;window.AlphaRigidGate?.stop();gate.hidden=true;gate.dataset.state='done';content.forEach(n=>n.inert=false);try{const url=new URL(location.href);url.searchParams.delete('entry');history.replaceState(history.state,'',url)}catch(_){} }
 const safety=setTimeout(finish,6500);
 async function artworkReady(){
  const artwork=gate.querySelector('.gate-artwork');if(!artwork)return false;
  // Decode the same image before exposing any of the composited SVG door layers.
  // SVG and Image share the document's image resource; this does not add a readiness probe.
  const picture=new Image();picture.src=new URL(artwork.getAttribute('href'),location.href).href;
  return Promise.race([picture.decode().then(()=>true,()=>false),wait(2500).then(()=>false)]);
 }
 async function open(){try{
  // Wait for local worker initialization, never for weather or background polling.
  let painted=false;
  if(!reduced){painted=await artworkReady();if(finished)return;if(painted){gate.classList.add('gate-scene-ready');await wait(200);}}
  if(finished)return;
  if(!reduced&&painted&&window.AlphaRigidGate){gate.dataset.state='opening';gate.classList.add('gate-opening');await window.AlphaRigidGate.open();if(finished)return;gate.dataset.state='opened';await wait(380);}
  if(finished)return;gate.dataset.state='revealing';gate.classList.add('gate-leaving');await wait(reduced?160:520);
  clearTimeout(safety);finish();
 }catch(_){clearTimeout(safety);finish()}}
 if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',open,{once:true});else open();
 addEventListener('pageshow',event=>{if(event.persisted){clearTimeout(safety);finish()}});
})();
