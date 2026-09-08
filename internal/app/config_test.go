package app

import (
	"github.com/leehyowon14/KLAP-cli/internal/settings"
	"testing"
)

func TestResetConfigSettingsRestoresDefaults(t *testing.T) {
	t.Setenv("KLAP_CONFIG_DIR", t.TempDir())
	settingsStore, err := settings.NewStore()
	if err != nil {
		t.Fatalf("settings.NewStore() error = %v", err)
	}
	service := &Service{settingsStore: settingsStore}
	if _, err := service.SetConfigValue("reminder.name", "To-do"); err != nil {
		t.Fatalf("SetConfigValue(reminder.name) error = %v", err)
	}
	if _, err := service.SetConfigValue("calendar.name", "시간표"); err != nil {
		t.Fatalf("SetConfigValue(calendar.name) error = %v", err)
	}
	if _, err := service.SetConfigValue("timetable-calendar.name", "수업시간표"); err != nil {
		t.Fatalf("SetConfigValue(timetable-calendar.name) error = %v", err)
	}
	if _, err := service.SetConfigValue("reminder.alarm-before-min", "60"); err != nil {
		t.Fatalf("SetConfigValue(reminder.alarm-before-min) error = %v", err)
	}
	if _, err := service.SetConfigValue("download.concurrency", "9"); err != nil {
		t.Fatalf("SetConfigValue(download.concurrency) error = %v", err)
	}
	if _, err := service.SetConfigValue("download.caffeinate", "false"); err != nil {
		t.Fatalf("SetConfigValue(download.caffeinate) error = %v", err)
	}
	if _, err := service.SetConfigValue("download.keep-partial", "true"); err != nil {
		t.Fatalf("SetConfigValue(download.keep-partial) error = %v", err)
	}
	if _, err := service.SetConfigValue("transcript.concurrency", "3"); err != nil {
		t.Fatalf("SetConfigValue(transcript.concurrency) error = %v", err)
	}

	got, err := service.ResetConfigSettings()
	if err != nil {
		t.Fatalf("ResetConfigSettings() error = %v", err)
	}
	if got.Reminder.ListName != settings.DefaultReminderListName {
		t.Fatalf("Reminder.ListName = %q", got.Reminder.ListName)
	}
	if got.Reminder.AlarmBeforeMin != 24*60 {
		t.Fatalf("Reminder.AlarmBeforeMin = %d", got.Reminder.AlarmBeforeMin)
	}
	if got.Calendar.Name != settings.DefaultAcademicCalendarName {
		t.Fatalf("Calendar.Name = %q", got.Calendar.Name)
	}
	if got.Calendar.TimetableName != settings.DefaultTimetableCalendarName {
		t.Fatalf("Calendar.TimetableName = %q", got.Calendar.TimetableName)
	}
	if got.Download.Dir != settings.DefaultDownloadDir() || got.Download.Concurrency != settings.DefaultDownloadConcurrency {
		t.Fatalf("Download = %+v", got.Download)
	}
	if !got.Download.Caffeinate {
		t.Fatal("Download.Caffeinate should reset to true")
	}
	if got.Download.KeepPartial {
		t.Fatal("Download.KeepPartial should reset to false")
	}
	if got.Transcript.Concurrency != settings.DefaultTranscriptConcurrency {
		t.Fatalf("Transcript.Concurrency = %d", got.Transcript.Concurrency)
	}
	if got.Term.Value != "" {
		t.Fatalf("Term.Value = %q", got.Term.Value)
	}
}

func TestSetConfigValueRejectsInvalidTranscriptConcurrency(t *testing.T) {
	t.Setenv("KLAP_CONFIG_DIR", t.TempDir())
	settingsStore, err := settings.NewStore()
	if err != nil {
		t.Fatalf("settings.NewStore() error = %v", err)
	}
	service := &Service{settingsStore: settingsStore}
	if _, err := service.SetConfigValue("transcript.concurrency", "4"); err == nil {
		t.Fatal("SetConfigValue(transcript.concurrency) expected error")
	}
}

func TestParseConfigBool(t *testing.T) {
	got, err := parseConfigBool("yes")
	if err != nil {
		t.Fatalf("parseConfigBool() error = %v", err)
	}
	if !got {
		t.Fatal("parseConfigBool() expected true")
	}
	got, err = parseConfigBool("off")
	if err != nil {
		t.Fatalf("parseConfigBool() off error = %v", err)
	}
	if got {
		t.Fatal("parseConfigBool() expected false")
	}
	if _, err := parseConfigBool("maybe"); err == nil {
		t.Fatal("parseConfigBool() expected error")
	}
}
