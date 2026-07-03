# Login Confirm Fixture

## Endpoint

```http
POST /usr/cmn/login/LoginConfirm.do
```

## Request

```json
{
  "loginToken": "<rsa_pkcs1_base64>",
  "redirectUrl": "",
  "redirectTabUrl": ""
}
```

## Success Fixtures

- `payloads/auth-login-confirm.success.json`

필수 구조:

```json
{
  "fieldErrors": [],
  "response": {
    "userId": "<user-id>"
  },
  "errorCount": 0,
  "loginRequired": false
}
```

## Failure Fixtures

- `payloads/auth-login-confirm.invalid-password.json`
- `payloads/auth-login-confirm.locked.json`
- `payloads/auth-login-confirm.session-lost.json`

## Parser Assertions

- HTTP 200이어도 `errorCount > 0`이면 로그인 실패다.
- `fieldErrors[0].message`를 사용자 표시 메시지로 사용한다.
- 성공 조건은 `loginRequired === false`, `errorCount === 0`, `response.userId` 존재다.

## Redaction Rules

- `loginToken`은 저장하지 않는다.
- `userId`는 `<user-id>`로 치환한다.
