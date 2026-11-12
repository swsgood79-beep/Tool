# 알파 복구 서버 v0.2 · 호환·연동 검증판

**원본과 100% 동일하다고 검증된 최종 복원본은 아닙니다.** 원본의 Go 메타데이터·어셈블리·57개 웹 자산, 고정 Pages의 JS와 제공하신 토스 OpenAPI 문서를 대조해 기존 v0.1 서버에 호환 읽기 및 외부 연동을 추가했습니다.

## 실행과 기존 자료

Windows 실행용 ZIP을 풀고 `AlphaWorkRecoveredServer.exe`를 실행합니다. 서버 창이 열린 상태에서 브라우저가 자동으로 열립니다. 종료는 서버 창에서 `Ctrl+C`입니다.

기본 관리자 주소는 `http://127.0.0.1:18791/`입니다. 다음과 같이 자료 위치를 정합니다.

1. `%AppData%\WanryoTalk\projects-v2.dat`가 있으면 **원래 회사 자료 폴더**를 사용합니다. 같은 Windows 계정에서 DPAPI를 풀어 수주·기록·담당자·업체 특이사항·사진·연결/시세/일정/푸시/패스키 설정을 읽습니다. 기존 bcrypt 비밀번호도 확인합니다.
2. 원본 파일이 없으면 `%AppData%\AlphaWorkRecoveredServer`를 사용합니다. 새 PC에서는 새 비밀번호를 설정합니다.
3. 원본 .dat와 `project-photos` 파일은 덮어쓰지 않습니다. **복구판의 변경은 같은 폴더의 `reconstructed-state.bin`에 저장합니다. 원본 프로그램으로의 역방향 쓰기는 구현하지 않았습니다.**
4. 원본 파일이 변경되면 추가 저장을 중단합니다. 원본과 복구판을 번갈아 써서 자료를 합치는 기능은 없습니다. 원본 프로그램을 종료한 뒤 이 복구판을 실행하세요.
5. v0.1에서 별도 폴더에 만든 시험 자료를 계속 쓰려면 `-data`로 그 폴더를 지정합니다.

```powershell
.\AlphaWorkRecoveredServer.exe -data "D:\AlphaRecoveredTest" -listen 127.0.0.1:18793
```

이 환경에는 실제 회사 PC의 암호화 자료가 없으므로, 실제 DPAPI 파일·사진·설정으로 호환성을 확인한 것은 아닙니다. 자동 시험은 원본 RTTI에서 확인한 필드/태그를 갖춘 시험 파일을 사용합니다. 읽기에 실패하면 원본을 수정하지 않고 중단합니다. Windows 계정이 다르면 원본 DPAPI 자료는 읽을 수 없습니다.

## 이번에 연결한 네 영역

| 영역 | 구현 | 확인한 범위 |
|---|---|---|
| 기존 자료 | UserConfigDir/WanryoTalk, projects-v2.dat, designers-v1.dat, client-notes-v1.dat, connection-v1.dat, phone-push-v1.dat, backup-v1.dat, project-photos/{id}.dat. 원본 JSON 스냅샷·해시도 보호된 복구판 상태에 보존 | 스키마 기반 시험 파일, 원본 불변·QR 키 유지·bcrypt·동시 변경 차단 |
| 알림·지문 | PC Windows 토스트, 기존 고정 휴대폰 페이지용 VAPID Web Push, 동일 고정 RP의 WebAuthn 등록/로그인, UV·서명·카운터·일회용 flow/ticket·소유 세션 확인 | 실제 ECDSA/CBOR 등록·서명, 재사용/출처 차단, 암호화된 푸시 HTTP 요청. Windows/휴대폰 실기기 수신은 미확인 |
| 시세·외부 접속 | 토스 lastPrice/timestamp, 일봉 closePrice로 전일 대비, 토큰/전일 종가 캐시, 업비트, midRate/validFrom 환율, 429 대기, Cloudflare 도구 체크섬·프로세스 관리·외부 검증, GitHub 고정 QR 포인터 갱신 | 제공된 OpenAPI 필드 및 대체 응답 시험. 실제 회사 IP·키·Cloudflare 연결은 미확인 |
| 일정·백업 | OneDrive 공유 다운로드 또는 동기화 파일, 5분 갱신, 동일 수주 여러 행·날짜·미정/ASAP 처리, age 암호화 전체 백업, GitHub 업로드 후 다운로드/복호화 검증, 새 PC 복원 | 실제 첨부 일정표 52개 수주·충돌 0건. 실제 age 왕복·잘못된 키·손상 차단, 원본 v1/v2 매니페스트 시험, GitHub 요청 대체 시험 |

