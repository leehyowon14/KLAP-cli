# internal/app

CLI와 TUI가 공유하는 headless application core다. [Service](service.go)는 compatibility facade이며 [Dependencies](dependencies.go)는 bootstrap이 생성한 store와 adapter를 받는다. `NewService`는 누락 dependency를 검증하고 직접 I/O를 하지 않는다.

## 소유하는 책임

- 선택 사용자·학기 해석, Session 로드와 저장, 만료 시 최대 1회 재로그인/retry
- feature별 정규화 request/result와 cache 정책
- Dashboard sync 및 conflict 결정, Room 다중 요일 조회
- Download/Transcript pipeline과 progress, cancel·join·소유 파일 cleanup
- typed config update와 durable sync-state migration 정책

## 소유하지 않는 책임

- CLI parser/flag·writer·terminal과 Bubble Tea 화면 상태
- HTTP client와 concrete store 생성, bridge 경로 탐색, subprocess 실행
- KLAS wire payload를 presentation 또는 조회 cache에 노출하는 계약

내부 facade에는 KLAS client factory/정규화 adapter 모델과 store 계약 타입 의존이 남아 있다. 모든 import가 순수 domain으로만 구성된 구조는 아니다. 외부 media·academic·download·macOS 구현은 consumer-owned port로 주입한다. workflow의 디렉터리 생성·cleanup 정책은 app에 남는다.

구조와 변경 이유는 [Architecture](../../docs/ARCHITECTURE.md), 각 경계의 후속 검토는 [ROADMAP](../../docs/ROADMAP.md)를 참고한다. 테스트는 fake HTTP/adapter/store를 사용하며 실제 계정이나 Swift 권한 호출을 요구하지 않는다.

```sh
go test ./internal/app
go test -race ./internal/app
```
