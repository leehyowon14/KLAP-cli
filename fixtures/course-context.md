# Course Context Fixture

## Endpoint

```http
POST /std/lis/evltn/LctrumHomeStdInfo.do
```

## Request

```json
{
  "selectYearhakgi": "2026,1",
  "selectSubj": "<subject-id>",
  "selectChangeYn": "Y"
}
```

## Success Fixtures

- `payloads/course-context.success.json`

응답 body는 앱/CLI에서 직접 사용하지 않는다. HTTP 성공이면 서버 세션의 현재 과목 컨텍스트가 바뀐 것으로 본다.

## Failure Fixtures

- `payloads/course-context.session-expired.html`
- `payloads/course-context.invalid-subject.json`

## Parser Assertions

- body shape에 의존하지 않는다.
- 이 API 성공 후에만 과목 의존 API를 호출한다.
- 실패하면 이후 강의/과제/공지 요청을 진행하지 않는다.

## Redaction Rules

- `noticeList`, `taskTop` 등 부가 목록이 포함되면 제목/작성자/ID를 치환한다.
