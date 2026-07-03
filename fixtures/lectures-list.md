# Lectures List Fixture

## Endpoint

```http
POST /std/lis/evltn/SelectOnlineCntntsStdList.do
```

## Request

```json
{}
```

선행 조건: `course-context.md`의 과목 컨텍스트 변경 성공.

## Success Fixtures

- `payloads/lectures-list.success.json`
- `payloads/lectures-list.empty.json`
- `payloads/lectures-list.with-project-items.json`
- `payloads/lectures-list.missing-dates.json`
- `payloads/lectures-list.missing-viewer-fields.json`

필수 후보 필드:

```json
[
  {
    "grcode": "<grcode>",
    "subj": "<subject-id>",
    "year": "2026",
    "hakgi": "1",
    "bunban": "01",
    "module": "01",
    "lesson": "001",
    "oid": "C000000000",
    "ptime": "50",
    "totalTime": "0",
    "weekNo": "1",
    "weeklyseq": "1",
    "moduletitle": "1주차",
    "sbjt": "강의 제목",
    "evltnSe": "on",
    "prog": 0,
    "sdateY": "20260306",
    "sdateH": "00",
    "sdateM": "00",
    "edateY": "20260312",
    "edateH": "23",
    "edateM": "59",
    "starting": "https://kwcommons.kw.ac.kr/em/<content-id>",
    "mvpLink": "https://kwcommons.kw.ac.kr/em/<content-id>&contents=..."
  }
]
```

## Failure Fixtures

- `payloads/lectures-list.session-expired.html`
- `payloads/lectures-list.non-array.json`
- `payloads/lectures-list.business-error.json`

## Parser Assertions

- 루트는 배열이어야 한다.
- `evltnSe === "proj"` 항목은 강의가 아니므로 제외한다.
- `prog >= 100`이면 수강 완료로 분류한다.
- 기간 필드를 파싱할 수 없으면 자동 수강 대상에서 제외한다.
- `mvpLink` 또는 `starting`에서 KWCommons 콘텐츠 ID를 추출한다.
- 뷰어 필수 필드가 부족하면 재생/자동 수강 불가로 분류한다.

## Redaction Rules

- `userId`, `subj`, `grcode`, `oid`, 콘텐츠 ID를 치환한다.
- 날짜/진도/숫자 타입은 유지한다.
