# ERP 연결 진단 도구 v0.2

이 문서는 기본 진단 모드의 설명입니다. 별도 엑셀 저장 시험 모드는 [EXPORT-TEST.md](EXPORT-TEST.md), 가려진 창 내보내기 시험은 [BACKGROUND-TEST.md](BACKGROUND-TEST.md)를 참고하세요.

현재 알파 PC 프로그램과 별도로 실행하는 Windows x64 확인용 도구입니다. PrimeERP의 버튼·목록을 자동으로 식별할 수 있는지 확인합니다. **ERP 자동 수집 기능은 아직 포함하지 않습니다.** 이 결과로 엑셀 내보내기 자동화를 구체화합니다.

1. ZIP을 내려받아 폴더에 압축을 풉니다.
2. PrimeERP에서 **발주대비미입고현황** 화면을 열어 둡니다.
3. `Alpha-ERP-Connection-Check.exe`를 실행합니다.
4. 확인 메시지에서 확인을 누르면 결과 폴더가 열립니다.
5. 폴더 안 **ERP-connection-check.json**을 대화에 첨부합니다.

결과 폴더는 실행 파일 옆에 매번 새로 만듭니다. 해당 위치에 저장할 수 없으면 Windows 임시 폴더의 AlphaERPCheck 아래에 만들고 그 폴더를 엽니다. 처음 확인하는 ERP 창은 최대 3개이며 UI Automation 확인은 창마다 최대 15초입니다.

창·컨트롤 종류/ID/위치/표준 패턴 지원 여부와 고정된 버튼/열 제목만 기록합니다. 입력창 값, 품목 행 내용, 도면/고객사/직원 이름, 비밀번호와 DB 연결 설정은 수집하지 않습니다. 결과를 자동으로 전송하지 않습니다. ERP 버튼을 누르거나 발주 등록/수정/삭제, 엑셀 저장을 실행하지 않습니다. 기존 알파 자료와 설정은 사용하지 않습니다.

Windows 기본 API로 먼저 확인한 뒤 기본 제공 Windows PowerShell/.NET UI Automation으로 추가 구조를 확인합니다. PowerShell이 차단되거나 UIA 응답이 없으면 Win32 결과와 제한 상태를 기록합니다. 검사 대상 ERP 프로세스를 종료하지 않습니다. ERP 창을 찾지 못하면 발주현황 화면을 연 뒤 다시 실행합니다. 읽히지 않는 항목이 있어도 결과 파일을 보내주시면 됩니다.

## 소스와 검증

ZIP의 source 폴더에 전체 도구 소스를 포함합니다. 추가 Go 라이브러리 없이 Windows 기본 API만 사용합니다. Linux에서 고정 라벨 선별/기존 파일 보존을 시험하고 Windows x64 교차 빌드와 vet를 실행했습니다. 이 개발 환경에서는 실제 회사 Windows/PrimeERP에서 실행하지 않았습니다. UI 구조를 알아야 다음 단계의 자동 조회·엑셀 추출을 구현할 수 있습니다.

```text
Go 1.24 이상
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o Alpha-ERP-Connection-Check.exe .
```

## v0.2 보완

최소화된 ERP 창은 펼친 뒤 화면 구조를 읽습니다. UIA JSON의 UTF-8 BOM/UTF-16 인코딩을 정규화합니다. 기존 v0.1 결과에서는 ERP 실행/컨트롤155개가 확인됐지만 최소화 좌표와 UIA 해석 실패가 있어 자동 추출 가능 여부를 확정하지 않았습니다. 인코딩 보완이 해당 PC의 UIA 실패 원인이라는 점은 아직 확정되지 않았습니다. ERP 발주/입고 자료와 버튼은 수정·실행하지 않습니다.
