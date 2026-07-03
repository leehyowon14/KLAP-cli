# Common Fixture Rules

## 공통 실패 응답

모든 JSON API는 아래 케이스를 fixture로 보관한다.

- HTTP 오류 상태
- JSON body 안의 `loginRequired: true`
- `errorCount > 0`
- `fieldErrors[].message`
- 로그인 페이지 HTML
- KLAS 장애/점검 HTML
- JSON이 아닌 문자열 응답

## 권장 파일명

```text
payloads/<api-name>.success.json
payloads/<api-name>.empty.json
payloads/<api-name>.session-expired.html
payloads/<api-name>.business-error.json
payloads/<api-name>.malformed.json
```

## 공통 Parser Assertions

- 로그인 페이지 HTML은 `SessionExpired`로 분류한다.
- HTTP 200이어도 `errorCount > 0`이면 실패다.
- 루트 shape가 명세와 다르면 `SchemaChanged`로 분류한다.
- 빈 배열/빈 목록은 API 실패가 아니다.
- 날짜 파싱 실패는 해당 항목의 날짜만 `nil` 또는 `unknown`으로 둔다.

## Redaction Rules

반드시 치환한다.

| 원본 | 치환값 |
| --- | --- |
| 학번, userId | `<user-id>` |
| 이름 | `<user-name>` |
| 과목 내부 ID | `<subject-id>` |
| grcode | `<grcode>` |
| registerId | `<register-id>` |
| IP | `<ip-address>` |
| 쿠키 | `<redacted-cookie>` |
| lecKey | `<lec-key>` |
| KWCommons 콘텐츠 ID | `<content-id>` |

가능하면 유지한다.

- 날짜 형식
- 숫자/문자열 타입
- null 여부
- 배열/객체 구조
- HTML tag 구조

## Capture Metadata

payload 옆에 같은 이름의 `.meta.json`을 둘 수 있다.

```json
{
  "capturedAt": "2026-06-04T00:00:00+09:00",
  "endpoint": "/path",
  "method": "POST",
  "httpStatus": 200,
  "contentType": "application/json;charset=utf-8",
  "scenario": "success",
  "redacted": true
}
```
