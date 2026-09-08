package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type commandHandler func(Runner, context.Context, *app.Service, []string) error

// A nil handler documents a bootstrap-owned mode, not a CLI command.
type commandSpec struct {
	Name    string
	Aliases []string
	Run     commandHandler
	Usage   []string
}

func commandRegistry() []commandSpec {
	return []commandSpec{
		{Name: "", Run: nil, Usage: []string{"  klap                   Bubble Tea 기반 TUI 실행"}},
		{Name: "auth", Run: Runner.runAuthCommand, Usage: []string{"  klap auth              KLAS 로그인 검증 후 계정 저장"}},
		{Name: "user", Run: Runner.runUser, Usage: []string{"  klap user list         저장된 학번 목록 출력", "  klap user select <학번> 현재 유저 선택", "  klap user rm <학번>    저장된 계정 삭제"}},
		{Name: "dashboard", Run: Runner.runDashboard, Usage: []string{"  klap dashboard         현재 학기 대시보드 출력", "  klap dashboard --refresh 캐시 무시 후 대시보드 갱신"}},
		{Name: "search", Run: Runner.runSearch, Usage: []string{"  klap search <키워드>   과목/과제/공지/온라인 강의/학사일정 통합 검색"}},
		{Name: "due", Run: Runner.runDue, Usage: []string{"  klap due              과제/온라인 강의/학사일정 데드라인 출력"}},
		{Name: "cache", Run: Runner.runCache, Usage: []string{"  klap cache status      캐시 상태 출력", "  klap cache clear       캐시 삭제"}},
		{Name: "course", Run: Runner.runCourse, Usage: []string{"  klap course list       현재 학기 수업 목록 출력"}},
		{Name: "subject", Run: Runner.runSubject, Usage: []string{"  klap subject search    과목 검색"}},
		{Name: "term", Run: Runner.runTerm, Usage: []string{"  klap term list         수강 학기 목록 출력", "  klap term select <학기번호|학기값> 현재 학기 선택"}},
		{Name: "assignment", Run: Runner.runAssignment, Usage: []string{"  klap assignment list   과제 목록 출력", "  klap assignment detail <과제ID> 과제 상세 출력", "  klap assignment open <과제ID> 과제 원문 열기", "  klap assignment remind 과제 마감 reminder 동기화"}},
		{Name: "notice", Run: Runner.runNotice, Usage: []string{"  klap notice list       강의 공지 목록 출력", "  klap notice detail <공지ID> 강의 공지 상세 출력", "  klap notice open <공지ID> 강의 공지 원문 열기"}},
		{Name: "timetable", Run: Runner.runTimetable, Usage: []string{"  klap timetable         현재 학기 시간표 출력"}},
		{Name: "attendance", Run: Runner.runAttendance, Usage: []string{"  klap attendance        출석 현황 출력", "  klap attendance detail <과목명|번호|학정번호> 주차별 출석 상세 출력", "  klap attendance cdp    CDP 출석내역 출력"}},
		{Name: "grade", Run: Runner.runGrade, Usage: []string{"  klap grade [학기]      성적 조회"}},
		{Name: "rank", Run: Runner.runRank, Usage: []string{"  klap rank [학기]       석차 조회"}},
		{Name: "evaluation", Run: Runner.runEvaluation, Usage: []string{"  klap evaluation list   수업평가 대상 목록 출력", "  klap evaluation submit <all|과목명|번호> 수업평가 자동 답변 미리보기"}},
		{Name: "syllabus", Run: Runner.runSyllabus, Usage: []string{"  klap syllabus <과목명|과목번호|학정번호> 강의계획서 출력"}},
		{Name: "room", Run: Runner.runRoom, Usage: []string{"  klap room free <강의실명> 강의실 빈 시간 출력", "  klap room busy <강의실명> 강의실 사용 목록 출력", "  klap room available --day <요일> --duration <교시범위> 조건에 맞는 빈 강의실 출력", "  klap room empty --day <요일> --duration <교시범위> room available alias", "  klap room index        강의실 시간표 인덱스 생성/출력", "  klap room cache clear  강의실 인덱스 캐시 삭제"}},
		{Name: "lecture", Run: Runner.runLecture, Usage: []string{"  klap lecture list      온라인 강의 목록 출력", "  klap lecture status    온라인 강의 수강 상태 출력", "  klap lecture download <과목명|과목번호|강의ID> 온라인 강의 다운로드", "  klap lecture download status 다운로드 폴더 상태 출력", "  klap lecture download open 다운로드 폴더 열기", "  klap lecture attend <강의ID> 특정 온라인 강의 자동 수강", "  klap lecture open <강의ID> 온라인 강의 열기"}},
		{Name: "attend", Run: Runner.runAttend, Usage: []string{"  klap attend <all|과목명|과목번호> 온라인 강의와 학습활동 자동 수강"}},
		{Name: "academic", Run: Runner.runAcademic, Usage: []string{"  klap academic list     학사일정 목록 출력"}},
		{Name: "config", Run: Runner.runConfig, Usage: []string{"  klap config list      전체 설정 출력", "  klap config set <key> <value> 설정 변경", "  klap config reset     설정 기본값 복원", "  klap config reminder  reminder 설정 확인/변경", "  klap config download  다운로드 폴더/동시성/절전 방지 설정 확인/변경"}},
		{Name: "tui", Run: nil, Usage: []string{"  klap tui               Bubble Tea 기반 TUI 실행"}},
		{Name: "help", Aliases: []string{"-h", "--help"}, Run: Runner.runHelpCommand, Usage: []string{"  klap help              CLI 명령 도움말 출력"}},
	}

}

func dispatchCommand(ctx context.Context, r Runner, args []string, registry []commandSpec) error {
	for _, command := range registry {
		if command.Run != nil && (command.Name == args[0] || containsCommandAlias(command.Aliases, args[0])) {
			return command.Run(r, ctx, r.Service, args[1:])
		}
	}
	return fmt.Errorf("unknown command: %s", args[0])
}

func containsCommandAlias(aliases []string, name string) bool {
	for _, alias := range aliases {
		if alias == name {
			return true
		}
	}
	return false
}

func (r Runner) runAuthCommand(ctx context.Context, service *app.Service, _ []string) error {
	return r.runAuth(ctx, service)
}

func (r Runner) runHelpCommand(_ context.Context, _ *app.Service, _ []string) error {
	r.printHelp()
	return nil
}

func (r Runner) printHelp() {
	_, _ = fmt.Fprintln(r.Out, "KLAP CLI\n\nUsage:")
	for _, command := range commandRegistry() {
		if len(command.Usage) > 0 {
			_, _ = fmt.Fprintln(r.Out, strings.Join(command.Usage, "\n"))
		}
	}
}
