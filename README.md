# KLAP CLI

광운대학교 KLAS를 대체하기 위한 크로스 플랫폼 CLI입니다.
현재는 one-shot CLI 명령으로 시작하며, 추후 Bubble Tea 기반 TUI와 같은 app core를 공유하는 하이브리드 앱으로 확장한다.

## 초기 명령

```sh
klap auth
klap user list
klap user select <학번>
klap user rm <학번>
klap term list
klap term select <학기번호|학기값>
klap course list
klap subject search --name 컴퓨터그래픽스 --professor 김동준
klap syllabus <과목명|과목번호|학정번호>
klap syllabus I040-3-3951-01 --term 2026-1
klap academic list
klap academic list --year 2026
klap attendance
klap attendance list
klap attendance detail <과목명|번호|학정번호>
klap assignment list
klap assignment list --course <과목명|번호>
klap assignment detail <과목번호:과제번호>
klap assignment open <과목번호:과제번호>
klap assignment remind
klap assignment remind --auto
klap notice open <과목번호:게시판번호:글번호>
klap lecture list --course <과목명|번호>
klap lecture status
klap lecture status --course <과목명|번호>
klap lecture download <과목명|과목번호>
klap lecture download <과목번호:강의콘텐츠ID>
klap lecture open <과목번호:강의콘텐츠ID>
klap lecture attend <과목번호:강의콘텐츠ID>
klap lecture attend <과목번호:lrn-학습활동번호>
klap attend <과목명|번호>
klap attend all
klap config reminder
klap config reminder --name "광운대학교"
klap config reminder --name "To-do" --use-existing-list
```

`klap auth`는 학번과 비밀번호를 입력받은 뒤 KLAS 로그인 API로 즉시 검증한다.
검증에 성공한 계정만 저장하며, 비밀번호와 세션 쿠키는 OS 보안 저장소에 저장한다.

`klap term select`로 선택한 학기는 `course`, `assignment`, `notice`, `timetable`, `lecture`, `attend` 명령에 공통 적용된다.
선택한 학기가 없으면 KLAS가 반환하는 최신 학기를 사용한다.

Reminder 기본 목록 이름은 `Kwangwoon Univ.`이고, 마감 1일 전 알림을 생성한다.
`--use-existing-list`를 쓰면 지정한 기존 목록만 사용하며, 없을 때 새로 만들지 않는다.

## 개발 실행

```sh
go run ./cmd/klap --help
```

## 아키텍처

TUI 확장을 고려한 패키지 경계와 마이그레이션 계획은 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)를 기준으로 한다.
