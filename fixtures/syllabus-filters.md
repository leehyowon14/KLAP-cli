# Syllabus Filter Fixtures

## Endpoints

```http
POST /std/cps/atnlc/CmmnGamokList.do
POST /std/cps/atnlc/CmmnHakgwaList.do
POST /std/cps/atnlc/CmmnMagerCodeList.do
```

## Request

공통 과목 목록:

```json
{}
```

학과 목록:

```json
{
  "selectYear": "2026",
  "selecthakgi": "1"
}
```

전공 목록:

```json
{
  "selectYear": "2026",
  "selecthakgi": "1",
  "selecthakgwa": "<department-code>"
}
```

## Success Fixtures

- `payloads/syllabus-common-subjects.success.json`
- `payloads/syllabus-common-subjects.empty.json`
- `payloads/syllabus-departments.success.json`
- `payloads/syllabus-majors.success.json`
- `payloads/syllabus-majors.empty.json`

필수 후보 필드:

```json
[
  {
    "code": "<code>",
    "codeName1": "공통 과목명"
  }
]
```

```json
[
  {
    "classCode": "<department-code>",
    "openMajorName": "학과명"
  }
]
```

## Failure Fixtures

- `payloads/syllabus-filters.session-expired.html`
- `payloads/syllabus-filters.business-error.json`
- `payloads/syllabus-filters.non-array.json`

## Parser Assertions

- 루트는 배열이어야 한다.
- 공통 과목과 전공은 `code`, `codeName1`을 사용한다.
- 학과는 `classCode`, `openMajorName`을 사용한다.
- 빈 배열은 선택지 없음 상태로 처리한다.

## Redaction Rules

- 코드 값은 fixture 간 join 검증이 필요하면 placeholder를 일관되게 유지한다.
