package tui

import (
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"testing"
)

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

func TestSyncPanelHidesDashboardStatus(t *testing.T) {
	m := model{
		active: screenDashboard,

		syncStatus: "동기화 완료: 과제 생성 1",
		dashboard: dashboardScreenModel{dashboardResult: app.DashboardResult{
			Term: app.Term{Value: "2026,1", Label: "2026년도 1학기"},
		}}, sync: syncScreenModel{syncPhase: "done"},
	}
	view := m.renderPanel(96)
	if !strings.Contains(view, "DONE") || !strings.Contains(view, "과제 생성 1") {
		t.Fatalf("renderPanel() missing sync result: %q", view)
	}
	if strings.Contains(view, "OVERVIEW") {
		t.Fatalf("renderPanel() should hide dashboard while sync panel is active: %q", view)
	}
}
