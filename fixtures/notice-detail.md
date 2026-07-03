# Notice Detail Fixture

## Endpoint

```http
POST /std/lis/sport/d052b8f845784c639f036b102fdc3023/BoardStdView.do
```

## Request

```json
{
  "selectYearhakgi": "2026,1",
  "selectSubj": "<subject-id>",
  "boardNo": "1161280",
  "masterNo": "1000000",
  "cmd": "select"
}
```

## Success Fixtures

- `payloads/notice-detail.success.json`
- `payloads/notice-detail.empty-content.json`
- `payloads/notice-detail.with-previous-next.json`
- `payloads/notice-detail.with-attachment.json`

필수 구조:

```json
{
  "board": {
    "boardNo": 1161280,
    "masterNo": 1000000,
    "title": "공지 제목",
    "content": "<p>공지 본문 HTML</p>",
    "topAt": "N",
    "atchFileId": null,
    "readCnt": 0,
    "userNm": "작성자",
    "registDt": "2026-05-29T02:20:00.000+09:00"
  },
  "boardPre": null,
  "boardNex": null
}
```

## Failure Fixtures

- `payloads/notice-detail.session-expired.html`
- `payloads/notice-detail.missing-master-no.json`
- `payloads/notice-detail.null-board.json`
- `payloads/notice-detail.template-page.html`

## Parser Assertions

- `board`가 없거나 null이면 상세 로드 실패다.
- `board.content`는 HTML을 plain text 또는 markdown-safe text로 변환한다.
- `masterNo` 없이 호출한 null 응답은 실패 케이스로 고정한다.
- `BoardViewStdPage.do` HTML은 본문 데이터로 사용하지 않는다.

## Redaction Rules

- 작성자, registerId, IP, 게시글 번호는 치환한다.
- HTML tag 구조와 줄바꿈은 유지한다.
