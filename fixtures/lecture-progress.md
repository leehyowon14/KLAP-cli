# Lecture Progress Fixtures

## Endpoints

```http
POST /spv/lis/lctre/viewer/ChkLctreCntntsView.do
POST /spv/lis/lctre/viewer/UpdateProgress.do
```

## Request

Form 요청. `lecture-viewer.md`의 요청 body에 `lecKey`를 추가한다.

```json
{
  "lecKey": "<lec-key>"
}
```

## Success Fixtures

- `payloads/lecture-check.success-empty.json`
- `payloads/lecture-progress.success-nested.json`
- `payloads/lecture-progress.success-root.json`
- `payloads/lecture-progress.completed.json`

응답 후보:

```json
{
  "data": {
    "totalTime": "10",
    "ptime": "50",
    "prog": 20
  }
}
```

```json
{
  "totalTime": "50",
  "ptime": "50",
  "prog": 100
}
```

## Failure Fixtures

- `payloads/lecture-check.session-expired.html`
- `payloads/lecture-progress.invalid-lec-key.json`
- `payloads/lecture-progress.business-error.json`

## Parser Assertions

- `ChkLctreCntntsView.do`는 빈 객체 `{}`를 성공으로 본다.
- `UpdateProgress.do`는 `data` 내부와 루트 진행 필드를 모두 허용한다.
- `prog === 100`이면 완료다.
- `lecKey`가 없으면 호출하지 않는다.

## Redaction Rules

- `lecKey`, 사용자/과목 식별자는 저장하지 않는다.
