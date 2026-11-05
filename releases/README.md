# 완료톡 v2.4.0 — GitHub 무료 고정 QR

**[v2.4.0 프로그램 ZIP 다운로드](https://github.com/swsgood79-beep/Tool/raw/refs/heads/main/releases/WanryoTalk-MyPC-Windows-x64-v2.4.0.zip)**

Windows 10/11 x64용입니다. 기존 프로그램의 **종료** 버튼을 누르고 새 ZIP의 `WanryoTalk.exe`를 실행하세요. 기존 수주·검사 기록·Excel 자동 저장은 유지됩니다.

## 고정 QR은 어떻게 동작하나요?

QR은 `https://swsgood79-beep.github.io/Tool/`의 고정 페이지를 엽니다. 프로그램이 실행 중인 회사 PC의 최신 연결 주소만 GitHub에 갱신하고, 페이지가 해당 PC의 작업자 화면으로 안내합니다. 재시작으로 Cloudflare 임시 주소가 바뀌어도 인쇄한 QR은 유지됩니다.

ngrok 계정이나 유료 도메인은 필요하지 않습니다. 대신 아래의 **GitHub 최초 설정 2가지**가 필요합니다. PC·프로그램·인터넷은 켜져 있어야 하며 GitHub/Cloudflare 서비스가 정상이어야 합니다.

## 처음 한 번: GitHub Pages 활성화

1. [Tool 저장소의 Pages 설정](https://github.com/swsgood79-beep/Tool/settings/pages)을 엽니다.
2. Build and deployment에서 **Source: Deploy from a branch**를 선택합니다.
3. **Branch: gh-pages**, 폴더 **/ (root)**를 선택하고 **Save**합니다.
4. 게시에는 몇 분 걸릴 수 있습니다. [고정 페이지](https://swsgood79-beep.github.io/Tool/)가 열리는지 확인하세요. 주소만 열었을 때 ‘작업장 QR을 다시 스캔해 주세요’라는 안내가 나오면 정상입니다.

정적 페이지 파일은 gh-pages 브랜치에 준비되어 있습니다. 페이지를 위한 파일만 들어 있고 EXE·작업자 기록·Excel은 올리지 않습니다.

## 처음 한 번: PC의 GitHub 연결 키 저장

1. [Fine-grained personal access token 만들기](https://github.com/settings/personal-access-tokens/new)를 엽니다.
2. Resource owner를 **swsgood79-beep**, Repository access를 **Only select repositories → Tool**로 제한합니다.
3. Repository permissions에서 **Contents: Read and write**를 지정합니다. Pages 관리나 Actions 권한은 필요하지 않습니다.
4. 발급한 키를 복사하고, EXE의 **고정 QR 설정**을 엽니다.
5. 연결 방식 **GitHub Pages 고정 QR**을 선택하고 **GitHub 연결 키**에 입력한 뒤 저장합니다. 키를 채팅·GitHub 파일·이메일에 보내지 마세요.
6. EXE에서 **인터넷 연결**을 누릅니다. ‘GitHub 고정 QR 연결됨’이 표시되면 **수주·QR 관리**에서 QR을 저장합니다.

기존 임시/ngrok QR은 **이번에 한 번 새로 출력**하세요. 이후 프로그램 업데이트·PC 재시작에도 이 QR을 사용합니다. 저장소/사용자 이름/Pages 주소 또는 PC 기록 파일을 바꾸거나 삭제하지 마세요. 인증키가 만료되면 새 키로 저장하면 되며 QR은 유지됩니다.

프로그램은 Pages가 게시되었는지 확인하고, PC 연결 주소 등록에 성공한 다음 연결됨을 표시합니다. 실제 연결 전에 QR을 출력한 경우 접수는 아직 불가능할 수 있습니다. 첫 등록이나 주소 반영에는 잠시 시간이 걸릴 수 있습니다.

## 저장되는 정보

- GitHub의 `qr-endpoint.json`: 현재 임시 접속 주소·연결 여부·갱신시각·연결 구분용 임시 ID만 저장합니다.
- 작업자 이름·검사 기록·Excel·프로그램 인증키·QR 접근 키는 해당 JSON에 넣지 않습니다.
- QR 접근 키는 URL의 fragment(`#k=...`)에 들어 있어 GitHub 페이지 요청/연결정보 파일 요청으로 전송하지 않습니다.
- GitHub 인증키는 `%APPDATA%\WanryoTalk\connection-v1.dat`에 Windows 계정으로 암호화하고, 설정 화면에 다시 노출하지 않습니다.
- PC가 꺼져 있으면 페이지가 안내하고 재시도할 수 있습니다. 비정상 종료 때 연결 상태가 남아도 PC 준비 상태를 확인하므로 죽은 주소로 바로 넘기지 않습니다.

## 기존 검사·Excel 기능

한 프로젝트의 출하검사와 세팅검사를 별도 작업자·시각으로 완료 접수합니다. 두 검사 모두 끝나면 작업자 목록에서 사라지고 관리자 기록은 남습니다.

문서 폴더 **완료톡/작업완료기록.xlsx** 하나에 한국 날짜별 시트를 추가합니다. EXE의 **엑셀 폴더**로 위치를 엽니다. 완료일자·완료시각·작업자·수주번호·프로젝트명·검사구분·기록번호를 저장합니다. Excel 파일이 잠겨 있으면 PC 원본에 먼저 기록하고 파일을 닫으면 자동 재시도합니다. 브라우저 Excel 다운로드는 사본입니다.

## 검증

실제 Chromium의 고정 QR → 변경된 PC 주소 이동, 오프라인/재시도, 접근 키 비전송, 검사와 Excel 다운로드 등을 포함한 테스트 **47개**와 경쟁 상태 검사를 통과했습니다. GitHub 갱신 API와 터널은 모의 서버로 검증했습니다.

이 개발 환경은 `github.io`·GitHub API 접근이 네트워크 정책으로 막혀 있어 실제 Pages 게시 상태와 사용자 PC의 실연결은 확인하지 못했습니다. Pages 활성화·개인 키 입력 후 실제 휴대폰 데이터로 시험 접수하세요. Windows 화면·DPAPI·소리·회사망 접속도 실기기 확인이 필요합니다. EXE의 디지털 코드 서명은 포함하지 않았습니다.
