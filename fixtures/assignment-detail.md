# Assignment Detail Fixture

## Endpoint

```http
POST /std/lis/evltn/TaskStdView.do
```

## Request

```json
{
  "selectYearhakgi": "2026,1",
  "selectSubj": "<subject-id>",
  "ordseq": "7"
}
```

## Success Fixtures

- `payloads/assignment-detail.success-unsubmitted.json`
- `payloads/assignment-detail.success-submitted.json`
- `payloads/assignment-detail.empty-contents.json`
- `payloads/assignment-detail.with-files.json`

필수 구조:

```json
{
  "rpt": {
    "ordseq": "7",
    "title": "과제명",
    "contents": "<p>과제 본문 HTML</p>",
    "expiredate": "2026-06-17 23:59:59",
    "reptype": "P",
    "submityn": "N",
    "submitfiletype": "상관없음",
    "filelimit": "30"
  },
  "smt": {
    "title": null,
    "contents": null,
    "finalscore": null,
    "tutorcontents": null
  }
}
```

## Failure Fixtures

- `payloads/assignment-detail.session-expired.html`
- `payloads/assignment-detail.missing-rpt.json`
- `payloads/assignment-detail.invalid-ordseq.json`
- `payloads/assignment-detail.template-page.html`

## Parser Assertions

- `rpt`가 없으면 상세 로드 실패다.
- `rpt.contents`는 HTML을 plain text 또는 markdown-safe text로 변환한다.
- `reptype` 값 `1`, `P`는 개인별로 표시한다.
- `reptype` 값 `2`는 팀별로 표시한다.
- `smt`는 null 필드가 많아도 성공이다.
- `TaskViewStdPage.do` HTML은 본문 데이터로 사용하지 않는다.

## Redaction Rules

- 제출 내용, 이름, 첨부 파일명은 익명화한다.
- HTML tag 구조는 유지한다.
