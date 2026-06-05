# KLAP CLI/TUI Hybrid Architecture

KLAP은 단순 CLI에서 시작하지만, 최종 형태는 CLI 명령과 Bubble Tea 기반 TUI가 같은 기능을 공유하는 하이브리드 애플리케이션이다.
따라서 네트워크/API 호출, 세션 관리, 파싱, Reminder 동기화 같은 업무 로직은 화면 계층에 두지 않는다.

## Goals

- `klap assignment list` 같은 one-shot CLI 명령을 유지한다.
- `klap tui` 또는 인자 없는 `klap`에서 TUI 앱으로 확장할 수 있게 한다.
- CLI와 TUI가 같은 인증, 수업, 과제, reminder 유스케이스를 호출한다.
- KLAS API, Keychain, Swift bridge는 교체 가능한 adapter로 둔다.
- 테스트는 화면이 아니라 headless app core를 기준으로 작성한다.

## High-Level Shape

```text
cmd/klap
  -> internal/cli      one-shot command parser and text output
  -> internal/tui      future interactive Bubble Tea shell

internal/cli
internal/tui
  -> internal/app      headless use cases and app state orchestration

internal/app
  -> internal/klas     KLAS HTTP adapter
  -> internal/account  account/session storage adapter
  -> internal/reminder macOS EventKit bridge adapter
```

## Package Responsibilities

### `cmd/klap`

Process entrypoint only.

Responsibilities:

- Build the root runtime.
- Decide presentation mode:
  - args present: run CLI command.
  - future no args or `tui`: run TUI.
- Return process exit code through errors.

Do not put API calls, parsing, or terminal rendering here.

### `internal/app`

Headless application core.

Responsibilities:

- Select current user.
- Load saved session and retry login on `klas.ErrSessionExpired`.
- Expose stable use cases:
  - `Authenticate`
  - `ListUsers`
  - `SelectUser`
  - `RemoveUser`
  - `ListCourses`
  - `ListAssignments`
  - `GetAssignmentDetail`
  - `SyncAssignmentReminders`
- Return structured results that CLI and TUI can render differently.

Rules:

- No Bubble Tea imports.
- No `fmt.Println`.
- No command-line flag parsing.
- No direct Swift script path assumptions outside reminder adapter configuration.

### `internal/cli`

One-shot command presentation.

Responsibilities:

- Parse argv.
- Call `internal/app` use cases.
- Render plain text tables/details.
- Keep output stable enough for a user to copy IDs.

Rules:

- Do not store app state beyond command-local variables.
- Do not call `internal/klas` directly after app core extraction.
- Do not own session retry logic.

### `internal/tui`

Interactive presentation for future TUI.

Responsibilities:

- Own Bubble Tea models, messages, key bindings, tabs, and screens.
- Call `internal/app` use cases from commands/effects.
- Maintain view-local state such as selected row, filter input, loading state, and errors.

Expected screens:

- Dashboard
- Courses
- Assignments
- Assignment detail
- Reminder sync status
- Users/settings

Rules:

- Do not parse KLAS response shapes.
- Do not read/write Keychain directly.
- Do not duplicate CLI command logic.

### `internal/klas`

KLAS HTTP adapter.

Responsibilities:

- Login flow:
  - `LoginSecurity.do`
  - RSA login token creation
  - `LoginConfirm.do`
- Cookie jar handling.
- KLAS API request/response parsing.
- Common failure classification:
  - HTTP error
  - login HTML
  - `loginRequired`
  - `errorCount`
  - schema change

Rules:

- Return domain-oriented structs, not raw UI strings.
- Keep endpoint-specific raw response structs private unless a use case needs them.

### `internal/account`

Account/session persistence adapter.

Responsibilities:

- Store password and session cookies in OS secure storage.
- Store non-secret user registry in config dir.
- Track selected/current student ID.

Rules:

- Never write password/session cookie plaintext into git-tracked paths.
- Registry may contain `studentId`, `userId`, timestamps, and current user only.

### `internal/reminder`

Reminder synchronization adapter.

Responsibilities:

- Accept structured assignment reminder payloads.
- On macOS, call Swift EventKit bridge.
- On non-macOS, return a clear unsupported error.

Rules:

- Keep EventKit details out of app/CLI/TUI.
- The bridge should be idempotent using stable assignment IDs.

## Current Command Surface

```text
klap auth
klap user list
klap user select <학번>
klap user rm <학번>
klap course list
klap assignment list
klap assignment list --course <과목명|번호>
klap assignment detail <과목번호:ordseq>
klap assignment remind
klap assignment remind --auto
```

Future TUI entrypoint:

```text
klap tui
```

Optionally, after the TUI is usable:

```text
klap
```

can launch TUI instead of printing help. Keep `klap --help` as the stable help path.

## Shared Domain IDs

Course list prints 1-based course numbers for user-facing selection.

Assignment IDs are currently:

```text
<course-list-number>:<ordseq>
```

Example:

```text
3:7
```

This is intentionally display-oriented. If KLAS exposes a more stable assignment identifier later, app core should introduce an internal stable ID while CLI can keep backward-compatible aliases.

## App Core Result Shapes

The app core should expose structured rows, not preformatted text.

```go
type CourseRow struct {
    Index int
    Name string
    TermLabel string
    TermValue string
}

type AssignmentRow struct {
    ID string
    CourseIndex int
    CourseName string
    Title string
    DueAt *time.Time
    Submitted bool
}
```

CLI renders these as text. TUI renders them as tables/lists/cards.

## Session Flow

```text
use case starts
  -> resolve selected user
  -> load saved session
  -> call KLAS
  -> if ErrSessionExpired:
       load saved password
       login again
       save new session
       retry once
  -> return structured result
```

This flow belongs in `internal/app`, not CLI or TUI.

## TUI State Model

TUI should use a small shell model with screen-specific child models.

```text
RootModel
  - currentUser
  - activeScreen
  - syncStatus
  - courses cache
  - assignments cache
  - error toast/status

Screens
  - DashboardModel
  - CoursesModel
  - AssignmentsModel
  - AssignmentDetailModel
  - UsersModel
```

Effects call app use cases and return messages:

```text
LoadAssignmentsCmd -> AssignmentsLoadedMsg | AppErrorMsg
SyncRemindersCmd   -> ReminderSyncedMsg    | AppErrorMsg
```

## Migration Plan

1. Create `internal/app` use cases and move current session retry/course/assignment orchestration there.
2. Keep `internal/cli` as argv parsing plus output formatting.
3. Add `klap tui` with a simple Bubble Tea root shell.
4. Move `auth` input form out of `internal/ui` into either:
   - `internal/cliui` for CLI-only forms, or
   - `internal/tui/auth` if it becomes a screen.
5. Add fixture-backed parser tests for `internal/klas`.
6. Add app-core tests with fake KLAS/account/reminder adapters.

## Dependency Rule

Allowed dependency direction:

```text
cmd -> cli/tui -> app -> adapters
```

Forbidden:

```text
app -> cli
app -> tui
klas -> cli/tui
account -> cli/tui
reminder -> cli/tui
```

If a package needs terminal rendering, it is not app core.
