# 내장 ERP 연결 구성요소

`erp-bridge.payload`는 PC 프로그램이 직접 관리하는 Windows 수집 실행 파일입니다. 전체 소스는 `erp-bridge-source/`에 있습니다. 독립 모듈이므로 부모 Go 테스트와 분리됩니다.

재빌드: `cd erp-bridge-source` 후 `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o ../erp-bridge.payload .`

본 프로그램 빌드는 이 payload를 포함합니다. 파일 해시는 `erp-bridge-manifest.json`에 있습니다. 원본 개발 모듈은 저장소의 `recovery/erp-bridge-check`이며 동기화 빌드는 `audit/build-managed-erp.py`를 사용합니다.
