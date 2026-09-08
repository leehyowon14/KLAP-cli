package tui

import (
	"context"
	"errors"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"github.com/leehyowon14/KLAP-cli/internal/testsupport"
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
	if len(m.lectures.lectureRows) != 1 {
		t.Fatalf("lectureRows = %+v", m.lectures.lectureRows)
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
		}}, due: dueScreenModel{dueResult: app.DueResult{
			From:  time.Date(2026, 6, 10, 0, 0, 0, 0, time.Local),
			Until: time.Date(2026, 6, 24, 0, 0, 0, 0, time.Local),
		}}, academic: academicScreenModel{academicResult: app.AcademicListResult{
			Year: "2026",
			Events: []app.AcademicEvent{{
				Year:  "2026",
				Month: "6월",
				Date:  "6.17(수)",
				Title: "종강",
			}},
		},
			academicMonth: 6}, lectures: lectureScreenModel{lectureRows: []app.LectureRow{{
			ID:         "1",
			CourseName: "강의",
			Lecture: app.Lecture{
				Title:        "영상",
				Progress:     "100",
				AchievedTime: "10",
				RequiredTime: "10",
			},
		}}}, config: configScreenModel{configSettings: app.ConfigSettings{
			Reminder: app.ReminderSettings{ListName: "Kwangwoon Univ.", AlarmBeforeMin: 1440},
			Calendar: app.CalendarSettings{Name: "학사일정", TimetableName: "시간표"},
			Download: app.DownloadSettings{
				Dir:         "downloads",
				Concurrency: 3,
				Caffeinate:  true,
			},
			Transcript: app.TranscriptSettings{Concurrency: 1},
		},
			configOptions:   app.CategoryOptions{Reminders: []string{"Kwangwoon Univ."}, Calendars: []string{"학사일정", "시간표"}},
			configChoiceKey: "calendar.name",
			configInput:     textinput.New()}, room: roomScreenModel{roomDaysSelected: map[int]bool{1: true},
			roomPeriodsSelected: map[int]bool{1: true}}, sync: syncScreenModel{syncConflicts: []app.SyncConflict{{
			Key:     "assignment:1",
			Scope:   "assignment",
			Title:   "기말 과제",
			Summary: "기말 과제 · 2026-06-17 23:59",
		}},
			syncConflictActions: map[string]app.SyncDecision{"assignment:1": app.SyncDecisionKeep}},
	}
	m.config.configInput.SetValue("입력값")
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

func TestIntegratedAttendFlowStartsProgress(t *testing.T) {
	m := model{
		ctx:     context.Background(),
		service: newTUITestService(t),
		active:  screenLectures, lectures: lectureScreenModel{lectureRows: []app.LectureRow{{
			ID:         "course/1:lecture/video",
			CourseName: "운영체제",
			Lecture: app.Lecture{
				ContentID: "video",
				Title:     "프로세스",
				Progress:  "25",
			},
		}}},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = updated.(model)
	if m.active != screenAttendConfirm || m.attend.attendRow.ID != "course/1:lecture/video" {
		t.Fatalf("confirm state active=%v row=%+v", m.active, m.attend.attendRow)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = updated.(model)
	if cmd == nil || m.active != screenAttendProgress || m.attend.attendProgress == nil {
		t.Fatalf("progress state active=%v progress nil=%t cmd nil=%t", m.active, m.attend.attendProgress == nil, cmd == nil)
	}
	defer m.attend.attendProgress.cancel()
	if m.attend.attendProgress.row.ID != "course/1:lecture/video" || m.attend.attendProgress.progress.Progress != 25 {
		t.Fatalf("attend progress = %+v", m.attend.attendProgress)
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
		active: screenAttendConfirm, attend: attendScreenModel{attendRow: app.LectureRow{ID: "1:video", Lecture: app.Lecture{ContentID: "video"}}},
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	got := updated.(model)
	if cmd != nil || got.active != screenLectures || got.attend.attendRow.ID != "" {
		t.Fatalf("cmd=%v active=%v attendRow=%+v", cmd, got.active, got.attend.attendRow)
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
