"""gTTS worker: JSON lines on stdin, one result per job. No UI work on this thread."""
import asyncio
import re
import array
import base64
import json
import math
import os
import queue
import sys
import tempfile
import threading
import time


def stretch_pcm(raw, sample_rate=26400, speed=1.1):
    """Small WSOLA overlap/add; speed changes duration while retaining voiced pitch.

    Mixer frequency is an output format, not a playback-speed control. No ffmpeg.
    """
    source = array.array('h', raw)
    if sys.byteorder != 'little':
        source.byteswap()
    size = int(len(source) / speed)
    if len(source) < 64:
        return raw
    result = array.array('h', [0]) * size
    window = max(64, int(sample_rate * .04))
    hop = window // 2
    search = int(sample_rate * .008)
    result[:min(window, size)] = source[:min(window, size)]
    for out in range(hop, size, hop):
        expected = min(int(out * speed), max(0, len(source) - window))
        left, right = max(0, expected-search), min(len(source)-window, expected+search)
        best, score = expected, -2.0
        # Match the previous grain's tail. Sparse correlation keeps Python CPU bounded.
        reference = result[out:min(out+hop, size):8]
        reference_norm = sum(x*x for x in reference)
        if reference_norm:
            for pos in range(left, max(left, right)+1, 8):
                candidate = source[pos:pos+len(reference)*8:8]
                norm = sum(x*x for x in candidate)
                correlation = sum(a*b for a,b in zip(reference,candidate)) / math.sqrt(reference_norm*norm) if norm else -1
                correlation -= abs(pos-expected)*1e-7
                if correlation > score:
                    best, score = pos, correlation
        for i in range(min(window, size-out, len(source)-best)):
            value = source[best+i]
            if i < hop:
                value = round(result[out+i]*(1-i/hop) + value*(i/hop))
            result[out+i] = value
    if sys.byteorder != 'little':
        result.byteswap()
    return result.tobytes()


