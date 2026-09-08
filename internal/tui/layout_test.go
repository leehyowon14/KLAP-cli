package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"testing"
	"time"
)

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
		height: 24, lectures: lectureScreenModel{lectureRows: []app.LectureRow{{
			ID:         "1",
			CourseName: "진로탐색및설계",
			Lecture: app.Lecture{
				Title:        "2026-1학기 진로탐색 및 설계 오리엔테이션 매우 긴 제목",
				Progress:     "100",
				AchievedTime: "10",
				RequiredTime: "10",
			},
		}}},
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
		{name: "auth", view: testChromeModel(screenAuth).View()},
		{name: "due", view: testChromeModel(screenDue).View()},
		{name: "assignments", view: testChromeModel(screenAssignments).View()},
		{name: "notices", view: testChromeModel(screenNotices).View()},
		{name: "lectures", view: testChromeModel(screenLectures).View()},
		{name: "assignment-detail", view: testChromeModel(screenAssignmentDetail).View()},
		{name: "notice-detail", view: testChromeModel(screenNoticeDetail).View()},
		{name: "academic", view: testChromeModel(screenAcademic).View()},
		{name: "sync-conflict", view: testChromeModel(screenSyncConflict).View()},
		{name: "config", view: testChromeModel(screenConfig).View()},
		{name: "config-choice", view: testChromeModel(screenConfigChoice).View()},
		{name: "config-input", view: testChromeModel(screenConfigInput).View()},
		{name: "room-day", view: testChromeModel(screenRoomDay).View()},
		{name: "room-period", view: testChromeModel(screenRoomPeriod).View()},
		{name: "download-select", view: testChromeModel(screenDownloadSelect).View()},
		{name: "download-confirm", view: testChromeModel(screenDownloadConfirm).View()},
		{name: "download-language", view: testChromeModel(screenDownloadLanguage).View()},
		{name: "download-progress", view: lectureDownloadModel{width: 80, height: 24, items: initialDownloadStatusLines([]app.LectureRow{{ID: "1", CourseName: "강의", Lecture: app.Lecture{Title: "영상"}}})}.View()},
		{name: "attend-confirm", view: testChromeModel(screenAttendConfirm).View()},
		{name: "attend-progress", view: lectureAttendModel{width: 80, height: 24, row: app.LectureRow{ID: "1:video", CourseName: "강의", Lecture: app.Lecture{Title: "영상"}}, progress: app.LectureProgress{Progress: 50, TotalTime: "5", PTime: "10"}}.View()},
	} {
		assertChromeInvariant(t, tt.name, tt.view, 80)
	}
}
