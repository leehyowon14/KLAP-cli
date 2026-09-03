# KLAP CLI/TUI Architecture

이 문서는 현재 구현과 목표 구조를 구분해 설명한다. 아래의 **Current Architecture**는 지금 실행되는 코드이며, **Target Architecture**는 이후 구조 리팩터링을 통해 도달할 방향이다.

## Current Architecture

### Runtime entrypoint

```text
cmd/klap
  -> internal/cli
       -> account + app + klas + tui + ui

internal/tui
  -> app + klas + settings

internal/app
  -> account + cache + settings + syncstate + klas
  -> calendar + reminder + category + transcript
  -> HTTP + filesystem + subprocess + bridge path discovery
```

`cmd/klap`은 process exit code와 stdout/stderr를 처리한다. `internal/cli`가 account store와 `app.Service`를 만들고 실행 mode를 고른다.

- 인자가 없거나 첫 인자가 `tui`이면 Bubble Tea TUI를 실행한다.
- 그 밖의 인자는 one-shot CLI command로 dispatch한다.
- `--help`, `-h`, `help`는 CLI help를 출력한다.

현재 composition과 presentation dispatch가 `internal/cli`에 함께 있으므로 `cmd/klap`이 완전한 composition root인 상태는 아니다.

### `internal/app`

`app.Service`는 CLI와 TUI가 공유하는 compatibility facade이자 현재 application orchestration의 중심이다.

현재 책임:

- 선택 사용자, Session load, 만료 후 재로그인과 1회 retry
- 학기, 과목, Dashboard, Due, Assignment, Notice, 출결, 성적, 강의평가, Syllabus, Lecture use case
- lecture download, resume, file naming, transcript orchestration
- Reminder, Calendar, timetable 동기화와 conflict 판단
- cache, settings, versioned sync state 사용과 legacy sync cache migration
- 학사일정 HTTP 호출과 일부 platform subprocess 실행
- 배포 archive 또는 환경 변수에서 macOS bridge 경로 탐색

이 패키지는 Bubble Tea를 import하지 않으며 구조화된 결과를 반환한다. 다만 concrete store와 adapter 생성, KLAS 타입 노출, HTTP/filesystem/subprocess 책임이 아직 남아 있다.

### `internal/cli`

one-shot command parser와 text renderer를 제공한다. 현재는 다음 책임도 함께 가진다.

- account store와 `app.Service` 생성
- CLI/TUI mode 선택
- CLI auth form을 위한 `internal/ui` 사용
- 일부 `account`, `klas`, terminal/platform API 직접 사용
- 전역 stdout 기반 출력

따라서 아직 순수한 `parse -> use case -> render` 경계는 아니다.

### `internal/tui`

Bubble Tea 기반 TUI는 현재 구현되어 있으며, 인자 없는 `klap`과 `klap tui`로 실행된다. 하나의 root model이 route와 대부분의 feature state 및 workflow를 소유한다.

현재 주요 화면과 하위 흐름:

- Auth와 Home
- Dashboard와 course별 Syllabus
- Due
- Assignments와 detail
- Notices와 detail
- Lectures
  - Download 선택, 전사 여부, 언어 선택, progress와 cancel/cleanup
  - Attend 확인, progress와 cancel
- Academic calendar
- Rooms 요일, 교시, 결과
- Config 일반, 일정, 다운로드 설정
- Reminder/Calendar sync conflict 결정

service I/O는 Bubble Tea command에서 실행하는 것을 기본으로 하지만, root model이 화면별 state와 다단계 workflow까지 크게 소유한다. `internal/klas`와 `internal/settings` 타입에 대한 직접 의존도 남아 있다.

### External adapters and persistence

- `internal/klas`: KLAS transport, 로그인, wire DTO, endpoint parser를 포함한다. KWCommons 과목 검색과 학사일정 관련 HTTP 책임도 아직 완전히 분리되지 않았다.
- `internal/account`: 사용자 registry와 Keychain password/Session을 관리한다. registry는 atomic JSON write와 credential rollback 정책을 가진다.
- `internal/cache`: 만료 가능한 조회 cache를 관리한다. sync baseline의 신규 저장소로 사용하지 않는다.
- `internal/syncstate`: versioned `sync-state.json`, atomic replace, process lock을 관리한다.
- `internal/settings`: 사용자 설정 JSON과 기본값을 관리한다.
- `internal/reminder`, `calendar`, `category`, `transcript`: macOS Swift subprocess protocol을 감싼다.
- `bridges/macos`: 실제 배포되는 Swift script 네 개와 SwiftPM `TranscriptBridge` source를 포함한다.

### Current session flow

```text
use case
  -> selected user 조회
  -> saved Session load
  -> KLAS request
  -> SessionExpired이면 password load와 login
  -> 새 Session을 durable store에 저장
  -> 같은 request를 최대 1회 retry
  -> structured result 또는 error 반환
```