class TTSWorker:
    def __init__(self, job_dir, mode, speak_backend=None, synth_factory=None):
        self.job_dir, self.mode = job_dir, mode
        self.jobs = queue.Queue()
        self.speak_backend = speak_backend
        self.synth_factory = synth_factory
        self.thread = threading.Thread(target=self._run, daemon=True)
        self.thread.start()

    def submit(self, job):
        completed = threading.Event()
        result = {}
        self.jobs.put((dict(job), result, completed))
        return result, completed

    def _run(self):
        while True:
            item = self.jobs.get()
            if item is None:
                self.jobs.task_done()
                return
            job, result, completed = item
            try:
                result.update(self._process(job))
            except Exception as exc:
                detail = str(exc).replace(str(job.get('text','')), '[message]')[:500]
                detail = re.sub(r'https?://[^\s]+', '[remote endpoint]', detail)
                result.update(ok=False, engine=job.get('voice','gtts'), error=type(exc).__name__, detail=detail)
            finally:
                completed.set()
                self.jobs.task_done()

    def _process(self, job):
        text = job.get('text', '')
        if not isinstance(text, str) or not text.strip() or len(text) > 1600:
            raise ValueError('invalid text')
        voice, speed = job.get('voice','gtts'), job.get('speed',1.1)
        repeat=job.get('repeat',1)
        if type(repeat) is not int or repeat not in (1,2):raise ValueError('repeat must be 1 or 2')
        if voice not in ('gtts','sunhi','injoon','sapi') or not isinstance(speed,(int,float)) or not math.isfinite(speed) or not .8 <= speed <= 1.5 or abs(speed*20-round(speed*20))>1e-7:
            raise ValueError('invalid voice/speed')
        with tempfile.TemporaryDirectory(prefix='speech-', dir=self.job_dir) as folder:
            path = os.path.join(folder, 'speech.wav' if voice=='sapi' else 'speech.mp3')
            if voice == 'gtts':
                factory = self.synth_factory
                if factory is None:
                    from gtts import gTTS
                    factory = gTTS
                factory(text=text, lang='ko', slow=False, timeout=(5,12)).save(path)
            elif voice in ('sunhi','injoon'):
                self._edge(text,path,voice,speed)
            else:
                self._sapi(text,path)
            if os.path.getsize(path) > 8*1024*1024:
                raise ValueError('audio too large')
            if self.mode == 'synth':
                with open(path,'rb') as f:
                    return dict(ok=True,mp3=base64.b64encode(f.read()).decode('ascii'))
            if self.speak_backend:
                self.speak_backend(path)
            else:
                self._play(path,job.get('chime'),voice,speed,repeat,job.get('not_before',0))
            return dict(ok=True)

    @staticmethod
    def _edge(text,path,voice,speed):
        import edge_tts
        model={'sunhi':'ko-KR-SunHiNeural','injoon':'ko-KR-InJoonNeural'}[voice]
        rate=f'{round((speed-1)*100):+d}%'
        async def generate():
            await edge_tts.Communicate(text,model,rate=rate,connect_timeout=5,receive_timeout=20).save(path)
        asyncio.run(generate())

    @staticmethod
    def _sapi(text,path):
        # Offline SAPI default voice, never silently replaced with another engine/voice.
        import comtypes
        from comtypes.client import CreateObject
        comtypes.CoInitialize()
        voice,stream=None,None
        opened=False
        try:
            voice=CreateObject('SAPI.SpVoice',dynamic=True)
            stream=CreateObject('SAPI.SpFileStream',dynamic=True)
            stream.Format.Type=22  # 22.05kHz / 16bit mono PCM WAV
            stream.Open(path,3,False)
            opened=True
            voice.AudioOutputStream=stream
            voice.Rate=0
            voice.Speak(text,16)  # SVSFIsNotXML: names/messages are plain text
        finally:
            if stream is not None and opened:
                stream.Close()
            voice,stream=None,None
            comtypes.CoUninitialize()

    @staticmethod
    def _play(path,chime,voice='gtts',speed=1.1,repeat=1,not_before=0):
        os.environ['PYGAME_HIDE_SUPPORT_PROMPT']='1'
        import pygame
        frequency=int(round(24000*speed)) if voice=='gtts' else 24000
        pygame.mixer.init(frequency=frequency,size=-16,channels=1,allowedchanges=0)
        try:
            if chime:
                channel=pygame.mixer.Sound(chime).play()
                while channel and channel.get_busy():time.sleep(.02)
            sound=pygame.mixer.Sound(path)
            raw=sound.get_raw()
            # edge-tts already applied +/-% during synthesis: do not speed it up twice.
            pcm=raw if voice in ('sunhi','injoon') or speed==1 else stretch_pcm(raw,frequency,speed)
            prepared=pygame.mixer.Sound(buffer=pcm)
            delay=max(0,min(2,float(not_before)-time.time()))
            if delay:time.sleep(delay)
            channel=prepared.play(loops=repeat-1)
            if channel is None:raise RuntimeError('No available audio output channel')
            while channel.get_busy():time.sleep(.02)
        finally:
            pygame.mixer.quit()

    def close(self):
        self.jobs.put(None)
        self.jobs.join()
        self.thread.join(timeout=1)


def main():
    mode, folder = sys.argv[1:3]
    if mode not in ('synth', 'speak'):
        raise ValueError('mode')
    os.makedirs(folder, exist_ok=True)
    worker = TTSWorker(folder, mode)
    try:
        for line in sys.stdin:
            try:
                job = json.loads(line)
                result, done = worker.submit(job)
                done.wait()
            except Exception:
                result = dict(ok=False, error='GTTS_AUDIO_FAILED')
            print(json.dumps(result, ensure_ascii=True), flush=True)
    finally:
        worker.close()


if __name__ == '__main__':
    main()
