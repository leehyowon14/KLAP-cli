package cli

import (
	"bytes"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
	"testing"
	"time"
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

func TestRoomQueryOptions(t *testing.T) {
	opts, err := roomQueryOptions([]string{"연", "102", "--term", "2026-1", "--building", "연구", "--refresh", "--user", "20260000"})
	if err != nil {
		t.Fatalf("roomQueryOptions() error = %v", err)
	}
	if opts.Room != "연 102" || opts.TermValue != "2026-1" || opts.Building != "연구" || !opts.Refresh || opts.User.StudentID != "20260000" {
		t.Fatalf("roomQueryOptions() = %+v", opts)
	}
	if _, err := roomQueryOptions([]string{"--term", "2026-1"}); err == nil {
		t.Fatal("roomQueryOptions() expected missing room error")
	}
}

func TestRoomIndexOptions(t *testing.T) {
	opts, err := roomIndexOptions([]string{"--term", "2026-1", "--building", "비마", "--user", "20260000"})
	if err != nil {
		t.Fatalf("roomIndexOptions() error = %v", err)
	}
	if opts.TermValue != "2026-1" || opts.Building != "비마" || opts.User.StudentID != "20260000" {
		t.Fatalf("roomIndexOptions() = %+v", opts)
	}
	if _, err := roomIndexOptions([]string{"비마관"}); err == nil {
		t.Fatal("roomIndexOptions() expected unknown positional error")
	}
}

func TestRoomAvailableOptions(t *testing.T) {
	opts, err := roomAvailableOptions([]string{"--day", "금요일", "--duration", "1-3", "--term", "2026-1", "--building", "새빛", "--refresh", "--user", "20260000"})
	if err != nil {
		t.Fatalf("roomAvailableOptions() error = %v", err)
	}
	if opts.Day != "금요일" || opts.Duration != "1-3" || opts.TermValue != "2026-1" || opts.Building != "새빛" || !opts.Refresh || opts.User.StudentID != "20260000" {
		t.Fatalf("roomAvailableOptions() = %+v", opts)
	}
	if _, err := roomAvailableOptions([]string{"--day", "금"}); err == nil {
		t.Fatal("roomAvailableOptions() expected missing duration error")
	}
}

func TestRenderProgressBar(t *testing.T) {
	got := renderProgressBar(25, 12)
	if got == "" || !strings.Contains(got, "25%") {
		t.Fatalf("renderProgressBar() = %q", got)
	}
}

func TestLectureProgressPrinterUpdatesCurrentTerminalLine(t *testing.T) {
	var output bytes.Buffer
	printer := newLectureProgressPrinter(&output, true)
	row := app.LectureRow{
		ID:         "1:video",
		CourseName: "자료구조",
		Lecture:    app.Lecture{Title: "01-DS-preliminary"},
	}

	printer.Print(row, app.LectureProgress{Progress: 2, TotalTime: "1", PTime: "41"})
	printer.Print(row, app.LectureProgress{Progress: 5, TotalTime: "2", PTime: "41"})
	printer.Clear()

	got := output.String()
	if strings.Contains(got, "\n") {
		t.Fatalf("inline progress contains newline: %q", got)
	}
	if count := strings.Count(got, "\r\x1b[2K"); count != 3 {
		t.Fatalf("clear-line sequence count = %d, output = %q", count, got)
	}
	if !strings.Contains(got, "2% 1/41분") || !strings.Contains(got, "5% 2/41분") {
		t.Fatalf("inline progress output = %q", got)
	}
	if printer.active {
		t.Fatal("Clear() should reset active state")
	}
}

func TestLectureProgressPrinterKeepsLineLogsWhenNotInteractive(t *testing.T) {
	var output bytes.Buffer
	printer := newLectureProgressPrinter(&output, false)
	row := app.LectureRow{ID: "1:video", Lecture: app.Lecture{Title: "영상"}}

	printer.Print(row, app.LectureProgress{Progress: 25})
	printer.Print(row, app.LectureProgress{Progress: 50})
	printer.Clear()

	got := output.String()
	if strings.Count(got, "\n") != 2 {
		t.Fatalf("non-interactive progress output = %q", got)
	}
	if strings.Contains(got, "\r") || strings.Contains(got, "\x1b[2K") {
		t.Fatalf("non-interactive output contains terminal controls: %q", got)
	}
}

func TestLectureProgressPrinterClearBeforeFirstUpdateDoesNothing(t *testing.T) {
	var output bytes.Buffer
	printer := newLectureProgressPrinter(&output, true)
	printer.Clear()
	if output.Len() != 0 {
		t.Fatalf("Clear() output before progress = %q", output.String())
	}
}

func TestLectureStatusPercent(t *testing.T) {
	if got := lectureStatusPercent(app.Lecture{ContentID: "content", Progress: "75"}); got != 75 {
		t.Fatalf("lectureStatusPercent() video = %v", got)
	}
	if got := lectureStatusPercent(app.Lecture{LearningSeq: "10", AchievedTime: "5", RequiredTime: "10"}); got != 50 {
		t.Fatalf("lectureStatusPercent() activity = %v", got)
	}
}

func TestHyperlinkURLs(t *testing.T) {
	got := hyperlinkURLs("링크: https://example.com/path.")
	want := "링크: \x1b]8;;https://example.com/path\x1b\\https://example.com/path\x1b]8;;\x1b\\."
	if got != want {
		t.Fatalf("hyperlinkURLs() = %q, want %q", got, want)
	}
}
