"""Decode the actual browser outputs; require preserved sound, portrait and duration."""
from pathlib import Path
import subprocess,json,re
results=[]
for i in range(4):
 p='/tmp/alpha-compressed-'+str(i)+'.webm'
 probe=json.loads(subprocess.check_output(['ffprobe','-v','error','-show_streams','-show_format','-of','json',p],stderr=subprocess.PIPE))
 video=next(x for x in probe['streams'] if x['codec_type']=='video');audio=next(x for x in probe['streams'] if x['codec_type']=='audio')
 volume=subprocess.run(['ffmpeg','-hide_banner','-i',p,'-af','volumedetect','-vn','-f','null','-'],capture_output=True,text=True,check=True).stderr
 mean=float(re.search(r'mean_volume: (-?[\d.]+) dB',volume).group(1));assert mean>-60
 assert video['width']==720 and video['height']==1280 and 4.5<float(probe['format']['duration'])<6.5
 results.append({'case':i,'width':video['width'],'height':video['height'],'video_codec':video['codec_name'],'audio_codec':audio['codec_name'],'duration':probe['format']['duration'],'mean_volume_db':mean,'bytes':int(probe['format']['size']),'result':'pass'})
Path('audit/shipping-video-media-v1.4.82.json').write_text(json.dumps({'scope':'ffprobe and decoded PCM volume check of real browser-compressed 5-second portrait MP4 fixture; actual phone recording not tested','results':results},indent=2)+'\n')
print('4 compressed clips retain portrait orientation, full duration and audible audio')
