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
from unittest.mock import patch
from types import SimpleNamespace
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

class EngineTests(unittest.TestCase):
 def test_edge_model_rate_and_no_second_speed_change(self):
  calls=[]
  class Communicate:
   def __init__(self,text,model,**kw):calls.append((text,model,kw['rate']))
   async def save(self,path):Path(path).write_bytes(b'ID3edge')
  with tempfile.TemporaryDirectory() as folder,patch.dict('sys.modules',{'edge_tts':SimpleNamespace(Communicate=Communicate)}):
   for voice,speed,model,rate in [('sunhi',1.1,'ko-KR-SunHiNeural','+10%'),('injoon',.9,'ko-KR-InJoonNeural','-10%')]:
    worker=h.TTSWorker(folder,'synth');result,done=worker.submit({'text':'test','voice':voice,'speed':speed});self.assertTrue(done.wait(2));worker.close();self.assertTrue(result['ok']);self.assertEqual(calls[-1],('test',model,rate))
 def test_selected_engine_error_is_visible_without_gtts_fallback(self):
  def fail(*args):raise OSError('HTTP403 selected voice blocked')
  with tempfile.TemporaryDirectory() as folder,patch.object(h.TTSWorker,'_edge',side_effect=fail):
   factory=lambda **kw:self.fail('unexpected gTTS fallback')
   worker=h.TTSWorker(folder,'synth',synth_factory=factory)
   result,done=worker.submit({'text':'test','voice':'sunhi','speed':1.1});self.assertTrue(done.wait(2));worker.close()
   self.assertFalse(result['ok']);self.assertEqual(result['engine'],'sunhi');self.assertIn('HTTP403',result['detail']);self.assertEqual(list(Path(folder).iterdir()),[])
 def test_sapi_offline_stream_uses_default_voice(self):
  calls=[];state={}
  class Stream:
   Format=SimpleNamespace(Type=0)
   def Open(self,path,*args):state['path']=path;calls.append('open')
   def Close(self):calls.append('close')
  class Voice:
   def Speak(self,text,flags):self_outer.assertEqual(flags,16);Path(state['path']).write_bytes(b'RIFFwave');calls.append(text)
  self_outer=self
  def create(name,**kw):calls.append(name);return Voice() if name=='SAPI.SpVoice' else Stream()
  com=SimpleNamespace(CoInitialize=lambda:calls.append('init'),CoUninitialize=lambda:calls.append('uninit'))
  with tempfile.TemporaryDirectory() as folder,patch.dict('sys.modules',{'comtypes':com,'comtypes.client':SimpleNamespace(CreateObject=create)}):
   worker=h.TTSWorker(folder,'synth');result,done=worker.submit({'text':'offline','voice':'sapi','speed':1.5});self.assertTrue(done.wait(2));worker.close();self.assertTrue(result['ok']);self.assertIn('offline',calls);self.assertEqual(calls[-1],'uninit')
 def test_pcm_speed_is_not_applied_twice_to_edge(self):
  rates=[];stretched=[]
  class Sound:
   def __init__(self,*args,**kw):pass
   def get_raw(self):return b'\0'*400
   def play(self):return SimpleNamespace(get_busy=lambda:False)
  mixer=SimpleNamespace(init=lambda **kw:rates.append(kw['frequency']),Sound=Sound,quit=lambda:None)
  with patch.dict('sys.modules',{'pygame':SimpleNamespace(mixer=mixer)}),patch.object(h,'stretch_pcm',side_effect=lambda raw,rate,speed:stretched.append(speed) or raw):
   h.TTSWorker._play('fake',None,'gtts',1.15);h.TTSWorker._play('fake',None,'sunhi',1.15);h.TTSWorker._play('fake',None,'sapi',.8)
  self.assertEqual(rates,[27600,24000,24000]);self.assertEqual(stretched,[1.15,.8])

if __name__=='__main__':unittest.main(verbosity=2)
