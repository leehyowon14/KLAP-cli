# Lecture Schedule Widget Fixture

## Endpoint

```http
POST /std/cmn/frame/LctrumSchdulInfo.do
```

## Request

```json
{
  "selectGrcode": "<grcode>",
  "selectYearhakgi": "2026,1",
  "selectSubj": "<subject-id>"
}
```

## Success Fixtures

- `payloads/lecture-schedule-widget.success.txt`
- `payloads/lecture-schedule-widget.online-room-unspecified.txt`
- `payloads/lecture-schedule-widget.empty.txt`

응답 예:

```text
화 10,11교시/미지정
```

## Failure Fixtures

- `payloads/lecture-schedule-widget.session-expired.html`
- `payloads/lecture-schedule-widget.business-error.json`

## Parser Assertions

- JSON으로 파싱하지 않는다.
- 문자열이 비어 있으면 일정 미등록이다.
- 형식 파싱에 실패해도 원문 문자열을 유지한다.
- 시간표 grid 계산에는 이 API가 아니라 `TimetableStdList.do`를 사용한다.

## Redaction Rules

- 강의실명은 필요하면 `<room>`으로 치환한다.
- 요일/교시 형식은 유지한다.
