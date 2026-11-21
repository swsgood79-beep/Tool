(()=>{'use strict';
// Only an upload copy is encoded. The file in the phone gallery is never modified.
async function shrink(file,{signal,progress=()=>{},threshold=20*1024*1024}={}){
 if(file.size<=threshold)return file;
 if(!window.MediaRecorder||!HTMLCanvasElement.prototype.captureStream)throw Error('이 브라우저에서는 영상 자동 압축을 지원하지 않습니다. 최신 Chrome 또는 알파 앱에서 시도해 주세요.');
 const video=document.createElement('video');video.playsInline=true;video.muted=true;video.preload='auto';video.style.cssText='position:fixed;width:1px;height:1px;opacity:0;pointer-events:none';document.body.append(video);
 const url=URL.createObjectURL(file);video.src=url;let audio,stream,recorder,frame,wake,hiddenHandler;let output;
 const check=()=>{if(signal?.aborted)throw Error('영상 준비를 취소했습니다.')};check();
 try{
  audio=new (window.AudioContext||window.webkitAudioContext)();await audio.resume();
  await new Promise((resolve,reject)=>{video.onloadedmetadata=resolve;video.onerror=()=>reject(Error('영상을 읽지 못했습니다. MP4로 촬영한 영상을 선택해 주세요.'));const abort=()=>reject(Error('영상 준비를 취소했습니다.'));signal?.addEventListener('abort',abort,{once:true});video.addEventListener('loadedmetadata',()=>signal?.removeEventListener('abort',abort),{once:true})});
  check();if(!video.videoWidth||!video.videoHeight||!Number.isFinite(video.duration)||video.duration<=0)throw Error('동영상 길이와 화면 크기를 확인할 수 없습니다.');
  const scale=Math.min(1,1280/Math.max(video.videoWidth,video.videoHeight));const canvas=document.createElement('canvas');canvas.width=Math.max(2,Math.round(video.videoWidth*scale/2)*2);canvas.height=Math.max(2,Math.round(video.videoHeight*scale/2)*2);const ctx=canvas.getContext('2d',{alpha:false});
  stream=canvas.captureStream(30);const source=audio.createMediaElementSource(video),dest=audio.createMediaStreamDestination();source.connect(dest);video.muted=false;for(const track of dest.stream.getAudioTracks())stream.addTrack(track);
  const types=['video/mp4;codecs=avc1.42E01E,mp4a.40.2','video/webm;codecs=vp8,opus','video/webm'];const type=types.find(t=>MediaRecorder.isTypeSupported(t));if(!type)throw Error('지원되는 동영상 압축 형식을 찾지 못했습니다.');
  recorder=new MediaRecorder(stream,{mimeType:type,videoBitsPerSecond:1800000,audioBitsPerSecond:128000});const chunks=[];let bytes=0;
  output=new Promise((resolve,reject)=>{
   let failed=false;
   const fail=message=>{if(failed)return;failed=true;reject(Error(message));if(recorder.state!=='inactive')recorder.stop()};
   hiddenHandler=()=>{if(document.hidden)fail('압축 중 화면이 닫혀 중단했습니다. 화면을 켠 상태에서 다시 저장해 주세요.')};document.addEventListener('visibilitychange',hiddenHandler);
   signal?.addEventListener('abort',()=>fail('영상 준비를 취소했습니다.'),{once:true});
   recorder.ondataavailable=e=>{if(e.data.size){bytes+=e.data.size;if(bytes>300*1024*1024){fail('압축한 영상도 300MB를 초과합니다. 영상을 짧게 나누어 등록해 주세요.');return}chunks.push(e.data)}};
   recorder.onerror=()=>fail('영상 압축 중 문제가 발생했습니다. 원본 영상은 그대로 보관되어 있습니다.');
   recorder.onstop=()=>{if(failed)return;if(video.currentTime<video.duration-.5){fail('영상 압축이 끝나지 않았습니다. 다시 시도해 주세요.');return}const blob=new Blob(chunks,{type:recorder.mimeType.split(';')[0]});if(!blob.size){fail('압축한 영상이 비어 있습니다.');return}const ext=blob.type.includes('mp4')?'.mp4':'.webm';resolve(new File([blob],file.name.replace(/\.[^.]+$/,'')+ext,{type:blob.type,lastModified:file.lastModified}))};
   video.onended=()=>{if(recorder.state!=='inactive')recorder.stop()};video.onerror=()=>fail('영상을 재생하지 못해 압축을 중단했습니다. MP4 영상을 선택해 주세요.');
  });
  // Attach a rejection handler before asynchronous playback, avoiding an unhandled
  // rejection if a cancellation/error occurs while play() is still pending.
  output.catch(()=>{});
  const draw=()=>{ctx.drawImage(video,0,0,canvas.width,canvas.height);progress(Math.min(99,Math.round(video.currentTime/video.duration*100)));if(!video.ended)frame=requestAnimationFrame(draw)};
  try{wake=await navigator.wakeLock?.request('screen')}catch(_){}
  ctx.drawImage(video,0,0,canvas.width,canvas.height);recorder.start(1000);await video.play();draw();let result=await output;if(result.type.includes('webm')){const fixed=await ysFixWebmDuration(result,video.duration*1000,{logger:false});result=new File([fixed],result.name,{type:fixed.type,lastModified:file.lastModified})}check();progress(100);return result.size<file.size?result:file;
 }finally{
  if(hiddenHandler)document.removeEventListener('visibilitychange',hiddenHandler);cancelAnimationFrame(frame);if(recorder?.state!=='inactive'&&recorder)recorder.stop();video.pause();video.removeAttribute('src');video.load();video.remove();stream?.getTracks().forEach(t=>t.stop());await audio?.close().catch(()=>{});await wake?.release().catch(()=>{});URL.revokeObjectURL(url)
 }
}
window.AlphaShipmentVideo={shrink};
})();
