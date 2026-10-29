package main

import (
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"testing"
)

func TestCalendarEntries(t *testing.T) {
	tests := []struct{ month, date, start, end string }{
		{"9월", "1(화)", "2026-09-01", "2026-09-01"},
		{"9월", "8(화)~10(목)", "2026-09-08", "2026-09-10"},
		{"9월", "09.28(월) ~ 10.02(금)", "2026-09-28", "2026-10-02"},
		{"12월", "12.30(수) ~ 01.02(토)", "2026-12-30", "2027-01-02"},
		{"9월", "추후 공지", "", ""},
	}
	for _, tt := range tests {
		got := calendarEntries([]app.AcademicEvent{{Year: "2026", Month: tt.month, Date: tt.date, Title: "일정"}})[0]
		if got.Start != tt.start || got.End != tt.end || got.Title != "일정" {
			t.Fatalf("%s: %+v", tt.date, got)
		}
	}
	if calendarEntries(nil) == nil {
		t.Fatal("empty rows must be an array")
	}
}
