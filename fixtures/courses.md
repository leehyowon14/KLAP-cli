# Courses Fixture

## Endpoint

```http
POST /std/cmn/frame/YearhakgiAtnlcSbjectList.do
```

## Request

```json
{}
```

## Success Fixtures

- `payloads/courses.success.json`
- `payloads/courses.multiple-terms.json`
- `payloads/courses.empty.json`

필수 구조:

```json
[
  {
    "label": "2026년도 1학기",
    "value": "2026,1",
    "subjList": [
      {
        "name": "수업명",
        "value": "<subject-id>"
      }
    ]
  }
]
```

## Failure Fixtures

- `payloads/courses.session-expired.html`
- `payloads/courses.missing-subj-list.json`
- `payloads/courses.non-array.json`

## Parser Assertions

- 루트는 배열이어야 한다.
- 최신 학기는 `data[0]`로 판단한다.
- `subjList`가 없는 학기는 schema error다.
- 과목의 `name` 또는 `value`가 없으면 해당 과목만 제외한다.

## Redaction Rules

- `value`는 `<subject-id>`로 치환할 수 있다.
- `name`은 테스트 가독성을 위해 익명 수업명으로 바꾼다.
