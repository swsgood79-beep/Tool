# v0.2 근거와 검증 범위

이 문서는 원본 동일성 인증서가 아닙니다. 기존 API-CONTRACTS.md/route-coverage.json은 v0.1 조사 당시 자료이며, 이번 추가 범위는 이 문서를 기준으로 봅니다.

| 근거 | 확인/구현 |
|---|---|
| `main.dataDir` → `os.UserConfigDir`, 문자열 WanryoTalk (길이 10) | Windows `%AppData%\WanryoTalk` 자동 감지 |
| `preparePCServer`, `openHost`의 파일명 참조 | projects-v2.dat, connection-v1.dat, phone-push-v1.dat, designers-v1.dat, client-notes-v1.dat, backup-v1.dat |
| `Host.photoPath`의 project-photos, .dat | 기존 JPEG 보호 파일 읽기 |
| HostData/WorkOrder/Completion/Event/InspectionState/ClientNote 등 RTTI | 기존 키·단계·요청·완료·읽음·사진·설정 변환. Completion.Event는 embedded로 JSON이 펼쳐짐 |
| encryptData의 CryptProtectData/CryptUnprotectData | 현재 Windows 계정 DPAPI. 실제 회사 파일로 확인하지 않음 |
| 원본 bcrypt 심볼 | 기존 bcrypt 관리자 비밀번호 확인 |
| `pages/admin/passkeys.js`, passkey-enter.js | /passkeys/register/{begin,finish}, /passkeys/auth/{begin,finish}, /mobile-admin/passkey/{enroll,start,enter,finish} |
| SDK v0.18.2 및 원본 storedPasskey 구조체 | RP/Origin 고정, UV/서명·카운터 확인, owner/revision/expiry, 일회용 흐름·티켓 |
| `pages/phone/push.js`, sw.js 및 phoneData/phonePayload | /phone/register + Authorization Bearer, VAPID 키, 암호화 payload와 project_id/request_id/pending_approval, 동일한 클릭 동선 |
| 원본 PlaySoundW/native toast 함수 | 새 PC 알림은 Windows 토스트로 재구현. 원본 창/숫자 배지와 같지 않음. TTS 호출은 확인하지 못함 |
| 제공된 토스 OpenAPI 1.2.28와 marketService 참조 | /oauth2/token, /api/v1/market-indicators/prices?symbols=KOSPI%2CKOSDAQ, /{symbol}/candles?interval=1d&count=10, /api/v1/exchange-rate?baseCurrency=USD&quoteCurrency=KRW |
| MarketIndicatorPriceResponse | symbol, lastPrice(문자열), timestamp(nullable) |
| MarketIndicatorCandle | timestamp, closePrice(문자열) |
| ExchangeRateResponse | midRate, validFrom/validUntil. 현재 표시는 midRate·validFrom 기준 |
| 원본 Upbit URL | /v1/ticker?markets=KRW-BTC%2CKRW-ETH, signed_change_rate×100 |
| tunnelBinary 및 고정 Pages JS | cloudflared-2026.10.7.exe, SHA-256 2493dbc168923bb3e59e5e7716f71deced92992eea9acd6e9d74f84ac7751343, 현 프로세스 ready 확인, 기존 포인터 스키마 |
| readScheduleSourceClient | api.onedrive.com/v1.0/shares/u!{base64url(shared-url)}/root/content |
| 실제 첨부 일정표 | 조립일정, 발송예정일/수주NO, 다행 수주·빈 행 및 미정 처리. 수주 52개·충돌 0개. 파일/행 내용은 소스에 포함하지 않음 |
| backupManifest/RecoveryKey 및 createEncryptedBackup | age X25519, manifest.json·records.xlsx·photos/{id}.jpg 원본 v1/v2 읽기 |
| 원본 GitHub 태그·라벨 | alpha-backup-{BoxID}, alpha-portable-backup-v1, alpha-pending-v1, alpha-verified-v1 |
| 새 업로드 | 원본 Release를 건드리지 않는 alpha-recovered-backup-{BoxID}; label recovered-v2, 다운로드/복호화 검증 후 최근 3개 |

시험은 서로 구분합니다.

- 회사 DPAPI/기존 사진/기존 백업: 원본 스키마에 맞춘 시험 파일만 검증. 실제 회사 파일은 없음.
- WebAuthn: 실제 P-256 ECDSA/CBOR 등록·서명 및 SDK 검증, challenge·UV·일회용 흐름. 실제 지문 센서는 사용하지 않음.
- Web Push: SDK가 만든 실제 암호화/VAPID HTTP 요청을 대체 전송 계층에서 검증. 푸시 제공자나 휴대폰은 사용하지 않음.
- 시세/GitHub: 원본 URL/문서 필드에 맞춘 대체 HTTP 응답과 요청 검증. 실제 계정·IP와 서버 접속은 검증하지 않음.
- OneDrive: 첨부 원본 XLSX와 생성 XLSX 파서 검증. 실제 공유 URL의 Microsoft 다운로드 권한은 검증하지 않음.
- Cloudflare: 설정·검증·프로세스 관리·포인터 소유권 코드를 작성. 실제 터널 도구를 실행하지 않음.
- 브라우저: Linux Chromium에서 로컬 업무·설정 화면 확인. Windows 네이티브 동작은 교차 빌드까지만 확인.

따라서 기존 회사 폴더에 실제 쓰기, 원래 네이티브 UI, 원래 복원 저널의 전 분기, 대용량 백업, 이미지 변환·일괄등록을 포함한 **원본 100% 동등성은 미확인/미완료**입니다.
