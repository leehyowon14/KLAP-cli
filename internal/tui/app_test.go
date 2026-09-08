package tui

import (
	"context"
	"errors"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"github.com/leehyowon14/KLAP-cli/internal/testsupport"
	"reflect"
	"strings"
	"testing"
	"time"
)

func newTUITestService(t *testing.T) *app.Service {
	t.Helper()
	return testsupport.NewService(t)
}

func TestEnterScreenUsesPrefetchedData(t *testing.T) {
	m := model{
		loadedScreens: map[screen]bool{screenAssignments: true},
		assignments: assignmentScreenModel{assignmentRows: []app.AssignmentRow{
			{ID: "1", CourseName: "오픈소스소프트웨어실습"},
		}},
	}
	updated, cmd := m.enterScreen(screenAssignments)
	got := updated.(model)
	if cmd != nil {
		t.Fatal("enterScreen() returned load command for prefetched screen")
	}
	if got.active != screenAssignments || got.loading {
		t.Fatalf("active=%v loading=%t", got.active, got.loading)
	}
	if len(got.assignments.assignmentRows) != 1 {
		t.Fatalf("assignmentRows = %+v", got.assignments.assignmentRows)
	}
}

func TestEnterScreenShowsLoadingForPendingPrefetch(t *testing.T) {
	m := model{
		loadingScreens: map[screen]bool{screenLectures: true},
	}
	updated, cmd := m.enterScreen(screenLectures)
	got := updated.(model)
	if cmd != nil {
		t.Fatal("enterScreen() started duplicate load for pending prefetch")
	}
	if got.active != screenLectures || !got.loading {
		t.Fatalf("active=%v loading=%t", got.active, got.loading)
	}
}

func TestInactiveLoadMsgCachesWithoutClobberingOtherScreens(t *testing.T) {
	m := model{
		active: screenHome,
		assignments: assignmentScreenModel{assignmentRows: []app.AssignmentRow{
			{ID: "1", CourseName: "컴퓨터그래픽스"},
		}},
	}
	m.applyLoadMsg(loadMsg{
		screen: screenLectures,
		lectures: []app.LectureRow{
			{ID: "lecture-1", CourseName: "오픈소스소프트웨어실습"},
		},
	})
	if !m.loadedScreens[screenLectures] || m.loading {
		t.Fatalf("loadedScreens=%+v loading=%t", m.loadedScreens, m.loading)
	}
	if len(m.lectureRows) != 1 {
		t.Fatalf("lectureRows = %+v", m.lectureRows)
	}
	if len(m.assignments.assignmentRows) != 1 || m.assignments.assignmentRows[0].ID != "1" {
		t.Fatalf("assignmentRows clobbered: %+v", m.assignments.assignmentRows)
	}
}

func TestPreparePrefetchStartsOnlyFirstScreen(t *testing.T) {
	m := (model{loadingScreens: map[screen]bool{}}).preparePrefetch([]screen{screenAssignments, screenLectures, screenDashboard})
	if !m.prefetchActive || m.prefetchCurrent != screenAssignments {
		t.Fatalf("prefetch active=%t current=%v", m.prefetchActive, m.prefetchCurrent)
	}
	if !m.loadingScreens[screenAssignments] || m.loadingScreens[screenLectures] || m.loadingScreens[screenDashboard] {
		t.Fatalf("loadingScreens = %+v", m.loadingScreens)
	}
	if len(m.prefetchQueue) != 2 || m.prefetchQueue[0] != screenLectures || m.prefetchQueue[1] != screenDashboard {
		t.Fatalf("prefetchQueue = %+v", m.prefetchQueue)
	}
}

func TestPrefetchLoadMsgStartsNextQueuedScreen(t *testing.T) {
	m := model{
		active:          screenHome,
		loadedScreens:   map[screen]bool{},
		loadingScreens:  map[screen]bool{screenAssignments: true},
		screenErrors:    map[screen]error{},
		prefetchActive:  true,
		prefetchCurrent: screenAssignments,
		prefetchQueue:   []screen{screenLectures},
	}
	updated, cmd := m.Update(loadMsg{screen: screenAssignments, prefetch: true,
		assignments: []app.AssignmentRow{{ID: "1"}}})
	got := updated.(model)
	if cmd == nil {
		t.Fatal("next prefetch command is nil")
	}
	if !got.loadedScreens[screenAssignments] || got.loadingScreens[screenAssignments] {
		t.Fatalf("assignments loaded/loading = %+v/%+v", got.loadedScreens, got.loadingScreens)
	}
	if !got.prefetchActive || got.prefetchCurrent != screenLectures || !got.loadingScreens[screenLectures] {
		t.Fatalf("next prefetch active=%t current=%v loading=%+v", got.prefetchActive, got.prefetchCurrent, got.loadingScreens)
	}
}

func TestPrefetchErrorIsStoredAsLoadedError(t *testing.T) {
	errBoom := errors.New("boom")
	m := model{}
	m.markScreenLoaded(screenLectures, errBoom)
	if !m.isScreenLoaded(screenLectures) {
		t.Fatal("failed screen should be considered loaded")
	}
	if got := m.screenError(screenLectures); got == nil || got.Error() != "boom" {
		t.Fatalf("screenError() = %v", got)
	}

	updated, cmd := m.enterScreen(screenLectures)
	got := updated.(model)
	if cmd != nil || got.loading || got.err == nil {
		t.Fatalf("enter failed screen cmd=%v loading=%t err=%v", cmd, got.loading, got.err)
	}
}

func TestDetailShortcutsAcceptKoreanKeyboardKeys(t *testing.T) {
	m := model{active: screenDashboard}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ㅠ")})
	got := updated.(model)
	if got.active != screenHome {
		t.Fatalf("active after korean b key = %v", got.active)
	}

	m = model{active: screenDashboard}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ㄱ")})
	got = updated.(model)
	if !got.loading || cmd == nil {
		t.Fatalf("refresh after korean r key loading=%t cmd nil=%t", got.loading, cmd == nil)
	}
}

