# ADR 0002: application과 presentation 책임 경계

상태: 적용됨

## 배경

CLI/TUI에서 store를 생성하거나 KLAS wire 모델을 조립하면 화면 변경이 인증·영속성·외부 API 변경과 결합된다. 다단계 다운로드와 동기화는 화면 전환 후에도 취소·회수 정책이 유지되어야 한다.

## 결정

- [bootstrap](../../internal/bootstrap/run.go)이 실행 모드와 concrete dependency를 조립한다. [NewService](../../internal/app/dependencies.go)는 필수 dependency를 검증하며 I/O를 수행하지 않는다.
- `app.Service`는 compatibility facade로 유지하고 feature별 파일과 app-owned 결과 DTO를 사용한다. 공유되는 Course·Term·Session은 zero-I/O `internal/domain`에 둔다.
- consumer가 사용하는 port를 선언한다. 하나의 거대한 gateway로 모든 endpoint를 묶지 않는다.
- Dashboard sync, Room 다중 요일 조회, Download worker·cancel·cleanup은 app이 소유한다. TUI는 의도와 결과를 표시한다.
- CLI는 registry 하나로 dispatch/help를 구성한다. TUI는 단일 package 안에서 child가 state/update/view를 소유하고 root는 route와 전역 상태를 관리한다.
- [architecture test](../../internal/architecture/dependencies_test.go)는 금지된 **직접** import를 차단한다. `cli -> app -> klas` 같은 합법적인 전이 의존은 금지하지 않는다.

## 결과와 제한

- presentation 테스트는 KLAS raw struct, Keychain, Swift 실행 없이 작성할 수 있다.
- 모든 facade 내부 타입까지 순수 port로 바뀐 것은 아니다. app 내부에는 KLAS client factory와 adapter의 정규화 모델, store 계약 타입에 대한 의존이 남는다. public 결과의 wire payload 노출과 concrete I/O 생성은 차단한다.
- 다운로드 취소 시 삭제 허용 범위는 app의 workflow 정책이다. filesystem API 전체를 기계적으로 adapter로 옮기지 않는다.
- package 수나 특정 파일명을 맞추기 위한 분리는 하지 않는다.