이 흐름은 `internal/app`에 있지만 feature별 retry block이 아직 공통 executor로 완전히 통합되지는 않았다.

### Current cache and sync-state policy

- 조회 cache는 `klap cache clear` 대상이다.
- Reminder/Calendar baseline은 `internal/syncstate`의 config 영역에 별도로 저장한다.
- legacy sync cache는 신규 sync state 저장 성공 후에만 제거한다.
- baseline 누락이나 손상은 자동 overwrite로 해석하지 않는다.
- persistent resource ID는 term, course identity, remote resource identity를 사용한다.
- CLI의 `<과목번호>:...` 형식은 화면 순번 기반 입력 alias로만 유지한다.

### Current macOS bridge and release policy

- Reminder, Calendar, Category, Transcript Swift script는 release archive의 `bridges/macos`에 포함된다.
- CI는 네 script를 모두 typecheck하고 Transcript SwiftPM target을 release mode로 build한다.
- transcript JSON/NDJSON protocol은 공유 fixture와 실제 배포 script smoke test로 검증한다.
- Homebrew Cask symlink 실행 시 실제 staged executable 경로를 해석해 bundled bridge를 찾는다.
- compiled `TranscriptBridge`와 배포 `transcribe.swift`의 중복은 아직 남아 있다.

## Target Architecture

### Dependency direction

```text
cmd/klap 또는 internal/bootstrap
  ├─ store와 adapter 생성
  ├─ app 조립
  └─ cli 또는 tui 실행

internal/cli
internal/tui
  -> app use-case interfaces + app-owned DTOs

internal/app
  -> consumer-owned narrow ports

adapters
  ├─ klas / kwcommons / academic
  ├─ account / settings / cache / syncstate
  └─ platform/macos
```

목표 의존 규칙:

- CLI와 TUI는 `internal/klas`, `account`, `settings`를 직접 import하지 않는다.
- app public result는 KLAS wire DTO를 노출하지 않는다.
- app은 concrete store, HTTP client, `os/exec`, bridge path를 직접 만들거나 사용하지 않는다.
- composition root가 dependency 누락과 초기화 실패를 process 시작 시점에 검증한다.
- 외부 adapter interface는 사용하는 app feature가 좁게 소유한다.

### Target application boundary

`app.NewService(Dependencies)`가 명시적으로 주입된 dependency를 조립하고, 기존 `Service` API는 소비자 이전 동안 compatibility facade로 유지한다.

목표 application 책임:

- app-owned request/result와 use-case orchestration
- 공통 Session executor와 typed remote errors
- Dashboard sync, Room multi-day query, Download pipeline 같은 workflow
- cancel, cleanup, progress event와 sync decision 정책

목표 application이 소유하지 않는 책임:

- CLI flag와 사용자 문구
- Bubble Tea state
- KLAS wire JSON
- Keychain, JSON file, EventKit, Speech, process 실행의 concrete 구현

### Target presentation boundary

CLI는 command registry 하나에서 dispatch와 help를 생성하고, `Runner`에 writer, terminal capability, opener, clock을 주입한다.

TUI root는 route, back stack, window size, global notification과 active child만 소유한다. Assignment, Notice, Lecture, Config, Room, Sync, Download, Attend의 state/update/view는 같은 `internal/tui` 패키지의 child model로 이동한다.

### Target adapter boundary

- KLAS host 전용 adapter와 KWCommons, academic adapter를 분리한다.
- download lifecycle은 검증된 뒤 독립 adapter로 분리한다.
- macOS 구현은 cancellable `ProcessRunner`를 공유한다.
- SwiftPM이 Reminder, Calendar, Category, Transcript의 실제 배포 executable을 모두 build한다.
- CI가 build한 동일 artifact만 release archive에 포함하고 중복 `transcribe.swift` entrypoint를 제거한다.

### Target dependency guards

CI에서 다음 의존을 차단한다.

```text
cli/tui -> klas
cli/tui -> account/settings
account -> klas
```

command registry, CLI help, README parity와 endpoint-to-presenter coverage도 자동 검사한다.

## Migration constraints

- package 이동 전에 동일 package 안에서 feature별 파일을 먼저 분리한다.
- 순수 이동과 동작 변경을 같은 커밋에 넣지 않는다.
- 기존 public `Service` API는 소비자 이전 전까지 유지한다.
- 화면 또는 endpoint마다 새 package를 만들지 않는다.
- 공유되는 안정된 zero-I/O 타입이 충분할 때만 하나의 `internal/domain` package 도입을 검토한다.

Canonical repository 결정은 [`adr/0001-canonical-repository.md`](adr/0001-canonical-repository.md)에 기록되어 있다.
