# v1.0 근거·변경·제한

이번 구현은 원본 소스 복구 또는 원본 동등성 인증이 아니라, v0.2에서 빠진 직접 저장과 Windows UI의 독립 구현입니다.

## 원본 근거

`../v2.39.1/evidence/go-version-m.txt`에는 systray·walk·webview 의존성이 없습니다. `audit/native-ui-evidence-v1.0.json`은 원본 네이티브 함수 22개와 문자열 참조 131건을 모았으며, `native-ui-original-assembly-v1.0.asm.gz`에는 해당 원본 어셈블리 7,572행이 들어 있습니다. 문자열 참조의 prefix는 상수 저장소의 주변 바이트까지 포함하므로 한 문자열의 정확한 길이를 뜻하지 않습니다.

| 원본 근거 | 구현 |
|---|---|
| `main.tray` 0x14079a920, `main.windowProc` 0x14079ab20, main_windows.go | raw Win32 클래스/메시지 루프, Shell_NotifyIconW 트레이, ShowWindow/브라우저 실행, 메뉴·설정·관리 연결 |
| toast_windows.go의 registerToastClass/fillToasts/repositionToasts/toastTick/paintToast/toastProc | OS 스레드에 고정된 자체 팝업, 큐/동시 3개, 마우스 호버 타이머, 닫기와 상세 열기 |
| `main.refreshUnreadBadge` 0x1407d3ea0, `main.unreadBadgeProc` 0x1407d4ae0 | topmost 미확인 숫자 창, 재표시·전체 목록 연결 |
| 원본 wndclass/msg/notifyIcon/paintStruct RTTI | x64 Win32 구조체 크기/오프셋 시험. NOTIFYICONDATAW 976, WNDCLASSEXW 80, MSG 48, PAINTSTRUCT 72 |
| CreateMutexW, `Local\WanryoTalkNotifier-v1`, `WanryoTalkMain` | 원본과 같은 단일 실행 뮤텍스/관리 창 클래스. 기존 창 찾기·앞으로 가져오기 |
| 원본 PlaySoundW | 새 팝업에 Windows SystemNotification 사운드. 원본 음원과 동일하다고 확인한 것은 아님 |
| HostData Version14, WorkOrder, Completion.Event, 각 보조 파일 RTTI | 원본 필드의 직접 직렬화 및 미지의 최상위/수주/기록 필드 유지 |
| CryptProtectData/CryptUnprotectData, photoPath | Windows 현재 계정 DPAPI, 기존 project-photos/{id}.dat JPEG |

## 저장 설계

원본과 같은 필드에 업무 상태를 기록합니다. 복구판의 추가 상태만 같은 projects-v2.dat의 `_RecoveredV1` 필드에 둡니다. 확장의 원본 부분/보조 파일 해시가 달라지면 원본 필드를 다시 읽으며, 실행 중 외부 변경은 중단합니다. 원본 JSON 판독기와 호환되는 형태지만 원본 EXE를 실행해 왕복 비교한 것은 아닙니다.

여섯 파일은 독립적인 원본 데이터 파일이므로 한 번의 OS rename으로 모두 바뀌지 않습니다. 각 새/이전 파일을 보호된 준비 폴더에 쓰고 prepared 저널을 저장한 뒤, 보조 파일부터 원본 호스트 파일까지 교체하고 committed 저널을 기록합니다. 재시작 시 prepared는 전부 되돌리고 committed는 새 상태를 유지합니다. 롤백 전에 현재 파일과 모든 이전 바이트를 확인합니다. 예상 밖 파일 변경은 덮어쓰지 않고 저널과 백업을 보존합니다. 전원 차단 내구성은 별도 시험하지 않았습니다.

최초 기존 .bak은 덮어쓰지 않습니다. 새로운 이미지 파일은 기존 이미지를 수정하지 않고 새 ID로 만들며 저장 실패 시 고립된 새 이미지가 남을 수 있습니다. 기존 파일 삭제 방지를 위해 자동 청소는 넣지 않았습니다. 수주 사진에 연결된 업체 사진은 독립 복사본으로 투영합니다.

## 검증 해석

`go-test-race-v1.0.txt`, `build-manifest-v1.0.json`, `browser-validation-v1.0.json`, `browser-integrations-validation-v1.0.json`이 이번 실행 결과입니다. 기존 v0.2 보고서는 그 당시의 상태로 보존합니다. 원본 웹 자산 57개는 기존 해시로 비교합니다.

Windows 프로그램은 x64 GUI 교차 빌드입니다. 네이티브 창을 실제 Windows에서 실행하거나 회사 DPAPI 자료를 사용하지 않았습니다. UI 픽셀/애니메이션/사운드, 실제 브라우저·푸시·지문·Cloudflare·토스·GitHub 계정 동작을 인증하지 않습니다. ngrok, 엑셀 일괄등록·기존 엑셀 병합·사진 자동 청소·대용량 백업 등 기존 차이가 남아 있으므로 원본 100% 상태라고 주장할 수 없습니다.
