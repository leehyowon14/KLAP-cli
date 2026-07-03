# Notices List Fixture

## Endpoint

```http
POST /std/lis/sport/d052b8f845784c639f036b102fdc3023/BoardStdList.do
```

## Request

```json
{
  "selectYearhakgi": "2026,1",
  "selectSubj": "<subject-id>",
  "currentPage": 0,
  "searchCondition": "ALL",
  "searchKeyword": null
}
```

## Success Fixtures

- `payloads/notices-list.success.json`
- `payloads/notices-list.empty.json`
- `payloads/notices-list.important.json`
- `payloads/notices-list.missing-page.json`

필수 구조:

```json
{
  "list": [
    {
      "boardNo": 1161280,
      "masterNo": 1000000,
      "title": "공지 제목",
      "topAt": "N",
      "atchFileId": null,
      "readCnt": 0,
      "userNm": "작성자",
      "registerId": "<register-id>",
      "registDt": "2026-05-29T02:20:00.000+09:00",
      "fileCnt": 0,
      "cmCnt": 0
    }
  ],
  "page": {
    "currentPage": 0,
    "pageSize": 10,
    "totalElements": 1,
    "totalPages": 1
  }
}
```

## Failure Fixtures

- `payloads/notices-list.session-expired.html`
- `payloads/notices-list.missing-list.json`
- `payloads/notices-list.business-error.json`

## Parser Assertions

- 루트는 객체여야 한다.
- `list`는 배열이어야 한다.
- `page`는 없어도 목록 표시는 성공이다.
- `boardNo` 또는 `masterNo`가 없으면 상세 진입을 비활성화한다.
- `registDt` 파싱 실패 시 서버 순서를 유지한다.

## Redaction Rules

- 작성자, registerId, 과목 식별자, 게시글 번호는 치환 가능하다.
- 날짜 형식은 유지한다.
