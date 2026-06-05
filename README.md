# KLAP CLI

광운대학교 KLAS를 대체하기 위한 크로스 플랫폼 CLI입니다.

## 초기 명령

```sh
klap auth
klap user list
klap user select <학번>
klap user rm <학번>
klap course list
```

`klap auth`는 학번과 비밀번호를 입력받은 뒤 KLAS 로그인 API로 즉시 검증한다.
검증에 성공한 계정만 저장하며, 비밀번호와 세션 쿠키는 OS 보안 저장소에 저장한다.

## 개발 실행

```sh
go run ./cmd/klap --help
```
