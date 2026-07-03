# Quizzes List Fixture

## Endpoint

```http
POST /std/lis/evltn/AnytmQuizStdList.do
GET /std/lis/evltn/AnytmQuizStdPage.do
```

## Request

```json
{
  "selectSubj": "<subject-id>",
  "selectYearhakgi": "2026,1",
  "selectChangeYn": "Y"
}
```

## Success Fixtures

- `payloads/quizzes-list.success.json`
- `payloads/quizzes-list.empty.json`
- `payloads/quizzes-list.submitted.json`
- `payloads/quizzes-list.expired.json`
- `payloads/quizzes-list.missing-edt.json`

필수 후보 필드:

```json
[
  {
    "quizseq": "5",
    "title": "퀴즈 제목",
    "sdt": "2026-05-01 00:00:00",
    "edt": "2026-05-07 23:59:59",
    "issubmit": "N"
  }
]
```

## Failure Fixtures

- `payloads/quizzes-list.session-expired.html`
- `payloads/quizzes-list.business-error.json`
- `payloads/quizzes-list.non-array.json`

## Parser Assertions

- 루트는 배열이어야 한다.
- `issubmit === "Y"`는 제출 완료로 분류한다.
- `edt`가 없거나 날짜로 파싱할 수 없으면 마감 알림 대상에서 제외한다.
- `AnytmQuizStdPage.do`는 원문 페이지 이동용으로만 쓰고 목록 데이터로 파싱하지 않는다.

## Redaction Rules

- 퀴즈 제목, 과목 ID, 사용자 식별자는 치환한다.
- `issubmit`, 날짜 문자열, 숫자/문자열 타입은 유지한다.
