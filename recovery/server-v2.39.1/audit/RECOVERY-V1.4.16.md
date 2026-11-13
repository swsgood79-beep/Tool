# v1.4.16 gTTS 전면 교체

사용자는 현재 회사 Go/웹 프로그램의 음성을 gTTS로 교체한다고 확인했다. 기존 신청·비고 알림만 유지하며 완료 알림은 새로 만들지 않는다. 공용 작업자는 신청만, 관리자는 신청·비고를 읽는다. 업무/지문/납기/알림 승인·반려·자료 저장 경로는 유지한다.

활성 Windows SAPI/PowerShell 코드는 desktop_speech_windows.go에서 제거했다. 활성 웹은 새 gtts-speech-v1.4.16.js로 바꾸고 SpeechSynthesis를 호출하지 않는다. 원본 web57개와 과거 릴리스 소스/비활성 버전 JS는 복구 기록으로 보존한다. 기존 버전을 삭제하지 않는 보관 원칙과 활성 TTS 교체를 구분한다.

공식 PSF NuGet Python3.12.10 x64 패키지와 PyPI gTTS2.5.4/pygame2.6.1 및 고정 실제 의존성 wheel을 tts_assets/python-gtts-win-x64.zip에 묶는다. runtime-manifest.json에 원본 다운로드 해시와 2083개 파일 해시, 출처를 기록하고 Python/패키지 라이선스를 보존한다. 실행 EXE에 ZIP·Python helper·manifest를 embed한다. 별도 Python/pip 설치나 첫 실행 온라인 설치가 필요 없다. 첫 음성 사용 때 %LocalAppData%/AlphaWorkCompleteRecovered/gtts-runtime/{contenthash}에 전체 체크섬을 검증해 준비한다. 변조/손상된 기존 캐시는 덮어쓰지 않고 실패를 표시한다. 회사 원본/복사 데이터 폴더는 건드리지 않는다.

Python helper는 gTTS(text=text,lang='ko',slow=False,timeout=(5,12))로 임시 MP3를 만든다. queue.Queue/daemon threading.Thread로 작업을 순차 처리하고 try/except/finally로 오류를 결과에 담고 다음 작업을 계속한다. 생성/디코딩/재생/인터넷 오류 때도 TemporaryDirectory를 정리한다. Go는 알림 작업자에서 helper를 호출해 UI를 막지 않으며, 시간 제한/끄기 취소와 Windows Job Object로 자식 프로세스를 관리한다. Go가 소유한 임시 job 디렉터리도 Wait 후 삭제한다. 정상 종료·끄기/실패 때 파일 핸들을 해제한 뒤 정리한다. OS/PC 강제 종료 순간의 임시 파일 삭제를 보장하는 것은 아니다.

pygame.mixer.init(frequency=26400,size=-16,channels=1,allowedchanges=0)은 출력 형식을 정할 뿐 MP3를 1.1배속으로 만드는 설정이 아니다. MP3를 Sound로 디코딩한 PCM에 표준 Python array/math 기반 WSOLA overlap/add로 길이를 원본/1.1로 바꿔 재생한다. 음높이를 가능한 한 유지하며 별도 ffmpeg/변환 프로그램을 런타임에 호출하지 않는다. 크롬 내장 TTS와 완전히 동일한 음색/운율/속도 체감은 보장하지 않는다.

웹은 서버 Python helper의 synth 모드에서 만든 MP3를 받아 HTMLAudio playbackRate=1.1/preservesPitch=true로 재생한다. 해당 모드는 pygame 장치를 열지 않는다. 임시 MP3는 서버가 바이트를 읽은 직후 삭제하고 응답은 no-store이다. 브라우저는 blob URL을 종료/끄기/오류 때 해제한다. AudioContext와 같은 media element를 클릭 안에서 무음 WAV로 열어 재생 권한을 요청한다. 실제 알림은 MP3 준비 후 띵똥 → 음성 순이다. 생성/재생 FIFO, 중복 ID 제거, 취소/연결 오류/중도 끄기와 미리듣기를 유지한다. 이전 목소리로 자동 대체하지 않는다.

POST /admin-speech/audio는 기존 관리자 인증/CSRF, /worker-speech/{board}/audio는 board 및 본문 검증을 요구한다. 클라이언트의 임의 text는 받지 않고 이벤트 ID의 서버 메시지 또는 고정 미리듣기 문장만 합성한다. 공용 음성은 비고/숨김/완료/반려를 합성하지 못한다. App.mu 아래에서 인증/문장을 복사한 뒤 잠금을 해제하고 외부 요청/프로세스를 실행하므로 수주·검사 HTTP 요청을 막지 않는다. 동시 합성/직접 재생 프로세스는 2개로 제한한다. CSP에 media-src self blob만 추가했다. 외부 JS/임의 오디오 주소를 열지 않는다.

Python 단위 시험은 FIFO/일부 네트워크 실패 후 다음 작업/임시 MP3 정리, MP3 바이트 반환, 1.1배 샘플 길이와 440Hz 음높이 보존을 검증했다. 실제 pygame2.6.1에서 로컬 합성 신호 MP3를 디코딩하고 SDL dummy 출력으로 재생했다. 이 시험에서만 개발용 ffmpeg로 입력 시험 신호를 만들며 배포 실행에는 ffmpeg가 없다. Go 시험은 권한/공용 비고 보호/서버 문장 사용/상태 불변/느린 합성 중 수주 응답/취소/실패 후 복구와 Windows runtime 전체 체크섬/PE x64를 검증한다. 기존 전체 race 시험, Linux/Windows vet와 Windows GUI 교차 빌드를 수행한다.

브라우저 시험은 Go 합성 화면과 모의 MP3 API/Audio 출력으로 관리자·공용 × PC/모바일 4조합을 검증한다. 클릭 권한 호출, API 경로/CSRF, MP3 playbackRate1.1와 pitch보존, 띵똥/순서/중복 제거, 생성 실패 후 복구, 생성 중 끄기/URL 해제 및 기존 SpeechSynthesis를 사용하지 않음을 검증한다. 실제 Google 음성 합성 서버는 개발 프록시에서 403 Forbidden으로 차단되어 호출·음색을 검증하지 못했다. 실제 Windows 오디오 장치/삼성 인터넷 잠금 재생 역시 미검증이다. 이를 실제 Google 목소리를 들은 시험이라고 주장하지 않는다.

gTTS 합성은 서버 PC의 인터넷과 Google 서비스가 필요하며 업무 메시지가 Google로 전송된다. 연결이 끊겨도 검사/화면/원래 알림은 유지하고 음성 실패만 표시한다. Google API 키/유료 계약을 추가하지 않았다. 소스/종속 런타임/실행 ZIP/체크섬/소스 ZIP 동일 EXE 재빌드 기록을 기존 복구 브랜치에 보관한다. 회사 실제 데이터/키/사진은 포함하지 않는다.
