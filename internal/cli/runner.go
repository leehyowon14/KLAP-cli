package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"io"
	"time"
)

type Runner struct {
	Service *app.Service
	Out     io.Writer
	ErrOut  io.Writer
	Opener  func(string, string) error
	Clock   func() time.Time
}

func (r Runner) normalized() Runner {
	if r.Out == nil {
		r.Out = io.Discard
	}
	if r.ErrOut == nil {
		r.ErrOut = io.Discard
	}
	if r.Clock == nil {
		r.Clock = time.Now
	}
	if r.Opener == nil {
		r.Opener = func(string, string) error { return errors.New("external opener is not configured") }
	}
	return r
}

func (r Runner) Run(ctx context.Context, args []string) error {
	r = r.normalized()
	service := r.Service
	if len(args) == 0 {
		return errors.New("missing CLI command")
	}

	switch args[0] {
	case "auth":
		return runAuth(ctx, service)
	case "user":
		return runUser(ctx, service, args[1:])
	case "dashboard":
		return runDashboard(ctx, service, args[1:])
	case "search":
		return runSearch(ctx, service, args[1:])
	case "due":
		return runDue(ctx, service, args[1:])
	case "cache":
		return runCache(ctx, service, args[1:])
	case "course":
		return runCourse(ctx, service, args[1:])
	case "subject":
		return runSubject(ctx, service, args[1:])
	case "term":
		return runTerm(ctx, service, args[1:])
	case "assignment":
		return r.runAssignment(ctx, service, args[1:])
	case "notice":
		return runNotice(ctx, service, args[1:])
	case "timetable":
		return runTimetable(ctx, service, args[1:])
	case "attendance":
		return runAttendance(ctx, service, args[1:])
	case "grade":
		return runGrade(ctx, service, args[1:])
	case "rank":
		return runRank(ctx, service, args[1:])
	case "evaluation":
		return runEvaluation(ctx, service, args[1:])
	case "syllabus":
		return runSyllabus(ctx, service, args[1:])
	case "room":
		return runRoom(ctx, service, args[1:])
	case "lecture":
		return runLecture(ctx, service, args[1:])
	case "attend":
		return runAttend(ctx, service, args[1:])
	case "academic":
		return runAcademic(ctx, service, args[1:])
	case "config":
		return r.runConfig(ctx, service, args[1:])
	case "help", "-h", "--help":
		r.printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func (r Runner) printHelp() {
	_, _ = fmt.Fprintln(r.Out, `KLAP CLI

Usage:
  klap                   Bubble Tea 기반 TUI 실행
  klap auth              KLAS 로그인 검증 후 계정 저장
  klap user list         저장된 학번 목록 출력
  klap user select <학번> 현재 유저 선택
  klap user rm <학번>    저장된 계정 삭제
  klap dashboard         현재 학기 대시보드 출력
  klap dashboard --refresh 캐시 무시 후 대시보드 갱신
  klap tui               Bubble Tea 기반 TUI 실행
  klap search <키워드>   과목/과제/공지/온라인 강의/학사일정 통합 검색
  klap due              과제/온라인 강의/학사일정 데드라인 출력
  klap cache status      캐시 상태 출력
  klap cache clear       캐시 삭제
  klap term list         수강 학기 목록 출력
  klap term select <학기번호|학기값> 현재 학기 선택
  klap course list       현재 학기 수업 목록 출력
  klap subject search    과목 검색
  klap assignment list   과제 목록 출력
  klap assignment detail <과제ID> 과제 상세 출력
  klap assignment open <과제ID> 과제 원문 열기
  klap assignment remind 과제 마감 reminder 동기화
  klap notice list       강의 공지 목록 출력
  klap notice detail <공지ID> 강의 공지 상세 출력
  klap notice open <공지ID> 강의 공지 원문 열기
  klap timetable         현재 학기 시간표 출력
  klap attendance        출석 현황 출력
  klap attendance detail <과목명|번호|학정번호> 주차별 출석 상세 출력
  klap attendance cdp    CDP 출석내역 출력
  klap grade [학기]      성적 조회
  klap rank [학기]       석차 조회
  klap evaluation list   수업평가 대상 목록 출력
  klap evaluation submit <all|과목명|번호> 수업평가 자동 답변 미리보기
  klap syllabus <과목명|과목번호|학정번호> 강의계획서 출력
  klap room free <강의실명> 강의실 빈 시간 출력
  klap room busy <강의실명> 강의실 사용 목록 출력
  klap room available --day <요일> --duration <교시범위> 조건에 맞는 빈 강의실 출력
  klap room empty --day <요일> --duration <교시범위> room available alias
  klap room index        강의실 시간표 인덱스 생성/출력
  klap room cache clear  강의실 인덱스 캐시 삭제
  klap academic list     학사일정 목록 출력
  klap lecture list      온라인 강의 목록 출력
  klap lecture status    온라인 강의 수강 상태 출력
  klap lecture download <과목명|과목번호|강의ID> 온라인 강의 다운로드
  klap lecture download status 다운로드 폴더 상태 출력
  klap lecture download open 다운로드 폴더 열기
  klap lecture attend <강의ID> 특정 온라인 강의 자동 수강
  klap lecture open <강의ID> 온라인 강의 열기
  klap attend <all|과목명|과목번호> 온라인 강의와 학습활동 자동 수강
  klap config list      전체 설정 출력
  klap config set <key> <value> 설정 변경
  klap config reset     설정 기본값 복원
  klap config reminder  reminder 설정 확인/변경
  klap config download  다운로드 폴더/동시성/절전 방지 설정 확인/변경`)
}
