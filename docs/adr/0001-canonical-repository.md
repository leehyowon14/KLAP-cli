# ADR-0001: Canonical Repository를 leehyowon14/KLAP-cli로 통일

- 상태: Accepted
- 결정일: 2026-09-03

## Context

저장소의 Go module은 `github.com/kw-klap/klap-cli`를 사용하지만, 실제 `origin`과 GoReleaser의 GitHub Release 및 Homebrew 배포 대상은 `leehyowon14/KLAP-cli`를 기준으로 설정되어 있다. 아직 공개 Release와 Homebrew Tap은 준비 전이지만, 서로 다른 identity가 남아 있으면 설치 문서와 import 경로가 예정된 배포 위치를 설명하지 못하고 향후 package 이동에서 잘못된 namespace가 더 확산될 수 있다.

현재 `kw-klap/klap-cli`로 repository를 이전한다는 확정 계획이나 선행 migration은 없다.

## Decision

Canonical repository와 Go module path를 다음 값으로 통일한다.

```text
github.com/leehyowon14/KLAP-cli
```

`go.mod`, 내부 import, GoReleaser, Homebrew, 설치 문서는 이 identity를 사용한다. GitHub repository 이름의 대소문자를 포함해 위 표기를 기준으로 삼는다.

## Consequences

- 기존 `github.com/kw-klap/klap-cli` import는 모두 새 module path로 변경한다.
- 현재 설정된 GitHub Release와 Homebrew 배포 대상은 repository 이동 없이 유지한다.
- 향후 `kw-klap` 조직으로 이전하려면 repository/release/Homebrew migration 계획을 먼저 세우고 이 결정을 대체하는 ADR을 추가한다.
- 아직 외부 소비자가 고정되지 않은 초기 버전에서 module path를 변경하므로 별도 호환 module은 제공하지 않는다.
