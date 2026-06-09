package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kw-klap/klap-cli/internal/app"
	"github.com/kw-klap/klap-cli/internal/klas"
)

func TestHomeViewShowsMenu(t *testing.T) {
	m := model{
		ctx: context.Background(),
		menu: []menuItem{
			{title: "Dashboard", help: "현재 학기 요약", screen: screenDashboard},
			{title: "Due", help: "다가오는 데드라인", screen: screenDue},
		},
	}

	view := m.View()
	if !strings.Contains(view, "https://github.com/leehyowon14/KLAP-GoLang") || !strings.Contains(view, "› 1.") || !strings.Contains(view, "Dashboard") || !strings.Contains(view, "_  __") {
		t.Fatalf("View() = %q", view)
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
	m.upsertTranscriptStatusLine(app.LectureTranscriptProgress{Lecture: row, Stage: "transcribe", OutputPath: "lecture.txt"})
	m.upsertStatusLine(app.LectureDownloadProgress{Lecture: row, Stage: "done", Path: "lecture.mp4", Bytes: 10, TotalBytes: 10})
	if got := m.items[0].status; got != "transcribe" {
		t.Fatalf("status = %q, want transcribe", got)
	}
	if got := itemProgressPercent(downloadStatusLine{status: "download", bytes: 5, total: 10}); got != 0.5 {
		t.Fatalf("itemProgressPercent() = %f", got)
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

func TestTranscriptLanguageDefaultsToKoreanAndMentionsCodeSwitching(t *testing.T) {
	m := model{downloadLanguage: 0}
	if got := m.selectedTranscriptLocale(); got != "ko-KR" {
		t.Fatalf("selectedTranscriptLocale() = %q", got)
	}
	view := m.renderDownloadLanguageView(96)
	if !strings.Contains(view, "ko-KR") || !strings.Contains(view, "language switching(code switching)") {
		t.Fatalf("renderDownloadLanguageView() = %q", view)
	}
}

func TestFormatConfigShowsDownloadConcurrency(t *testing.T) {
	view := formatConfig(app.ConfigSettings{
		Download: app.DownloadSettings{Dir: "downloads", Concurrency: 7, Caffeinate: true},
	})
	if !strings.Contains(view, "concurrency  7") {
		t.Fatalf("formatConfig() missing concurrency: %q", view)
	}
	if !strings.Contains(view, "caffeinate  true") {
		t.Fatalf("formatConfig() missing caffeinate: %q", view)
	}
}
