package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/kw-klap/klap-cli/internal/klas"
)

func TestCourseFilterRequiresValue(t *testing.T) {
	_, err := courseFilter([]string{"--course"})
	if err == nil {
		t.Fatal("courseFilter() expected error")
	}
}

func TestUserFlag(t *testing.T) {
	if got := userFlag([]string{"--user", "20260000"}); got != "20260000" {
		t.Fatalf("userFlag() = %q", got)
	}
}

func TestLooksLikeLectureID(t *testing.T) {
	if !looksLikeLectureID("7:content-123") {
		t.Fatal("looksLikeLectureID() expected true")
	}
	if looksLikeLectureID("7") {
		t.Fatal("looksLikeLectureID() expected false for course number")
	}
	if looksLikeLectureID("오픈소스소프트웨어실습") {
		t.Fatal("looksLikeLectureID() expected false for course name")
	}
}

func TestIntervalFlag(t *testing.T) {
	got, err := intervalFlag([]string{"--interval", "5"})
	if err != nil {
		t.Fatalf("intervalFlag() error = %v", err)
	}
	if got != 5*time.Second {
		t.Fatalf("intervalFlag() = %v", got)
	}
}

func TestAcademicYearFlag(t *testing.T) {
	got, err := academicYearFlag([]string{"--year", "2026"})
	if err != nil {
		t.Fatalf("academicYearFlag() error = %v", err)
	}
	if got != "2026" {
		t.Fatalf("academicYearFlag() = %q", got)
	}
	if _, err := academicYearFlag([]string{"--year", "26"}); err == nil {
		t.Fatal("academicYearFlag() expected error for invalid year")
	}
}

func TestDashboardRefreshFlag(t *testing.T) {
	if !dashboardRefreshFlag([]string{"--refresh"}) {
		t.Fatal("dashboardRefreshFlag() expected true")
	}
	if unknown := firstUnknownDashboardArg([]string{"--refresh", "--user", "20260000"}); unknown != "" {
		t.Fatalf("firstUnknownDashboardArg() = %q", unknown)
	}
	if unknown := firstUnknownDashboardArg([]string{"--bad"}); unknown != "--bad" {
		t.Fatalf("firstUnknownDashboardArg() unknown = %q", unknown)
	}
}

func TestSyllabusOptions(t *testing.T) {
	opts, err := syllabusOptions([]string{"I040-3-3951-01", "--term", "2026-1", "--user", "20250000"})
	if err != nil {
		t.Fatalf("syllabusOptions() error = %v", err)
	}
	if opts.Selector != "I040-3-3951-01" || opts.TermValue != "2026-1" || opts.User.StudentID != "20250000" {
		t.Fatalf("syllabusOptions() = %+v", opts)
	}
	if _, err := syllabusOptions([]string{"--term"}); err == nil {
		t.Fatal("syllabusOptions() expected error")
	}
}

func TestSubjectSearchOptions(t *testing.T) {
	opts, err := subjectSearchOptions([]string{"--name", "컴퓨터그래픽스", "--professor", "김동준", "--term", "2026-1"})
	if err != nil {
		t.Fatalf("subjectSearchOptions() error = %v", err)
	}
	if opts.Name != "컴퓨터그래픽스" || opts.Professor != "김동준" || opts.TermValue != "2026-1" {
		t.Fatalf("subjectSearchOptions() = %+v", opts)
	}
	if _, err := subjectSearchOptions([]string{}); err == nil {
		t.Fatal("subjectSearchOptions() expected error")
	}
	if _, err := subjectSearchOptions([]string{"--name", "컴퓨터그래픽스", "--user", "20250000"}); err == nil {
		t.Fatal("subjectSearchOptions() expected error for --user")
	}
}

func TestRenderProgressBar(t *testing.T) {
	got := renderProgressBar(25, 12)
	if got == "" || !strings.Contains(got, "25%") {
		t.Fatalf("renderProgressBar() = %q", got)
	}
}

func TestLectureStatusPercent(t *testing.T) {
	if got := lectureStatusPercent(klas.Lecture{ContentID: "content", Progress: "75"}); got != 75 {
		t.Fatalf("lectureStatusPercent() video = %v", got)
	}
	if got := lectureStatusPercent(klas.Lecture{LearningSeq: "10", AchievedTime: "5", RequiredTime: "10"}); got != 50 {
		t.Fatalf("lectureStatusPercent() activity = %v", got)
	}
}

func TestParseReminderConfigArgs(t *testing.T) {
	name, useExistingList, ok, err := parseReminderConfigArgs([]string{"--name", "To-do", "--use-existing-list"})
	if err != nil {
		t.Fatalf("parseReminderConfigArgs() error = %v", err)
	}
	if !ok || name != "To-do" || !useExistingList {
		t.Fatalf("parseReminderConfigArgs() = %q, %v, %v", name, useExistingList, ok)
	}
}

func TestHyperlinkURLs(t *testing.T) {
	got := hyperlinkURLs("링크: https://example.com/path.")
	want := "링크: \x1b]8;;https://example.com/path\x1b\\https://example.com/path\x1b]8;;\x1b\\."
	if got != want {
		t.Fatalf("hyperlinkURLs() = %q, want %q", got, want)
	}
}
