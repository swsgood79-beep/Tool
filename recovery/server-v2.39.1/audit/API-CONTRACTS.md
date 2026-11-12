# 프론트엔드 계약과 구현 범위

원본의 폼·JS 호출 전체 위치는 `frontend-contracts.json`에 있습니다. Axios/WebSocket/EventSource 호출은 이 57개 자산에서 발견되지 않았습니다. 외부 고정 Pages의 코드는 이 내장 자산 조사 범위에 포함되지 않습니다. 아래는 원본 화면에서 직접 확인한 계약이며, 동적 변수 경로·반환값의 전체 의미를 역컴파일로 증명한 명세는 아닙니다.

| 원본 요청 | 확인한 요청/응답 | 이 서버 |
|---|---|---|
| GET `/status/{board}` | `active: number[]`, `stages/completed_due: {id,generation,revision,pending}[]`, `shipments: ShipmentSummary[]`, `date`, `schedule` | 구현 |
| GET `/card/{board}/{id}` | 프로젝트 카드 HTML, data-project/generation/revision | 구현 |
| POST `/complete/{board}` | project, inspection, submission, worker | 단독 신청. 관리자가 승인해야 완료 |
| POST `/combined/{board}` | 위 필드 + combined_submission | 통합 신청. 두 단계 원자적으로 저장 |
| POST `/note/{board}` | project, submission, worker, note | 독립 비고 기록·안 읽은 알림 |
| GET `/orders/panel` | `#order-items` HTML, data-day/schedule-hint | 구현 |
| GET `/inspection-requests/panel` | `#inspection-request-items` HTML | 구현 |
| GET/POST `/projects/{id}/inspection` | inspection, request, csrf, optional return_queue | 승인·전체/부분 통합 승인 |
| GET `/project-info/{board}/{id}` / `/projects/{id}/info` | 특이사항 HTML, data-info-id/revision | 구현, 마스터 변경 시 revision 증가 |
| GET `/project-photo/{board}/{project}/{photo}` / `/projects/{id}/photos/{photo}` | 이미지 | 구현 |
| GET `/designers.json` | `{ID,Name,Phone}[]` | 구현 (Revision도 포함) |
| POST `/designers` | csrf, action, name, phone; 수정은 id/revision | 구현 |
| GET `/client-notes/list.json` | `{ID,Name}[]` | 구현 (원본 정보도 포함) |
| GET `/client-notes/{id}/data` | `{ID,Name,Text,Photos,Revision}` | 구현 |
| GET `/client-notes/{id}/photos/{photo}` | 이미지 | 구현 |
| POST `/records/{id}/read` | csrf; 성공 HTTP 상태 | 구현, JSON `{ok:true}` |
| GET `/markets/{board}` | `marketResult`와 `marketQuote`, `marketFX` | 데이터 미구현, enabled:false·빈 quotes |
| GET `/market-settings/status` | status 문자열 | 미구현 안내 |
| GET `/backup/status` | status/error/busy 등 (`backup.js` 원문 참고) | 미구현 안내 |
| GET `/schedule/status.json` | schedule 상태 (`schedule.js` 원문 참고) | 미구현 안내 |
| GET `/mobile-admin/session` | 로그인 여부 (`mobile-login.js` 참고) | 원본 모바일 세션 프로토콜 미구현 |
| POST `/mobile-admin/login` | password, csrf, next | 별도 복구판 로그인으로 처리. 원본 세션 호환 아님 |
| POST `/mobile-admin/passkey/finish` | handoff ticket (`passkey-enter.js` 참고) | 미구현 안내 |

`ShipmentSummary`의 원본 JSON 키: project_id, number, name, info_url, photo_count, setting_worker, shipping_worker. 원본 stageStatus/시장/이벤트 구조체의 타입·JSON 태그는 `main-struct-layouts.json`에 전부 보관했습니다.

HTML 폼의 프로젝트 CRUD, 디렉터리 CRUD, archive, unread/read-all, records edit/cancel/delete도 구현했습니다. 프로젝트 폼에 `shipping_due_date`를 추가해 외부 납기 동기화 없이 수동 검사할 수 있게 했습니다. 원본 import/backup/settings/mobile/passkey/schedule/market 연결 설정 폼은 구현하지 않았습니다.

# 사용자 함수와의 대응

함수 주소·원래 파일·시작 줄은 `main-functions.json`에서 확인합니다. 이 대응은 원본 호출과 기능 영역의 대응이며, C/Go 역컴파일로 함수 본문을 그대로 복원했다는 뜻이 아닙니다.

| 원본 함수/파일 영역 | 새 구현 |
|---|---|
| `Host.addCombinedRoutes.func1`, `Host.approveCombined`, `inspection_combined.go` | inspection.go requestInspection/approve |
| `Host.approveInspection`, `inspection_review.go` | inspection.go approve, routes.go 검사 화면/POST |
| `Host.commit`, `host.go` | model.go transaction/persist (별도 포맷) |
| `project_edit.go`, `client_notes.go`, `designers.go` | routes.go CRUD와 projectView의 연결 원본 반영 |
| `notes.go`, `unread.go`, `corrections.go` | 비고·읽음·기록 수정/취소 |
| `workbook.go`, `host_csv.go` | workbook.go OOXML·CSV (새 구현) |
| `mobile_admin.go` | auth.go의 독립 로컬 비밀번호 세션. 원본 모바일 프로토콜 미복원 |
| `passkeys.go`, `phone.go`, `tunnel.go`, `market.go`, `schedule_sync.go`, `backup_*` | 미복원 |

따라서 이 서버는 **원래 기능을 100% 수행하는 복원본으로 판정할 수 없습니다.** 개별 업무 기능의 새 구현과 시험 결과를 제공하는 단계입니다.
