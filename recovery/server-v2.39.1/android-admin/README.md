# 알파 관리자 Android 1.0.1

관리자 전용 APK. 기존 관리자 웹 화면을 유지하면서 Android WebView·FCM 알림·시스템 카메라/파일 선택·다운로드를 사용합니다. 작업자 PWA와 별개입니다.

- package: `com.alpharobotics.admin`, minSdk 26 / targetSdk 35
- JDK 17, Android SDK 35, Gradle 8.9, AGP 8.7.3
- `./gradlew :app:assembleDebug :app:testDebugUnitTest :app:lintDebug`
- release: 환경 변수 `ALPHA_ADMIN_KEYSTORE`, `ALPHA_ADMIN_STORE_PASSWORD`, `ALPHA_ADMIN_KEY_PASSWORD`를 비공개 빌드 환경에 지정한 뒤 `./gradlew :app:assembleRelease :app:testReleaseUnitTest :app:lintRelease`. 키 alias는 `alpha-admin`입니다.
- 배포 APK 업데이트에는 최초 배포와 같은 서명키가 필요합니다. 서명키·비밀번호는 공개 소스에 포함하지 않습니다.
- Firebase 등록은 회사 PC 설정 화면에서 수행하며 APK에는 서비스 계정 비밀키를 포함하지 않습니다. `google-services.json` Gradle 플러그인이 필요하지 않은 동적 FirebaseOptions 초기화를 사용합니다.
- 앱 로그인은 일반 관리자 인증 후 별도 grant를 등록합니다. grant 비밀값은 Android Keystore AES-GCM으로 저장하며 앱 종료·같은 서명 업데이트 후에도 복원합니다. 서버 저장값은 비밀값의 해시입니다. 일반 브라우저 인증 기간과 FCM 갱신 키는 분리합니다.
- 로그아웃/앱 연결 해제/개인 계정 비밀번호 변경·중지 시 grant를 더 사용할 수 없습니다. 앱 삭제/데이터 초기화 시 다시 로그인합니다.
- HTTPS의 검증된 회사 tunnel만 WebView 내부로 열며, TLS 오류 무시·JavaScript bridge·평문 HTTP는 사용하지 않습니다. UI 동작에 JavaScript가 필요하므로 켜져 있습니다. 자동 다크닝은 끄고 사이트의 명시적 테마를 사용합니다.
- 설정 방법은 상위 폴더의 `ANDROID-ADMIN-SETUP.md`를 참조하세요. 실기기 FCM 시험은 사용자 Firebase 연결 후 필요합니다.

APK 1.0.1의 User-Agent 표시로 현장 메뉴를 선택합니다. 서버는 인증된 세션에만 전용 자산을 제공하며 User-Agent를 권한 판단에 사용하지 않습니다. 일반 PC/작업자 페이지에는 기존 자산을 유지합니다.
