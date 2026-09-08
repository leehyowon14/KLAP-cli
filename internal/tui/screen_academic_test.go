package tui

import (
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"testing"
)

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
