# Q&A Board List Fixture

## Endpoint

```http
POST /std/lis/sport/{questionBoardUid}/BoardStdList.do
GET /std/lis/sport/573f918c23984ae8a88c398051bb1263/BoardQnaListStdPage.do
```

## Request

```json
{
  "cmd": null,
  "pageInit": true,
  "selectYearhakgi": "2026,1",
  "selectSubj": "<subject-id>",
  "selectChangeYn": "Y",
  "searchCondition": "ALL",
  "searchKeyword": "",
  "currentPage": 0
}
```

`questionBoardUid`는 강의 홈 HTML의 `BoardQnaListStdPage.do` 링크에서 추출한다.

## Success Fixtures

- `payloads/qna-board-list.success.json`
- `payloads/qna-board-list.empty.json`
- `payloads/qna-board-list.multi-page-first.json`
- `payloads/qna-board-list.multi-page-next.json`

필수 구조:

```json
{
  "list": [
    {
      "boardNo": "1",
      "masterNo": "10",
      "title": "질문 제목",
      "registDt": "2026-06-01 12:00:00",
      "userNm": "작성자"
    }
  ],
  "page": {
    "totalPages": "1",
    "totalElements": "1"
  }
}
```

## Failure Fixtures

- `payloads/qna-board-list.session-expired.html`
- `payloads/qna-board-list.business-error.json`
- `payloads/qna-board-list.missing-page.json`

## Parser Assertions

- `list`가 없으면 구조 변경으로 본다.
- `page.totalPages`가 있으면 전체 페이지를 순회할 수 있다.
- `registDt`가 24시간 이내인 항목은 신규 글 후보로 분류할 수 있다.
- UID가 없으면 Q&A 집계를 비활성화한다.

## Redaction Rules

- 작성자, 제목, board/master 번호는 치환한다.
