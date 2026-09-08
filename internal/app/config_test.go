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
	if _, err := service.UpdateConfig(ConfigUpdate{ReminderName: configPointer("To-do")}); err != nil {
		t.Fatalf("UpdateConfig(reminder.name) error = %v", err)
	}
	if _, err := service.UpdateConfig(ConfigUpdate{AcademicCalendarName: configPointer("시간표")}); err != nil {
		t.Fatalf("UpdateConfig(calendar.name) error = %v", err)
	}
	if _, err := service.UpdateConfig(ConfigUpdate{TimetableCalendarName: configPointer("수업시간표")}); err != nil {
		t.Fatalf("UpdateConfig(timetable-calendar.name) error = %v", err)
	}
	if _, err := service.UpdateConfig(ConfigUpdate{ReminderAlarmBeforeMin: configPointer(60)}); err != nil {
		t.Fatalf("UpdateConfig(reminder.alarm-before-min) error = %v", err)
	}
	if _, err := service.UpdateConfig(ConfigUpdate{DownloadConcurrency: configPointer(9)}); err != nil {
		t.Fatalf("UpdateConfig(download.concurrency) error = %v", err)
	}
	if _, err := service.UpdateConfig(ConfigUpdate{DownloadCaffeinate: configPointer(false)}); err != nil {
		t.Fatalf("UpdateConfig(download.caffeinate) error = %v", err)
	}
	if _, err := service.UpdateConfig(ConfigUpdate{DownloadKeepPartial: configPointer(true)}); err != nil {
		t.Fatalf("UpdateConfig(download.keep-partial) error = %v", err)
	}
	if _, err := service.UpdateConfig(ConfigUpdate{TranscriptConcurrency: configPointer(3)}); err != nil {
		t.Fatalf("UpdateConfig(transcript.concurrency) error = %v", err)
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

func TestUpdateConfigRejectsInvalidTranscriptConcurrency(t *testing.T) {
	t.Setenv("KLAP_CONFIG_DIR", t.TempDir())
	settingsStore, err := settings.NewStore()
	if err != nil {
		t.Fatalf("settings.NewStore() error = %v", err)
	}
	service := &Service{settingsStore: settingsStore}
	if _, err := service.UpdateConfig(ConfigUpdate{TranscriptConcurrency: configPointer(4)}); err == nil {
		t.Fatal("UpdateConfig(transcript.concurrency) expected error")
	}
}
