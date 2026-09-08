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

func TestParseAcademicEvents(t *testing.T) {
	body := []byte(`
		<!--
		<h3>2025 학사일정</h3>
		<table><tbody><tr><td>3월</td><td>1(토)</td><td>오래된 일정</td><td></td></tr></tbody></table>
		-->
		<h3>2026 학사일정</h3>
		<table><tbody>
			<tr><td rowspan="2">3월</td><td>3(화)</td><td>2026학년도 1학기 개강(학기개시일)</td><td>3월2일(월) - 대체공휴일</td></tr>
			<tr><td>28(토)</td><td>수업일수 4분의 1</td><td>15주 기준</td></tr>
			<tr><td>8월</td><td>3(월)~31(월)</td><td>2학기 복학신청</td></tr>
		</tbody></table>
		<h3>2027 학사일정</h3>
		<table><tbody>
			<tr><td>3월</td><td>2(화)</td><td>2027학년도 1학기 개강</td><td></td></tr>
		</tbody></table>
	`)

	events, err := parseAcademicEvents(body, "2026")
	if err != nil {
		t.Fatalf("parseAcademicEvents() error = %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("parseAcademicEvents() len = %d, want 3: %+v", len(events), events)
	}
	if events[0].Month != "3월" || events[0].Date != "3(화)" || events[0].Title != "2026학년도 1학기 개강(학기개시일)" {
		t.Fatalf("parseAcademicEvents()[0] = %+v", events[0])
	}
	if events[1].Month != "3월" || events[1].Date != "28(토)" || events[1].Note != "15주 기준" {
		t.Fatalf("parseAcademicEvents()[1] = %+v", events[1])
	}
	if events[2].Month != "8월" || events[2].Date != "3(월)~31(월)" || events[2].Title != "2학기 복학신청" {
		t.Fatalf("parseAcademicEvents()[2] = %+v", events[2])
	}
}
