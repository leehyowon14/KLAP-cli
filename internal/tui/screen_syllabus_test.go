package tui

import (
	"context"
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"testing"
)

func TestDashboardSyllabusShortcutOnlyWorksOnCoursePage(t *testing.T) {
	result := app.DashboardResult{
		Term: app.Term{Value: "2026,1", Label: "2026년도 1학기"},
		Courses: []app.DashboardCourse{{
			Index: 1,
			Name:  "컴퓨터그래픽스",
		}},
	}

	summary := model{active: screenDashboard,
		dashboard: dashboardScreenModel{dashboardResult: result}}
	updated, cmd := summary.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	got := updated.(model)
	if got.active != screenDashboard || cmd != nil {
		t.Fatalf("summary shortcut active=%v cmd nil=%t", got.active, cmd == nil)
	}

	course := model{active: screenDashboard,
		dashboard: dashboardScreenModel{dashboardResult: result, dashboardPage: 1}}
	updated, cmd = course.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	got = updated.(model)
	if got.active != screenSyllabus || !got.loading || got.syllabus.syllabusCourseIndex != 1 || cmd == nil {
		t.Fatalf("course shortcut active=%v loading=%t index=%d cmd nil=%t", got.active, got.loading, got.syllabus.syllabusCourseIndex, cmd == nil)
	}
}

