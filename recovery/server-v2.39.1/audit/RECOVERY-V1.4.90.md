# PC v1.4.90 · 검사자 자동 귀속/과거 보완/대표 이름

Stage/Event의 optional Inspector와 State.OwnerName/InspectionAudits를 기존 _RecoveredV1 확장 상태 및 백업에 저장합니다. 인증된 검사 완료 경로가 현재 계정의 이름/아이디/ID를 승인 transaction에 전달합니다. 자동 승인과 사진 스냅샷 캡처/실패가 같은 원자성 범위이며 새 신청/취소는 기존 단계 초기화로 검사자를 이어받지 않습니다. 이름은 매번 계정 조회로 과거를 바꾸지 않고 스냅샷으로 보관합니다.

과거 보완은 owner/CSRF를 확인하고 완료된 Event/현재 Stage/ShippingRecord.Stage를 project/kind/event 또는 단계 완료시각 키로 묶습니다. 모든 사본이 미기록인 완료 검사만 후보로 삼고 revision/선택 범위/확인 체크를 확인합니다. 원래 날짜·작업자·보관일은 유지하며 감사 로그에 처리 주체/시각/대상을 담습니다. 기존 이름 덮어쓰기와 낡은 폼 제출을 막습니다. 자동 귀속은 동료 검사 관리자도 자신의 세션으로 처리하며 구매 계정은 기존 gate로 차단합니다.

대표 이름은 별도 OwnerName으로 저장하며 wssong/PasswordHash/Integration.Revision/기존 로그인은 유지합니다. 네이티브 계정 이름도 새 이름을 사용합니다. UI는 승인 예정 이름·완료 기록·출하 상세 두 카드와 보완 관리 화면을 제공합니다. 출하 XLSX는 정보 다음에 두 검사자 카드, 월별 검사 엑셀은 별도 검사자 기록 시트를 더하며 사용자 원본 6개 열/시트는 보존합니다. 원래 web 57개 자산과 APK 1.0.11 유지.

서버 HTTP·권한/CSRF·서로 다른 세팅/출하 승인·통합·위조 이름·날짜 보존·원본 형식 재실행·백업·Excelize 검사·LibreOffice 렌더 및 Chromium PC/휴대폰 밝게/어둡게 화면을 검증합니다. 회사 운영 데이터나 실명은 테스트/소스에 포함하지 않고 합성 이름/사진을 사용합니다. 실제 Windows GUI·Android·Microsoft Excel은 실행하지 않았습니다.
