# v1.4.17 TTS 설정·테스트와 4개 엔진

회사 프로그램은 Go/Win32 창과 웹 GUI, 포함된 Python 음성 작업자로 구성된다. Windows 창의 ‘음성 설정·테스트’를 누르면 일회용 관리자 연결로 로컬 /tts-settings GUI가 열린다. 기존 관리자 설정 메뉴에도 진입 링크를 제공한다. 별도의 Python/Tkinter 앱을 새로 만들지 않고 기존 회사 프로그램의 GUI 설정 영역을 추가했다.

체크박스, 요청한 네 목소리 옵션, min0.8/max1.5/step0.05 슬라이더, 실시간 1.10x 숫자, 테스트 버튼을 제공한다. 새 설정 기본값은 ON/gTTS/1.10이고 기존 OFF는 이관해 유지한다. 테스트 문장은 정확히 ‘아리아니 공구14님이 테스트 완료’이다. 이는 고정 미리듣기이며 실제 검사 완료 알림을 추가하지 않는다. 실제 업무 알림은 이전처럼 관리자 신청·비고, 공용 신청만이다.

운영 자료 폴더의 tts_config.json에 enabled/voice/speed를 자동 저장하고 State.Integration.TTS와 기존 DesktopSpeechMuted에 사본을 반영한다. JSON 쓰기와 상태 저장 실패 시 이전 JSON을 되돌리고 적용하지 않는다. 파일 손상은 화면 오류와 음성 중단으로 처리하며 기본 엔진으로 몰래 바꾸지 않는다. 정상 업무 화면은 계속 열린다. 설정은 재시작/운영 복사/암호화 portable 백업 복원에서도 보존한다. 복원은 sidecar를 복원된 설정에서 다시 만든다. 실제 회사의 GitHub 데이터 백업 성공을 확인한 것은 아니다.

설정 변경/상태/테스트는 기존 관리자 인증·CSRF와 localRequest 경계를 요구한다. 원격 관리자나 board만으로 PC 설정을 바꾸지 못한다. PC 직접 음성 ON/OFF와 각 브라우저의 음성 켜기는 구분한다. 로컬 브라우저 음성으로 전환하면 이전 동작처럼 PC 직접 음성은 꺼짐으로 저장하고 중복을 막는다. 설정 화면의 수동 테스트는 OFF 상태에서도 같은 FIFO로 재생한다. 이 PC의 다른 브라우저 음성 임대가 활성이라면 테스트는 안내 오류를 반환해 소리가 겹치지 않게 한다.

수동 테스트는 클릭 시점의 TTSConfig 사본을 queue job에 붙인다. 뒤에서 다른 목소리로 설정을 바꿔도 대기 중인 테스트의 목소리/배속이 바뀌지 않는다. 실제 신규 신청은 재생 작업을 시작할 때 최신 설정을 읽는다. queue.Queue/daemon Python 작업자와 Go 단일 알림 작업자는 유지하며 HTTP/GUI 잠금 밖에서 합성·재생한다. UI/수주 처리를 기다리게 하지 않는다.

1) gTTS(lang=ko,slow=False)는 임시 MP3를 만든다. pygame 출력 frequency=round(24000*speed)만으로 배속이 생긴다고 가정하지 않고 디코딩된 PCM에 WSOLA 시간 조정을 별도로 적용한다. 2) edge-tts SunHiNeural, 3) InJoonNeural은 speed를 ±%로 변환해 합성에 넘긴다. 합성 후 pygame/브라우저 속도는 1.0으로 해 두 번 가속하지 않는다. 4) SAPI는 comtypes로 서버 Windows의 기본 SpVoice를 그대로 사용하며 인터넷 없이 PCM WAV를 만든다. SVSFIsNotXML=16으로 메시지를 코드/SSML로 해석하지 않는다. 기본 SAPI 음성이 남성/영어일 수 있으며 이는 해당 옵션 자체의 기본값이다. 다른 엔진이 실패했을 때 SAPI로 자동 전환하는 기능은 없다.

모든 분기에서 예외는 선택된 엔진명과 Python 예외 종류/원인을 반환한다. 메시지 내용 및 URL 토큰은 오류 문자열에서 제거한다. HTTP 403, NoAudioReceived, COM 등록/장치 오류 등의 이유를 GUI 상태에 표시한다. 실패 후 다음 작업은 같은 엔진으로 정상 처리하며 자동 fallback은 없다. 임시 MP3/WAV는 finally 및 Go job cleanup으로 제거한다. 브라우저도 응답 Content-Type(MP3/WAV)과 X-TTS-Playback-Rate/Requested-Speed/Voice를 따라 재생하며 엔진을 변경하지 않는다.

포함 런타임에 edge-tts7.3.1/comtypes1.5.0과 의존성을 추가했다. 실제 wheel 버전·SHA/파일 SHA/라이선스는 runtime-manifest.json과 번들 ZIP에 보존한다. Go EXE 사용자는 pip 설치가 없다. 직접 Python 소스를 실행할 때만 pip install gTTS pygame edge-tts comtypes 또는 tts_assets/requirements.txt를 사용한다.

검증: Go race 전체 시험, 기본값/범위/저장·재시작/손상/저장 실패 rollback/portable 백업 복원/설정 GUI·OFF 수동 테스트/클릭 시점 선택 고정/엔진별 배속 헤더·오류 원인과 fallback 없음, 기존 권한/작업 흐름을 검증한다. Python 8개 시험은 모델·±rate/중복 가속 금지, SAPI COM 호출 계약, 선택 엔진 실패와 temp 정리, 실제 pygame MP3 디코딩(SDL dummy), 시간 길이·음높이와 큐를 검증한다. 브라우저는 설정 GUI 2크기와 4엔진×관리자/공용×2크기 16조합으로 조절·자동 저장·수동 재생 요청/MP3·WAV/1회 배속/끄기·실패 복구를 검증한다. SAPI WAV를 초기 무음 WAV와 혼동한 첫 모의 오디오 시험 도구를 수정했다. 회사 프로그램 오류가 아니라 테스트 모형의 형식 구분 문제였다.

개발 환경에서 실제 Google/Microsoft 서비스 합성, 실제 Windows COM/설치된 기본 목소리·스피커 출력은 확인하지 못했다. Python SAPI 호출 및 온라인 엔진은 모의 응답, pygame은 실제 더미 오디오, Windows는 GUI 교차 빌드로 확인한 범위를 구분한다. 회사 PC에서 테스트 재생으로 최종 확인해야 한다. 크롬 내장 Google 음성과의 완전 동일성은 보장하지 않는다.

기존 web57개/버전 ZIP은 삭제하지 않는다. 전체 소스/종속 런타임/Windows ZIP/체크섬/소스 ZIP 동일 EXE 재빌드 기록을 복구 브랜치에 보관한다. 회사 실제 tts_config.json·수주·사진·키는 업로드하지 않는다.
