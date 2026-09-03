# KLAP CLI

광운대학교 KLAS를 대체하기 위한 크로스 플랫폼 CLI입니다.
기본 실행은 Bubble Tea 기반 TUI이며, 세부 작업은 one-shot CLI 명령으로도 수행할 수 있다.

## 설치

Canonical repository는 `github.com/leehyowon14/KLAP-cli`이다.
현재 공개 Release와 Homebrew Tap은 준비 중이므로 Go toolchain으로 설치한다.

```sh
go install github.com/leehyowon14/KLAP-cli/cmd/klap@latest
```

첫 tagged release와 `leehyowon14/homebrew-klap` Tap 준비가 끝난 뒤에는 다음 명령을 사용한다.

```sh
brew tap leehyowon14/klap
brew install --cask klap
```

## 초기 명령

```sh
klap
klap auth
klap user list
klap user select <학번>
klap user rm <학번>
klap dashboard
klap dashboard --refresh
klap tui
klap search <키워드>
klap search <키워드> --type assignment
klap due
klap due --week
klap due --days 30
klap cache status
klap cache clear
klap cache clear assignment
klap term list
klap term select <학기번호|학기값>
klap course list
klap course list --refresh
klap subject search --name 컴퓨터그래픽스 --professor 김동준
klap syllabus <과목명|과목번호|학정번호>
klap syllabus I040-3-3951-01 --term 2026-1
klap academic list
klap academic list --year 2026
klap academic list --refresh
klap attendance
klap attendance list
klap attendance detail <과목명|번호|학정번호>
klap attendance cdp
klap grade
klap grade list
klap grade 2025-1
klap grade list 2025-2
klap rank
klap rank list
klap rank 2025-1
klap rank --term 2025-1
klap evaluation list
klap evaluation submit all
klap evaluation submit all --yes
klap evaluation submit <과목명|번호> --yes
klap assignment list
klap assignment list --course <과목명|번호>
klap assignment list --refresh
klap assignment detail <과목번호:과제번호>
klap assignment open <과목번호:과제번호>
klap assignment remind
klap assignment remind --auto
klap notice open <과목번호:게시판번호:글번호>
klap lecture list --course <과목명|번호>
klap lecture status
klap lecture status --course <과목명|번호>
klap lecture status --refresh
klap lecture download status
klap lecture download open
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
klap config list
klap config set term 2026-1
klap config set reminder.name "광운대학교"
klap config set reminder.use-existing-list true
klap config download
klap config download --dir ~/Downloads/KLAP
klap config download --concurrency 8
```

`klap auth`는 학번과 비밀번호를 입력받은 뒤 KLAS 로그인 API로 즉시 검증한다.
검증에 성공한 계정만 저장하며, 비밀번호와 세션 쿠키는 OS 보안 저장소에 저장한다.

`klap term select`로 선택한 학기는 `course`, `assignment`, `notice`, `timetable`, `lecture`, `attend` 명령에 공통 적용된다.
선택한 학기가 없으면 KLAS가 반환하는 최신 학기를 사용한다.

`klap evaluation submit`은 기본적으로 제출하지 않고 미리보기만 출력한다.
실제 저장 API는 `--yes`가 있을 때만 호출하며, 공학인증 추가 문항은 기본 제외한다.

`klap dashboard`와 주요 목록 조회는 기본 5분 동안 디스크 캐시를 사용한다.
즉시 새로 조회하려면 `--refresh`를 사용하고, 특정 캐시를 지우려면 `klap cache clear assignment`처럼 scope를 지정한다.

`klap search <키워드>`는 현재 학기 과목, 과제, 공지, 온라인 강의와 올해 학사일정을 한 번에 검색한다.
`--type course|assignment|notice|lecture|academic`으로 검색 범위를 좁힐 수 있다.

`klap due`는 과제, 미완료 온라인 강의/학습활동, 학사일정을 마감순으로 합쳐 보여준다.
기본 범위는 14일이며 `--week` 또는 `--days 30`으로 조정할 수 있다.

`klap lecture download <과목명|과목번호>`는 일반 CLI 출력으로 과목 내 강의를 다운로드한다.
`klap tui`의 Lectures 화면에서는 `d`를 눌러 Space로 받을 강의를 선택하고, 이어서 전사 여부를 선택한 뒤 같은 위치에서 progress bar를 갱신하며 다운로드한다.
동시 다운로드 수는 `klap config download --concurrency <수>`로 바꾼다.
`--dir`이 없으면 `klap config download --dir <경로>`로 저장한 기본 다운로드 폴더를 사용한다.
전사는 macOS 26 이상의 Apple Speech.framework를 사용하며, 결과는 다운로드 파일 옆의 `.txt` 파일로 저장한다.
`klap lecture download status`와 `klap lecture download open`으로 받은 파일을 확인할 수 있다.

Reminder 기본 목록 이름은 `Kwangwoon Univ.`이고, 마감 1일 전 알림을 생성한다.
`--use-existing-list`를 쓰면 지정한 기존 목록만 사용하며, 없을 때 새로 만들지 않는다.

`klap config list`는 현재 설정을 한 번에 보여준다.
`klap config set <key> <value>`는 `term`, `reminder.name`, `reminder.use-existing-list`, `download.dir`, `download.concurrency`를 지원한다.

## 개발 실행

```sh
go run ./cmd/klap
```

## 릴리즈

CI/CD와 Homebrew 배포 절차는 [docs/RELEASE.md](docs/RELEASE.md)를 기준으로 한다.

## 아키텍처

TUI 확장을 고려한 패키지 경계와 마이그레이션 계획은 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)를 기준으로 한다.
