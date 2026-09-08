package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultReminderListName = "Kwangwoon Univ."

const DefaultAcademicCalendarName = "학사일정"

const DefaultTimetableCalendarName = "시간표"

const DefaultDownloadConcurrency = 3

const DefaultTranscriptConcurrency = 1

const MaxTranscriptConcurrency = 3

type Store struct {
	path string
}

func NewStore() (*Store, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("config dir 생성 실패: %w", err)
	}
	return &Store{path: filepath.Join(dir, "settings.json")}, nil
}

func (s *Store) Load() (Settings, error) {
	bytes, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("settings 읽기 실패: %w", err)
	}

	settings := Default()
	if err := json.Unmarshal(bytes, &settings); err != nil {
		return Settings{}, fmt.Errorf("settings 파싱 실패: %w", err)
	}
	settings.Normalize()
	return settings, nil
}

func (s *Store) Save(settings Settings) error {
	settings.Normalize()
	bytes, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("settings 직렬화 실패: %w", err)
	}
	if err := os.WriteFile(s.path, bytes, 0o600); err != nil {
		return fmt.Errorf("settings 저장 실패: %w", err)
	}
	return nil
}

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

func (s *Settings) Normalize() {
	if s.Reminder.ListName == "" {
		s.Reminder.ListName = DefaultReminderListName
	}
	if s.Reminder.AlarmBeforeMin <= 0 {
		s.Reminder.AlarmBeforeMin = 24 * 60
	}
	s.Calendar.Normalize()
	s.Download.Normalize()
	s.Transcript.Normalize()
}

func (c *Calendar) Normalize() {
	if c.AcademicName == "" {
		if c.Name != "" {
			c.AcademicName = c.Name
			c.AcademicUseExistingList = c.UseExistingList
		} else {
			c.AcademicName = DefaultAcademicCalendarName
		}
	}
	if c.TimetableName == "" {
		c.TimetableName = DefaultTimetableCalendarName
	}
	c.Name = c.AcademicName
	c.UseExistingList = c.AcademicUseExistingList
}

func DefaultDownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join("Documents", "KLAP")
	}
	return filepath.Join(home, "Documents", "KLAP")
}

func DownloadCaffeinateEnabled(download Download) bool {
	download.Normalize()
	return download.Caffeinate != nil && *download.Caffeinate
}

func (d *Download) Normalize() {
	if d.Dir == "" || d.Dir == "downloads" {
		d.Dir = DefaultDownloadDir()
	}
	if d.Concurrency <= 0 {
		d.Concurrency = DefaultDownloadConcurrency
	}
	if d.Caffeinate == nil {
		d.Caffeinate = boolPtr(true)
	}
}

func (t *Transcript) Normalize() {
	if t.Concurrency <= 0 {
		t.Concurrency = DefaultTranscriptConcurrency
	}
	if t.Concurrency > MaxTranscriptConcurrency {
		t.Concurrency = MaxTranscriptConcurrency
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func configDir() (string, error) {
	if override := os.Getenv("KLAP_CONFIG_DIR"); override != "" {
		return override, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config dir 확인 실패: %w", err)
	}
	return filepath.Join(configDir, "klap"), nil
}
