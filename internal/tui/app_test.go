package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kw-klap/klap-cli/internal/app"
	"github.com/kw-klap/klap-cli/internal/klas"
)

func TestHomeViewShowsMenu(t *testing.T) {
	m := model{
		width: 96,
		ctx:   context.Background(),
		menu: []menuItem{
			{title: "Dashboard", help: "현재 학기 요약", screen: screenDashboard},
			{title: "Due", help: "다가오는 데드라인", screen: screenDue},
		},
	}

	view := m.View()
	if !strings.Contains(view, "https://github.com/leehyowon14/KLAP-GoLang") || !strings.Contains(view, "Beyond KLAS, in your terminal.") || !strings.Contains(view, "Kwangwoon Learning Automation Project") || !strings.Contains(view, "› 1.") || !strings.Contains(view, "Dashboard") || !strings.Contains(view, "_  __") {
		t.Fatalf("View() = %q", view)
	}
}

func TestHomeViewLinesFitWidth(t *testing.T) {
	m := model{
		width: 96,
		menu: []menuItem{
			{title: "Dashboard", help: "현재 학기 요약", screen: screenDashboard},
		},
	}
	for index, line := range strings.Split(m.View(), "\n") {
		if width := lipgloss.Width(line); width > m.width {
			t.Fatalf("home line %d width = %d > %d: %q", index, width, m.width, line)
		}
	}
}

