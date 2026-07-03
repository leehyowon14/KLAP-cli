# Session Update Fixture

## Endpoint

```http
POST /usr/cmn/login/UpdateSession.do
```

## Request

KLAS 공통 frame HTML 안의 세션 연장 함수가 호출하는 API다. 실제 body 유무는 추가 캡처가 필요하다.

```json
{}
```

## Success Fixtures

- `payloads/session-update.success-empty.json`
- `payloads/session-update.success-string.txt`
- `payloads/session-update.success-cookie-only.http`

## Failure Fixtures

- `payloads/session-update.session-expired.html`
- `payloads/session-update.business-error.json`
- `payloads/session-update.http-error.html`

## Parser Assertions

- HTTP `2xx`이고 로그인 페이지 HTML이 아니면 세션 연장 성공 후보로 본다.
- `Set-Cookie`가 있으면 저장된 세션 쿠키를 갱신한다.
- 실패해도 즉시 로그아웃하지 않고 다음 API 호출의 세션 만료 판정을 우선한다.

## Redaction Rules

- `SESSION`, `WMONID`, `JSESSIONID` 등 모든 쿠키를 제거한다.

## Open Questions

- 요청 body가 항상 비어 있는지 확인 필요
- 성공 body shape 확인 필요
