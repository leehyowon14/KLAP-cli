# Timetable Years Fixture

## Endpoint

```http
POST /std/cps/atnlc/AtnlcYearList.do
```

## Request

```json
{
  "searchYear": "2026",
  "searchHakgi": "1",
  "searchPgmNo": ""
}
```

## Success Fixtures

- `payloads/timetable-years.success.json`
- `payloads/timetable-years.empty.json`

필수 구조:

```json
[
  {
    "year": "2026"
  },
  {
    "year": "2025"
  }
]
```

## Failure Fixtures

- `payloads/timetable-years.session-expired.html`
- `payloads/timetable-years.non-array.json`

## Parser Assertions

- 루트는 배열이어야 한다.
- `year`가 없는 항목은 제외한다.
- 빈 배열이면 현재 선택 학기를 기본값으로 사용한다.

## Redaction Rules

- 연도 값은 유지한다.
