# Graduate Syllabus Fixtures

## Endpoints

```http
GET /std/cps/atnlc/LectrePlanGdhlStdPage.do
POST /std/cps/atnlc/LectrePlanDaList.do
```

## Request

대학원 강의계획서 검색 페이지의 `appModule.$data`를 보낸다.

```json
{
  "selectYear": "2026",
  "selecthakgi": "1",
  "selectGdhlitem": "<graduate-school-code>",
  "selectText": "과목명",
  "selectProfsr": "교수명"
}
```

## Success Fixtures

- `payloads/syllabus-graduate-list.success.json`
- `payloads/syllabus-graduate-list.empty.json`
- `payloads/syllabus-graduate-page.html`

필수 후보 필드:

```json
[
  {
    "thisYear": "2026",
    "hakgi": "1",
    "gwamokKname": "대학원 수업명",
    "memberName": "교수명",
    "openGwamokNo": "<course-code>",
    "bunbanNo": "01"
  }
]
```

## Failure Fixtures

- `payloads/syllabus-graduate-list.session-expired.html`
- `payloads/syllabus-graduate-list.business-error.json`
- `payloads/syllabus-graduate-list.non-array.json`

## Parser Assertions

- 루트는 배열이어야 한다.
- `selectGdhlitem`이 없으면 호출하지 않는다.
- 페이지 HTML은 검색 화면 렌더링/원문 이동 확인용으로만 사용한다.

## Redaction Rules

- 대학원 코드, 과목 코드, 교수명은 치환한다.
