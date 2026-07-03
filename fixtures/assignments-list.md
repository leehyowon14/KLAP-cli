# Assignments List Fixture

## Endpoint

```http
POST /std/lis/evltn/TaskStdList.do
```

## Request

```json
{
  "selectYearhakgi": "2026,1",
  "selectSubj": "<subject-id>",
  "currentPage": 0
}
```

## Success Fixtures

- `payloads/assignments-list.success.json`
- `payloads/assignments-list.empty.json`
- `payloads/assignments-list.submitted.json`
- `payloads/assignments-list.missing-due-date.json`

필수 구조:

```json
[
  {
    "ordseq": "7",
    "weeklyseq": "15",
    "weeklysubseq": "1",
    "title": "과제명",
    "startdate": "2026-05-01 00:00:00",
    "expiredate": "2026-06-17 23:59:59",
    "submityn": "N"
  }
]
```

## Failure Fixtures

- `payloads/assignments-list.session-expired.html`
- `payloads/assignments-list.non-array.json`
- `payloads/assignments-list.business-error.json`

## Parser Assertions

- 루트는 배열이어야 한다.
- `ordseq`가 없으면 상세 조회가 불가능하므로 제외한다.
- `title`이 없으면 제외하거나 `제목 없음`으로 표시한다.
- `expiredate` 파싱 실패는 항목 보존, 마감 미확정 처리다.
- `submityn === "Y"`면 제출 완료다.

## Redaction Rules

- 과제 제목은 의미 없는 샘플 제목으로 바꿀 수 있다.
- 날짜 형식과 null 여부는 유지한다.