func TestListFormatsHideInternalIDs(t *testing.T) {
	due := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)

	assignments := formatAssignments([]app.AssignmentRow{{
		ID:         "7:7",
		CourseName: "오픈소스소프트웨어실습",
		Assignment: app.Assignment{
			Title: "기말고사 대체 과제",
			DueAt: &due,
		},
	}})
	if strings.Contains(assignments, "7:7") {
		t.Fatalf("formatAssignments() leaked internal id: %q", assignments)
	}

	notices := formatNotices([]app.NoticeRow{{
		ID:         "2:1151742:1",
		CourseName: "창의설계입문",
		Notice: app.Notice{
			Title:      "최종 발표 일정 안내",
			Registered: &due,
		},
	}})
	if strings.Contains(notices, "2:1151742:1") {
		t.Fatalf("formatNotices() leaked internal id: %q", notices)
	}

	lectures := formatLectures([]app.LectureRow{{
		ID:         "1:6a0ebcc046111",
		CourseName: "진로탐색및설계",
		Lecture: app.Lecture{
			Title:       "최신 면접 따라잡기",
			ModuleTitle: "1주차",
			ContentID:   "6a0ebcc046111",
			Progress:    "10",
		},
	}})
	if strings.Contains(lectures, "1:6a0ebcc046111") {
		t.Fatalf("formatLectures() leaked internal id: %q", lectures)
	}
}

func TestDetailLinesIncludeKlasURLAndBody(t *testing.T) {
	due := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	lines := assignmentDetailLines(app.AssignmentDetailResult{
		ID:         "7:1",
		CourseName: "오픈소스소프트웨어실습",
		DetailURL:  "https://klas.kw.ac.kr/assignment",
		Detail: app.AssignmentDetail{
			Title:       "기말고사 대체 과제",
			ContentText: "GitHub repository 주소",
			DueAt:       &due,
		},
	}, 96)
	view := strings.Join(lines, "\n")
	for _, want := range []string{"기말고사 대체 과제", "오픈소스소프트웨어실습", "https://klas.kw.ac.kr/assignment", "GitHub repository 주소"} {
		if !strings.Contains(view, want) {
			t.Fatalf("assignmentDetailLines() missing %q: %q", want, view)
		}
	}
}

func TestLectureContentGroupsAlignsTitleColumn(t *testing.T) {
	groups := lectureContentGroups([]app.LectureRow{
		{
			CourseName: "Gen-AI",
			Lecture: app.Lecture{
				Progress:    "100",
				ContentID:   "a",
				ModuleTitle: "Basics of Python I",
				Title:       "Python Basics I",
			},
		},
		{
			CourseName: "Gen-AI",
			Lecture: app.Lecture{
				Progress:    "100",
				ContentID:   "b",
				ModuleTitle: "Basics of Python II / Goal of Data Science and Data Storytelling",
				Title:       "Python_Basics_II",
			},
		},
	}, 80)

	if len(groups) != 1 || len(groups[0].lines) != 2 {
		t.Fatalf("lectureContentGroups() = %+v", groups)
	}
	firstTitleAt := strings.Index(groups[0].lines[0], "Python Basics I")
	secondTitleAt := strings.Index(groups[0].lines[1], "Python_Basics_II")
	if firstTitleAt < 0 || secondTitleAt < 0 {
		t.Fatalf("title not found: %q / %q", groups[0].lines[0], groups[0].lines[1])
	}
	firstTitleWidth := lipgloss.Width(groups[0].lines[0][:firstTitleAt])
	secondTitleWidth := lipgloss.Width(groups[0].lines[1][:secondTitleAt])
	if firstTitleWidth != secondTitleWidth {
		t.Fatalf("title columns not aligned: %q / %q", groups[0].lines[0], groups[0].lines[1])
	}
	if strings.Contains(groups[0].lines[1], "\n") {
		t.Fatalf("lecture line contains newline: %q", groups[0].lines[1])
	}
}

func TestLectureCompletedDetectsPercentAndMinuteProgress(t *testing.T) {
	if !lectureCompleted(app.Lecture{ContentID: "content", Progress: "100"}) {
		t.Fatal("content lecture with 100% progress should be completed")
	}
	if lectureCompleted(app.Lecture{ContentID: "content", Progress: "99"}) {
		t.Fatal("content lecture below 100% should not be completed")
	}
	if !lectureCompleted(app.Lecture{AchievedTime: "10", RequiredTime: "10"}) {
		t.Fatal("minute based lecture with achieved >= required should be completed")
	}
	if lectureCompleted(app.Lecture{AchievedTime: "9", RequiredTime: "10"}) {
		t.Fatal("minute based lecture below required time should not be completed")
	}
}

func TestDuePageLinesSplitByKind(t *testing.T) {
	dueAt := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	result := app.DueResult{Items: []app.DueItem{
		{Kind: "과제", CourseName: "컴퓨터그래픽스", Title: "과제1", DueAt: dueAt},
		{Kind: "온라인 강의", CourseName: "오픈소스", Title: "HuggingFace", DueAt: dueAt},
		{Kind: "학사일정", Title: "종강", DueAt: dueAt},
	}}

	summary := strings.Join(duePageLines(result, 0, 96), "\n")
	if !strings.Contains(summary, "OVERVIEW") || !strings.Contains(summary, "FOCUS") || !strings.Contains(summary, "3 items") || !strings.Contains(summary, "HuggingFace") {
		t.Fatalf("summary lines = %q", summary)
	}
	assignments := strings.Join(duePageLines(result, 1, 96), "\n")
	if !strings.Contains(assignments, "과제1") || strings.Contains(assignments, "HuggingFace") {
		t.Fatalf("assignment due lines = %q", assignments)
	}
	academic := strings.Join(duePageLines(result, 3, 96), "\n")
	if !strings.Contains(academic, "종강") || strings.Contains(academic, "과제1") {
		t.Fatalf("academic due lines = %q", academic)
	}
}

func TestAcademicCalendarRendersMultiDayEvents(t *testing.T) {
	result := app.AcademicListResult{
		Year: "2026",
		Events: []app.AcademicEvent{{
			Year:  "2026",
			Month: "6월",
			Date:  "06.22(월) ~ 06.24(수)",
			Title: "보강주간",
		}},
	}
	view := renderAcademicMonthCalendar(result, 6, 96, result.Events[0])
	for _, want := range []string{"월  화  수", "22", "23", "24"} {
		if !strings.Contains(view, want) {
			t.Fatalf("renderAcademicMonthCalendar() missing %q: %q", want, view)
		}
	}
	list := renderAcademicEventList(academicMonthEvents(result, 6), 0, 96, 8)
	if !strings.Contains(list, "22일-24일  보강주간") {
		t.Fatalf("renderAcademicEventList() should collapse range: %q", list)
	}
}

func TestAcademicCalendarRendersMonthEvents(t *testing.T) {
	result := app.AcademicListResult{
		Year: "2026",
		Events: []app.AcademicEvent{
			{Year: "2026", Month: "3월", Date: "3(화)", Title: "개강"},
			{Year: "2026", Month: "4월", Date: "1(수)", Title: "다른 달"},
		},
	}

	view := renderAcademicEventList(academicMonthEvents(result, 3), 0, 96, 8)
	if !strings.Contains(view, "개강") || strings.Contains(view, "다른 달") {
		t.Fatalf("renderAcademicEventList() = %q", view)
	}
}

