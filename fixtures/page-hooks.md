# KLAS Page Hook Fixtures

## Endpoints

extension이 DOM enhancement 대상으로 잡는 KLAS HTML page 경로다. 네이티브 데이터 소스가 아니라 원문 이동, route 감지, HTML fallback 검증용으로만 사용한다.

```http
GET /std/cmn/frame/Frame.do
GET /std/cps/atnlc/LectrePlanStdPage.do
GET /std/cps/atnlc/popup/LectrePlanStdView.do
GET /std/cps/atnlc/popup/LectrePlanStdNumPopup.do
GET /std/cps/atnlc/LectrePlanGdhlStdPage.do
GET /std/cps/inqire/AtnlcScreStdPage.do
GET /std/cps/inqire/GradScreStdPage.do
GET /std/cps/inqire/StandStdPage.do
GET /std/cps/inqire/LctreEvlViewStdPage.do
GET /std/cps/inqire/LctreEvlStdPage.do
GET /std/cps/inqire/ToeicStdPage.do
GET /std/lis/evltn/LctrumHomeStdPage.do
GET /std/lis/evltn/OnlineCntntsStdPage.do
GET /spv/lis/lctre/viewer/LctreCntntsViewSpvPage.do
GET /std/cps/atnlc/TimetableStdPage.do
GET /usr/cmn/login/LoginForm.do
GET /std/ads/admst/MyInfoStdPage.do
GET /std/ext/grdtn/GrdtnYnImprtyResnStdPage.do
```

## Success Fixtures

- `payloads/page-frame.success.html`
- `payloads/page-syllabus.success.html`
- `payloads/page-syllabus-popup.success.html`
- `payloads/page-grade.success.html`
- `payloads/page-toeic.success.html`
- `payloads/page-online-lecture.success.html`
- `payloads/page-viewer.success.html`
- `payloads/page-login.success.html`

## Failure Fixtures

- `payloads/page-hooks.session-expired.html`
- `payloads/page-hooks.common-error.html`
- `payloads/page-hooks.http-404.html`

## Parser Assertions

- HTML page fixture는 JSON API 대체 데이터로 쓰지 않는다.
- 로그인 페이지 HTML은 세션 만료로 분류한다.
- viewer page는 KWCommons 콘텐츠 ID 추출 후보 script만 검증한다.
- frame page는 `UpdateSession.do` 함수와 시간표 select 존재 여부만 검증한다.

## Redaction Rules

- 사용자명, 학번, 학과 링크 추정에 쓰인 숫자는 치환한다.
- script/function 이름과 API path 문자열은 유지한다.
