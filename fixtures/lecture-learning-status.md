# Lecture Learning Status Fixtures

## Endpoints

```http
POST /std/lis/evltn/SelectLrnSttusStd.do
POST /std/lis/evltn/CertiStdCheck.do
POST /std/lis/evltn/SaveLrnStatus.do
```

## Request

`SelectLrnSttusStd.do`와 `CertiStdCheck.do`는 KLAS 온라인 강의 페이지의 `viewForm` 또는 `lrnCerti.$data`와 같은 필드를 보낸다.

```json
{
  "grcode": "<grcode>",
  "subj": "<subject-id>",
  "year": "2026",
  "hakgi": "1",
  "bunban": "01",
  "module": "01",
  "lesson": "001",
  "oid": "<oid>",
  "weeklyseq": "1",
  "weeklysubseq": "1"
}
```

`SaveLrnStatus.do`는 현재 구현에서 학습 상태 저장 후보 API로 사용한다.

```json
{
  "grcode": "<grcode>",
  "subj": "<subject-id>",
  "year": "2026",
  "hakgi": "1",
  "lrnStatus": "Y"
}
```

## Success Fixtures

- `payloads/lecture-learning-status.success-y.txt`
- `payloads/lecture-learning-status.success-n.txt`
- `payloads/lecture-certification-check.success.json`
- `payloads/lecture-save-learning-status.success.json`

응답 후보:

```text
Y
```

```text
N
```

```json
{}
```

## Failure Fixtures

- `payloads/lecture-learning-status.session-expired.html`
- `payloads/lecture-learning-status.business-error.json`
- `payloads/lecture-learning-status.unexpected-string.txt`
- `payloads/lecture-certification-check.invalid-payload.json`

## Parser Assertions

- `SelectLrnSttusStd.do` 응답이 `Y` 또는 `N`이면 뷰어 열기 가능 상태로 본다.
- 로그인 페이지 HTML이 내려오면 세션 만료로 분류한다.
- `CertiStdCheck.do`는 HTTP 성공과 공통 실패 플래그가 없으면 인증 체크 통과로 본다.
- `SaveLrnStatus.do`는 `UpdateProgress.do`와 같은 진행률 응답 shape도 허용한다.

## Redaction Rules

- 과목 ID, `grcode`, `oid`, 인증 관련 값은 치환한다.
