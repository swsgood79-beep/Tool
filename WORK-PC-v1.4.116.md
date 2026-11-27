# PC v1.4.116 최신 복구 메모

## 상태

구현, 합성 테스트, Windows 교차 빌드, 배포 소스 ZIP 재빌드가 완료되었습니다. 공개 업데이트 정보와 Windows/소스 ZIP 전체 다운로드 해시 검증까지 완료되었습니다. 작업 브랜치는 work-pc-v1.4.116이고 정식 브랜치는 recovery-v2.39.1입니다. 정식 배포 커밋은 c68c5a2이며 공개 업데이트는 PC v1.4.116 / APK v1.0.11입니다.

## 완료한 작업

- 대표 관리자 직접 실행형 가공 테스트 기록 초기화. 계정 ID, 미리보기, 명시 체크, DPAPI 보호 백업 및 복호화 확인, 변경 감지, 후속 타 작업자/입고 확인/입고 수량 보류, 실행 이력.
- 수주 전체 거래처/품목의 미입고 잔량 팝업. 기존 INI 재사용, 발주/가공의뢰 복합 키, 행별 수주 우선, 확인 불가 로봇 표시, 결과 잘림/실패 오류, 새로고침/조회 시각.
- 모바일 320/360/390/430px와 PC 1100px 실제 Chromium 화면 검증. 모든 자료는 합성입니다.
- 기존 계정/권한/로그인/수주/필요 수량/다른 작업자 기록 보존 테스트. 실제 업무 초기화는 실행하지 않았습니다.

## 수정 파일

recovery/server-v2.39.1의 machining_reset.go, machining_reset_test.go, model.go, machining_http.go, order_outstanding.go, order_outstanding_test.go, erp_db_probe.go, erp_db_probe_windows.go, erp-db-probe.ps1, extras/machining.html, extras/machining-reset.html, extras/machining-v1.4.116.js, extras/order-outstanding-v1.4.116.css/js, team_native_push.go, backup_stream.go, backup_sync.go, production.go, release116_browser_test.go와 audit의 1.4.116 검증 자료입니다. 루트 AGENTS.md에 사용자 지속 작업 기준을 기록했습니다.

## 검증 결과

최종 go test ./... 통과 (120.252초), 관련 race 통과, Chromium 7개 사례 통과, Linux/Windows go vet 통과, Windows GUI 교차 빌드 통과. 소스 ZIP의 SHA256SUMS 전체 확인과 Windows 실행파일 동일 해시 재빌드, 내장 ERP 보조 프로그램 동일 재빌드, APK v1.0.11 기존 해시 확인이 통과했습니다. 증빙은 recovery/server-builds/AlphaWorkComplete-Recovered-source-PC-v1.4.116-App-v1.0.11-rebuild.json입니다.

실제 회사 ERP 결과 대조, 실제 Windows GUI/DPAPI 및 휴대폰 수신 실기는 미검증입니다. 회사 내부 DB에 접근하지 않았습니다. 실제 업무 자료/업로드/백업/비밀번호는 추가하지 않았습니다.

## 배포 완료 확인

정식 브랜치 recovery-v2.39.1에 검증된 트리를 반영했고 pc-update.json을 v1.4.116으로 갱신했습니다. 공개 Windows ZIP 38,641,770바이트와 소스 ZIP 35,851,624바이트를 전체 다운로드하여 SHA-256이 패키징 결과 및 공개 해시 파일과 일치함을 확인했습니다. 공개 소스 재빌드 증빙의 실행파일 해시도 업데이트 정보와 일치합니다. 확인 시각은 2026-11-27 15:50 KST입니다. 공개 검증 증빙은 루트 PUBLIC-PC-v1.4.116.json입니다.

사용자는 기존 프로그램의 **PC 업데이트 확인**으로 설치합니다. 실제 회사 기록 초기화는 사용자가 대표 관리자 화면에서 미리보기와 백업 후 직접 실행합니다. 실제 업무 초기화를 대신 실행하지 않았습니다.

## 남은 현장 확인

회사 ERP의 수주별 전체 거래처 조회 결과를 현장 ERP 화면과 대조해야 합니다. 실제 Windows GUI/DPAPI 및 휴대폰 수신 실기는 클라우드에서 검증하지 못했습니다. 추가 수정은 AGENTS.md 기준대로 별도 작업 브랜치에서 작은 변경마다 소스/복구 메모 커밋·푸시 후 검증하여 새 버전으로 배포합니다.

세부 구현 과정은 recovery/server-v2.39.1/audit/RESUME-PC-v1.4.116.md에 있습니다. 이 루트 메모가 패키징 이후의 최신 상태입니다.
