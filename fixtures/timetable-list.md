# Timetable List Fixture

## Endpoint

```http
POST /std/cps/atnlc/TimetableStdList.do
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

- `payloads/timetable-list.success.json`
- `payloads/timetable-list.empty.json`
- `payloads/timetable-list.online-periods.json`
- `payloads/timetable-list.unspecified-room.json`
- `payloads/timetable-list.multi-span.json`

필수 구조:

```json
[
  {
    "wtTime": "1",
    "wtHasSchedule": "Y",
    "wtSpan_2": "1",
    "wtSubj_2": "<subject-id>",
    "wtSubjNm_2": "수업명",
    "wtLocHname_2": "강의실",
    "wtProfNm_2": "교수명",
    "wtSubjPrintSeq_2": 1,
    "wtMoreBackSpan_2": "N",
    "wtMoreLine_2": 0
  }
]
```

요일 suffix:

| 요일 | suffix |
| --- | --- |
| 월 | `_1` |
| 화 | `_2` |
| 수 | `_3` |
| 목 | `_4` |
| 금 | `_5` |
| 토 | `_6` |

## Failure Fixtures

- `payloads/timetable-list.session-expired.html`
- `payloads/timetable-list.non-array.json`
- `payloads/timetable-list.missing-wt-time.json`
- `payloads/timetable-list.invalid-span.json`

## Parser Assertions

- 루트는 배열이어야 한다.
- 각 행의 `wtTime`은 숫자로 변환 가능해야 한다.
- `wtHasSchedule !== "Y"` 행은 빈 교시다.
- `wtSubjNm_<n>`과 `wtSubj_<n>`가 모두 없으면 빈 셀이다.
- `wtSpan_<n>`이 없거나 숫자가 아니면 `1`로 처리한다.
- `wtTime <= 8`이고 강의실이 있으면 정규 시간표 grid에 표시한다.
- `wtTime > 8`이면 온라인 수업으로 분리한다.
- 강의실이 비어 있거나 미지정이면 온라인 수업으로 분리한다.

## Redaction Rules

- 과목 ID, 교수명, 강의실은 치환 가능하다.
- `wtTime`, `wtSpan_*`, suffix 구조는 유지한다.

## Open Questions

- `wtMoreBackSpan_<n>`의 정확한 UI 의미
- `wtMoreLine_<n>`의 정확한 UI 의미
- `wtSubjPrintSeq_<n>`가 동일 교시 복수 수업에 어떤 순서로 쓰이는지
