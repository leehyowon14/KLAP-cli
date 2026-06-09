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