func TestFormatSyncStatus(t *testing.T) {
	got := formatReminderSyncStatus("과제", app.ReminderSyncResult{EligibleCount: 2})
	if !strings.Contains(got, "과제") || !strings.Contains(got, "생성 0") {
		t.Fatalf("formatReminderSyncStatus() = %q", got)
	}
	calendar := formatCalendarSyncStatus("학사일정", app.CalendarSyncResult{})
	if calendar != "학사일정 대상 없음" {
		t.Fatalf("formatCalendarSyncStatus() = %q", calendar)
	}
}

func TestFooterUsesKoreanSyncLabel(t *testing.T) {
	m := model{active: screenDashboard}
	footer := m.footerHelp()
	if strings.Contains(footer, "sync") || !strings.Contains(footer, "동기화") {
		t.Fatalf("footerHelp() = %q", footer)
	}
	if strings.Contains(footer, "←→") || !strings.Contains(footer, "←/→") {
		t.Fatalf("footerHelp() should use separated arrows: %q", footer)
	}
}

func testChromeModel(active screen) model {
	dueAt := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	m := model{
		active: active,
		width:  80,
		height: 24,

		dueResult: app.DueResult{
			From:  time.Date(2026, 6, 10, 0, 0, 0, 0, time.Local),
			Until: time.Date(2026, 6, 24, 0, 0, 0, 0, time.Local),
		},

		lectureRows: []app.LectureRow{{
			ID:         "1",
			CourseName: "강의",
			Lecture: app.Lecture{
				Title:        "영상",
				Progress:     "100",
				AchievedTime: "10",
				RequiredTime: "10",
			},
		}},
		academicResult: app.AcademicListResult{
			Year: "2026",
			Events: []app.AcademicEvent{{
				Year:  "2026",
				Month: "6월",
				Date:  "6.17(수)",
				Title: "종강",
			}},
		},
		academicMonth: 6,

		configSettings: app.ConfigSettings{
			Reminder: app.ReminderSettings{ListName: "Kwangwoon Univ.", AlarmBeforeMin: 1440},
			Calendar: app.CalendarSettings{Name: "학사일정", TimetableName: "시간표"},
			Download: app.DownloadSettings{
				Dir:         "downloads",
				Concurrency: 3,
				Caffeinate:  true,
			},
			Transcript: app.TranscriptSettings{Concurrency: 1},
		},
		configOptions:       app.CategoryOptions{Reminders: []string{"Kwangwoon Univ."}, Calendars: []string{"학사일정", "시간표"}},
		configChoiceKey:     "calendar.name",
		configInput:         textinput.New(),
		roomDaysSelected:    map[int]bool{1: true},
		roomPeriodsSelected: map[int]bool{1: true},
		syncConflicts: []app.SyncConflict{{
			Key:     "assignment:1",
			Scope:   "assignment",
			Title:   "기말 과제",
			Summary: "기말 과제 · 2026-06-17 23:59",
		}},
		syncConflictActions: map[string]app.SyncDecision{"assignment:1": app.SyncDecisionKeep},
		assignments: assignmentScreenModel{assignmentRows: []app.AssignmentRow{{
			ID:         "1",
			CourseName: "강의",
			Assignment: app.Assignment{
				Title: "과제",
				DueAt: &dueAt,
			},
		}},

			assignmentDetail: app.AssignmentDetailResult{
				ID:         "1",
				CourseName: "강의",
				DetailURL:  "https://klas.kw.ac.kr",
				Detail:     app.AssignmentDetail{Title: "과제", DueAt: &dueAt, ContentText: "본문"},
			}},
		notices: noticeScreenModel{noticeRows: []app.NoticeRow{{
			ID:         "1",
			CourseName: "강의",
			Notice: app.Notice{
				Title:      "공지",
				Registered: &dueAt,
			},
		}},

			noticeDetail: app.NoticeDetailResult{
				ID:         "1",
				CourseName: "강의",
				DetailURL:  "https://klas.kw.ac.kr",
				Detail:     app.NoticeDetail{Title: "공지", Registered: &dueAt, ContentText: "본문"},
			}},
		dashboard: dashboardScreenModel{dashboardResult: app.DashboardResult{
			Term: app.Term{Value: "2026,1", Label: "2026년도 1학기"},
			Courses: []app.DashboardCourse{{
				Index: 1,
				Name:  "강의",
			}},
		}},
	}
	m.configInput.SetValue("입력값")
	return m
}

func assertChromeInvariant(t *testing.T, name string, view string, terminalWidth int) {
	t.Helper()
	lines := strings.Split(view, "\n")
	first := ""
	second := ""
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if first == "" {
			first = line
			continue
		}
		second = line
		break
	}
	if !strings.Contains(first, "KLAP") {
		t.Fatalf("%s first visible line should be header, got %q", name, first)
	}
	if strings.Contains(first, "─") || !strings.Contains(second, "─") {
		t.Fatalf("%s rule order failed: first=%q second=%q", name, first, second)
	}
	for index, line := range lines {
		if width := lipgloss.Width(line); width > terminalWidth {
			t.Fatalf("%s line %d width = %d > %d: %q", name, index, width, terminalWidth, line)
		}
	}
}

