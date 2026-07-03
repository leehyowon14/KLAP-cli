# Lecture Viewer Fixture

## Endpoint

```http
POST /spv/lis/lctre/viewer/LctreCntntsViewSpvPage.do
```

## Request

Form 요청.

```json
{
  "grcode": "<grcode>",
  "subj": "<subject-id>",
  "year": "2026",
  "hakgi": "1",
  "bunban": "01",
  "module": "01",
  "oid": "C000000000",
  "ptime": "50",
  "weeklyseq": "1",
  "weeklysubseq": "1",
  "totalTime": "0",
  "prog": "0",
  "lesson": "001",
  "profYN": "Y",
  "previewYN": "N",
  "late": "N"
}
```

## Success Fixtures

- `payloads/lecture-viewer.success.html`

HTML/스크립트 안에 `lecKey`가 있어야 한다.

```js
"lecKey": "<lec-key>"
```

## Failure Fixtures

- `payloads/lecture-viewer.missing-lec-key.html`
- `payloads/lecture-viewer.session-expired.html`
- `payloads/lecture-viewer.invalid-weeklyseq.html`

## Parser Assertions

- 정규식 `"lecKey"\\s*:\\s*'([^']+)'` 또는 실제 확인된 quoting 변형을 지원한다.
- `lecKey`가 없으면 자동 수강을 중단한다.
- HTML이 로그인 페이지면 세션 만료로 분류한다.

## Redaction Rules

- `lecKey`는 `<lec-key>`로 치환한다.
- HTML은 key 추출에 필요한 script 주변만 저장한다.
