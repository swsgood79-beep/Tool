# v1.4.8 납기표 다운로드와 시장 상태 확인

v1.4.7은 eaf25b9로 업로드 완료했다. 본 버전은 그 기능을 포함한다.

OneDrive: 원본 main.readScheduleSourceClient 문자열/어셈블리에는 download=1 직접 URL과 /shares/u!.../root/content 재시도, UA/Accept/no-cache 헤더가 있었다. 복구 코드에는 API 하나만 있었고 storage.live.com 이동도 허용하지 않았다. 직접 다운로드와 원본 링크(redeem의 허용된 1drv.ms/onedrive.live.com URL만) 재시도, API fallback을 추가한다. 쿠키는 다운로드별 메모리 jar를 사용한다. 최대4후보/전체20초/32MB/8회 리디렉션/기존 Microsoft HTTPS 도메인 제한을 적용한다. 엑셀 대신 HTML/로그인 응답이면 다른 방식으로 재시도한다. 응답 본문/공유 비밀 URL을 오류 화면에 출력하지 않는다. 요청429는 재시도하지 않는다. 다운로드 중 라벨을 표시하고 실패해도 기존 납기 및 마지막 조회 성공 시각은 보존한다.

시장 설정: 원본 JS는 /market-settings/status.json과 message를 요구하지만 서버는 /market-settings/status와 status만 반환했다. 두 경로를 호환 제공하고 같은 MarketResult를 통해 message/state/summary/checked를 반환한다. 확인 버튼은 추가 공급자 호출 없이 캐시 상태를 읽는다. JS는 정상/일부 지연/실패/꺼짐을 구분하고5초마다 갱신한다. 성공 메시지를 Market.Error로 빨갛게 그리는 템플릿 부분은 런타임에서 제거한다. 제공처 API 호출/계산식은 바꾸지 않는다.

검증: schedule_download_test.go(합성 OneDrive URL/모의 HTTP400/HTML/원본 short URL/storage 이동/외부 이동 거부/실패시 데이터 및 성공 시각 보존), market_status_test.go(경로·응답 호환/외부 추가 호출 없음/성공·지연·꺼짐/인증/성공 오류색 방지), browser-market-status-v1.4.8.json(실제 JS 정상/부분실패/꺼짐/다시 확인).

실제 회사 OneDrive 파일에는 접속하지 못했다. 2026-11-13 개발 환경의 HTTP proxy가 onedrive.live.com CONNECT를403으로 거절했다. 이는 사용자 PC의 HTTP400과 별개다. 회사 URL의 다운로드 성공이나 실제 계정/Windows를 검증했다는 의미가 아니다. GitHub 게시 포인터/푸시/실제 데이터에는 도구로 쓰지 않았다. 원본57개 웹 자산은 보존한다.
