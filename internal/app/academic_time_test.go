package app

import (
	"testing"
	"time"
)

func TestAcademicEventDueAt(t *testing.T) {
	got, ok := academicEventDueAt(AcademicEvent{
		Year:  "2026",
		Month: "6월",
		Date:  "06.17(수)",
		Title: "기말고사",
	})
	if !ok {
		t.Fatal("academicEventDueAt() expected ok")
	}
	if got.Year() != 2026 || got.Month() != time.June || got.Day() != 17 {
		t.Fatalf("academicEventDueAt() = %s", got)
	}
}

func TestAcademicEventRangeParsesMultiDayEvent(t *testing.T) {
	startAt, endAt, ok := AcademicEventRange(AcademicEvent{
		Year:  "2026",
		Month: "6월",
		Date:  "06.22(월) ~ 06.26(금)",
		Title: "보강주간",
	})
	if !ok {
		t.Fatal("AcademicEventRange() expected ok")
	}
	if startAt.Year() != 2026 || startAt.Month() != time.June || startAt.Day() != 22 {
		t.Fatalf("startAt = %s", startAt)
	}
	if endAt.Year() != 2026 || endAt.Month() != time.June || endAt.Day() != 27 {
		t.Fatalf("endAt = %s", endAt)
	}
}
