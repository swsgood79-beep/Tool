# 알파 관리자 Android 1.0.3

## 현재: APK 1.0.6 / 서버 1.4.71

자동 세션 복원 진입점을 제거했습니다. 업무 화면은 직접 passwordLogin 또는 지문 버튼의 인증 성공 이후에만 열립니다. onStop에서 화면을 잠그고 AuthEpoch를 갱신하여 백그라운드 이전 인증 응답이 나중에 자동으로 열지 못하게 합니다. 파일 선택기 복귀는 기존 업무를 유지합니다. 서버 grant·푸시 연결은 로그아웃 전까지 유지하며, 아래 이전 버전 자동 복원 설명은 이 버전에서 대체됩니다.

## 현재: APK 1.0.5 / 서버 1.4.69

인증된 앱은 onStop에서 잠그지 않습니다. onResume은 같은 WebView/입력 상태를 유지합니다. 새 프로세스는 저장된 grant와 암호화된 secret으로 서버 복원 후 account_key를 확인하고 업무 화면을 엽니다. signed_out·누락된 키·진행 중 로그인은 자동 복원을 막고, 일시적인 연결 실패는 키를 지우지 않습니다. 직접 로그아웃과 서버 권한 검증은 유지합니다. 아래 이전 버전의 매번 로그인 선택 화면 설명은 이 버전에서 대체됩니다.

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

APK 1.0.2: BiometricPrompt(28+)/FingerprintManager(26~27)로 앱 잠금을 해제합니다. 인증 성공은 프로세스에만 보관하며 WebView·복원 호출은 해제 후 진행합니다. 저장된 로그인 연결과 FCM 수신은 유지합니다. 웹 패스키는 앱에서 브라우저로 연결하지 않습니다. OS 지문 센서의 실기기 시험은 별도로 필요합니다.

앱 내부 업데이트: update.json을 고정 저장소에서 읽고, 새 version_code일 때만 안내합니다. 다운로드는 고정 raw GitHub APK 경로로 제한하고 SHA-256·패키지명·버전·기존 앱 서명을 확인합니다. FileProvider로 Android 설치 창을 열며 사용자 확인과 최초 설치 출처 허용이 필요합니다. 업무 입력 중에는 설치를 보류합니다. **새 APK를 배포할 때 update.json의 version_code/version_name/apk_url/sha256도 함께 갱신하세요.** 서버 화면만 바뀌는 릴리스에서는 이 manifest를 갱신하지 않습니다. 현재 APK에서 아직 없는 다음 버전은 원격 manifest 갱신 후 안내됩니다.

APK 1.0.3: 네이티브 로그인 폼을 먼저 표시하고 실제 버튼 클릭에서만 지문 인증을 요청합니다. PasswordLogin은 /login CSRF와 해당 경로의 쿠키로 정상 인증하며, 비밀번호는 저장하지 않습니다. LoginPolicy는 입력 아이디와 지문 연결 계정을 구분합니다. 서버 account_username은 표시 보조 필드로 v1.4.66 서버와도 호환합니다. 자동 APK 확인은 로그인 전에서도 앱 foreground에만 수행합니다.


## APK 1.0.4 / 서버 1.4.68

인증된 bootstrap의 account_label을 상단 담당자 이름에 사용합니다. 상단 로그아웃 버튼은 로컬 로그인·알림을 즉시 차단하고 해당 세션/로그인 grant/푸시 기기만 해제합니다. 이전 비동기 응답은 AuthEpoch로 걸러서 로그아웃 후 이름이나 푸시 등록을 복원하지 않습니다. 기존 알림 채널 ID와 서명 키를 유지합니다. 실제 Android UI·지문 센서·설치·FCM 수신은 빌드 환경에서 실행하지 않았습니다.
