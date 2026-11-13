import array
import base64
import importlib.util
import math
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import threading
import unittest
import wave

spec=importlib.util.spec_from_file_location('tts_helper',Path(__file__).with_name('tts_helper.py'))
h=importlib.util.module_from_spec(spec);spec.loader.exec_module(h)

class HelperTests(unittest.TestCase):
 def test_queue_fifo_failure_survival_and_immediate_temp_cleanup(self):
  with tempfile.TemporaryDirectory() as folder:
   calls=[];paths=[];gate=threading.Event();started=threading.Event()
   class Factory:
    def __init__(self,**kw):
     self.text=kw['text'];self.kw=kw
     self_outer.assertEqual(kw['lang'],'ko');self_outer.assertIs(kw['slow'],False)
    def save(self,path):
     paths.append(path);Path(path).write_bytes(b'ID3synthetic')
     if self.text=='failure':raise OSError('synthetic network interruption')
   self_outer=self
   def play(path):
    self.assertTrue(Path(path).exists());text=Path(path).parent.name;calls.append(text)
    if len(calls)==1:started.set();gate.wait(2)
   worker=h.TTSWorker(folder,'speak',play,Factory)
   first,d1=worker.submit({'text':'one'});self.assertTrue(started.wait(2))
   failed,df=worker.submit({'text':'failure'});last,d2=worker.submit({'text':'three'})
   self.assertFalse(d2.is_set());gate.set();self.assertTrue(d2.wait(3));worker.close()
   self.assertTrue(first['ok']);self.assertFalse(failed['ok']);self.assertTrue(last['ok']);self.assertEqual(len(calls),2)
   self.assertTrue(all(not Path(p).exists() for p in paths));self.assertEqual(list(Path(folder).iterdir()),[])
 def test_synthesis_returns_mp3_and_deletes_file(self):
  class Factory:
   def __init__(self,**kw):pass
   def save(self,path):Path(path).write_bytes(b'ID3audio')
  with tempfile.TemporaryDirectory() as folder:
   worker=h.TTSWorker(folder,'synth',synth_factory=Factory);result,done=worker.submit({'text':'sample'});self.assertTrue(done.wait(2));worker.close()
   self.assertEqual(base64.b64decode(result['mp3']),b'ID3audio');self.assertEqual(list(Path(folder).iterdir()),[])
 def test_speed_duration_and_pitch(self):
  rate=26400;raw=array.array('h',(int(12000*math.sin(2*math.pi*440*i/rate)) for i in range(rate*2)))
  data=array.array('h',h.stretch_pcm(raw.tobytes(),rate,1.1))
  self.assertEqual(len(data),int(len(raw)/1.1))
  crossings=sum(a<=0<b for a,b in zip(data,data[1:]));frequency=crossings/(len(data)/rate)
  self.assertLess(abs(frequency-440),6)
 def test_real_pygame_mp3_decode_and_pcm_playback(self):
  # Local synthetic signal and dummy SDL output; not Google/Windows speaker verification.
  with tempfile.TemporaryDirectory() as folder:
   wav=Path(folder)/'tone.wav';mp3=Path(folder)/'tone.mp3'
   with wave.open(str(wav),'wb') as out:
    out.setparams((1,2,24000,0,'NONE','not compressed'))
    out.writeframes(array.array('h',(int(10000*math.sin(2*math.pi*440*i/24000)) for i in range(4800))).tobytes())
   subprocess.run(['ffmpeg','-loglevel','error','-i',str(wav),str(mp3)],check=True)
   os.environ['SDL_AUDIODRIVER']='dummy';os.environ['PYGAME_HIDE_SUPPORT_PROMPT']='1'
   h.TTSWorker._play(str(mp3),None)

if __name__=='__main__':unittest.main(verbosity=2)
