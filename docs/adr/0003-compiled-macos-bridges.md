# ADR 0003: 검증된 macOS bridge artifact 배포

상태: 적용됨

## 배경

SwiftPM에서 검증한 source와 배포하는 별도 Swift script가 다르면 CI 통과가 실제 배포 실행 경로의 검증을 의미하지 않는다.

## 결정

- [SwiftPM package](../../bridges/macos/Package.swift)가 ReminderBridge, CalendarBridge, CategoryBridge, TranscriptBridge 네 executable을 생성한다. EventKitCore와 SpeechCore는 공통 로직과 비파괴 테스트의 경계다.
- [공통 build script](../../scripts/build-macos-bridges.sh)가 arm64/x86_64를 각각 빌드한 뒤 universal binary를 생성한다. [재사용 workflow](../../.github/workflows/bridge-artifacts.yml)는 Swift core 테스트와 artifact 검증을 수행하고 tar로 전달한다.
- CI snapshot과 tag release는 전달받은 bridge를 재빌드하지 않는다. Darwin archive에만 포함하고 [archive verifier](../../scripts/verify-release-archives.sh)가 원본과 정확한 byte 일치를 검사한다.
- standalone binary에 고유 식별자와 권한 usage description을 포함한다. 공통 deployment target은 macOS 12이며 실제 Speech 전사는 macOS 26 이상에서만 실행한다.
- [ProcessRunner](../../internal/platform/macos/process_runner.go)는 protocol stdout과 stderr를 분리하고 context 취소·typed 오류·process 회수를 처리한다. desktop launcher는 기존 비동기 시작 계약을 보존하므로 이후 종료 오류를 사용자에게 소급 반환하지 않는다.

## 결과와 제한

- 기본 실행에 중복 Swift script 또는 사용자 Mac의 Swift interpreter가 필요하지 않다. 명시적 환경변수로 지정한 사용자 script는 호환 지원한다.
- Go 단독 설치에는 bridge가 포함되지 않는다. macOS bridge 기능은 배포 archive 또는 개발용 Swift build 산출물이 필요하다.
- metadata와 ad hoc signature 검증은 Developer ID 배포 서명·notarization·실제 TCC 권한 승인 검증을 대체하지 않는다.
- 로컬 CLT가 Intel compatibility library를 제공하지 않으면 universal build는 실패해야 한다. architecture 검사를 생략하거나 macOS 최소 버전을 올려 통과시키지 않는다.
