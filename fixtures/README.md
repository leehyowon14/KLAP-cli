# KLAS API Fixtures

이 폴더의 Markdown은 KLAS API fixture 수집 기준과 reverse-engineering 연구 메모다. 현재 실행되는 payload fixture 파일 모음이 아니며, 아래 예시의 `payloads/` 디렉터리는 아직 존재하지 않는다.

구현된 adapter와 실제 `_test.go`의 sanitized fixture는 [API_CONTRACTS](../docs/API_CONTRACTS.md)에서 찾는다. 실제 응답 원본을 추가할 때는 개인정보와 세션 정보를 먼저 제거한다.

## 목적

- API 응답 구조가 바뀌었는지 빠르게 확인한다.
- 파서 테스트를 실제 KLAS 응답 shape에 맞춘다.
- CLI 구현에서 Swift 앱의 추측성 파싱을 반복하지 않는다.
- 성공 응답뿐 아니라 실패, 세션 만료, 빈 목록, 누락 필드 케이스를 고정한다.

## 파일 구성

| 문서 | API |
| --- | --- |
| [common.md](common.md) | 공통 저장 규칙, 실패 응답, 민감정보 제거 |
| [auth-login-security.md](auth-login-security.md) | `POST /usr/cmn/login/LoginSecurity.do` |
| [auth-login-confirm.md](auth-login-confirm.md) | `POST /usr/cmn/login/LoginConfirm.do` |
| [courses.md](courses.md) | `POST /std/cmn/frame/YearhakgiAtnlcSbjectList.do` |
| [course-context.md](course-context.md) | `POST /std/lis/evltn/LctrumHomeStdInfo.do` |
| [lectures-list.md](lectures-list.md) | `POST /std/lis/evltn/SelectOnlineCntntsStdList.do` |
| [lecture-viewer.md](lecture-viewer.md) | `POST /spv/lis/lctre/viewer/LctreCntntsViewSpvPage.do` |
| [lecture-progress.md](lecture-progress.md) | `ChkLctreCntntsView.do`, `UpdateProgress.do` |
| [lecture-learning-status.md](lecture-learning-status.md) | `SelectLrnSttusStd.do`, `CertiStdCheck.do`, `SaveLrnStatus.do` |
| [lecture-download.md](lecture-download.md) | KWCommons 콘텐츠 ID, 동영상 URL 조회 |
| [kwcommons-slides.md](kwcommons-slides.md) | KWCommons `slide_list_*.xml`, slide image URL |
| [assignments-list.md](assignments-list.md) | `POST /std/lis/evltn/TaskStdList.do` |
| [assignment-detail.md](assignment-detail.md) | `POST /std/lis/evltn/TaskStdView.do` |
| [projects-list.md](projects-list.md) | `POST /std/lis/evltn/PrjctStdList.do` |
| [quizzes-list.md](quizzes-list.md) | `POST /std/lis/evltn/AnytmQuizStdList.do` |
| [notices-list.md](notices-list.md) | `POST /std/lis/sport/.../BoardStdList.do` |
| [notice-detail.md](notice-detail.md) | `POST /std/lis/sport/.../BoardStdView.do` |
| [qna-board-list.md](qna-board-list.md) | `POST /std/lis/sport/{questionBoardUid}/BoardStdList.do` |
| [lecture-schedule-widget.md](lecture-schedule-widget.md) | `POST /std/cmn/frame/LctrumSchdulInfo.do` |
| [timetable-years.md](timetable-years.md) | `POST /std/cps/atnlc/AtnlcYearList.do` |
| [timetable-list.md](timetable-list.md) | `POST /std/cps/atnlc/TimetableStdList.do` |
| [session-update.md](session-update.md) | `POST /usr/cmn/login/UpdateSession.do` |
| [syllabus-filters.md](syllabus-filters.md) | `CmmnGamokList.do`, `CmmnHakgwaList.do`, `CmmnMagerCodeList.do` |
| [syllabus-popup.md](syllabus-popup.md) | `CultureOptOneInfo.do`, `LectrePlanStdView.do`, `LectrePlanStdCrtNum.do` |
| [syllabus-graduate.md](syllabus-graduate.md) | `LectrePlanGdhlStdPage.do`, `LectrePlanDaList.do` |
| [page-hooks.md](page-hooks.md) | extension이 route로 잡는 KLAS HTML page 경로 |
| [everytime-lectures.md](everytime-lectures.md) | Everytime 강의평 검색/상세 API, packaged JSON |

## 추천 디렉터리 구조

실제 fixture payload를 추가할 때는 문서와 payload를 분리한다.

```text
fixtures/
  README.md
  common.md
  assignments-list.md
  payloads/
    assignments-list.success.json
    assignments-list.empty.json
    assignments-list.session-expired.html
    assignment-detail.success-submitted.json
    timetable-list.success.json
```

## Payload 저장 규칙

- 실제 학번, 사용자 ID, 쿠키, IP, 토큰은 저장하지 않는다.
- 수업명, 과제명, 공지 제목은 파서 테스트에 필요한 범위에서는 유지할 수 있다.
- 사용자 실명은 `<user-name>`, 학번은 `<user-id>`, 과목 ID는 `<subject-id>`로 치환한다.
- `SESSION`, `WMONID`, `JSESSIONID` 등 쿠키는 저장하지 않는다.
- HTML 응답은 본문 구조 검증에 필요한 최소 영역만 저장한다.
- 실패 응답은 HTTP status, content-type, body shape를 함께 기록한다.

## Fixture 작성 템플릿

각 API 문서는 아래 항목을 포함한다.

```text
## Endpoint
## Request
## Success Fixtures
## Failure Fixtures
## Parser Assertions
## Redaction Rules
## Open Questions
```

## CLI 테스트 기준

CLI 파서 테스트는 최소 다음을 검증한다.

- 성공 fixture에서 필요한 모델 필드를 생성한다.
- 빈 목록 fixture는 실패가 아니라 empty state로 처리한다.
- 세션 만료 HTML은 JSON 파싱 오류가 아니라 auth error로 분류한다.
- 필수 식별자 누락 항목은 제외하거나 비활성화한다.
- 날짜/시간 파싱 실패는 전체 동기화 실패로 번지지 않는다.

## 참고

- 상세 API와 목록 API는 분리해서 테스트한다.
- KLAS 원문 페이지 HTML은 네이티브/CLI 데이터 소스로 쓰지 않는다.
- [API.md](../API.md)는 과거 연구 기록이다. 현재 구현·테스트로 고정한 계약은 [API_CONTRACTS](../docs/API_CONTRACTS.md)를 기준으로 한다.