func TestSyncPanelHidesDashboardStatus(t *testing.T) {
	m := model{
		active:     screenDashboard,
		syncPhase:  "done",
		syncStatus: "동기화 완료: 과제 생성 1",
		dashboard: dashboardScreenModel{dashboardResult: app.DashboardResult{
			Term: app.Term{Value: "2026,1", Label: "2026년도 1학기"},
		}},
	}
	view := m.renderPanel(96)
	if !strings.Contains(view, "DONE") || !strings.Contains(view, "과제 생성 1") {
		t.Fatalf("renderPanel() missing sync result: %q", view)
	}
	if strings.Contains(view, "OVERVIEW") {
		t.Fatalf("renderPanel() should hide dashboard while sync panel is active: %q", view)
	}
}

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
	if got.active != screenSyllabus || !got.loading || got.syllabusCourseIndex != 1 || cmd == nil {
		t.Fatalf("course shortcut active=%v loading=%t index=%d cmd nil=%t", got.active, got.loading, got.syllabusCourseIndex, cmd == nil)
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
	m := model{active: screenSyllabus, syllabusCursor: 3,
		dashboard: dashboardScreenModel{dashboardPage: 2}}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(model)
	if got.active != screenDashboard || got.dashboard.dashboardPage != 2 || got.syllabusCursor != 0 {
		t.Fatalf("back active=%v dashboardPage=%d syllabusCursor=%d", got.active, got.dashboard.dashboardPage, got.syllabusCursor)
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

func TestLectureDownloadFormatting(t *testing.T) {
	row := app.LectureRow{
		CourseName: "오픈소스소프트웨어실습",
		Lecture: app.Lecture{
			ModuleTitle: "14주차",
			Title:       "기말 보강 영상",
		},
	}
	if got := lectureDownloadLabel(row); got != "오픈소스소프트웨어실습 · 14주차 · 기말 보강 영상" {
		t.Fatalf("lectureDownloadLabel() = %q", got)
	}
	progress := app.LectureDownloadProgress{Bytes: 512, TotalBytes: 1024}
	if got := lectureDownloadPercent(progress); got != 0.5 {
		t.Fatalf("lectureDownloadPercent() = %f", got)
	}
	if got := formatDownloadBytes(1536); got != "1.5 KB" {
		t.Fatalf("formatDownloadBytes() = %q", got)
	}
	if got := downloadStageLabel("transcribe"); got != "전사중" {
		t.Fatalf("downloadStageLabel(transcribe) = %q", got)
	}
}

func TestLectureDownloadProgressListUsesProgressRows(t *testing.T) {
	rows := []app.LectureRow{
		{ID: "1:a", CourseName: "컴퓨터그래픽스", Lecture: app.Lecture{ContentID: "a", ModuleTitle: "1주차", Title: "소개"}},
		{ID: "1:b", CourseName: "컴퓨터그래픽스", Lecture: app.Lecture{ContentID: "b", ModuleTitle: "2주차", Title: "렌더링"}},
	}
	m := lectureDownloadModel{
		width:  96,
		height: 24,
		items:  initialDownloadStatusLines(rows),
	}
	view := m.renderProgressList(96)
	if strings.Contains(view, "QUEUE") {
		t.Fatalf("renderProgressList() leaked QUEUE header: %q", view)
	}
	if !strings.Contains(view, "컴퓨터그래픽스 · 1주차 · 소개") || !strings.Contains(view, "░") {
		t.Fatalf("renderProgressList() missing progress rows: %q", view)
	}
}

func TestLectureDownloadTranscriptStatusPreservesDoneOverwrite(t *testing.T) {
	row := app.LectureRow{ID: "1:a", CourseName: "컴퓨터그래픽스", Lecture: app.Lecture{ContentID: "a", Title: "소개"}}
	m := lectureDownloadModel{items: initialDownloadStatusLines([]app.LectureRow{row})}
	m.upsertTranscriptStatusLine(app.LectureTranscriptProgress{Lecture: row, Stage: "transcribe", OutputPath: "lecture.txt", Progress: 0.42})
	m.upsertStatusLine(app.LectureDownloadProgress{Lecture: row, Stage: "done", Path: "lecture.mp4", Bytes: 10, TotalBytes: 10})
	if got := m.items[0].status; got != "transcribe" {
		t.Fatalf("status = %q, want transcribe", got)
	}
	if got := itemProgressPercent(m.items[0]); got != 0.42 {
		t.Fatalf("transcript itemProgressPercent() = %f", got)
	}
	if got := itemProgressPercent(downloadStatusLine{status: "download", bytes: 5, total: 10}); got != 0.5 {
		t.Fatalf("download itemProgressPercent() = %f", got)
	}
}

func TestLectureDownloadTranscribedTextOmitsPath(t *testing.T) {
	m := lectureDownloadModel{}
	got := m.itemProgressText(downloadStatusLine{status: "transcribed", path: "/tmp/lecture.txt"})
	if !strings.Contains(got, "전사완료") || strings.Contains(got, "lecture.txt") {
		t.Fatalf("itemProgressText(transcribed) = %q", got)
	}
}

func TestLectureDownloadProgressCursorWraps(t *testing.T) {
	rows := []app.LectureRow{
		{ID: "1:a", CourseName: "A", Lecture: app.Lecture{ContentID: "a"}},
		{ID: "1:b", CourseName: "A", Lecture: app.Lecture{ContentID: "b"}},
	}
	m := lectureDownloadModel{items: initialDownloadStatusLines(rows)}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	got := updated.(lectureDownloadModel)
	if got.cursor != 1 {
		t.Fatalf("up from top cursor = %d, want 1", got.cursor)
	}
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyDown})
	got = updated.(lectureDownloadModel)
	if got.cursor != 0 {
		t.Fatalf("down from bottom cursor = %d, want 0", got.cursor)
	}
}

