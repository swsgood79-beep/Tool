# v2.39.1 실행파일 정적 분석 · 복구 자료

**전체 프로그램의 Go 소스 복원본이 아닙니다. 기존 프로그램과 100% 동일하게 동작하는 재빌드는 아직 할 수 없습니다.**

최신 배포 ZIP의 실행파일을 실행하지 않고 분석했습니다. 원본과 기존 자료는 변경·삭제하지 않았습니다. 이 폴더의 자료는 별도 복구 브랜치에 저장하며 실제 배포 실행파일이나 접속 주소를 교체하지 않습니다.

## 원본과 확인된 결과

- 원본: 저장소의 `releases/AlphaWorkComplete-Windows-x64-v2.39.1.zip` 안 `AlphaWorkComplete.exe`
- 실행파일: 78,287,872바이트
- SHA-256: `cac5fd55c9eb294a228969f1f09f4c01512ea4f1674c80bdb407bae8edf8e7e8`
- 빌드 정보: Go 1.27.3, Windows/amd64, 모듈 `wanryotalk`, 의존 모듈 38개
- 내장 웹 자료: **57개**, 합계 2,362,496바이트. HTML·JS·CSS·PNG를 바이너리의 포인터/길이로 추출했습니다. 전부 원본 바이트 및 Go 1.27 컴파일러의 16바이트 해시와 일치합니다.
- 함수 정보: 전체 **19,911개**, `main.` 함수 **711개**. 주소·크기·소스 시작 줄 및 가능한 파일 경로를 복구했습니다. 최적화로 제거된 함수나 원래 변수명까지 복원한 것은 아닙니다.
- 파일 경로: Go 런타임의 파일 테이블 1,459개. 이 중 프로그램의 `wanryotalk/` 파일 경로는 59개입니다. **경로가 복구된 것이며 해당 Go 파일 내용이 복구된 것은 아닙니다.**
- `.text` 전체 어셈블리 및 프로그램 함수 어셈블리, ASCII·UTF-8·UTF-16LE 문자열을 보관했습니다. 문자열 덤프는 출력 가능한 바이트 구간이며 모든 Go 문자열의 의미상 경계를 보장하지 않습니다.

## 파일 안내

| 경로 | 내용 |
|---|---|
| `evidence/assets/web/` | 원본에서 그대로 추출한 화면 자료 |
| `evidence/asset-manifest.json` | 각 파일의 원본 위치·크기·SHA-256·컴파일러 해시 |
| `evidence/symbols.json` | 복구한 함수 이름·주소·소스 위치 |
| `evidence/application-functions.txt` | 프로그램 함수 711개 목록 |
| `evidence/all-functions.asm.gz` | 전체 기계어 디스어셈블리 |
| `evidence/application-functions.asm.gz` | 프로그램 함수별 디스어셈블리 |
| `evidence/strings-*.txt.gz` | 인코딩별 문자열 덤프 |
| `evidence/go-version-m.txt` | 공식 Go 1.27.4 도구로 조회한 빌드 정보 |
| `evidence/go-nm.txt` | 일반 심볼 테이블이 제거되어 nm가 실패한 결과. 런타임 pclntab으로 별도 복구함 |
| `evidence/modules.json` | 원본 실행파일에 기록된 의존 모듈·버전·해시 |
| `recorded-project-module/go.mod` | 빌드 정보로 재구성한 모듈/의존성 목록. 원래 direct/indirect 구분·사용되지 않은 의존성·추가 지시문은 알 수 없음 |
| `recorded-project-module/go.sum` | 원본 빌드 정보에서 확인한 모듈 해시만 기록. 원래 go.sum 전체가 아님 |
| `evidence/source-file-paths.txt` | Go 런타임 파일 테이블 경로 |
| `evidence/web-route-references.json` | 화면 코드의 fetch/form/link 참조. 서버 라우트 전체 목록이 아님 |
| `evidence/embedded-pe-scan.json` | PE 헤더 서명 기반 중첩 실행파일 위치·섹션 길이 분석 |
| `evidence/verification.json` | 원본 바이트·해시·함수 주소 범위 검증 결과 |
| `main.go`, `go.mod` | **새로 작성한 자료 검증·내보내기 도구. 원래 프로그램의 서버 코드가 아님** |
| `tools/` | 실행파일을 실행하지 않는 추출·어셈블리 분석·검증 Python 도구 |

## 아직 복구되지 않은 부분

검사 신청·승인·취소, 인증/패스키, 암호화 저장과 백업, 엑셀, 일정 동기화, PC 알림, 웹 푸시, 터널 관리 등의 **Go 서버 구현은 고수준 소스로 복원되지 않았습니다.** 어셈블리를 Go로 옮기고 저장 형식·Windows 동작·기존 데이터 호환성을 검증하는 별도 작업이 필요합니다. 이 자료를 원본 소스처럼 빌드해 기존 앱을 대체하면 안 됩니다.

고객 수주 데이터·사진·설정은 실행파일 안에 들어 있는 웹 에셋과 다릅니다. 이번 복구 자료가 회사 PC의 실제 운영 데이터 백업을 대신하지 않습니다.

## 복구 도구 실행

Go 1.27.3 이상에서 이 폴더를 기준으로 실행합니다. 외부 모듈을 설치하지 않는 표준 라이브러리 도구입니다.

```sh
go run .
go run . -binary /path/to/AlphaWorkComplete.exe
go run . -binary /path/to/AlphaWorkComplete.exe -export /path/to/NEW-directory
```

내보내기 대상 폴더가 이미 있으면 오류로 중단합니다. 기존 파일을 덮어쓰지 않습니다.

원본을 직접 다시 분석하려면 Python 3와 GNU binutils가 필요합니다. 새로운 출력 폴더를 지정하세요.

```sh
python3 tools/extract_binary.py /path/to/AlphaWorkComplete.exe /path/to/NEW-evidence
python3 tools/disassemble.py /path/to/AlphaWorkComplete.exe /path/to/NEW-evidence
python3 tools/verify_recovery.py /path/to/AlphaWorkComplete.exe /path/to/NEW-evidence
```

추출기는 이 바이너리의 PE32+/Go 1.27 pclntab/embed 레이아웃을 대상으로 합니다. 다른 Go 버전에 그대로 적용한다고 보장하지 않습니다. Go 1.27 컴파일러 해시는 1KiB 이하 파일은 SHA-256의 첫 바이트를 반전하고, 더 큰 파일은 `0x01` 접두 바이트를 넣어 계산합니다. 일반 파일 SHA-256과 구분해 기록했습니다.

검증 도구의 컴파일·원본 검증·57개 내보내기 바이트 대조·기존 경로 덮어쓰기 차단·다른 실행파일 거부를 확인했습니다. 이는 복구 자료 검증이며 원래 업무 프로그램의 동작 검증은 아닙니다.
