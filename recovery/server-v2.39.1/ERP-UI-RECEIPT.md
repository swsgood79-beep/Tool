# ERP 입고 화면 인식 확인

반자동 연동은 Alpha 선택→ERP 검색/선택 준비→사용자 ERP 저장→Alpha 읽기전용 잔량 갱신 흐름입니다. 현재 단계는 실제 ERP 표와 컨트롤을 읽을 수 있는지 진단하는 기능입니다.

회사 PC에서 ERP 입고등록→Data 불러오기 발주 검색 창(PG03F2120)을 열어 목록이 보이게 둡니다. Alpha 대표관리자→가공 작업 관리→DB 진단·발주 미리보기→ERP 화면 인식 확인→진단 결과 받기. 파일이름 ERP-ui-receipt-check.json. 읽기만 하므로 이 진단을 위해 ERP 저장을 누를 필요는 없습니다.

보고서는 해당 PrimeXAR 프로세스의 정확한 단일 PG03F2120 팝업 범위에서 visible 이름/값, 컨트롤ID/종류, grid행열, 지원pattern과 현재toggle/selection 상태를 수집합니다. 비밀번호컨트롤/하위트리는 제외합니다. 다른 창 탐색/클릭/포커스변경/타이핑/체크변경/저장은 수행하지 않습니다. 기존DB INI도 읽지 않습니다. 동일 Windows 사용자/대화형 데스크톱에서 ERP와 같은 권한 수준의 접근이 필요합니다. 창미검출/권한제한/시간초과는 해당 상태로 표시합니다.

회사 운영폴더에 DPAPI 보호된 ERP-ui-receipt-check-*.dat를 저장하고 복호 검증 후 JSON 다운로드를 제공합니다. JSON은 평문이며 품목/발주 등 업무 내용이 포함되므로 비공개로 취급합니다. 자동 외부 업로드 없음. 실제 결과를 공개 저장소에 올리지 않습니다. 요소/깊이/출력제한, 일부읽기실패를 표시하며 지원pattern 존재만으로 정확한 자동매칭 가능을 단정하지 않습니다. 가려지거나 가상화된 행은 보이지 않을 수 있습니다.

실제 회사 Windows UIA 실행은 현장검증 대상입니다. 합성 fixture의 브라우저/API/보호저장, PowerShell7 구문 검사 및 Windows 교차빌드를 확인합니다.

## PC125 창 탐색 보완

최상위 UIA 목록에서 찾지 못하더라도 같은 ERP 프로세스의 네이티브 하위창/MDI를 함께 조사합니다. 정확한 PG03F2120 제목 후보를 HWND로 중복 제거하고, 없으면 구조 컨테이너를 제한된 깊이/개수로 탐색합니다. 여러 후보는 자동 선택하지 않습니다. 미검출 결과에도 발견한 모듈 코드/창 종류/부모 관계와 개수/탐색제한을 남기므로 결과파일로 원인을 분석할 수 있습니다. 다른 ERP 창의 임의 입력값/전체 제목은 탐색정보에 포함하지 않습니다.

## PC126 구형 접근성 읽기

식별한 입고 팝업에 대해 UIA와 MSAA를 함께 읽습니다. UIA 제공자 등록 결과와 관리형 Legacy API 지원 여부를 기록합니다. MSAA는 같은 PID의 확인된 HWND에서 OBJID_CLIENT를 얻어 accName/accValue/accRole/accState와 children을 읽습니다. protected 상태인 요소/하위트리는 텍스트 읽기 전에 제외하며800노드·깊이16·텍스트1500문자 등으로 제한합니다. 결과 ui_receipt.msaa에는 노드의 부모관계/child ID/role/state/값과 제한·오류가 담깁니다. 구형 방식에서 역할·상태를 얻었다는 사실만으로 자동 체크 가능한 행 식별이 입증되는 것은 아닙니다.

상태 표시의 ‘구형 접근성 N개’는 실제 ERP를 읽은 JSON 결과에서 확인해야 합니다. 공개 스크린샷과 브라우저 검증에는 합성 자료만 사용합니다. 사용자 파일에는 실제 업무 내용이 있으므로 공개 저장소에 올리지 않습니다.

구현 참고는 Microsoft의 client-side provider 등록 문서와 MSAA/IAccessible 접근성 인터페이스입니다.
- https://learn.microsoft.com/en-us/dotnet/framework/ui-automation/register-a-client-side-provider-assembly
- https://learn.microsoft.com/en-us/windows/win32/winauto/uiauto-accessingmsaaservers

### PC127 bounded cell samples

Read-only diagnostic prioritizes descendants of the first8 accessible row objects instead of exhausting800 nodes on row summaries. sampled_rows/skipped_rows/row_limit explicitly distinguish partial samples from full lists. ordinal is accessibility child order, NOT ERP purchase sequence. Optional default_action is a description only, never executed. Password guard and node/depth/queue caps remain. Real checkbox recognition/action capability and exact duplicate business-row identity still require company report; no automation action enabled.

### PC128 usable preparation helper

Authorized manager with machining menu uses basic order-card outstanding popup selection/quantities→prepare→Windows topmost helper. Full business row identity, grouped supplier/unit/PO.20-minute memory-only helper task; source scripts contain no business values. User manually sets ERP vendor/opens Data-load popup and confirms vendor. Optional labelled native EDIT input and unique shallow toolbar Search only, otherwise copy. No automatic vendor/row/quantity/final-save or direct DB write. Native company helper/filters pending field use. Refresh outstanding after manual ERP save. No repeated diagnostic upload required.
