# Projects List Fixture

## Endpoint

```http
POST /std/lis/evltn/PrjctStdList.do
GET /std/lis/evltn/PrjctStdPage.do
```

## Request

```json
{
  "selectSubj": "<subject-id>",
  "selectYearhakgi": "2026,1",
  "selectChangeYn": "Y"
}
```

선행 조건: 과목 컨텍스트가 대상 수업으로 맞춰져 있거나 요청 body에 `selectSubj`, `selectYearhakgi`를 포함한다.

## Success Fixtures

- `payloads/projects-list.success.json`
- `payloads/projects-list.empty.json`
- `payloads/projects-list.submitted.json`
- `payloads/projects-list.reexpiredate.json`
- `payloads/projects-list.missing-dates.json`

필수 후보 필드:

```json
[
  {
    "ordseq": "3",
    "weeklyseq": "7",
    "weeklysubseq": "1",
    "title": "팀 프로젝트명",
    "startdate": "2026-05-01 00:00:00",
    "expiredate": "2026-06-14 23:59:59",
    "reexpiredate": "2026-06-16 23:59:59",
    "submityn": "N"
  }
]
```

## Failure Fixtures

- `payloads/projects-list.session-expired.html`
- `payloads/projects-list.business-error.json`
- `payloads/projects-list.non-array.json`

## Parser Assertions

- 루트는 배열이어야 한다.
- `submityn === "Y"`는 제출 완료로 분류한다.
- `expiredate`가 지났고 `reexpiredate`가 미래면 추가 제출 기한을 마감일로 사용한다.
- `ordseq`가 없으면 상세/제출 페이지 이동이 불가능한 항목으로 표시한다.
- `PrjctStdPage.do`는 원문 페이지 이동용으로만 쓰고 목록 데이터로 파싱하지 않는다.

## Redaction Rules

- 팀명, 제출자명, 과목 ID, 프로젝트 제목은 치환한다.
- 날짜 형식, 제출 여부, null 여부는 유지한다.
