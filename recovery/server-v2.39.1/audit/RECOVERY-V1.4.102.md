# PC v1.4.102 · 휴대폰 수주 검색 완료 처리

휴대폰 수주 검색에서 키보드의 **검색/Enter를 누르면 키보드를 닫고 검색 결과 위치로 이동**하도록 수정했습니다. 관리자 브라우저·관리자 앱·작업자 화면에 함께 적용합니다. 입력 중 실시간 검색과 한글 조합, 검색어·단계 필터를 유지하고, 결과가 없으면 결과 없음 안내로 이동합니다. 검색어 지우기는 키보드를 유지하며 PC에서는 입력 포커스를 그대로 둡니다. **회사 PC를 v1.4.102로 업데이트한 뒤 휴대폰 화면을 다시 열거나 새로고침하세요. APK v1.0.11은 그대로 사용합니다.**

기존 검색은 input 필터링만 있고 Enter/search 포커스 종료 처리가 없었습니다. admin.html/work.html에만 order-search-v1.4.102.js를 로드합니다. 두 수주 검색 필드만 대상으로 Enter 및 Android search 이벤트에서 필터를 갱신한 뒤 터치/모바일에서는 blur합니다. 키보드가 닫히는 visualViewport resize가 멈춘 후 보이는 결과/결과 없음으로 이동하며 이후 다른 입력에 포커스하면 예약 이동을 취소합니다. 한글 composition/229 이벤트, 빈 검색창의 네이티브 지우기는 완료 처리에서 제외합니다. PC 포커스/다른 폼 Enter/필터/페이지 상태와 원본 web57개는 유지합니다.

Chromium 터치 크기 관리자/작업자/네이티브 앱 UA의 밝게/어둡게 6뷰와 PC 1뷰에서 Enter/native search의 blur·결과 위치/조합 보호/검색 결과 없음/빈값 지우기/검색어 유지/PC 포커스를 확인합니다. 실물 삼성/Android 키보드의 OS 동작은 실행하지 않았으며 브라우저 포커스·뷰포트 동작을 검증합니다. 전체 Go, Linux/Windows vet, Windows GUI 교차 빌드, 소스 ZIP 재빌드 일치를 확인합니다. 실제 회사 자료·폰 알림·설치 상태를 변경하지 않습니다. PC 1.4.100 이후 Web 1.4.101 설치 화면 개선을 포함한 전체 소스를 보존하고 APK 1.0.11은 유지합니다.

시각 확인 재실행은 audit/order-search-preview-v1.4.102.go.fixture를 임시 order_search_preview_test.go로 복사해 ORDER_SEARCH_PREVIEW=1 go test -run ^TestOrderSearchPreview$ -count=1 -v . 실행 후 제거합니다. 임시 서버/계정만 사용합니다.