PC 알림은 원본의 자체 팝업 창 대신 Windows 토스트를 사용합니다. 원본의 음성 합성(TTS) 호출 근거는 확인하지 못했으며, 알림 관련 원본에는 `PlaySoundW`가 확인됩니다. 원본의 관리 창·트레이·떠 있는 안 읽은 숫자 창은 복원하지 않았습니다. 읽지 않은 개수와 신청 목록은 관리자 웹 화면에서 유지됩니다.

### 인터넷·휴대폰·지문

관리자 **설정 → 고정 QR 설정**에서 기존 GitHub 키를 유지/저장하고 **인터넷 연결 · PC 알림 → 인터넷 연결**을 누릅니다. 원본 폴더의 `cloudflared-2026.10.7.exe`를 SHA-256으로 확인해 재사용하며, 없으면 공식 GitHub 릴리스에서 받아 같은 해시를 확인합니다. 현재는 GitHub Pages/Cloudflare 방식만 구현했습니다. 이전 ngrok 설정값은 보존하지만 ngrok 연결 실행은 미구현입니다.

외부 주소는 `/ready/{board}`의 현재 프로세스 세션으로 확인한 뒤 게시합니다. 포인터에는 QR 접근 키나 수주 자료를 넣지 않습니다. 종료 시 다른 프로세스가 게시한 세션은 오프라인으로 덮어쓰지 않습니다. **개발·시험 중 실제 main 포인터를 변경한 적은 없습니다.** 연결을 누른 본인 PC에서 설정된 키로 게시됩니다.

휴대폰 관리자 접속을 켜고 패스키 등록 화면에서 비밀번호를 확인하면 기존 고정 Pages 주소로 이동합니다. RP는 `swsgood79-beep.github.io`, 허용 Origin은 `https://swsgood79-beep.github.io`입니다. 등록된 키가 없으면 지문 로그인 버튼으로 로그인되지 않습니다. 생체 원본이나 패스키 개인키는 서버에 전달되지 않습니다.

휴대폰 알림은 인터넷 연결 후 **내 휴대폰 알림 → 연결 QR 생성 → 휴대폰에서 허용** 순서입니다. 기존 구독/VAPID 키도 읽습니다. 잠금 수신과 삼성 인터넷·패스키 보관함·Windows 알림 정책은 실제 기기에서 확인해야 합니다.

### 시장·납기

토스 키는 기존 설정에서 읽거나 시장 설정에 입력합니다. 값은 Windows DPAPI로 보호하며 작업자에게 키·토큰을 보내지 않습니다. 토스는 허용 공인 IP 설정이 필요합니다. 코인과 주식 값을 만들어 내지 않습니다. 조회 실패 시 기존 값에 지연 표시를 사용합니다.

제공하신 OpenAPI의 실제 필드인 `lastPrice`, `timestamp`, `closePrice`, `midRate`, `validFrom`을 반영했습니다. 시세 시각이 없으면 조회 시각 `checked_at`으로 구분합니다. 일봉은 날짜별로 캐시하고 환율은 1분 캐시합니다.

