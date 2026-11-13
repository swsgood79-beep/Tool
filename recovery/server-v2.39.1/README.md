# 알파작업완료확인 v1.4 · 복구 운영판

수주·검사 신청/승인·동시 검사·담당자/업체 특이사항·캘린더·엑셀 일괄등록/병합·알림·외부 접속·시세·납기·백업 코드를 연결한 운영 전환용 배포본입니다. **원본과 100% 같은 소스를 복구한 것은 아니며 실제 Windows/회사 계정/휴대폰 확인이 남아 있습니다.**

먼저 **[START-HERE.md](START-HERE.md)**를 읽으세요. 실행 파일은 `AlphaWorkComplete-Recovered.exe`입니다. 기존 프로그램 폴더를 덮어쓰거나 원본 데이터를 삭제하지 않습니다.

| 실행 파일 | 용도 |
|---|---|
| start-production.cmd | 기본 운영 시작. 최초 원본을 새 운영 폴더로 복사, 다음부터 운영 자료 재개 |
| promote-test-to-production.cmd | 최초에만 테스트 업무 자료로 운영 전환. 원본 접속/인증 설정 복구 |
| start-empty-for-restore.cmd | 새 PC의 빈 운영 자료/암호화 백업 복원용 |
| start-test.cmd / continue-test.cmd | 외부 연동을 막은 격리 테스트 |
| copy-existing-data.cmd | 최신 업무 자료의 새 테스트 복사본 생성 |

운영 자료는 `%AppData%\AlphaWorkCompleteRecovered`, 로컬 주소는 `http://127.0.0.1:18791/`입니다. 테스트 자료는 기존 `%AppData%\AlphaWorkRecoveryTest-v1.1`, 포트는 28791을 유지합니다. Windows 운영 뮤텍스는 원본과 같아 동시 운영을 막고 테스트는 별도 뮤텍스·클래스·쿠키를 사용합니다.

운영 모드는 설정을 지우거나 외부 전송을 강제로 차단하지 않습니다. 기존 설정에 따라 시세/일정/암호화 백업을 갱신합니다. 준비된 Pages/Cloudflare 설정은 실행 시 연결을 시작하며 수동 연결 버튼도 유지합니다. **개발 검증 중 운영 GitHub 포인터를 게시하거나 실제 계정/휴대폰에 메시지를 보내지 않았습니다.**

파일 직접 저장/최초 .bak/다중 파일 저널 복구는 운영 복사본 안에서 작동합니다. 원본 WanryoTalk는 읽기 전용으로 복사하며 정상 운영 중 다시 덮어쓰지 않습니다. 테스트에서 운영 전환은 명시적으로 선택하며 이미 시작한 운영 자료를 대체하지 않습니다.

제공받은 완료 엑셀의 읽기도 확인했으며 실제 회사 파일·개인정보는 소스/ZIP에 포함하지 않습니다. 엑셀 저장 설정은 출력용 사본 연결이고 업무 상태를 역으로 가져오는 기능이 아닙니다. 실제 프로그램 상태는 원본/테스트 데이터 복사에서 가져옵니다.

이전 기능 안내와 검증은 audit/RECOVERY-V1.3-TEST.md 및 audit/RECOVERY-V1.3.1-TEST.md, 이번 운영 모드 검증은 audit/RECOVERY-V1.4.md와 build-manifest-v1.4.json을 참고하세요. 과거 보고서는 당시 상태로 보존했습니다. 원본 웹 자산 57개는 기존 SHA-256 그대로 유지합니다.

남은 제한: Windows GUI/DPAPI·회사 실제 자료/계정·삼성 인터넷의 실제 수신/지문은 개발 환경에서 미검증입니다. ngrok 실행·원본과 같은 사진 자동 청소·256MB 초과 백업·원본과 픽셀/소리/애니메이션까지 같은 동작은 구현/검증하지 않았습니다. 엑셀 원본에 대한 자동 양방향 동기화 대신 업로드 사본에 병합합니다.

Go 1.27.3 이상에서 build-windows.ps1 또는 다음 명령으로 빌드합니다.

```powershell
$env:GOOS='windows'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
go build -trimpath -buildvcs=false -ldflags='-s -w -H=windowsgui' -o AlphaWorkComplete-Recovered.exe .
```

Linux는 암호화 없는 검증용입니다. --headless --no-browser로 시험할 수 있지만 실제 회사 자료 운영은 Windows 현재 계정의 DPAPI를 전제로 합니다.
