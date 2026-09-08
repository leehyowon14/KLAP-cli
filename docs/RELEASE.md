# Release

Canonical repository와 Go module은 `github.com/leehyowon14/KLAP-cli`다. GitHub Actions와 GoReleaser v2.17.1 설정이 배포 경로를 정의한다. 이 문서의 절차가 실제 publish·Homebrew 설치·notarization 완료를 뜻하지는 않는다.

## CI

[CI workflow](../.github/workflows/ci.yml)는 main push와 pull request에서 실행된다.

- Linux/macOS/Windows: Go test/build/vet/format, architecture guard 및 registry/help/README parity
- pinned golangci-lint v2.12.2
- [재사용 Swift workflow](../.github/workflows/bridge-artifacts.yml): Xcode 26.6, 순수 core 테스트, 네 universal executable 빌드, metadata·signature·빈 Transcript JSON/NDJSON 검사
- macOS snapshot: 검증된 bridge artifact를 다운로드하고 재빌드 없이 포함; archive manifest·원본 byte 일치·native CLI smoke 및 누락/치환 거부 테스트

Workflow YAML 변경은 로컬에서 actionlint로도 검사한다(현재 CI의 자동 step은 아님). Xcode 버전을 바꾸면 Intel compatibility library와 양 architecture artifact 검증을 함께 재실행한다.

## 로컬 검증

Go CLI만 빌드하는 명령은 Swift bridge를 만들지 않는다.

```sh
go test ./...
go build ./...
go vet ./...
go test -race ./...
golangci-lint run
```

macOS bridge는 macOS 26 SDK API를 제공하고 arm64/x86_64 runtime library를 모두 포함하는 **Xcode 환경**이 필요하다. 현재 공통 CI 선택은 Xcode 26.6이다. 설치된 Xcode가 아닌 CLT만 선택되어 있으면 그 toolchain의 library/테스트 framework 부족으로 실패할 수 있다.

```sh
swift test --disable-xctest --package-path bridges/macos
bash scripts/build-macos-bridges.sh
goreleaser check
goreleaser release --snapshot --clean --skip=publish
bash scripts/verify-release-archives.sh
bash scripts/test-release-archives.sh
```

공통 build script의 결과는 `bridges/macos/.build/artifacts/`다. 이 ignored 디렉터리는 Git에 추가하지 않는다. 직접 개발용 `swift build -c release --package-path bridges/macos` 결과도 bootstrap이 탐색하지만 release는 위 공통 script의 artifact만 사용한다.

## Archive 계약

기존 이름의 platform archive 다섯 개를 만든다.

- Darwin x86_64 / arm64: 해당 Go `klap` + 네 universal bridge
- Linux x86_64 / arm64: Go `klap`
- Windows x86_64: Go `klap.exe`

Darwin archive의 bridge 경로:

```text
bridges/macos/ReminderBridge
bridges/macos/CalendarBridge
bridges/macos/CategoryBridge
bridges/macos/TranscriptBridge
```

별도 Swift script를 배포하지 않는다. standalone bridge에는 권한 usage description과 고유 식별자가 들어 있고 ad hoc signature를 검사한다. deployment target은 macOS 12이지만 실제 Speech 전사는 macOS 26 이상에서만 실행한다. Developer ID 서명, notarization, Gatekeeper 및 각 OS의 실제 권한 검증은 [후속 검증](ROADMAP.md)이다.

## Tagged release

`v*` tag push가 [release workflow](../.github/workflows/release.yml)를 시작한다. tag 생성과 push는 명시적으로 배포할 때만 수행한다.

1. 공통 Swift workflow에서 테스트한 bridge를 artifact로 전달한다.
2. release job은 다운로드한 artifact를 snapshot archive에 넣어 검증한다.
3. 같은 bridge artifact를 유지한 채 GoReleaser release를 실행한다.
4. checksums, draft GitHub Release, Homebrew Cask를 생성한다.

GoReleaser는 Darwin/다른 OS build 및 archive ID를 구분한다. Homebrew Cask는 Darwin ID를 사용하고 `klap` binary 이름과 archive 파일명은 유지한다.

`GITHUB_TOKEN`은 Actions가 제공한다. Homebrew publish에는 `leehyowon14/homebrew-klap` repository와 그 repository에 쓰기 가능한 `HOMEBREW_TAP_TOKEN` secret이 필요하다. 실제 secret 설정과 publish 성공은 로컬 snapshot 검증 범위가 아니다.

## 설치와 탐색

배포 archive 전체를 유지해야 bundled bridge를 찾을 수 있다. Homebrew Cask는 staged archive의 `klap`을 연결하며 bootstrap은 symlink의 실제 위치를 해석한다. 설치된 bundled binary가 현재 작업 디렉터리의 개발 artifact보다 우선한다.

`go install .../cmd/klap@latest`로는 bridge가 설치되지 않는다. macOS 동기화·전사는 Darwin archive를 사용하거나 source checkout에서 bridge를 별도 빌드해야 한다.

명시적 override는 계속 지원한다.

- `KLAP_REMINDER_BRIDGE`
- `KLAP_CALENDAR_BRIDGE`
- `KLAP_CATEGORY_BRIDGE`
- `KLAP_TRANSCRIPT_BRIDGE`

override는 compiled executable 또는 사용자 정의 `.swift` script를 가리킬 수 있다. 후자의 경우에만 Swift interpreter가 필요하다.

## Rollback

이전 release의 **archive 전체**로 되돌린다. Go binary와 bridge를 서로 다른 release에서 섞지 않는다. source 변경은 목적별 commit을 revert하되 dependency가 있는 후속 변경과 함께 검토한다. 사용자 registry, cache, 설정, sync state를 삭제하는 것은 rollback 절차가 아니다.
