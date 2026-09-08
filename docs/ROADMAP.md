# ROADMAP

아래는 현재 기능 설명이 아닌 미구현 또는 별도 검증이 필요한 후속 작업이다. 일정·완료를 약속하는 목록이 아니다.

- Developer ID 서명·notarization·Gatekeeper 및 Homebrew 실제 설치 검증. 현재 archive는 ad hoc signed bridge를 포함하며 해당 검증을 완료했다고 간주하지 않는다.
- 별도 동의된 계정/데이터로 KLAS, Keychain, EventKit, 실제 음성이 있는 Speech의 end-to-end 검증. 현재 자동 검증은 fake HTTP/process와 공유 fixture를 사용한다.
- macOS 12·14·26 각각에서 권한 UX와 실행 호환성 검증. deployment target 및 availability 검사만으로 모든 OS의 실기기 검증을 대신하지 않는다.
- TUI 전용 통합 검색 입력 UX. CLI의 `search`와 TUI 목록·상세/원문 열기는 이미 구현되어 있으므로 미래 기능으로 분류하지 않는다.
- app 내부 KLAS client factory 및 store 계약 타입의 추가 port 분리 필요성 검토. 현재 facade와 테스트된 retry 의미를 보존하는 독립 slice로만 진행한다.
- API 연구 메모의 미확정 endpoint·첨부/제출 흐름을 sanitized fixture로 검증. 연구 기록에 endpoint가 있다는 이유로 현재 CLI 지원 계약으로 간주하지 않는다.
