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