func TestEnterScreenUsesPrefetchedData(t *testing.T) {
	m := model{
		loadedScreens: map[screen]bool{screenAssignments: true},
		assignmentRows: []app.AssignmentRow{
			{ID: "1", CourseName: "오픈소스소프트웨어실습"},
		},
	}
	updated, cmd := m.enterScreen(screenAssignments)
	got := updated.(model)
	if cmd != nil {
		t.Fatal("enterScreen() returned load command for prefetched screen")
	}
	if got.active != screenAssignments || got.loading {
		t.Fatalf("active=%v loading=%t", got.active, got.loading)
	}
	if len(got.assignmentRows) != 1 {
		t.Fatalf("assignmentRows = %+v", got.assignmentRows)
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
		assignmentRows: []app.AssignmentRow{
			{ID: "1", CourseName: "컴퓨터그래픽스"},
		},
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
	if len(m.assignmentRows) != 1 || m.assignmentRows[0].ID != "1" {
		t.Fatalf("assignmentRows clobbered: %+v", m.assignmentRows)
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
	updated, cmd := m.Update(loadMsg{screen: screenAssignments, prefetch: true, assignments: []app.AssignmentRow{{ID: "1"}}})
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

func TestHomeNavigation(t *testing.T) {
	m := model{
		ctx: context.Background(),
		menu: []menuItem{
			{title: "Dashboard", screen: screenDashboard},
			{title: "Due", screen: screenDue},
		},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	got := updated.(model)
	if got.cursor != 1 {
		t.Fatalf("cursor after down = %d", got.cursor)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyUp})
	got = updated.(model)
	if got.cursor != 0 {
		t.Fatalf("cursor after up = %d", got.cursor)
	}
}

func TestHomeNavigationAcceptsKoreanKeyboardKeys(t *testing.T) {
	m := model{
		ctx: context.Background(),
		menu: []menuItem{
			{title: "Dashboard", screen: screenDashboard},
			{title: "Due", screen: screenDue},
		},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ㅓ")})
	got := updated.(model)
	if got.cursor != 1 {
		t.Fatalf("cursor after korean j key = %d", got.cursor)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ㅏ")})
	got = updated.(model)
	if got.cursor != 0 {
		t.Fatalf("cursor after korean k key = %d", got.cursor)
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
		Assignment: klas.Assignment{
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
		Notice: klas.Notice{
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
		Lecture: klas.Lecture{
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

func TestCoursePagedContentGroupsByCourse(t *testing.T) {
	m := model{
		active: screenLectures,
		width:  96,
		lectureRows: []app.LectureRow{
			{CourseName: "컴퓨터그래픽스", Lecture: klas.Lecture{Title: "렌더링", ModuleTitle: "1주차", Progress: "20", ContentID: "a"}},
			{CourseName: "오픈소스소프트웨어실습", Lecture: klas.Lecture{Title: "Git", ModuleTitle: "2주차", Progress: "0", ContentID: "b"}},
			{CourseName: "컴퓨터그래픽스", Lecture: klas.Lecture{Title: "셰이딩", ModuleTitle: "3주차", Progress: "30", ContentID: "c"}},
		},
	}

	groups := m.contentGroups(96)
	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, want 2", len(groups))
	}
	if groups[0].name != "컴퓨터그래픽스" || len(groups[0].lines) != 2 {
		t.Fatalf("first group = %#v", groups[0])
	}
	if groups[1].name != "오픈소스소프트웨어실습" || len(groups[1].lines) != 1 {
		t.Fatalf("second group = %#v", groups[1])
	}
}

func TestCoursePagedCursorAndCourseWrap(t *testing.T) {
	due := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	m := model{
		active: screenAssignments,
		width:  96,
		assignmentRows: []app.AssignmentRow{
			{CourseName: "컴퓨터그래픽스", Assignment: klas.Assignment{Title: "과제1", DueAt: &due}},
			{CourseName: "컴퓨터그래픽스", Assignment: klas.Assignment{Title: "과제2", DueAt: &due}},
			{CourseName: "오픈소스소프트웨어실습", Assignment: klas.Assignment{Title: "기말", DueAt: &due}},
		},
	}

	m.moveContentCursor(-1)
	if m.contentCursor != 1 {
		t.Fatalf("cursor after wrapping up = %d, want 1", m.contentCursor)
	}
	m.moveContentCursor(1)
	if m.contentCursor != 0 {
		t.Fatalf("cursor after wrapping down = %d, want 0", m.contentCursor)
	}
	m.moveContentCourse(-1)
	if m.contentCourse != 1 || m.contentCursor != 0 {
		t.Fatalf("course/cursor after wrapping left = %d/%d, want 1/0", m.contentCourse, m.contentCursor)
	}
	m.moveContentCourse(1)
	if m.contentCourse != 0 || m.contentCursor != 0 {
		t.Fatalf("course/cursor after wrapping right = %d/%d, want 0/0", m.contentCourse, m.contentCursor)
	}
}

func TestCoursePagedRenderShowsCurrentCourseOnly(t *testing.T) {
	due := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	m := model{
		active:        screenAssignments,
		width:         96,
		height:        24,
		contentCourse: 1,
		assignmentRows: []app.AssignmentRow{
			{CourseName: "컴퓨터그래픽스", Assignment: klas.Assignment{Title: "과제1", DueAt: &due}},
			{CourseName: "오픈소스소프트웨어실습", Assignment: klas.Assignment{Title: "기말고사 대체 과제", DueAt: &due}},
		},
	}

	view := m.renderCoursePagedPanel(96)
	if !strings.Contains(view, "오픈소스소프트웨어실습") || !strings.Contains(view, "기말고사 대체 과제") {
		t.Fatalf("renderCoursePagedPanel() missing current course: %q", view)
	}
	if strings.Contains(view, "과제1") {
		t.Fatalf("renderCoursePagedPanel() leaked other course: %q", view)
	}
}

func TestCoursePagedSelectionUsesCurrentCourseAndCursor(t *testing.T) {
	due := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	m := model{
		active:        screenAssignments,
		width:         96,
		contentCourse: 1,
		contentCursor: 1,
		assignmentRows: []app.AssignmentRow{
			{ID: "1:1", CourseName: "컴퓨터그래픽스", Assignment: klas.Assignment{Title: "과제1", DueAt: &due}},
			{ID: "2:1", CourseName: "오픈소스소프트웨어실습", Assignment: klas.Assignment{Title: "과제A", DueAt: &due}},
			{ID: "2:2", CourseName: "오픈소스소프트웨어실습", Assignment: klas.Assignment{Title: "과제B", DueAt: &due}},
		},
	}

	row, ok := m.selectedAssignmentRow()
	if !ok || row.ID != "2:2" {
		t.Fatalf("selectedAssignmentRow() = %+v, %t", row, ok)
	}
}

func TestKlasShortcutDoesNotMoveListCursor(t *testing.T) {
	due := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	m := model{
		active:        screenAssignments,
		width:         96,
		contentCursor: 1,
		assignmentRows: []app.AssignmentRow{
			{ID: "1:1", CourseName: "컴퓨터그래픽스", DetailURL: "https://klas.example/1", Assignment: klas.Assignment{Title: "과제1", DueAt: &due}},
			{ID: "1:2", CourseName: "컴퓨터그래픽스", DetailURL: "https://klas.example/2", Assignment: klas.Assignment{Title: "과제2", DueAt: &due}},
		},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	got := updated.(model)
	if got.contentCursor != 1 {
		t.Fatalf("contentCursor after k = %d, want 1", got.contentCursor)
	}
}

func TestDetailLinesIncludeKlasURLAndBody(t *testing.T) {
	due := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	lines := assignmentDetailLines(app.AssignmentDetailResult{
		ID:         "7:1",
		CourseName: "오픈소스소프트웨어실습",
		DetailURL:  "https://klas.kw.ac.kr/assignment",
		Detail: klas.AssignmentDetail{
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

func TestNoticeContentGroupsMarksPinnedNotices(t *testing.T) {
	registered := time.Date(2026, 6, 1, 10, 0, 0, 0, time.Local)
	groups := noticeContentGroups([]app.NoticeRow{{
		CourseName: "컴퓨터그래픽스",
		Notice: klas.Notice{
			Title:      "중요 공지",
			Top:        true,
			Registered: &registered,
		},
	}}, 96)

	if len(groups) != 1 || len(groups[0].lines) != 1 {
		t.Fatalf("noticeContentGroups() = %+v", groups)
	}
	if !strings.Contains(groups[0].lines[0], "Pinned") || !strings.Contains(groups[0].lines[0], "중요 공지") {
		t.Fatalf("pinned notice line = %q", groups[0].lines[0])
	}
	if strings.Index(groups[0].lines[0], "Pinned") < strings.Index(groups[0].lines[0], "중요 공지") {
		t.Fatalf("pinned badge should be rendered after title: %q", groups[0].lines[0])
	}
}

func TestLectureContentGroupsAlignsTitleColumn(t *testing.T) {
	groups := lectureContentGroups([]app.LectureRow{
		{
			CourseName: "Gen-AI",
			Lecture: klas.Lecture{
				Progress:    "100",
				ContentID:   "a",
				ModuleTitle: "Basics of Python I",
				Title:       "Python Basics I",
			},
		},
		{
			CourseName: "Gen-AI",
			Lecture: klas.Lecture{
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
	if !lectureCompleted(klas.Lecture{ContentID: "content", Progress: "100"}) {
		t.Fatal("content lecture with 100% progress should be completed")
	}
	if lectureCompleted(klas.Lecture{ContentID: "content", Progress: "99"}) {
		t.Fatal("content lecture below 100% should not be completed")
	}
	if !lectureCompleted(klas.Lecture{AchievedTime: "10", RequiredTime: "10"}) {
		t.Fatal("minute based lecture with achieved >= required should be completed")
	}
	if lectureCompleted(klas.Lecture{AchievedTime: "9", RequiredTime: "10"}) {
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

func TestWrapHelpKeepsTrailingCommands(t *testing.T) {
	help := "←/→ 과목  ↑↓ 스크롤  enter 상세  k KLAS  s 동기화  b/esc 뒤로  r 새로고침  q 종료"
	view := wrapHelp(help, 36)
	if !strings.Contains(view, "\n") || !strings.Contains(view, "q 종료") {
		t.Fatalf("wrapHelp() = %q", view)
	}
}

func TestHeaderRendersSingleLineWithoutTUI(t *testing.T) {
	m := model{active: screenDashboard, loadedAt: time.Date(2026, 6, 10, 17, 29, 5, 0, time.Local)}
	header := m.renderHeader(48)
	if strings.Contains(header, "\n") || strings.Contains(header, "tui") || strings.Contains(header, "17:29") {
		t.Fatalf("renderHeader() = %q", header)
	}
	if !strings.Contains(header, "KLAP") || !strings.Contains(header, "Dashboard") {
		t.Fatalf("renderHeader() missing labels: %q", header)
	}
	if lipgloss.Width(header) > 48 {
		t.Fatalf("renderHeader() width = %d: %q", lipgloss.Width(header), header)
	}
}

func TestViewLinesFitTerminalWidth(t *testing.T) {
	m := model{
		active: screenLectures,
		width:  80,
		height: 24,
		lectureRows: []app.LectureRow{{
			ID:         "1",
			CourseName: "진로탐색및설계",
			Lecture: klas.Lecture{
				Title:        "2026-1학기 진로탐색 및 설계 오리엔테이션 매우 긴 제목",
				Progress:     "100",
				AchievedTime: "10",
				RequiredTime: "10",
			},
		}},
	}
	view := m.View()
	for index, line := range strings.Split(view, "\n") {
		if width := lipgloss.Width(line); width > m.width {
			t.Fatalf("line %d width = %d > %d: %q", index, width, m.width, line)
		}
	}
}

func TestViewStartsWithHeaderBeforeRule(t *testing.T) {
	for _, tt := range []struct {
		name string
		view string
	}{
		{name: "dashboard", view: testChromeModel(screenDashboard).View()},
		{name: "due", view: testChromeModel(screenDue).View()},
		{name: "assignments", view: testChromeModel(screenAssignments).View()},
		{name: "notices", view: testChromeModel(screenNotices).View()},
		{name: "lectures", view: testChromeModel(screenLectures).View()},
		{name: "assignment-detail", view: testChromeModel(screenAssignmentDetail).View()},
		{name: "notice-detail", view: testChromeModel(screenNoticeDetail).View()},
		{name: "academic", view: testChromeModel(screenAcademic).View()},
		{name: "config", view: testChromeModel(screenConfig).View()},
		{name: "config-choice", view: testChromeModel(screenConfigChoice).View()},
		{name: "config-input", view: testChromeModel(screenConfigInput).View()},
		{name: "room-day", view: testChromeModel(screenRoomDay).View()},
		{name: "room-period", view: testChromeModel(screenRoomPeriod).View()},
		{name: "download-select", view: testChromeModel(screenDownloadSelect).View()},
		{name: "download-confirm", view: testChromeModel(screenDownloadConfirm).View()},
		{name: "download-language", view: testChromeModel(screenDownloadLanguage).View()},
		{name: "download-progress", view: lectureDownloadModel{width: 80, height: 24, items: initialDownloadStatusLines([]app.LectureRow{{ID: "1", CourseName: "강의", Lecture: klas.Lecture{Title: "영상"}}})}.View()},
		{name: "confirm", view: confirmModel{title: "확인", message: "진행할까요?", width: 80}.View()},
		{name: "lecture-select", view: lectureSelectionModel{width: 80, height: 24, rows: []app.LectureRow{{ID: "1", CourseName: "강의", Lecture: klas.Lecture{Title: "영상"}}}, selected: map[string]bool{"1": true}}.View()},
	} {
		assertChromeInvariant(t, tt.name, tt.view, 80)
	}
}

func testChromeModel(active screen) model {
	dueAt := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	m := model{
		active: active,
		width:  80,
		height: 24,
		dashboardResult: app.DashboardResult{
			Term: klas.Term{Value: "2026,1", Label: "2026년도 1학기"},
			Courses: []app.DashboardCourse{{
				Index: 1,
				Name:  "강의",
			}},
		},
		dueResult: app.DueResult{
			From:  time.Date(2026, 6, 10, 0, 0, 0, 0, time.Local),
			Until: time.Date(2026, 6, 24, 0, 0, 0, 0, time.Local),
		},
		assignmentRows: []app.AssignmentRow{{
			ID:         "1",
			CourseName: "강의",
			Assignment: klas.Assignment{
				Title: "과제",
				DueAt: &dueAt,
			},
		}},
		noticeRows: []app.NoticeRow{{
			ID:         "1",
			CourseName: "강의",
			Notice: klas.Notice{
				Title:      "공지",
				Registered: &dueAt,
			},
		}},
		lectureRows: []app.LectureRow{{
			ID:         "1",
			CourseName: "강의",
			Lecture: klas.Lecture{
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
		assignmentDetail: app.AssignmentDetailResult{
			ID:         "1",
			CourseName: "강의",
			DetailURL:  "https://klas.kw.ac.kr",
			Detail:     klas.AssignmentDetail{Title: "과제", DueAt: &dueAt, ContentText: "본문"},
		},
		noticeDetail: app.NoticeDetailResult{
			ID:         "1",
			CourseName: "강의",
			DetailURL:  "https://klas.kw.ac.kr",
			Detail:     klas.NoticeDetail{Title: "공지", Registered: &dueAt, ContentText: "본문"},
		},
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
		dashboardResult: app.DashboardResult{
			Term: klas.Term{Value: "2026,1", Label: "2026년도 1학기"},
		},
	}
	view := m.renderPanel(96)
	if !strings.Contains(view, "DONE") || !strings.Contains(view, "과제 생성 1") {
		t.Fatalf("renderPanel() missing sync result: %q", view)
	}
	if strings.Contains(view, "OVERVIEW") {
		t.Fatalf("renderPanel() should hide dashboard while sync panel is active: %q", view)
	}
}

func TestDashboardShowsLastSyncing(t *testing.T) {
	last := time.Date(2026, 6, 10, 15, 4, 5, 0, time.Local)
	m := model{
		active:     screenDashboard,
		lastSyncAt: last,
		dashboardResult: app.DashboardResult{
			Term: klas.Term{Value: "2026,1", Label: "2026년도 1학기"},
		},
	}
	view := m.renderDashboardPagedPanel(96)
	if !strings.Contains(view, "Last Syncing") || !strings.Contains(view, "2026-06-10 15:04:05") {
		t.Fatalf("renderDashboardPagedPanel() missing last sync: %q", view)
	}
}

func TestDetailScrollClampsAtEdges(t *testing.T) {
	content := strings.Repeat("본문 줄\n", 40)
	m := model{
		active:       screenNoticeDetail,
		width:        80,
		height:       14,
		noticeDetail: app.NoticeDetailResult{ID: "1", CourseName: "강의", DetailURL: "https://klas.kw.ac.kr", Detail: klas.NoticeDetail{Title: "공지", ContentText: content}},
	}
	m.moveDetailCursor(1)
	if m.detailCursor != 1 {
		t.Fatalf("detailCursor after first down = %d", m.detailCursor)
	}
	for i := 0; i < 100; i++ {
		m.moveDetailCursor(1)
	}
	bottom := m.detailCursor
	if bottom <= 1 {
		t.Fatalf("detailCursor bottom = %d", bottom)
	}
	for i := 0; i < 100; i++ {
		m.moveDetailCursor(-1)
	}
	if m.detailCursor != 0 {
		t.Fatalf("detailCursor after up clamp = %d", m.detailCursor)
	}
}

func TestDashboardFormatUsesScanSections(t *testing.T) {
	due := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	result := app.DashboardResult{
		Term: klas.Term{Value: "2026,1", Label: "2026년도 1학기"},
		Assignments: []app.AssignmentRow{{
			ID:         "7:7",
			CourseName: "오픈소스소프트웨어실습",
			Assignment: klas.Assignment{
				Title: "기말고사 대체 과제",
				DueAt: &due,
			},
		}},
		Notices: []app.NoticeRow{{
			ID:         "2:1151742:1",
			CourseName: "창의설계입문",
			Notice: klas.Notice{
				Title:      "최종 발표 일정 안내",
				Registered: &due,
			},
		}},
		Attendance: app.DashboardAttendance{
			TotalCourses: 7,
			Completed:    138,
			Absent:       4,
		},
	}

	view := formatDashboard(result)
	for _, want := range []string{"OVERVIEW", "FOCUS", "LATEST", "Due", "Attendance", "기말고사 대체 과제", "최종 발표 일정 안내"} {
		if !strings.Contains(view, want) {
			t.Fatalf("formatDashboard() missing %q: %q", want, view)
		}
	}
	if strings.Contains(view, "7:7") || strings.Contains(view, "2:1151742:1") {
		t.Fatalf("formatDashboard() leaked internal id: %q", view)
	}
	if strings.Contains(view, "Attendance출석") {
		t.Fatalf("formatDashboard() metric spacing failed: %q", view)
	}
}

func TestDashboardPagedPanelShowsCoursePages(t *testing.T) {
	due := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	result := app.DashboardResult{
		Term: klas.Term{Value: "2026,1", Label: "2026년도 1학기"},
		Courses: []app.DashboardCourse{
			{
				Index: 1,
				Name:  "컴퓨터그래픽스",
				Assignments: []app.AssignmentRow{{
					CourseName: "컴퓨터그래픽스",
					Assignment: klas.Assignment{Title: "과제1", DueAt: &due},
				}},
				Notices: []app.NoticeRow{{
					CourseName: "컴퓨터그래픽스",
					Notice:     klas.Notice{Title: "강의 공지", Registered: &due},
				}},
				Attendance: &app.DashboardAttendanceRow{Completed: 10, Absent: 1},
			},
		},
	}
	m := model{active: screenDashboard, dashboardResult: result, dashboardPage: 1}

	view := m.renderDashboardPagedPanel(96)
	for _, want := range []string{"컴퓨터그래픽스", "과제1", "강의 공지", "STATUS"} {
		if !strings.Contains(view, want) {
			t.Fatalf("renderDashboardPagedPanel() missing %q: %q", want, view)
		}
	}
}

func TestDashboardPageWraps(t *testing.T) {
	m := model{dashboardResult: app.DashboardResult{Courses: []app.DashboardCourse{{Name: "A"}, {Name: "B"}}}}
	m.moveDashboardPage(-1)
	if m.dashboardPage != 2 {
		t.Fatalf("dashboardPage after left wrap = %d", m.dashboardPage)
	}
	m.moveDashboardPage(1)
	if m.dashboardPage != 0 {
		t.Fatalf("dashboardPage after right wrap = %d", m.dashboardPage)
	}
}

func TestLectureDownloadFormatting(t *testing.T) {
	row := app.LectureRow{
		CourseName: "오픈소스소프트웨어실습",
		Lecture: klas.Lecture{
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
		{ID: "1:a", CourseName: "컴퓨터그래픽스", Lecture: klas.Lecture{ContentID: "a", ModuleTitle: "1주차", Title: "소개"}},
		{ID: "1:b", CourseName: "컴퓨터그래픽스", Lecture: klas.Lecture{ContentID: "b", ModuleTitle: "2주차", Title: "렌더링"}},
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
	row := app.LectureRow{ID: "1:a", CourseName: "컴퓨터그래픽스", Lecture: klas.Lecture{ContentID: "a", Title: "소개"}}
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

func TestLectureDownloadCancelCleanupRemovesArtifacts(t *testing.T) {
	root := t.TempDir()
	videoPath := filepath.Join(root, "컴퓨터그래픽스", "video", "lecture.mp4")
	transcriptPath := filepath.Join(root, "컴퓨터그래픽스", "transcription", "lecture.txt")
	for _, path := range []string{videoPath, videoPath + ".part", transcriptPath, transcriptPath + ".part"} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", path, err)
		}
		if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", path, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	m := lectureDownloadModel{
		ctx:    ctx,
		cancel: cancel,
		items: []downloadStatusLine{{
			status:         "transcribe",
			path:           transcriptPath,
			downloadPath:   videoPath,
			transcriptPath: transcriptPath,
		}},
	}
	m.cancelAndCleanup()
	if !m.canceling {
		t.Fatal("cancelAndCleanup() did not mark canceling")
	}
	for _, path := range []string{videoPath, videoPath + ".part", transcriptPath, transcriptPath + ".part"} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("artifact %s still exists or stat failed: %v", path, err)
		}
	}
}

func TestLectureDownloadTranscriptQueueRespectsConcurrency(t *testing.T) {
	rowA := app.LectureRow{ID: "1:a", CourseName: "A", Lecture: klas.Lecture{ContentID: "a", Title: "A"}}
	rowB := app.LectureRow{ID: "1:b", CourseName: "A", Lecture: klas.Lecture{ContentID: "b", Title: "B"}}
	m := lectureDownloadModel{
		request: LectureDownloadRequest{
			Transcribe:            true,
			TranscriptConcurrency: 1,
		},
		transcriptStarted: make(map[string]bool),
		transcriptRunning: make(map[string]bool),
	}

	cmds := m.enqueueTranscriptForDownloadProgress(app.LectureDownloadProgress{Lecture: rowA, Path: "a.mp4", Stage: "done"})
	if len(cmds) != 1 || m.transcriptActive != 0 || len(m.transcriptQueue) != 1 || !m.transcriptStartScheduled {
		t.Fatalf("first enqueue cmds=%d active=%d queue=%d scheduled=%t", len(cmds), m.transcriptActive, len(m.transcriptQueue), m.transcriptStartScheduled)
	}

	cmds = m.enqueueTranscriptForDownloadProgress(app.LectureDownloadProgress{Lecture: rowB, Path: "b.mp4", Stage: "done"})
	if len(cmds) != 0 || m.transcriptActive != 0 || len(m.transcriptQueue) != 2 {
		t.Fatalf("second enqueue cmds=%d active=%d queue=%d", len(cmds), m.transcriptActive, len(m.transcriptQueue))
	}

	m.transcriptStartScheduled = false
	cmds = m.startTranscriptWorkers()
	if len(cmds) != 1 || m.transcriptActive != 1 || len(m.transcriptQueue) != 0 || len(m.transcriptRunning) != 2 {
		t.Fatalf("batch worker cmds=%d active=%d queue=%d running=%d", len(cmds), m.transcriptActive, len(m.transcriptQueue), len(m.transcriptRunning))
	}
}

func TestLectureDownloadTranscribesSkippedVideoWhenTranscriptMissing(t *testing.T) {
	root := t.TempDir()
	videoPath := filepath.Join(root, "컴퓨터그래픽스", "video", "lecture.mp4")
	if err := os.MkdirAll(filepath.Dir(videoPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(video) error = %v", err)
	}
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}

	row := app.LectureRow{ID: "1:a", CourseName: "컴퓨터그래픽스", Lecture: klas.Lecture{ContentID: "a", Title: "소개"}}
	m := lectureDownloadModel{
		request: LectureDownloadRequest{
			Transcribe:            true,
			TranscriptConcurrency: 1,
		},
		transcriptStarted: make(map[string]bool),
		transcriptRunning: make(map[string]bool),
	}
	cmds := m.enqueueTranscriptsForResult(lectureDownloadDoneMsg{all: app.LectureDownloadAllResult{Items: []app.LectureDownloadItem{{
		Lecture: row,
		Path:    videoPath,
		Skipped: true,
	}}}})
	if len(cmds) != 1 || m.transcriptActive != 0 || len(m.transcriptQueue) != 1 {
		t.Fatalf("missing transcript cmds=%d active=%d queue=%d", len(cmds), m.transcriptActive, len(m.transcriptQueue))
	}
	if len(m.items) != 1 || m.items[0].status != "transcribe" || m.items[0].skipped {
		t.Fatalf("missing transcript status = %+v", m.items)
	}
	if got := m.itemProgressText(m.items[0]); !strings.Contains(got, "전사중") || strings.Contains(got, "건너뜀") {
		t.Fatalf("transcribing skipped item text = %q", got)
	}
	if got := itemProgressPercent(m.items[0]); got >= 1 {
		t.Fatalf("transcribing skipped item progress = %f, want in-progress", got)
	}

	transcriptPath := filepath.Join(root, "컴퓨터그래픽스", "transcription", "lecture.txt")
	if err := os.MkdirAll(filepath.Dir(transcriptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(transcript) error = %v", err)
	}
	if err := os.WriteFile(transcriptPath, []byte("text"), 0o644); err != nil {
		t.Fatalf("WriteFile(transcript) error = %v", err)
	}
	m = lectureDownloadModel{
		request: LectureDownloadRequest{
			Transcribe:            true,
			TranscriptConcurrency: 1,
		},
		transcriptStarted: make(map[string]bool),
		transcriptRunning: make(map[string]bool),
	}
	cmds = m.enqueueTranscriptsForResult(lectureDownloadDoneMsg{all: app.LectureDownloadAllResult{Items: []app.LectureDownloadItem{{
		Lecture: row,
		Path:    videoPath,
		Skipped: true,
	}}}})
	if len(cmds) != 0 || m.transcriptActive != 0 {
		t.Fatalf("existing transcript cmds=%d active=%d", len(cmds), m.transcriptActive)
	}
}

func TestLectureDownloadProgressCursorWraps(t *testing.T) {
	rows := []app.LectureRow{
		{ID: "1:a", CourseName: "A", Lecture: klas.Lecture{ContentID: "a"}},
		{ID: "1:b", CourseName: "A", Lecture: klas.Lecture{ContentID: "b"}},
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

func TestConfirmAcceptsKoreanKeyboardKeys(t *testing.T) {
	m := confirmModel{value: false}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ㅛ")})
	got := updated.(confirmModel)
	if !got.value || cmd == nil {
		t.Fatalf("confirm korean yes value=%t cmd nil=%t", got.value, cmd == nil)
	}
}

func TestLectureSelectionToggleAll(t *testing.T) {
	rows := []app.LectureRow{
		{ID: "1:a", Lecture: klas.Lecture{ContentID: "a"}},
		{ID: "1:b", Lecture: klas.Lecture{ContentID: "b"}},
		{ID: "1:empty"},
	}
	model := lectureSelectionModel{
		rows:     rows,
		selected: map[string]bool{"1:a": true, "1:b": true},
	}
	model.toggleAll()
	if model.selected["1:a"] || model.selected["1:b"] {
		t.Fatalf("toggleAll() expected selected rows off: %+v", model.selected)
	}
	model.toggleAll()
	if !model.selected["1:a"] || !model.selected["1:b"] || model.selected["1:empty"] {
		t.Fatalf("toggleAll() expected downloadable rows on only: %+v", model.selected)
	}
}

func TestModelDownloadSelectionIDs(t *testing.T) {
	m := model{
		downloadRows: []app.LectureRow{
			{ID: "1:a", CourseName: "A", Lecture: klas.Lecture{ContentID: "a"}},
			{ID: "1:b", CourseName: "B", Lecture: klas.Lecture{ContentID: "b"}},
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
			{ID: "1:a", CourseName: "A", Lecture: klas.Lecture{ContentID: "a"}},
			{ID: "1:b", CourseName: "A", Lecture: klas.Lecture{ContentID: "b"}},
			{ID: "2:c", CourseName: "B", Lecture: klas.Lecture{ContentID: "c"}},
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
		{ID: "1:a", CourseName: "A", Lecture: klas.Lecture{ContentID: "a"}},
	}})
	got := updated.(model)
	if len(got.downloadSelected) != 0 {
		t.Fatalf("downloadSelected default = %+v, want empty", got.downloadSelected)
	}
}

func TestDownloadSelectIgnoresSelectionWhileLoading(t *testing.T) {
	m := model{
		active:             screenDownloadSelect,
		loading:            true,
		downloadRows:       []app.LectureRow{{ID: "1:a", CourseName: "A", Lecture: klas.Lecture{ContentID: "a"}}},
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

func TestDownloadSelectionLeftRightChangesCourse(t *testing.T) {
	m := model{
		active: screenDownloadSelect,
		downloadRows: []app.LectureRow{
			{ID: "1:a", CourseName: "A", Lecture: klas.Lecture{ContentID: "a"}},
			{ID: "2:b", CourseName: "B", Lecture: klas.Lecture{ContentID: "b"}},
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
			{ID: "1:a", CourseName: "A", Lecture: klas.Lecture{ContentID: "a"}},
			{ID: "1:b", CourseName: "A", Lecture: klas.Lecture{ContentID: "b"}},
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
	if rows[0].key != "reminder.name" || !rows[0].cycle || !rows[0].editable {
		t.Fatalf("reminder row = %+v", rows[0])
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
