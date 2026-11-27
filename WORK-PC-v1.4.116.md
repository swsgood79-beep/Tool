# PC v1.4.116 최신 복구 메모

## 상태

구현, 합성 테스트, Windows 교차 빌드, 배포 소스 ZIP 재빌드가 완료되었습니다. 아직 공개 배포 확인 전입니다. 작업 브랜치는 work-pc-v1.4.116이고 정식 브랜치는 recovery-v2.39.1입니다. 정식 브랜치는 현재 aea43c4에서 v1.4.115로 유지됩니다.

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

## 다음 단계

1. 작업 브랜치의 패키지 커밋이 GitHub에 올라간 것을 확인합니다.
2. recovery-v2.39.1이 여전히 aea43c4인지 확인합니다. 작업 브랜치의 최종 검증된 트리를 병합하여 앞서 되돌린 중간 기능을 복원합니다. 다른 작업자의 새 변경이 있다면 보존하여 통합합니다.
3. 새 1.4.116 Windows/소스 ZIP, 해시, 재빌드 증빙 및 pc-update.json을 함께 푸시합니다. 기존 버전 파일은 수정하지 않습니다.
4. 공개 업데이트 JSON 버전/URL/해시와 두 공개 ZIP 전체 다운로드 SHA-256을 확인합니다. 이 검증 전에는 배포 완료로 보고하지 않습니다.
5. 이 루트 메모에 최종 공개 확인 상태를 기록하고 푸시합니다. 기존 프로그램의 PC 업데이트 확인으로 사용하도록 안내하고 가상 자료 실제 구현 화면을 보여줍니다.

세부 구현 과정은 recovery/server-v2.39.1/audit/RESUME-PC-v1.4.116.md에 있습니다. 이 루트 메모가 패키징 이후의 최신 상태입니다.
