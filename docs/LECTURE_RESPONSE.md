# 강의 목록 정규화 검증

2026-09-29 로그인 세션으로 과목 컨텍스트를 전환한 뒤
`SelectOnlineCntntsStdList.do`를 읽어 확인했다. 수강/진도 저장 API는 호출하지 않았다.
6개 과목 응답 중 2개에 총 15개 항목이 있었다. 개인 식별자와 제목은 기록하지 않는다.

| 응답 필드 | 타입 | 정규화 |
|---|---|---|
| firstEdu | string 또는 null | FirstStartedAt, KST 첫 학습 시각 |
| firstEnd | string 또는 null | FirstCompletedAt, KST 첫 완료 시각 |
| achivTime | 숫자 문자열 | AchievedTime 우선 사용 |
| rcognTime | 숫자 문자열 | RequiredTime 우선 사용 |
| prog | number | Progress |
| startDate / endDate | string | 수강 가능 기간, 실제 학습 시각과 별개 |

날짜 형식은 `yyyy-MM-dd HH:mm`이었다. 빈 값, null, 잘못된 날짜는 nil로 유지한다.
진도 100인 응답에서 firstEdu와 firstEnd가 함께 확인됐고,
진도 0인 응답에는 둘 다 null이거나 firstEdu만 있었다.
실제 지각 완료 샘플은 이번 조회에 없었다. 지각 표시는 전달된 시각과 기한을 비교한
앱의 분류이며 학교의 공식 출석 인정 결과를 의미하지 않는다.

미수강 샘플의 값은 `prog=0, achivTime="0", rcognTime="60",
ptime="50", totalTime="60"`이었다. 목록의 totalTime을 누적 수강 시간으로
사용하면 안 된다. achivTime → learnTime → totAchivTime 순서로 사용하고,
필드가 모두 없으면 알 수 없는 값으로 유지한다. 문자열 "0"도 유효한 값이다.

UpdateProgress의 totalTime은 별도 API의 진행 응답이므로 기존 파싱을 유지한다.
세션 만료, 컨텍스트 실패, 응답 구조 오류는 기존 오류 경로를 따른다.
