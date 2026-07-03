# Login Security Fixture

## Endpoint

```http
POST /usr/cmn/login/LoginSecurity.do
```

## Request

JSON body 없음.

## Success Fixtures

- `payloads/auth-login-security.success.json`

필수 구조:

```json
{
  "publicKey": "<rsa-public-key-body>"
}
```

메타데이터에는 `Set-Cookie` 존재 여부만 기록하고 쿠키 값은 저장하지 않는다.

## Failure Fixtures

- `payloads/auth-login-security.missing-public-key.json`
- `payloads/auth-login-security.no-cookie.json`
- `payloads/auth-login-security.html`

## Parser Assertions

- `publicKey`가 비어 있으면 로그인 시작 실패다.
- PEM 헤더/푸터 없이 내려온 public key body를 PEM으로 복원할 수 있어야 한다.
- `Set-Cookie` 값은 저장하지 않지만 쿠키가 있었는지는 테스트한다.

## Redaction Rules

- `publicKey`는 테스트 암호화를 실제로 수행해야 하면 유지 가능하다.
- 저장이 부담되면 `<rsa-public-key-body>`로 치환하고 parser test에서는 shape만 검증한다.
