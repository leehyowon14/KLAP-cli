# KLAP CLI/TUI Architecture

이 문서는 현재 구현을 설명한다. 미구현 기능과 추가 검증은 [ROADMAP](ROADMAP.md), 결정의 이유는 [ADR 0002](adr/0002-application-boundaries.md)와 [ADR 0003](adr/0003-compiled-macos-bridges.md)에 둔다.

## Runtime과 의존 경계

```text
cmd/klap -> bootstrap
              ├─ store와 adapter 생성 -> app.NewService(Dependencies)
              └─ mode 선택 -> cli.Runner 또는 tui.Run

cli / tui -> app use cases + app-owned results
app       -> session executor + consumer-owned ports
bootstrap -> klas / kwcommons / academic / download
          -> account / settings / cache / syncstate
          -> platform/macos -> SwiftPM executable
```

이 그림은 실행·조립 경계다. **app의 모든 compile-time adapter 의존이 제거된 것은 아니다.** app 내부에는 KLAS client factory와 정규화 모델, account/settings/cache 계약 타입에 대한 의존이 남는다. public 결과에서는 wire DTO embedding을 제거했고, concrete store 생성·직접 HTTP·subprocess·bridge 경로 탐색은 하지 않는다.

- [cmd/klap/main.go](../cmd/klap/main.go): stderr와 process exit code.
- [bootstrap/run.go](../internal/bootstrap/run.go): 인자 없음 또는 `tui`는 TUI, 나머지는 CLI. [dependency 구성](../internal/bootstrap/service.go)과 [bridge 탐색](../internal/bootstrap/bridge_paths.go)도 bootstrap 책임이다.
- [app/dependencies.go](../internal/app/dependencies.go): dependency 누락/typed nil을 시작 시 검증한다. `NewService` 자체는 I/O를 하지 않는다.
- [domain](../internal/domain): 여러 경계에서 공유하는 zero-I/O Course, Term, Session 타입.

## Application core

[app.Service](../internal/app/service.go)는 compatibility facade다. 구현은 feature별 파일, app-owned DTO는 `*_models.go`로 분리한다.

- [Session executor](../internal/app/session.go)는 만료에 한해 재로그인 후 최대 1회 retry하며, 새 Session 저장 실패는 성공으로 숨기지 않는다. [request session](../internal/app/request_session.go)은 사용자·Session·client를 요청 단위로 재사용한다.
- [Dashboard sync](../internal/app/dashboard_sync.go): 화면 대신 sync 순서와 결과를 조정한다.
- [Room workflow](../internal/app/room_workflow.go): 여러 요일 조회를 app에서 처리한다.
- [Download pipeline](../internal/app/download_pipeline.go)과 [run](../internal/app/download_run.go): worker 수명주기, progress, cancel, join 및 cleanup을 소유한다. cleanup은 이 run이 추적하는 새 파일에 한정하며 기존 resume artifact를 보호한다.
- [typed config update](../internal/app/config_update.go): presentation의 key/value 편집을 검증된 변경으로 적용한다.

app은 UI 문자열·CLI flag를 workflow 계약으로 사용하지 않는다. adapter 구현은 주입된 좁은 port로 교체할 수 있다. 다운로드 디렉터리 생성과 cleanup 같은 **workflow 소유 filesystem 정책**은 app에 남아 있으며 파일 I/O 전체를 제거한 구조는 아니다.

## Presentation

### CLI

[Runner](../internal/cli/runner.go)에 Service, In/Out/ErrOut, Opener, Clock, Terminal을 주입한다. [registry](../internal/cli/registry.go) 하나에서 dispatch와 help를 구성하고 feature 파일은 parse → use case → writer 기반 render 순서로 읽힌다. auth form도 CLI가 소유한다.

[registry_test.go](../internal/cli/registry_test.go)는 command/alias 중복, dispatch 인자 전달, help alias와 README command block의 일치를 검사한다.

### TUI

[root.go](../internal/tui/root.go)는 route, 전역 loading/error, window size, prefetch coordinator와 child를 보유한다. global message 처리 뒤 active child로 위임한다. 화면별 row/cursor/편집/진행 상태와 update/view는 같은 package의 `screen_*.go` child가 소유한다.

