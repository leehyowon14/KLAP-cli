package settings

import (
	"os"
	"path/filepath"
)

const DefaultReminderListName = "Kwangwoon Univ."

const DefaultAcademicCalendarName = "학사일정"

const DefaultTimetableCalendarName = "시간표"

const DefaultDownloadConcurrency = 3

const DefaultTranscriptConcurrency = 1

const MaxTranscriptConcurrency = 3

func Default() Settings {
	return Settings{
		Reminder: Reminder{
			ListName:       DefaultReminderListName,
			AlarmBeforeMin: 24 * 60,
		},
		Calendar: Calendar{
			Name:          DefaultAcademicCalendarName,
			AcademicName:  DefaultAcademicCalendarName,
			TimetableName: DefaultTimetableCalendarName,
		},
		Download: Download{
			Dir:         DefaultDownloadDir(),
			Concurrency: DefaultDownloadConcurrency,
			Caffeinate:  boolPtr(true),
		},
		Transcript: Transcript{
			Concurrency: DefaultTranscriptConcurrency,
		},
	}
}

func DefaultDownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join("Documents", "KLAP")
	}
	return filepath.Join(home, "Documents", "KLAP")
}

func boolPtr(value bool) *bool {
	return &value
}
