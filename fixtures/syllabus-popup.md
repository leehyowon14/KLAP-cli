# Syllabus Popup Fixtures

## Endpoints

```http
POST /std/cps/atnlc/CultureOptOneInfo.do
GET /std/cps/atnlc/popup/LectrePlanStdView.do
GET /std/cps/atnlc/popup/LectrePlanStdFixedView.do
GET /std/cps/atnlc/popup/LectrePlanStdNumPopup.do
POST /std/cps/atnlc/popup/LectrePlanStdCrtNum.do
```

## Request

`CultureOptOneInfo.do`는 강의계획서 검색 페이지의 `appModule.$data`를 보낸다.

```json
{
  "selectYear": "2026",
  "selecthakgi": "1",
  "selectSubj": "<subject-id>"
}
```

수강인원 조회:

```json
{
  "currentNum": "0",
  "gwamokName": "수업명",
  "numText": "",
  "randomNum": "",
  "selectGrcode": "<grcode>",
  "selectSubj": "<subject-id>",
  "selectYear": "2026",
  "selectYearHakgi": "2026,1",
  "selecthakgi": "1",
  "stopFlag": ""
}
```

## Success Fixtures

- `payloads/syllabus-culture-opt.false.json`
- `payloads/syllabus-culture-opt.true.json`
- `payloads/syllabus-popup-view.html`
- `payloads/syllabus-popup-fixed-view.html`
- `payloads/syllabus-popup-current-number.success.json`

응답 후보:

```json
{
  "cultureOpt": false
}
```

```json
{
  "currentNum": "42"
}
```

## Failure Fixtures

- `payloads/syllabus-popup.session-expired.html`
- `payloads/syllabus-popup-current-number.business-error.json`
- `payloads/syllabus-popup-current-number.missing-current-num.json`

## Parser Assertions

- `cultureOpt`가 truthy면 fixed view URL을 사용한다.
- HTML popup은 네이티브 데이터 소스가 아니라 원문 열기 fallback으로만 사용한다.
- `LectrePlanStdCrtNum.do`는 `currentNum`이 숫자 문자열이면 성공으로 본다.

## Redaction Rules

- 교수명, 전화번호, 이메일, 수강인원 상세 값은 필요 최소 범위로 치환한다.
