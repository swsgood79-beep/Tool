"""gTTS worker: JSON lines on stdin, one result per job. No UI work on this thread."""
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
        self.jobs.put((job, result, completed))
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
            except Exception:
                # Never print names/messages or network credentials in diagnostics.
                result.update(ok=False, error='GTTS_AUDIO_FAILED')
            finally:
                completed.set()
                self.jobs.task_done()

    def _process(self, job):
        text = job.get('text', '')
        if not isinstance(text, str) or not text.strip() or len(text) > 1600:
            raise ValueError('invalid text')
        factory = self.synth_factory
        if factory is None:
            from gtts import gTTS
            factory = gTTS
        with tempfile.TemporaryDirectory(prefix='speech-', dir=self.job_dir) as folder:
            path = os.path.join(folder, 'speech.mp3')
            factory(text=text, lang='ko', slow=False, timeout=(5, 12)).save(path)
            if os.path.getsize(path) > 8*1024*1024:
                raise ValueError('audio too large')
            if self.mode == 'synth':
                with open(path, 'rb') as f:
                    return dict(ok=True, mp3=base64.b64encode(f.read()).decode('ascii'))
            if self.speak_backend:
                self.speak_backend(path)
            else:
                self._play(path, job.get('chime'))
            # Temporary MP3 is deleted on success, playback failure and network failure.
            return dict(ok=True)

    @staticmethod
    def _play(path, chime):
        os.environ['PYGAME_HIDE_SUPPORT_PROMPT'] = '1'
        import pygame
        pygame.mixer.init(frequency=26400, size=-16, channels=1, allowedchanges=0)
        try:
            if chime:
                channel = pygame.mixer.Sound(chime).play()
                while channel and channel.get_busy():
                    time.sleep(.02)
            sound = pygame.mixer.Sound(path)
            pcm = stretch_pcm(sound.get_raw(), sample_rate=26400, speed=1.1)
            channel = pygame.mixer.Sound(buffer=pcm).play()
            if channel is None:
                raise RuntimeError('no mixer channel')
            while channel.get_busy():
                time.sleep(.02)
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
