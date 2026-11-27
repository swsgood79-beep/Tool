# PC v1.4.118 ERP 조회 실행 실패 복구 메모

## 확인한 원인과 구현

기준은 공개 recovery-v2.39.1의 1233543, PC v1.4.117 / APK v1.0.11입니다. 새 작업 브랜치는 work-pc-v1.4.118입니다. 사용자가 제공한 진단은 실행 단계/DB 오류 코드 없이 failed였습니다. 사용자 첨부 파일과 화면은 공개 저장소에 포함하지 않습니다.

v1.4.117의 ERP 스크립트를 전체 UTF-16 Base64로 -EncodedCommand에 넣으면 스크립트만 34,284자가 되어 Windows CreateProcess 명령 줄 한도 32,767자를 초과합니다. 기존 코드는 cmd.Start 실패 이유를 기록하지 않아 단순 failed로만 보였습니다. v1.4.116 스크립트 인자는 30,272자, v1.4.115는 23,976자입니다.

짧은 고정 부트스트랩만 실행 인자로 전달하고 전체 UTF-8 스크립트는 표준입력으로 전달하도록 수정했습니다. 같은 PowerShell/INI/SQL 및 프로세스 종료 관리 구조를 유지합니다. 프로세스 시작/준비/종료/응답 해석 실패 단계를 구분하고 Windows 시작 오류 번호를 보관합니다. 자동 반영 보류에도 실제 실패 문구 또는 조회 잘림을 표시합니다.

## 수정 파일

recovery/server-v2.39.1/erp_db_powershell.go, erp_db_probe_windows.go, erp_db_probe.go, erp_db_sync.go, production.go.

## 남은 작업과 다음 단계

현재는 구현 소스 중간 저장 단계입니다. 실제 크기 스크립트/한글 경로/각 조회 모드/Windows 인자 길이/표준입력 전달 회귀 테스트, 합성 Chromium 실패/회복 화면, 전체 Go 테스트/race/vet/Windows 교차 빌드, 소스 ZIP 재빌드, 공개 다운로드/업데이트 검증이 남았습니다. 기존 운영 자료와 DB 설정을 변경하거나 실제 회사 DB에 연결하지 않았습니다. 공개 업데이트는 아직 v1.4.117입니다.

## 실행 방식 회귀 검증 통과 및 화면 보완

진단/기간 미리보기/자동 반영/수주별 미입고 4개 모드에서 전체 스크립트는 기존 한도보다 크지만 실행 인자는 2,048자 미만이며 동일한 고정 부트스트랩만 사용함을 확인했습니다. 한글/따옴표 경로와 수주값, 전체 표준입력 자식 프로세스 전달, 시작 오류 206 및 각 실패 단계 디코딩, 기존 자동 반영/수주 조회 회귀 테스트가 통과했습니다.

신규 테스트 파일은 erp_db_powershell_test.go입니다. 실패 상태를 녹색 정상 표시와 혼동하지 않도록 경고색을 추가하고, 실제 Chromium 합성 오류/복구 테스트를 release118_browser_test.go와 audit/browser-erp-launch-v1.4.118.cjs에 준비했습니다. 신규 화면 자산은 extras/erp-db-check-v1.4.118.js, extras/erp-db-status-v1.4.118.css이며 extras/machining.html에서 사용합니다. 다음 단계는 화면과 전체 검증 및 새 버전 패키징입니다.

## 화면 검증 통과와 배포 문서 준비

합성 Windows 시작 오류 206을 사용하는 실제 Chromium 320/390/430/1100px 4개 경고 화면과 정상 재시도 1개 사례가 통과했습니다. 가로 넘침, 오류 상세 표시, 기존 수주/수량/설정 보존과 정상 경고색 해제를 확인했습니다. 5장 캡처는 모두 합성 자료입니다.

README.md, START-HERE.md, CHANGELOG.md, audit/RECOVERY-V1.4.118.md 및 패키징/소스 동일 재빌드 스크립트를 준비했습니다. 루트 AGENTS.md의 최신 복구 메모도 1.4.118로 갱신했습니다. 전체 Go 테스트와 최종 빌드 검증 진행 후 결과를 저장하고 패키징합니다. 공개 업데이트는 아직 v1.4.117입니다.

## 최종 검증 통과

전체 go test ./... (63.588초), 관련 race (11.032초), Linux/Windows vet, Windows GUI 교차 빌드, 실제 Chromium 5개 사례가 모두 통과했습니다. 로그는 audit/*v1.4.118.txt에 저장했습니다. 실행파일은 /workspace/work/AlphaWorkComplete-Recovered-v1.4.118.exe입니다. 신규 코드 변경 없이 전체 추적 소스 파일을 패키징하고 Windows 실행파일 동일 해시 재빌드를 진행합니다. 실제 회사 Windows PowerShell/DB 실행은 현장 확인 대상입니다. 공개 업데이트는 아직 v1.4.117입니다.