func TestLectureAttendShortcutOpensConfirmation(t *testing.T) {
	m := model{
		active: screenLectures,
		width:  96,
		lectureRows: []app.LectureRow{{
			ID:         "1:video",
			CourseName: "운영체제",
			Lecture: app.Lecture{
				ContentID: "video",
				Title:     "프로세스",
				Progress:  "25",
			},
		}},
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	got := updated.(model)
	if cmd != nil {
		t.Fatal("attend confirmation should not start attendance")
	}
	if got.active != screenAttendConfirm || got.attendRow.ID != "1:video" {
		t.Fatalf("active=%v attendRow=%+v", got.active, got.attendRow)
	}
	if !strings.Contains(got.View(), "이 강의를 수강할까요?") {
		t.Fatalf("confirmation view = %q", got.View())
	}
}

func TestLectureAttendShortcutAcceptsKoreanKeyboardKey(t *testing.T) {
	m := model{
		active: screenLectures,
		lectureRows: []app.LectureRow{{
			ID:      "1:video",
			Lecture: app.Lecture{ContentID: "video", Progress: "25"},
		}},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ㅁ")})
	if got := updated.(model); got.active != screenAttendConfirm {
		t.Fatalf("active = %v, want screenAttendConfirm", got.active)
	}
}

func TestIntegratedAttendFlowStartsProgress(t *testing.T) {
	m := model{
		ctx:     context.Background(),
		service: newTUITestService(t),
		active:  screenLectures,
		lectureRows: []app.LectureRow{{
			ID:         "course/1:lecture/video",
			CourseName: "운영체제",
			Lecture: app.Lecture{
				ContentID: "video",
				Title:     "프로세스",
				Progress:  "25",
			},
		}},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = updated.(model)
	if m.active != screenAttendConfirm || m.attendRow.ID != "course/1:lecture/video" {
		t.Fatalf("confirm state active=%v row=%+v", m.active, m.attendRow)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = updated.(model)
	if cmd == nil || m.active != screenAttendProgress || m.attendProgress == nil {
		t.Fatalf("progress state active=%v progress nil=%t cmd nil=%t", m.active, m.attendProgress == nil, cmd == nil)
	}
	defer m.attendProgress.cancel()
	if m.attendProgress.row.ID != "course/1:lecture/video" || m.attendProgress.progress.Progress != 25 {
		t.Fatalf("attend progress = %+v", m.attendProgress)
	}
}

func TestValidateLectureAttendRejectsInvalidStates(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.Local)
	before := now.Add(time.Hour)
	after := now.Add(-time.Hour)
	tests := []struct {
		name string
		row  app.LectureRow
	}{
		{name: "missing id", row: app.LectureRow{Lecture: app.Lecture{ContentID: "video"}}},
		{name: "unsupported", row: app.LectureRow{ID: "1:unsupported"}},
		{name: "completed video", row: app.LectureRow{ID: "1:video", Lecture: app.Lecture{ContentID: "video", Progress: "100"}}},
		{name: "completed activity", row: app.LectureRow{ID: "1:lrn-1", Lecture: app.Lecture{LearningSeq: "1", AchievedTime: "10", RequiredTime: "10"}}},
		{name: "not started", row: app.LectureRow{ID: "1:video", Lecture: app.Lecture{ContentID: "video", StartAt: &before}}},
		{name: "expired", row: app.LectureRow{ID: "1:video", Lecture: app.Lecture{ContentID: "video", EndAt: &after}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateLectureAttend(tt.row, now); err == nil {
				t.Fatalf("validateLectureAttend(%+v) returned nil", tt.row)
			}
		})
	}

	valid := app.LectureRow{ID: "1:video", Lecture: app.Lecture{ContentID: "video", Progress: "99"}}
	if err := validateLectureAttend(valid, now); err != nil {
		t.Fatalf("validateLectureAttend(valid) error = %v", err)
	}
}

func TestAttendConfirmationCanCancel(t *testing.T) {
	m := model{
		active:    screenAttendConfirm,
		attendRow: app.LectureRow{ID: "1:video", Lecture: app.Lecture{ContentID: "video"}},
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	got := updated.(model)
	if cmd != nil || got.active != screenLectures || got.attendRow.ID != "" {
		t.Fatalf("cmd=%v active=%v attendRow=%+v", cmd, got.active, got.attendRow)
	}
}

func TestLectureAttendModelTracksProgressAndCompletion(t *testing.T) {
	canceled := false
	m := lectureAttendModel{
		cancel: func() { canceled = true },
		row:    app.LectureRow{ID: "1:video"},
	}
	updated, _ := m.Update(lectureAttendProgressMsg{progress: app.LectureProgress{Progress: 40, TotalTime: "4", PTime: "10"}})
	got := updated.(lectureAttendModel)
	if got.progress.Progress != 40 || got.done {
		t.Fatalf("progress=%+v done=%t", got.progress, got.done)
	}

	updated, _ = got.Update(lectureAttendDoneMsg{result: app.LectureAttendResult{
		Lecture:  app.LectureRow{ID: "1:video"},
		Progress: app.LectureProgress{Progress: 100, Completed: true},
	}})
	got = updated.(lectureAttendModel)
	if !got.done || got.progress.Progress != 100 || !canceled {
		t.Fatalf("done=%t progress=%+v canceled=%t", got.done, got.progress, canceled)
	}
}

type fakeLectureAttender struct {
	id       string
	progress app.LectureProgress
}

func (f *fakeLectureAttender) AttendLecture(_ context.Context, id string, opts app.LectureAttendOptions) (app.LectureAttendResult, error) {
	f.id = id
	if opts.OnProgress != nil {
		opts.OnProgress(app.LectureRow{ID: id}, f.progress)
	}
	return app.LectureAttendResult{
		Lecture:  app.LectureRow{ID: id},
		Progress: f.progress,
	}, nil
}

func TestLectureAttendRunCallsServiceAndEmitsProgress(t *testing.T) {
	progress := app.LectureProgress{Progress: 75, TotalTime: "9", PTime: "12"}
	service := &fakeLectureAttender{progress: progress}
	m := lectureAttendModel{
		ctx:     context.Background(),
		service: service,
		row:     app.LectureRow{ID: "2:video"},
		updates: make(chan tea.Msg, 1),
	}

	msg := m.run()()
	done, ok := msg.(lectureAttendDoneMsg)
	if !ok || done.err != nil || done.result.Progress.Progress != 75 {
		t.Fatalf("run() message = %#v", msg)
	}
	if service.id != "2:video" {
		t.Fatalf("AttendLecture id = %q", service.id)
	}
	progressMsg, ok := (<-m.updates).(lectureAttendProgressMsg)
	if !ok || progressMsg.progress.Progress != 75 {
		t.Fatalf("progress message = %#v", progressMsg)
	}
}

func TestInitialAttendProgressForLearningActivity(t *testing.T) {
	progress := initialAttendProgress(app.LectureRow{Lecture: app.Lecture{
		LearningSeq:  "1",
		AchievedTime: "3",
		RequiredTime: "12",
	}})
	if progress.Progress != 25 || progress.TotalTime != "3" || progress.PTime != "12" {
		t.Fatalf("initialAttendProgress() = %+v", progress)
	}
}

func TestModelDownloadSelectionIDs(t *testing.T) {
	m := model{
		downloadRows: []app.LectureRow{
			{ID: "1:a", CourseName: "A", Lecture: app.Lecture{ContentID: "a"}},
			{ID: "1:b", CourseName: "B", Lecture: app.Lecture{ContentID: "b"}},
		},
		downloadSelected: map[string]bool{"1:b": true},
	}
	ids := m.selectedDownloadIDs()
	if len(ids) != 1 || ids[0] != "1:b" {
		t.Fatalf("selectedDownloadIDs() = %v", ids)
	}
}

func TestDownloadSelectionGroupsByCourse(t *testing.T) {
	m := model{
		downloadRows: []app.LectureRow{
			{ID: "1:a", CourseName: "A", Lecture: app.Lecture{ContentID: "a"}},
			{ID: "1:b", CourseName: "A", Lecture: app.Lecture{ContentID: "b"}},
			{ID: "2:c", CourseName: "B", Lecture: app.Lecture{ContentID: "c"}},
		},
		downloadSelected: map[string]bool{},
	}
	groups := m.downloadGroups()
	if len(groups) != 2 || groups[0].name != "A" || len(groups[0].rows) != 2 || groups[1].name != "B" {
		t.Fatalf("downloadGroups() = %+v", groups)
	}

	m.toggleDownloadCurrent()
	if !m.downloadSelected["1:a"] || !m.downloadSelected["1:b"] || m.downloadSelected["2:c"] {
		t.Fatalf("course toggle selected = %+v", m.downloadSelected)
	}
	m.toggleDownloadAll()
	if !m.downloadSelected["2:c"] {
		t.Fatalf("global toggle should include every course: %+v", m.downloadSelected)
	}
	m.toggleDownloadAll()
	if m.downloadSelected["1:a"] || m.downloadSelected["1:b"] || m.downloadSelected["2:c"] {
		t.Fatalf("global toggle should clear every course: %+v", m.downloadSelected)
	}
}

func TestDownloadRowsStartUnselected(t *testing.T) {
	m := model{active: screenDownloadSelect}
	updated, _ := m.Update(downloadRowsMsg{rows: []app.LectureRow{
		{ID: "1:a", CourseName: "A", Lecture: app.Lecture{ContentID: "a"}},
	}})
	got := updated.(model)
	if len(got.downloadSelected) != 0 {
		t.Fatalf("downloadSelected default = %+v, want empty", got.downloadSelected)
	}
}

func TestIntegratedDownloadFlowStartsSelectedTranscript(t *testing.T) {
	m := model{
		ctx:     context.Background(),
		service: newTUITestService(t),
		active:  screenDownloadSelect,
		downloadRows: []app.LectureRow{{
			ID:         "course/1:lecture/video",
			CourseName: "운영체제",
			Lecture:    app.Lecture{ContentID: "video", Title: "프로세스"},
		}},
		downloadSelected: map[string]bool{},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.active != screenDownloadConfirm {
		t.Fatalf("active after selection = %v, want screenDownloadConfirm", m.active)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = updated.(model)
	if m.active != screenDownloadLanguage || !m.downloadTranscribe {
		t.Fatalf("confirm state active=%v transcribe=%t", m.active, m.downloadTranscribe)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if cmd == nil || m.active != screenDownloadProgress || m.downloadProgress == nil {
		t.Fatalf("progress state active=%v progress nil=%t cmd nil=%t", m.active, m.downloadProgress == nil, cmd == nil)
	}
	defer m.downloadProgress.cancel()
	request := m.downloadProgress.request
	if !request.Transcribe || request.TranscriptLocale != "ko-KR" || len(request.LectureIDs) != 1 || request.LectureIDs[0] != "course/1:lecture/video" {
		t.Fatalf("download request = %+v", request)
	}
}

func TestDownloadSelectIgnoresSelectionWhileLoading(t *testing.T) {
	m := model{
		active:             screenDownloadSelect,
		loading:            true,
		downloadRows:       []app.LectureRow{{ID: "1:a", CourseName: "A", Lecture: app.Lecture{ContentID: "a"}}},
		downloadSelected:   map[string]bool{},
		downloadCourse:     0,
		downloadCursor:     1,
		downloadTranscribe: false,
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	got := updated.(model)
	if got.downloadSelected["1:a"] {
		t.Fatalf("loading selection changed: %+v", got.downloadSelected)
	}
	if got.active != screenDownloadSelect {
		t.Fatalf("active = %v, want screenDownloadSelect", got.active)
	}
}

func TestRoomFlowSelectsDaysAndPeriods(t *testing.T) {
	m := model{active: screenHome}
	updated, _ := m.startRoomFlow()
	got := updated.(model)
	if got.active != screenRoomDay || len(got.roomDaysSelected) != 0 {
		t.Fatalf("startRoomFlow() active=%v selected=%v", got.active, got.roomDaysSelected)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	got = updated.(model)
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyDown})
	got = updated.(model)
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	got = updated.(model)
	if days := got.selectedRoomDays(); len(days) != 2 || days[0] != 1 || days[1] != 2 {
		t.Fatalf("selectedRoomDays() = %v, want [1 2]", days)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = updated.(model)
	if got.active != screenRoomPeriod {
		t.Fatalf("active after day enter = %v", got.active)
	}
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	got = updated.(model)
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyDown})
	got = updated.(model)
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyDown})
	got = updated.(model)
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	got = updated.(model)
	if periods := got.selectedRoomPeriods(); len(periods) != 2 || periods[0] != 1 || periods[1] != 3 {
		t.Fatalf("selectedRoomPeriods() = %v, want [1 3]", periods)
	}
}

func TestFormatRoomAvailableResults(t *testing.T) {
	view := formatRoomAvailableResults([]app.RoomAvailableResult{{
		Weekday: 5,
		Periods: []int{1, 3},
		Rooms: []app.RoomAvailableRoom{
			{Room: "새빛관102"},
			{Room: "새빛관103"},
		},
	}})
	if !strings.Contains(view, "금 1, 3교시 비어있음") || !strings.Contains(view, "새빛관102") || !strings.Contains(view, "새빛관103") {
		t.Fatalf("formatRoomAvailableResults() = %q", view)
	}

	empty := formatRoomAvailableResults([]app.RoomAvailableResult{{Weekday: 1, Periods: []int{6, 7, 8}}})
	if !strings.Contains(empty, "조건에 맞는 빈 강의실이 없습니다") {
		t.Fatalf("formatRoomAvailableResults(empty) = %q", empty)
	}
}

func TestRoomAvailableGroupsByBuilding(t *testing.T) {
	results := []app.RoomAvailableResult{{
		Weekday: 2,
		Periods: []int{4, 5, 6},
		Rooms: []app.RoomAvailableRoom{
			{Room: "새빛관103"},
			{Room: "비마관502"},
			{Room: "새빛관101"},
		},
	}}
	groups := roomAvailableBuildingGroups(results)
	if len(groups) != 2 {
		t.Fatalf("groups = %+v", groups)
	}
	if groups[0].Building != "비마관" || groups[1].Building != "새빛관" {
		t.Fatalf("building order = %+v", groups)
	}
	if groups[1].Rows[0].Room != "새빛관101" || groups[1].Rows[1].Room != "새빛관103" {
		t.Fatalf("room sort = %+v", groups[1].Rows)
	}
}

func TestRoomResultNavigationWrapsByBuildingAndRows(t *testing.T) {
	m := model{roomResults: []app.RoomAvailableResult{{
		Weekday: 2,
		Periods: []int{4},
		Rooms: []app.RoomAvailableRoom{
			{Room: "비마관502"},
			{Room: "새빛관101"},
			{Room: "새빛관103"},
		},
	}}}
	m.moveRoomResultPage(1)
	if m.roomResultPage != 1 || m.roomResultCursor != 0 {
		t.Fatalf("page=%d cursor=%d", m.roomResultPage, m.roomResultCursor)
	}
	m.moveRoomResultCursor(-1)
	if m.roomResultCursor != 1 {
		t.Fatalf("cursor wrap = %d, want 1", m.roomResultCursor)
	}
	m.moveRoomResultPage(1)
	if m.roomResultPage != 0 || m.roomResultCursor != 0 {
		t.Fatalf("page wrap=%d cursor=%d", m.roomResultPage, m.roomResultCursor)
	}
}

func TestRoomResultViewFitsHeightAndKeepsHeader(t *testing.T) {
	rooms := make([]app.RoomAvailableRoom, 0, 30)
	for index := 1; index <= 30; index++ {
		rooms = append(rooms, app.RoomAvailableRoom{Room: fmt.Sprintf("새빛관%03d", index)})
	}
	m := model{
		active: screenRoomResult,
		width:  96,
		height: 18,
		roomResults: []app.RoomAvailableResult{{
			Weekday: 2,
			Periods: []int{4, 5, 6},
			Rooms:   rooms,
		}},
	}
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) > m.height {
		t.Fatalf("room result view lines = %d > %d\n%s", len(lines), m.height, view)
	}
	if !strings.Contains(lines[0], "KLAP") || !strings.Contains(view, "1-7 / 30") {
		t.Fatalf("room result view = %q", view)
	}
}

func TestDownloadSelectionLeftRightChangesCourse(t *testing.T) {
	m := model{
		active: screenDownloadSelect,
		downloadRows: []app.LectureRow{
			{ID: "1:a", CourseName: "A", Lecture: app.Lecture{ContentID: "a"}},
			{ID: "2:b", CourseName: "B", Lecture: app.Lecture{ContentID: "b"}},
		},
		downloadSelected: map[string]bool{},
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	got := updated.(model)
	if got.downloadCourse != 1 || got.downloadCursor != 0 {
		t.Fatalf("right key course=%d cursor=%d", got.downloadCourse, got.downloadCursor)
	}
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyLeft})
	got = updated.(model)
	if got.downloadCourse != 0 || got.downloadCursor != 0 {
		t.Fatalf("left key course=%d cursor=%d", got.downloadCourse, got.downloadCursor)
	}
}

func TestDownloadSelectCursorWraps(t *testing.T) {
	m := model{
		active: screenDownloadSelect,
		downloadRows: []app.LectureRow{
			{ID: "1:a", CourseName: "A", Lecture: app.Lecture{ContentID: "a"}},
			{ID: "1:b", CourseName: "A", Lecture: app.Lecture{ContentID: "b"}},
		},
		downloadSelected: map[string]bool{},
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	got := updated.(model)
	if got.downloadCursor != 2 {
		t.Fatalf("up from top downloadCursor = %d, want 2", got.downloadCursor)
	}
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyDown})
	got = updated.(model)
	if got.downloadCursor != 0 {
		t.Fatalf("down from bottom downloadCursor = %d, want 0", got.downloadCursor)
	}
}

func TestTranscriptLanguageDefaultsToKoreanWithoutWarning(t *testing.T) {
	m := model{downloadLanguage: 0}
	if got := m.selectedTranscriptLocale(); got != "ko-KR" {
		t.Fatalf("selectedTranscriptLocale() = %q", got)
	}
	view := m.renderDownloadLanguageView(96)
	if !strings.Contains(view, "ko-KR") || !strings.Contains(view, "전사 주 언어를 선택하세요.") || strings.Contains(view, "language switching(code switching)") {
		t.Fatalf("renderDownloadLanguageView() = %q", view)
	}
}

func TestFormatConfigShowsDownloadConcurrency(t *testing.T) {
	view := formatConfig(app.ConfigSettings{
		Reminder: app.ReminderSettings{ListName: "To-do", AlarmBeforeMin: 1440},
		Calendar: app.CalendarSettings{
			Name:                     "학사일정",
			UseExistingList:          true,
			TimetableName:            "시간표",
			TimetableUseExistingList: true,
		},
		Download:   app.DownloadSettings{Dir: "downloads", Concurrency: 7, Caffeinate: true, KeepPartial: false},
		Transcript: app.TranscriptSettings{Concurrency: 2},
	})
	if !strings.Contains(view, "Calendar") || !strings.Contains(view, "academic-name  학사일정") || !strings.Contains(view, "timetable-name  시간표") {
		t.Fatalf("formatConfig() missing calendar: %q", view)
	}
	if !strings.Contains(view, "concurrency  7") {
		t.Fatalf("formatConfig() missing concurrency: %q", view)
	}
	if !strings.Contains(view, "caffeinate  true") {
		t.Fatalf("formatConfig() missing caffeinate: %q", view)
	}
	if !strings.Contains(view, "keep-partial  false") {
		t.Fatalf("formatConfig() missing keep-partial: %q", view)
	}
	if !strings.Contains(view, "Transcript") || !strings.Contains(view, "concurrency  2") {
		t.Fatalf("formatConfig() missing transcript concurrency: %q", view)
	}
}

func TestConfigRowsExposeCategorySelection(t *testing.T) {
	settings := app.ConfigSettings{
		Reminder: app.ReminderSettings{ListName: "To-do", UseExistingList: true, AlarmBeforeMin: 60},
		Calendar: app.CalendarSettings{
			Name:                     "학사일정",
			UseExistingList:          true,
			TimetableName:            "시간표",
			TimetableUseExistingList: true,
		},
		Download: app.DownloadSettings{
			Dir:         "downloads",
			Concurrency: 4,
			Caffeinate:  true,
			KeepPartial: false,
		},
		Transcript: app.TranscriptSettings{Concurrency: 1},
	}
	options := app.CategoryOptions{
		Reminders: []string{"개인", "To-do"},
		Calendars: []string{"개인", "학사일정", "시간표"},
	}
	rows := configRows(settings, options)

	if len(rows) == 0 {
		t.Fatal("configRows() returned no rows")
	}
	var reminderRow configRow
	for _, row := range rows {
		if row.key == "reminder.name" {
			reminderRow = row
			break
		}
	}
	if reminderRow.key != "reminder.name" || !reminderRow.cycle || !reminderRow.editable {
		t.Fatalf("reminder row = %+v", reminderRow)
	}
	scheduleRows := configRowsForPage(settings, options, configPageSchedule)
	foundAcademicCalendar := false
	foundTimetableCalendar := false
	for _, row := range scheduleRows {
		if row.key == "calendar.name" && row.value == "학사일정" && row.page == configPageSchedule {
			foundAcademicCalendar = true
		}
		if row.key == "timetable-calendar.name" && row.value == "시간표" && row.page == configPageSchedule {
			foundTimetableCalendar = true
		}
	}
	if !foundAcademicCalendar || !foundTimetableCalendar {
		t.Fatalf("configRows() missing calendar category row: %+v", rows)
	}
}

func TestConfigRowsShowCurrentUserAndTerm(t *testing.T) {
	m := model{
		configSettings: app.ConfigSettings{
			Term: app.TermSettings{Value: "2026-1", Label: "2026년도 1학기"},
		},
		configUsers: []app.UserRow{
			{User: app.User{StudentID: "20250001"}},
			{User: app.User{StudentID: "20250002"}, Current: true},
		},
		configTerms: []app.TermRow{
			{Term: app.Term{Value: "2025-2", Label: "2025년도 2학기"}},
			{Term: app.Term{Value: "2026-1", Label: "2026년도 1학기"}, Current: true},
		},
	}
	rows := m.currentConfigRows()
	if len(rows) < 2 || rows[0].key != "user.current" || rows[0].value != "20250002" {
		t.Fatalf("user config row = %+v", rows)
	}
	if rows[1].key != "term.current" || rows[1].value != "2026-1  2026년도 1학기" {
		t.Fatalf("term config row = %+v", rows[1])
	}
	m.configSettings.Term = app.TermSettings{Value: "2024-1", Label: "2024년도 1학기"}
	m.configTerms[1].Current = false
	if got := currentTermLabel(m.configTerms, m.configSettings.Term); got != "선택 필요" {
		t.Fatalf("currentTermLabel(unmatched) = %q", got)
	}
}

func TestConfigChoicesUseRegisteredUsersAndTerms(t *testing.T) {
	m := model{
		configUsers: []app.UserRow{
			{User: app.User{StudentID: "20250001"}},
			{User: app.User{StudentID: "20250002"}, Current: true},
		},
		configTerms: []app.TermRow{
			{Term: app.Term{Value: "2025-2", Label: "2025년도 2학기"}},
			{Term: app.Term{Value: "2026-1", Label: "2026년도 1학기"}, Current: true},
		},
	}
	m.configChoiceKey = "user.current"
	if got := m.currentConfigChoices(); !reflect.DeepEqual(got, []string{"20250001", "20250002"}) {
		t.Fatalf("user choices = %#v", got)
	}
	if got := m.currentConfigChoiceIndex(); got != 1 {
		t.Fatalf("user choice index = %d", got)
	}
	m.configChoiceKey = "term.current"
	if got := m.currentConfigChoices(); !reflect.DeepEqual(got, []string{"2025-2  2025년도 2학기", "2026-1  2026년도 1학기"}) {
		t.Fatalf("term choices = %#v", got)
	}
	if got := m.currentConfigChoiceIndex(); got != 1 {
		t.Fatalf("term choice index = %d", got)
	}
	if got := termSelectorFromChoice("2026-1  2026년도 1학기"); got != "2026-1" {
		t.Fatalf("termSelectorFromChoice() = %q", got)
	}
}

func TestConfigCursorWraps(t *testing.T) {
	m := model{
		configSettings: app.ConfigSettings{
			Reminder:   app.ReminderSettings{ListName: "To-do", AlarmBeforeMin: 1440},
			Calendar:   app.CalendarSettings{Name: "학사일정", TimetableName: "시간표"},
			Download:   app.DownloadSettings{Dir: "downloads", Concurrency: 3, Caffeinate: true},
			Transcript: app.TranscriptSettings{Concurrency: 1},
		},
		configOptions: app.CategoryOptions{Reminders: []string{"To-do"}, Calendars: []string{"시간표"}},
		configPage:    configPageDownload,
	}
	m.moveConfigCursor(-1)
	if m.configCursor != len(configRowsForPage(m.configSettings, m.configOptions, configPageDownload))-1 {
		t.Fatalf("configCursor after up wrap = %d", m.configCursor)
	}
	m.moveConfigCursor(1)
	if m.configCursor != 0 {
		t.Fatalf("configCursor after down wrap = %d", m.configCursor)
	}
}

func TestConfigPageNavigationResetsCursor(t *testing.T) {
	m := model{
		configSettings: app.ConfigSettings{
			Reminder:   app.ReminderSettings{ListName: "To-do", AlarmBeforeMin: 1440},
			Calendar:   app.CalendarSettings{Name: "학사일정", TimetableName: "시간표"},
			Download:   app.DownloadSettings{Dir: "downloads", Concurrency: 3, Caffeinate: true},
			Transcript: app.TranscriptSettings{Concurrency: 1},
		},
		configOptions: app.CategoryOptions{Reminders: []string{"To-do"}, Calendars: []string{"시간표"}},
		configCursor:  2,
	}
	m.moveConfigPage(1)
	if m.configPage != configPageSchedule || m.configCursor != 0 {
		t.Fatalf("config page/cursor = %d/%d", m.configPage, m.configCursor)
	}
}

func TestConfigArrowKeysNavigatePages(t *testing.T) {
	m := model{
		active: screenConfig,
		configSettings: app.ConfigSettings{
			Reminder:   app.ReminderSettings{ListName: "To-do", AlarmBeforeMin: 1440},
			Calendar:   app.CalendarSettings{Name: "학사일정", TimetableName: "시간표"},
			Download:   app.DownloadSettings{Dir: "downloads", Concurrency: 3, Caffeinate: true},
			Transcript: app.TranscriptSettings{Concurrency: 1},
		},
		configOptions: app.CategoryOptions{Reminders: []string{"To-do"}, Calendars: []string{"시간표"}},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	got := updated.(model)
	if got.configPage != configPageSchedule {
		t.Fatalf("configPage after right = %d", got.configPage)
	}
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyLeft})
	got = updated.(model)
	if got.configPage != configPageGeneral {
		t.Fatalf("configPage after left = %d", got.configPage)
	}
}