service I/O와 외부 URL 열기는 `tea.Cmd` 실행 시점에 수행한다. Config 저장 및 Download 준비에는 stale completion을 무시하는 generation 경계가 있다. 다운로드 worker와 파일 삭제 정책은 TUI가 결정하지 않는다. 현재 화면과 entrypoint는 [TUI README](../internal/tui/README.md)를 참고한다.

## Adapter와 저장소

| 경계 | 책임 |
| --- | --- |
| [klas](../internal/klas) | KLAS host의 transport·로그인·private wire DTO·endpoint parser, typed Network/HTTP/SessionExpired/RemoteBusiness/Schema 오류 |
| [kwcommons](../internal/kwcommons) | 공개 media URL 해석; KLAS 인증 cookie jar를 공유하지 않는 HTTP client |
| [academic](../internal/academic) | 학사일정 HTTP fetch/parser; app이 연도·cache 정책 소유 |
| [download](../internal/download) | HTTP transfer, Range resume, file naming, 폴더 status scan |
| [account](../internal/account) | registry와 OS secret store; atomic write, 실패 보상·typed partial failure |
| [settings](../internal/settings) | 설정 schema, 기본값, migration과 atomic 저장 |
| [cache](../internal/cache) | TTL이 있는 조회 cache; durable sync baseline과 별도 |
| [syncstate](../internal/syncstate) | versioned sync baseline, process lock, atomic replace, legacy migration |
| [platform/macos](../internal/platform/macos) | Reminder/Calendar/Category/Transcript, WakeLock와 공통 process 실행; desktop launcher의 OS별 명령 선택도 이 경계로 집중 |

`internal/reminder`, `calendar`, `category`, `transcript`는 JSON protocol 모델이며 concrete subprocess adapter가 아니다.

## 데이터 보호 계약

- 조회 cache 삭제는 config 영역의 sync state를 삭제하지 않는다.
- legacy baseline은 신규 저장 성공 후에만 제거한다. 기준 누락·손상만으로 EventKit 항목을 강제 overwrite하지 않는다.
- 영속 리소스 ID는 term + course identity + remote identity를 사용한다. CLI 과목 순번은 입력 호환 alias일 뿐이다.
- 목록 cache는 정규화 필드만 직렬화한다. 수강에 필요한 remote action payload는 action 시 다시 조회한다.
- durable store와 Session 오류는 반환한다. 조회 cache write는 best-effort로 무시할 수 있으며 별도 오류 관측 hook은 없다. TTL과 삭제 범위는 [cache 정책 구현](../internal/app/cache.go)에 둔다.

## macOS 실행·배포

SwiftPM 네 executable이 공통 EventKitCore/SpeechCore를 사용한다. 별도 배포 Swift script는 없다. 배포 bridge는 macOS 12 target, 실제 Speech 전사는 macOS 26 이상 availability 검사를 가진다.

`ProcessRunner.Run`은 JSON/NDJSON stdout과 stderr를 분리하며 decoder 실패·context 취소 시 child를 회수한다. `Start`는 desktop launcher의 비동기 시작 계약을 보존하고 stdout을 상속하는 자손 때문에 직접 child 회수가 막히지 않게 한다.

[재사용 workflow](../.github/workflows/bridge-artifacts.yml)가 검증한 universal artifact만 Darwin archive에 넣는다. archive 검증은 원본 byte 일치, 두 architecture, metadata/signature, 빈 Transcript 요청과 native CLI smoke를 확인한다. ad hoc signature는 notarization을 뜻하지 않는다. 개발·배포 명령은 [RELEASE](RELEASE.md)를 참고한다.

## 자동 경계 검사

[architecture test](../internal/architecture/dependencies_test.go)는 `go list -deps -json`의 직접 Imports와 presentation/account test imports를 검사한다.

- CLI/TUI → KLAS, account, settings, bootstrap, platform 금지
- CLI → TUI 및 account → KLAS 금지
- app production → 직접 HTTP/exec 및 새 concrete I/O adapter 역참조 금지

`cli -> app -> klas` 같은 전이 의존은 허용한다. 플랫폼 build constraint는 CI의 Linux/macOS/Windows 검사에서 각각 적용된다. endpoint별 구현·fixture·presenter 추적은 별도 API contract 문서를 따른다.

Canonical identity는 [ADR 0001](adr/0001-canonical-repository.md)에 기록되어 있다.