‘조립일정’의 수주NO/발송예정일을 읽습니다. 같은 수주의 여러 행 중 빈 날짜는 확정 날짜와 충돌로 보지 않습니다. 명시적 ‘미정’/ASAP는 납기에서 제외하고, 서로 다른 확정 날짜나 해석할 수 없는 날짜는 기존 납기를 유지합니다. 본인 공유 주소의 실제 다운로드 권한·OneDrive 접속은 확인이 필요합니다. 첨부 원본 일정표는 소스나 ZIP에 포함하지 않았습니다.

### 백업·복원

복구키 JSON을 PC 밖에 보관한 뒤 자동 백업을 켭니다. 백업에는 복구판 상태·원본 JSON 스냅샷·연결 설정·키·담당자·업체·복호화한 사진·생성 엑셀이 들어가며 **전체 ZIP을 age X25519로 암호화한 .age 파일만 업로드**합니다. 복구키는 업로드하지 않습니다.

새 백업은 `alpha-recovered-backup-{BoxID}`라는 **별도 Release**에 저장합니다. 다운로드한 암호문 SHA-256과 실제 복호화/매니페스트 검증 후 성공 처리하며 복구판 백업만 최근 3개를 보관합니다. 원본 `alpha-backup-{BoxID}`와 기존 `alpha-verified-v1` 파일은 삭제하지 않습니다.

복원은 자료가 없는 새 복구 서버에서만 가능합니다. 새 형식이 없으면 기존 원본 Release에서 검증된 백업을 조회할 수 있으며 원본 Version 1/2 매니페스트와 photos/{id}.jpg 읽기를 구현했습니다. 실제 회사 백업으로는 아직 확인하지 않았습니다. 사진을 새 Windows 계정으로 다시 DPAPI 암호화합니다. **이 복구판이 만든 백업을 원본 프로그램으로 되돌려 복원하는 호환성은 보장하지 않습니다.**

현재 백업/복원은 메모리 사용을 제한하기 위해 256MB 상한이 있습니다. 원본의 약 1.9GB 처리 방식과 같지 않습니다. 이 상한을 넘는 자료의 스트리밍 처리, 원본 복원 저널 전체 동작도 미복원입니다.

## 나머지 차이

원본 파일에 대한 쓰기 호환, 원본 Windows 관리 창·트레이·숫자 배지·자체 알림 UI, ngrok, 수주 엑셀 일괄등록, 원본과 동일한 이미지 변환, 원본 엑셀 파일/수정 내역의 직접 병합, 사진 자동 청소는 미복원입니다. 이 항목 때문에 **원본 100% 상태로 완성했다고 말할 수 없습니다.** 변경 내용은 복구판에만 있으므로 원본 프로그램을 다시 열 때 두 자료가 자동으로 합쳐지지 않습니다.

Linux는 검증용이며 DPAPI 파일 보호와 Windows 네이티브 알림을 실행하지 않습니다. Windows 실행파일은 교차 빌드하며, Windows 실기기 실행·계정/IP·지문·휴대폰 알림·실제 계정의 외부 서비스 왕복은 별도 확인이 필요합니다.

## 근거와 빌드

`audit/main-struct-layouts.json`의 사용자 구조체 99개/필드 611개, `main-functions.json`의 사용자 함수 711개, `frontend-contracts.json` 및 `main-string-references.json`을 참고했습니다. `audit/RECOVERY-V0.2.md`에 API/경로와 검증 범위를 정리했습니다. 웹 자산 57개는 추출 원본과 바이트가 동일하며, 서버가 안내/설정 링크/수동 납기 입력란을 렌더링 때 추가합니다.

Go 1.27.3 이상. 원본에서 확인한 age·webpush·webauthn·excelize 버전을 go.mod/go.sum에 고정했습니다.

```sh
go mod download
go test -race ./...
go vet ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o AlphaWorkRecoveredServer.exe .
```

PowerShell에서는 `$env:GOOS="windows"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"`을 설정한 뒤 `go build`를 실행합니다. 원본 바이너리를 다시 실행하거나 기존 배포 실행파일을 교체하지 않았습니다. GitHub 복구 브랜치의 소스·실행용 ZIP으로 제공합니다.