func TestDashboardSyllabusShortcutAcceptsKoreanKeyboardKey(t *testing.T) {
	m := model{
		active: screenDashboard,
		dashboard: dashboardScreenModel{dashboardPage: 1,
			dashboardResult: app.DashboardResult{
				Term:    app.Term{Value: "2026,1"},
				Courses: []app.DashboardCourse{{Index: 1, Name: "컴퓨터그래픽스"}},
			}},
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ㅔ")})
	got := updated.(model)
	if got.active != screenSyllabus || cmd == nil {
		t.Fatalf("korean shortcut active=%v cmd nil=%t", got.active, cmd == nil)
	}
}

func TestSyllabusBackReturnsToSelectedDashboardCourse(t *testing.T) {
	m := model{active: screenSyllabus,
		dashboard: dashboardScreenModel{dashboardPage: 2},
		syllabus:  syllabusScreenModel{syllabusCursor: 3},
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(model)
	if got.active != screenDashboard || got.dashboard.dashboardPage != 2 || got.syllabus.syllabusCursor != 0 {
		t.Fatalf("back active=%v dashboardPage=%d syllabusCursor=%d", got.active, got.dashboard.dashboardPage, got.syllabus.syllabusCursor)
	}
}

func TestSyllabusLinesShowCoursePlan(t *testing.T) {
	result := app.SyllabusResult{
		Term:      app.Term{Value: "2026,1", Label: "2026년도 1학기"},
		SubjectID: "U202613951I040013",
		Course:    app.Course{Name: "컴퓨터그래픽스"},
		Syllabus: app.Syllabus{
			CourseCode:     "I040-3-3951-01",
			FullName:       "컴퓨터그래픽스",
			CourseType:     "전공선택",
			Credits:        "3",
			CurrentNum:     "42",
			Professor:      "김교수",
			ProfessorTitle: "교수",
			Competency:     "창의융합",
			Summary:        "그래픽스의 기본 원리를 학습한다.",
			Purpose:        "렌더링 파이프라인을 이해한다.",
			BookName:       "Computer Graphics",
			Times:          []app.SyllabusTime{{Weekday: "월", Periods: []int{1, 2}, Room: "새빛관 101"}},
			Evaluation:     app.SyllabusEvaluation{Attendance: 10, Midterm: 30, Final: 30, Report: 30},
			Schedule:       []app.SyllabusWeek{{Week: 1, Topic: "그래픽스 개요", SubNote: "실습 환경 구성"}},
		},
	}
	lines := syllabusLines(result, 96)
	view := strings.Join(lines, "\n")
	for _, want := range []string{"컴퓨터그래픽스", "I040-3-3951-01", "김교수", "월 1,2교시", "수강인원: 42명 (A: 16명, B: 33명)", "개요", "학습목표", "평가", "1주차", "그래픽스 개요"} {
		if !strings.Contains(view, want) {
			t.Fatalf("syllabusLines() missing %q: %q", want, view)
		}
	}
	for index, line := range lines {
		if strings.HasPrefix(line, "대표역량") {
			if index+1 >= len(lines) || lines[index+1] != "수강인원: 42명 (A: 16명, B: 33명)" {
				t.Fatalf("enrollment line is not directly below competency: %+v", lines)
			}
			return
		}
	}
	t.Fatal("competency line is missing")
}

func TestFormatSyllabusEnrollment(t *testing.T) {
	tests := []struct {
		name    string
		current string
		want    string
	}{
		{name: "zero", current: "0", want: "수강인원: 0명 (A: 0명, B: 0명)"},
		{name: "floor fractional quota", current: "42", want: "수강인원: 42명 (A: 16명, B: 33명)"},
		{name: "trim whitespace", current: " 50 ", want: "수강인원: 50명 (A: 20명, B: 40명)"},
		{name: "missing", current: "", want: "수강인원: 확인 필요"},
		{name: "invalid", current: "unknown", want: "수강인원: 확인 필요"},
		{name: "negative", current: "-1", want: "수강인원: 확인 필요"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatSyllabusEnrollment(tt.current); got != tt.want {
				t.Fatalf("formatSyllabusEnrollment(%q) = %q, want %q", tt.current, got, tt.want)
			}
		})
	}
}

type syllabusScreenStub struct {
	calls   int
	ctx     context.Context
	options app.SyllabusOptions
	err     error
}

func (s *syllabusScreenStub) Syllabus(ctx context.Context, options app.SyllabusOptions) (app.SyllabusResult, error) {
	s.calls++
	s.ctx = ctx
	s.options = options
	return app.SyllabusResult{SubjectID: "subject"}, s.err
}
func TestSyllabusChildLoadAndRefreshDeferIO(t *testing.T) {
	ctx := context.Background()
	service := &syllabusScreenStub{err: errors.New("syllabus failed")}
	child := syllabusScreenModel{}
	child.Start(7)
	child.syllabusCursor = 9
	action, handled := child.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ㄱ")}, 80, 20, true, ctx, service, "2026,1")
	if !handled || !action.setLoading || !action.loading || !action.setError || action.cmd == nil || service.calls != 0 || child.syllabusCursor != 0 {
		t.Fatal("refresh did not defer IO/reset cursor")
	}
	result := action.cmd().(syllabusMsg)
	if service.calls != 1 || service.ctx != ctx || service.options.Selector != "7" || service.options.TermValue != "2026,1" || !errors.Is(result.err, service.err) {
		t.Fatal("query contract changed")
	}
	child.Loaded(result.result)
	if child.syllabusResult.SubjectID != "subject" {
		t.Fatal("loaded state lost")
	}
	child.Start(2)
	if child.syllabusResult.SubjectID != "" || child.syllabusCourseIndex != 2 {
		t.Fatal("new request retained old result")
	}
}
func TestSyllabusChildScrollAndStaleRootMessage(t *testing.T) {
	child := syllabusScreenModel{syllabusResult: app.SyllabusResult{SubjectID: "id", Syllabus: app.Syllabus{Summary: strings.Repeat("line\n", 40)}}}
	for i := 0; i < 100; i++ {
		child.Update(tea.KeyMsg{Type: tea.KeyDown}, 80, 14, false, context.Background(), nil, "")
	}
	bottom := child.syllabusCursor
	if bottom <= 0 {
		t.Fatal("scroll did not advance")
	}
	child.Update(tea.KeyMsg{Type: tea.KeyUp}, 80, 14, true, context.Background(), nil, "")
	if child.syllabusCursor != bottom {
		t.Fatal("loading allowed scrolling")
	}
	for i := 0; i < 100; i++ {
		child.Update(tea.KeyMsg{Type: tea.KeyUp}, 80, 14, false, context.Background(), nil, "")
	}
	if child.syllabusCursor != 0 {
		t.Fatal("scroll lower clamp changed")
	}
	m := model{active: screenDashboard, syllabus: child}
	updated, cmd := m.Update(syllabusMsg{result: app.SyllabusResult{SubjectID: "stale"}})
	if updated.(model).syllabus.syllabusResult.SubjectID != "id" || cmd != nil {
		t.Fatal("stale result clobbered inactive child")
	}
}
