(() => {
 'use strict';
 // One rigid rotation per leaf in 3D, projected by a fixed orthographic game camera.
 // No interpolation between skew/scale matrices. Vertical hinges remain stationary.
 const yaw=35*Math.PI/180,slope=156/450;
 const sy=Math.sin(yaw),cy=Math.cos(yaw),sp=slope/Math.tan(yaw);
 const gate=document.querySelector('[data-entry-gate]');if(!gate)return;const svg=gate.querySelector('svg');
 const aperture=svg.querySelector('g[clip-path="url(#gate-aperture)"]');
 const ns='http://www.w3.org/2000/svg';
 const leaves=[['.gate-leaf-left',390,1,[614,230],[610,864]],['.gate-leaf-right',840,-1,[614,230],[610,864]]].map(([selector,hinge,side,top,bottom])=>{
  const face=svg.querySelector(selector),back=face.cloneNode(true),edge=document.createElementNS(ns,'path');
  back.classList.remove('gate-leaf','gate-leaf-left','gate-leaf-right');back.classList.add('rigid-back');back.style.filter='brightness(.40)';back.style.opacity='1';
  edge.setAttribute('fill','#534a35');edge.setAttribute('stroke','#9a8153');edge.setAttribute('stroke-width','1.2');
  aperture.insertBefore(back,face);aperture.insertBefore(edge,face);
  return {face,back,edge,hinge,side,top,bottom};
 });
 const defs=svg.querySelector('defs');
 const filter=document.createElementNS(ns,'filter');filter.id='rigid-bloom';filter.setAttribute('x','-80%');filter.setAttribute('y','-80%');filter.setAttribute('width','260%');filter.setAttribute('height','260%');
 const blur=document.createElementNS(ns,'feGaussianBlur');blur.setAttribute('stdDeviation','50');filter.append(blur);defs.append(filter);
 const bloom=svg.querySelector('.gate-light').cloneNode();bloom.classList.remove('gate-light');bloom.setAttribute('fill','#ffd478');bloom.setAttribute('filter','url(#rigid-bloom)');bloom.style.opacity='0';aperture.before(bloom);
 const spill=document.createElementNS(ns,'ellipse');spill.setAttribute('cx','595');spill.setAttribute('cy','990');spill.setAttribute('rx','460');spill.setAttribute('ry','160');spill.setAttribute('transform','rotate(19 595 990)');spill.setAttribute('fill','#ffcf70');spill.setAttribute('filter','url(#rigid-bloom)');spill.style.opacity='0';aperture.before(spill);
 const haloGradient=document.createElementNS(ns,'radialGradient');haloGradient.id='rigid-halo';
 for(const [offset,color,opacity] of [['0','#fff8dd','.68'],['.28','#ffe5a8','.48'],['.62','#ffca72','.20'],['1','#ffc05b','0']]){const stop=document.createElementNS(ns,'stop');stop.setAttribute('offset',offset);stop.setAttribute('stop-color',color);stop.setAttribute('stop-opacity',opacity);haloGradient.append(stop)}defs.append(haloGradient);
 const halo=document.createElementNS(ns,'ellipse');halo.setAttribute('cx','610');halo.setAttribute('cy','700');halo.setAttribute('rx','640');halo.setAttribute('ry','590');halo.setAttribute('fill','url(#rigid-halo)');halo.style.mixBlendMode='screen';halo.style.opacity='0';aperture.before(halo);
 const rayGradient=document.createElementNS(ns,'linearGradient');rayGradient.id='rigid-rays';rayGradient.setAttribute('x1','0');rayGradient.setAttribute('y1','0');rayGradient.setAttribute('x2','0');rayGradient.setAttribute('y2','1');for(const [offset,opacity] of [['0','.85'],['1','0']]){const stop=document.createElementNS(ns,'stop');stop.setAttribute('offset',offset);stop.setAttribute('stop-color','#fff2bc');stop.setAttribute('stop-opacity',opacity);rayGradient.append(stop)}defs.append(rayGradient);
 const rays=document.createElementNS(ns,'path');rays.setAttribute('d','M440 560L530 590L190 1160L30 1140Z M610 610L710 640L1080 1190L850 1210Z M530 650L590 670L530 1250L370 1230Z');rays.setAttribute('fill','url(#rigid-rays)');rays.setAttribute('filter','url(#rigid-bloom)');rays.style.mixBlendMode='screen';rays.style.opacity='0';aperture.before(rays);
 const light=svg.querySelector('.gate-light');light.setAttribute('fill','#fffbea');light.style.transition='none';
 gate.querySelector('.gate-artwork').style.transition='none';
 let animation=0,settle=null;
 function stop(){cancelAnimationFrame(animation);animation=0;if(settle){const done=settle;settle=null;done()}}
 function matrix(angle,hinge,side){
  const c=Math.cos(angle),s=Math.sin(angle),a=c+side*Math.tan(yaw)*s,b=slope*(c-1)-side*sp*s;
  return [a,b,0,1,hinge*(1-a),-hinge*b];
 }
 function apply(leaf,angle){
  const m=matrix(angle,leaf.hinge,leaf.side),thickness=10;
  // A physical thickness vector normal to the rotating door plane.
  const dx=thickness*(-leaf.side*cy*Math.sin(angle)+sy*Math.cos(angle));
  const dy=thickness*(-leaf.side*sp*sy*Math.sin(angle)-sp*cy*Math.cos(angle));
  const f=p=>[m[0]*p[0]+m[4],p[1]+m[1]*p[0]+m[5]];
  const top=f(leaf.top),bottom=f(leaf.bottom);
  leaf.face.style.transform=`matrix(${m.join(',')})`;
  leaf.back.style.transform=`matrix(${[m[0],m[1],0,1,m[4]+dx,m[5]+dy].join(',')})`;
  leaf.edge.setAttribute('d',`M${top}L${bottom}L${bottom[0]+dx},${bottom[1]+dy}L${top[0]+dx},${top[1]+dy}Z`);
 }
 function set(progress){
  const t=Math.max(0,Math.min(1,progress)),ease=t*t*(3-2*t);
  for(const leaf of leaves)apply(leaf,ease*(leaf.side===1?130:112)*Math.PI/180);
  light.style.opacity=.66*Math.pow(ease,1.4);bloom.style.opacity=.98*Math.pow(ease,1.7);spill.style.opacity=.48*ease*ease;halo.style.opacity=.95*ease*ease;rays.style.opacity=.34*ease*ease;
  gate.querySelector('.gate-artwork').style.filter=`brightness(${.67+.33*ease})`;
  gate.querySelector('.gate-caption').style.opacity=1-ease;
 }
 window.AlphaRigidGate={set,matrix,stop,open(){stop();return new Promise(resolve=>{settle=resolve;const start=performance.now();function tick(now){const t=Math.min(1,(now-start)/1800);set(t);if(t<1)animation=requestAnimationFrame(tick);else{animation=0;settle=null;resolve()}}animation=requestAnimationFrame(tick)})}};
 set(0);
})();
