# v1.4.6 엑셀 폴더 열기

원인: idFolder가 웹 URL용 rundll32 url.dll,FileProtocolHandler에 Windows 폴더 경로를 넘기고 실패를 표시하지 않았다. 폴더 전용 openDataFolder로 분리하고 절대 경로/디렉터리 확인 후 ShellExecuteW(open, SW_SHOWNORMAL)를 사용한다. 웹 브라우저 경로와 인터넷 도구의 콘솔 숨김 설정은 유지한다. ShellExecute 반환값 <=32는 실패로 처리한다. 실패 메시지에 원인/저장 경로/Windows키+R 대안을 표시한다. 데이터를 이동·삭제하거나 기존 원본에 쓰지 않는다.

folder_open_test.go: 한글/공백 경로가 그대로 전달되는지, shell 실패가 사라지지 않는지, 파일/없는 폴더를 실행하지 않는지 확인한다. Windows 빌드와 vet는 수행하나 실제 Windows 탐색기 실행은 이 환경에서 검증하지 못했다. 이전 브라우저 결과는 해당 버전 당시 결과로 보존한다. v1.4.5 완료 수주 필터와 v1.4.4 신청 알림/비고 알림 정책은 유지한다.
