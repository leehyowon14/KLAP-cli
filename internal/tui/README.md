# internal/tui

현재 동작하는 Bubble Tea presentation이다. 인자 없는 `klap` 또는 `klap tui`로 실행한다. 계정 검증과 데이터·동기화·다운로드 작업은 `internal/app`을 호출하며 KLAS/account/settings/platform 구현을 직접 import하지 않는다.

## 실행과 상태 소유권

- [root.go](root.go): context, route, 전역 상태와 child 보유; 외부 opener는 bootstrap에서 주입한다.
- [router.go](router.go), [child.go](child.go): 화면 이동과 active child 위임.
- [prefetch.go](prefetch.go): 전역 데이터 로딩과 stale/background 응답 처리.
- [theme.go](theme.go), [layout.go](layout.go), [keys.go](keys.go): 공통 표현과 키 처리.
- 각 `screen_*.go`: 해당 화면 state/update/view. 모든 service I/O는 `tea.Cmd` 안에서 수행한다.

## 현재 화면

| 흐름 | 구현 진입점 |
| --- | --- |
| 저장 계정 확인·로그인 | [screen_auth.go](screen_auth.go) |
| Home 메뉴와 선택 | [screen_home.go](screen_home.go) |
| Dashboard와 과목별 Syllabus | [screen_dashboard.go](screen_dashboard.go), [screen_syllabus.go](screen_syllabus.go) |
| 과제 목록·상세 | [screen_assignments.go](screen_assignments.go) |
| 공지 목록·상세 | [screen_notices.go](screen_notices.go) |
| Due 통합 마감 | [screen_due.go](screen_due.go) |
| 월별 학사일정 | [screen_academic.go](screen_academic.go) |
| 강의 목록·상태 | [screen_lectures.go](screen_lectures.go) |
| 사용자·학기·동기화·다운로드 설정 편집 | [screen_config.go](screen_config.go) |
| Rooms 요일·교시·조회 결과 | [screen_room.go](screen_room.go) |
| 동기화 충돌 keep/apply 선택 | [screen_sync.go](screen_sync.go) |
| Download 선택·전사 언어·진행·취소 | [screen_download.go](screen_download.go) |
| Attend 확인·진행·취소 | [screen_attend.go](screen_attend.go) |
| 선택된 KLAS 원문 열기 | [external_url.go](external_url.go) |

공통 course-paged state는 [course_pager.go](course_pager.go)를 사용한다. 화면마다 별도 package를 만들지 않는다.

## 다단계 작업

Download는 강의 선택 → 전사 여부 → 언어 선택 → 준비 → 진행 순서다. 준비 응답에는 generation을 적용하고 background run 메시지는 전역 lifecycle 경계에서 처리한다. worker queue, 종료 대기, 파일 cleanup은 app 소유이며 TUI가 파일을 삭제하지 않는다.

Config는 비동기 저장 완료 전 pending 상태를 유지하고 stale 완료를 무시한다. 저장 실패 시 편집 내용을 보존한다. Room 다중 요일 조회와 Dashboard sync 순서도 app workflow에 위임한다.

이전 화면 복귀·진행 취소·Home 이동은 각 child의 키 정책을 따른다. 사용자용 키와 실행 예시는 [루트 README](../../README.md#tui-화면과-실행-흐름)를 참고한다. TUI 전용 통합 검색 UX 같은 후속 목표는 [ROADMAP](../../docs/ROADMAP.md)에만 기록한다.

## 테스트

```sh
go test ./internal/tui
go test -race ./internal/tui ./internal/app
```

child 테스트는 fake service로 I/O 지연 실행, loading/error, 선택·back, stale 응답을 검사한다. Root 테스트는 route/global message와 진행 화면 중 background 응답 처리를 검사한다. 실제 KLAS 로그인·EventKit 권한·브라우저 실행 없이 수행한다.
