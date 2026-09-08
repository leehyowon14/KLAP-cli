package settings

import (
	"testing"
)

func TestDefaultReminderSettings(t *testing.T) {
	got := Default()
	if got.Reminder.ListName != "Kwangwoon Univ." {
		t.Fatalf("ListName = %q", got.Reminder.ListName)
	}
	if got.Reminder.UseExistingList {
		t.Fatal("UseExistingList should default to false")
	}
	if got.Reminder.AlarmBeforeMin != 1440 {
		t.Fatalf("AlarmBeforeMin = %d", got.Reminder.AlarmBeforeMin)
	}
	if got.Calendar.Name != DefaultAcademicCalendarName {
		t.Fatalf("Calendar.Name = %q", got.Calendar.Name)
	}
	if got.Calendar.TimetableName != DefaultTimetableCalendarName {
		t.Fatalf("Calendar.TimetableName = %q", got.Calendar.TimetableName)
	}
	if got.Calendar.UseExistingList {
		t.Fatal("Calendar.UseExistingList should default to false")
	}
	if got.Term.Value != "" {
		t.Fatalf("Term.Value = %q", got.Term.Value)
	}
	if got.Download.Dir != DefaultDownloadDir() {
		t.Fatalf("Download.Dir = %q", got.Download.Dir)
	}
	if got.Download.Concurrency != DefaultDownloadConcurrency {
		t.Fatalf("Download.Concurrency = %d", got.Download.Concurrency)
	}
	if !DownloadCaffeinateEnabled(got.Download) {
		t.Fatal("Download.Caffeinate should default to true")
	}
	if got.Download.KeepPartial {
		t.Fatal("Download.KeepPartial should default to false")
	}
	if got.Transcript.Concurrency != DefaultTranscriptConcurrency {
		t.Fatalf("Transcript.Concurrency = %d", got.Transcript.Concurrency)
	}
}