func TestCategoryChoicesAppendDirectInput(t *testing.T) {
	got := categoryChoices([]string{"개인", "회사"}, "시간표")
	want := []string{"개인", "회사", "시간표", directInputChoice}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("categoryChoices() = %#v, want %#v", got, want)
	}
}

func TestConfigEnterOpensChoiceViewForCategories(t *testing.T) {
	m := model{
		active: screenConfig,
		configSettings: app.ConfigSettings{
			Reminder: app.ReminderSettings{ListName: "To-do", AlarmBeforeMin: 1440},
		},
		configOptions: app.CategoryOptions{Reminders: []string{"개인", "To-do"}},
		configCursor:  2,
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(model)
	if got.active != screenConfigChoice || got.configChoiceKey != "reminder.name" {
		t.Fatalf("config choice state = %v/%q", got.active, got.configChoiceKey)
	}
}

func TestConfigEnterOpensInputViewForTextRows(t *testing.T) {
	m := model{
		active: screenConfig,
		configSettings: app.ConfigSettings{
			Reminder: app.ReminderSettings{ListName: "To-do", AlarmBeforeMin: 1440},
			Download: app.DownloadSettings{Dir: "downloads", Concurrency: 3},
		},
		configPage: configPageDownload,
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(model)
	if got.active != screenConfigInput || got.configEditing != "download.dir" {
		t.Fatalf("config input state = %v/%q", got.active, got.configEditing)
	}
}
