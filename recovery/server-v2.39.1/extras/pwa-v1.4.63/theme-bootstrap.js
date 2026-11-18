(()=>{'use strict';try{
 const theme=localStorage.getItem('alpha-app-theme-v1');if(!['system','light','dark'].includes(theme))return;
 document.documentElement.dataset.appTheme=theme;
 const meta=document.querySelector('meta[name=color-scheme]');if(meta)meta.content=theme==='light'?'only light':theme==='dark'?'dark':'light dark';
 const color=document.querySelector('meta[name=theme-color]');if(color)color.content=theme==='dark'||theme==='system'&&matchMedia('(prefers-color-scheme:dark)').matches?'#101720':'#f3f5f8';
}catch(_){}})();
