package tui

import (
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"testing"
	"time"
)

func TestDashboardShowsLastSyncing(t *testing.T) {
	last := time.Date(2026, 6, 10, 15, 4, 5, 0, time.Local)
	m := model{
		active:     screenDashboard,
		lastSyncAt: last,
		dashboardResult: app.DashboardResult{
			Term: app.Term{Value: "2026,1", Label: "2026년도 1학기"},
		},
	}
	view := m.renderDashboardPagedPanel(96)
	if !strings.Contains(view, "Last Syncing") || !strings.Contains(view, "2026-06-10 15:04:05") {
		t.Fatalf("renderDashboardPagedPanel() missing last sync: %q", view)
	}
}

func TestDashboardFormatUsesScanSections(t *testing.T) {
	due := time.Date(2026, 6, 17, 23, 59, 0, 0, time.Local)
	result := app.DashboardResult{
		Term: app.Term{Value: "2026,1", Label: "2026년도 1학기"},
		Assignments: []app.AssignmentRow{{
			ID:         "7:7",
			CourseName: "오픈소스소프트웨어실습",
			Assignment: app.Assignment{
				Title: "기말고사 대체 과제",
				DueAt: &due,
			},
		}},
		Notices: []app.NoticeRow{{
			ID:         "2:1151742:1",
			CourseName: "창의설계입문",
			Notice: app.Notice{
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
		Term: app.Term{Value: "2026,1", Label: "2026년도 1학기"},
		Courses: []app.DashboardCourse{
			{
				Index: 1,
				Name:  "컴퓨터그래픽스",
				Assignments: []app.AssignmentRow{{
					CourseName: "컴퓨터그래픽스",
					Assignment: app.Assignment{Title: "과제1", DueAt: &due},
				}},
				Notices: []app.NoticeRow{{
					CourseName: "컴퓨터그래픽스",
					Notice:     app.Notice{Title: "강의 공지", Registered: &due},
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

func TestDashboardFooterShowsSyllabusShortcutOnlyOnCoursePage(t *testing.T) {
	summary := model{active: screenDashboard}
	if strings.Contains(summary.footerHelp(), "p 강의계획서") {
		t.Fatalf("summary footer = %q", summary.footerHelp())
	}
	course := model{active: screenDashboard, dashboardPage: 1}
	if !strings.Contains(course.footerHelp(), "p 강의계획서") {
		t.Fatalf("course footer = %q", course.footerHelp())
	}
}
