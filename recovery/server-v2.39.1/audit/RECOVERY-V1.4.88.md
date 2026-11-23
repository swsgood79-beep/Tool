# PC v1.4.88 · 디스크 스트림 분할 백업

실행 백업은 buildPortableFile의 ZIP/age 파일 스트림을 사용합니다. 사진/첨부 조각당 최대 9MiB씩 읽고 metadata/records를 포함해 archive를 만든 뒤 verifyPortableFile로 디스크에서 전 항목 해시/참조 첨부 길이를 검증합니다. App.mu 밖에서 큰 I/O를 수행하고 snapshot clone과 backupPins로 동시 삭제를 보호합니다. 기존 작은 portable/decodePortable는 단일 파일 백업 호환 및 시험을 위해 256MiB 제한을 유지하며, 운영 백업/새 복원은 v3 경로를 사용합니다.

암호문은 200MiB SectionReader로 POST하고 각 조각을 재다운로드해 정확한 크기/해시를 검사합니다. 암호화 index는 recovered-v3-pending으로 올려 검증 후 PATCH label recovered-v3으로 완료 등록합니다. 정상 세대 목록은 완료 인덱스만 선택합니다. 실패 정리는 인덱스부터 역순으로 하고 첫 삭제 실패 시 조각 삭제를 중단해 불확실한 완료 등록에도 정상 payload를 남깁니다. 로컬 LastSuccess 저장까지 끝난 후 완료 3세트를 보존하며 오래된 세트는 index부터 지워 중간 정리 실패 시 불완전 세트를 노출하지 않습니다. 원본 alpha-verified-v1 및 다른 Release는 유지합니다. GitHub asset 목록 페이지 조회, 요청20분/작업2시간 기한을 적용합니다.

복원은 전체 원격 조각 및 암호문 합본/ZIP/age/내부 체크섬을 먼저 검증하고 빈 대상인지 재검사 후 보호 파일과 상태를 적용합니다. 기존 단일 백업 복원을 지원합니다. 현재 복구키/운영 자료 경로/자동 백업 사용 여부는 바꾸지 않습니다. LastAttempt/LastParts/LastError는 저장하며 snapshotHash에서는 제외합니다. 검증 안전 상한은 전체1TiB/metadata128MiB/항목100만이며 증분·중단 재개 백업은 포함하지 않습니다.

합성 260MiB 첨부 실제 분할·복원, 원본 해시, 전송·검증·완료 등록 실패 및 이전 세트 보존, 손상/누락 적용 방지, 기존 단일 백업, 원래 Release 보호, 동시 작업 조회, file pin, API pagination, 키 다시 받기 동일값, race, Chromium 4개뷰의 성공/실패/보존 시각·용량을 검증합니다. 회사 데이터/비밀키는 검증 자료와 소스에 포함하지 않습니다. 원래 web 57개 자산 및 APK 1.0.11 유지. 실제 회사 GitHub 백업/새 Windows 계정 DPAPI 복원/Windows GUI는 실행하지 않았습니다.
