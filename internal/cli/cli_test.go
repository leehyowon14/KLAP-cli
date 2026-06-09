package cli

import (
	"context"
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

func TestSearchOptions(t *testing.T) {
	opts, err := searchOptions([]string{"기말", "--type", "assignment", "--refresh", "--user", "20260000"})
	if err != nil {
		t.Fatalf("searchOptions() error = %v", err)
	}
	if opts.Query != "기말" || opts.Type != "assignment" || !opts.Refresh || opts.User.StudentID != "20260000" {
		t.Fatalf("searchOptions() = %+v", opts)
	}
	if _, err := searchOptions([]string{}); err == nil {
		t.Fatal("searchOptions() expected error")
	}
}

func TestDueOptions(t *testing.T) {
	opts, err := dueOptions([]string{"--week", "--refresh", "--user", "20260000"})
	if err != nil {
		t.Fatalf("dueOptions() error = %v", err)
	}
	if opts.Days != 7 || !opts.Refresh || opts.User.StudentID != "20260000" {
		t.Fatalf("dueOptions() = %+v", opts)
	}
	opts, err = dueOptions([]string{"--days", "30"})
	if err != nil {
		t.Fatalf("dueOptions() days error = %v", err)
	}
	if opts.Days != 30 {
		t.Fatalf("dueOptions() days = %d", opts.Days)
	}
}

func TestParseDownloadConfigArgs(t *testing.T) {
	opts, ok, err := parseDownloadConfigArgs([]string{"--dir", "downloads/course", "--concurrency", "12", "--no-caffeinate", "--keep-partial"})
	if err != nil {
		t.Fatalf("parseDownloadConfigArgs() error = %v", err)
	}
	if !ok || opts.Dir != "downloads/course" || opts.Concurrency != 12 || opts.Caffeinate == nil || *opts.Caffeinate || opts.KeepPartial == nil || !*opts.KeepPartial {
		t.Fatalf("parseDownloadConfigArgs() = %+v, %v", opts, ok)
	}
	if _, _, err := parseDownloadConfigArgs([]string{"--concurrency", "0"}); err == nil {
		t.Fatal("parseDownloadConfigArgs() expected concurrency error")
	}
	if _, _, err := parseDownloadConfigArgs([]string{"--bad"}); err == nil {
		t.Fatal("parseDownloadConfigArgs() expected error")
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

func TestConfigResetRejectsExtraArgs(t *testing.T) {
	err := runConfig(context.Background(), nil, []string{"reset", "download"})
	if err == nil || !strings.Contains(err.Error(), "klap config reset") {
		t.Fatalf("runConfig(reset extra) error = %v", err)
	}
}

func TestHyperlinkURLs(t *testing.T) {
	got := hyperlinkURLs("링크: https://example.com/path.")
	want := "링크: \x1b]8;;https://example.com/path\x1b\\https://example.com/path\x1b]8;;\x1b\\."
	if got != want {
		t.Fatalf("hyperlinkURLs() = %q, want %q", got, want)
	}
}
