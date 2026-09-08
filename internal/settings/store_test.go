package settings

import (
	"testing"
)

func TestNormalizeReminderSettings(t *testing.T) {
	settings := Settings{}
	settings.Normalize()
	if settings.Reminder.ListName != "Kwangwoon Univ." {
		t.Fatalf("ListName = %q", settings.Reminder.ListName)
	}
	if settings.Reminder.AlarmBeforeMin != 1440 {
		t.Fatalf("AlarmBeforeMin = %d", settings.Reminder.AlarmBeforeMin)
	}
	if settings.Calendar.Name != DefaultAcademicCalendarName {
		t.Fatalf("Calendar.Name = %q", settings.Calendar.Name)
	}
	if settings.Calendar.TimetableName != DefaultTimetableCalendarName {
		t.Fatalf("Calendar.TimetableName = %q", settings.Calendar.TimetableName)
	}
	if settings.Download.Dir != DefaultDownloadDir() {
		t.Fatalf("Download.Dir = %q", settings.Download.Dir)
	}
	if settings.Download.Concurrency != DefaultDownloadConcurrency {
		t.Fatalf("Download.Concurrency = %d", settings.Download.Concurrency)
	}
	if !DownloadCaffeinateEnabled(settings.Download) {
		t.Fatal("Download.Caffeinate should normalize to true")
	}
	if settings.Download.KeepPartial {
		t.Fatal("Download.KeepPartial should normalize to false")
	}
	if settings.Transcript.Concurrency != DefaultTranscriptConcurrency {
		t.Fatalf("Transcript.Concurrency = %d", settings.Transcript.Concurrency)
	}
}

func TestNormalizeMigratesLegacyCalendarName(t *testing.T) {
	settings := Settings{Calendar: Calendar{Name: "기존 캘린더", UseExistingList: true}}
	settings.Normalize()
	if settings.Calendar.AcademicName != "기존 캘린더" || settings.Calendar.Name != "기존 캘린더" {
		t.Fatalf("Calendar academic migration = %+v", settings.Calendar)
	}
	if !settings.Calendar.AcademicUseExistingList || !settings.Calendar.UseExistingList {
		t.Fatalf("Calendar use-existing migration = %+v", settings.Calendar)
	}
	if settings.Calendar.TimetableName != DefaultTimetableCalendarName {
		t.Fatalf("Calendar.TimetableName = %q", settings.Calendar.TimetableName)
	}
}

func TestNormalizeCapsTranscriptConcurrency(t *testing.T) {
	settings := Settings{Transcript: Transcript{Concurrency: MaxTranscriptConcurrency + 10}}
	settings.Normalize()
	if settings.Transcript.Concurrency != MaxTranscriptConcurrency {
		t.Fatalf("Transcript.Concurrency = %d", settings.Transcript.Concurrency)
	}
}

func TestNormalizeMigratesLegacyDownloadDefault(t *testing.T) {
	settings := Settings{Download: Download{Dir: "downloads"}}
	settings.Normalize()
	if settings.Download.Dir != DefaultDownloadDir() {
		t.Fatalf("Download.Dir = %q", settings.Download.Dir)
	}
}

func TestNormalizePreservesExplicitCaffeinateFalse(t *testing.T) {
	disabled := false
	settings := Settings{Download: Download{Caffeinate: &disabled}}
	settings.Normalize()
	if DownloadCaffeinateEnabled(settings.Download) {
		t.Fatal("Download.Caffeinate explicit false should be preserved")
	}
}
