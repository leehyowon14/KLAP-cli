package settings

import "testing"

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

func TestNormalizeReminderSettings(t *testing.T) {
	settings := Settings{}
	settings.Normalize()
	if settings.Reminder.ListName != "Kwangwoon Univ." {
		t.Fatalf("ListName = %q", settings.Reminder.ListName)
	}
	if settings.Reminder.AlarmBeforeMin != 1440 {
		t.Fatalf("AlarmBeforeMin = %d", settings.Reminder.AlarmBeforeMin)
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
