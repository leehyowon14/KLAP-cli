package app

import (
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/category"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
	"strconv"
	"strings"
)

type DownloadSettings struct {
	Dir         string
	Concurrency int
	Caffeinate  bool
	KeepPartial bool
}

type TranscriptSettings struct {
	Concurrency int
}

type ConfigSettings struct {
	Reminder   ReminderSettings
	Calendar   CalendarSettings
	Download   DownloadSettings
	Transcript TranscriptSettings
	Term       TermSettings
}

type ReminderSettings struct {
	ListName        string
	UseExistingList bool
	AlarmBeforeMin  int
}

type CalendarSettings struct {
	Name                     string
	UseExistingList          bool
	TimetableName            string
	TimetableUseExistingList bool
}

type CategoryOptions struct {
	Reminders []string
	Calendars []string
}

func (s *Service) ReminderSettings() (ReminderSettings, error) {
	current, err := s.loadSettings()
	if err != nil {
		return ReminderSettings{}, err
	}
	return ReminderSettings{
		ListName:        current.Reminder.ListName,
		UseExistingList: current.Reminder.UseExistingList,
		AlarmBeforeMin:  current.Reminder.AlarmBeforeMin,
	}, nil
}

func (s *Service) SetReminderConfig(name string, useExistingList bool) (ReminderSettings, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ReminderSettings{}, errors.New("리마인더 목록 이름은 비워둘 수 없습니다")
	}

	current, err := s.loadSettings()
	if err != nil {
		return ReminderSettings{}, err
	}
	current.Reminder.ListName = name
	current.Reminder.UseExistingList = useExistingList
	if err := s.saveSettings(current); err != nil {
		return ReminderSettings{}, err
	}
	return ReminderSettings{
		ListName:        current.Reminder.ListName,
		UseExistingList: current.Reminder.UseExistingList,
		AlarmBeforeMin:  current.Reminder.AlarmBeforeMin,
	}, nil
}

func (s *Service) SetCalendarConfig(name string, useExistingList bool) (CalendarSettings, error) {
	return s.SetAcademicCalendarConfig(name, useExistingList)
}

func (s *Service) SetAcademicCalendarConfig(name string, useExistingList bool) (CalendarSettings, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CalendarSettings{}, errors.New("캘린더 이름은 비워둘 수 없습니다")
	}
	current, err := s.loadSettings()
	if err != nil {
		return CalendarSettings{}, err
	}
	current.Calendar.AcademicName = name
	current.Calendar.AcademicUseExistingList = useExistingList
	if err := s.saveSettings(current); err != nil {
		return CalendarSettings{}, err
	}
	return calendarSettingsFrom(current), nil
}

func (s *Service) SetTimetableCalendarConfig(name string, useExistingList bool) (CalendarSettings, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CalendarSettings{}, errors.New("시간표 캘린더 이름은 비워둘 수 없습니다")
	}
	current, err := s.loadSettings()
	if err != nil {
		return CalendarSettings{}, err
	}
	current.Calendar.TimetableName = name
	current.Calendar.TimetableUseExistingList = useExistingList
	if err := s.saveSettings(current); err != nil {
		return CalendarSettings{}, err
	}
	return calendarSettingsFrom(current), nil
}

func (s *Service) DownloadSettings() (DownloadSettings, error) {
	current, err := s.loadSettings()
	if err != nil {
		return DownloadSettings{}, err
	}
	return downloadSettingsFrom(current), nil
}

func (s *Service) TranscriptSettings() (TranscriptSettings, error) {
	current, err := s.loadSettings()
	if err != nil {
		return TranscriptSettings{}, err
	}
	return transcriptSettingsFrom(current), nil
}

func downloadSettingsFrom(current settings.Settings) DownloadSettings {
	return DownloadSettings{
		Dir:         current.Download.Dir,
		Concurrency: current.Download.Concurrency,
		Caffeinate:  settings.DownloadCaffeinateEnabled(current.Download),
		KeepPartial: current.Download.KeepPartial,
	}
}

func transcriptSettingsFrom(current settings.Settings) TranscriptSettings {
	return TranscriptSettings{Concurrency: current.Transcript.Concurrency}
}

func calendarSettingsFrom(current settings.Settings) CalendarSettings {
	current.Normalize()
	return CalendarSettings{
		Name:                     current.Calendar.AcademicName,
		UseExistingList:          current.Calendar.AcademicUseExistingList,
		TimetableName:            current.Calendar.TimetableName,
		TimetableUseExistingList: current.Calendar.TimetableUseExistingList,
	}
}

func (s *Service) SetDownloadDir(dir string) (DownloadSettings, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return DownloadSettings{}, errors.New("다운로드 폴더 경로가 필요합니다")
	}
	current, err := s.loadSettings()
	if err != nil {
		return DownloadSettings{}, err
	}
	current.Download.Dir = dir
	if err := s.saveSettings(current); err != nil {
		return DownloadSettings{}, err
	}
	return downloadSettingsFrom(current), nil
}

func (s *Service) SetDownloadConfig(dir string, concurrency int, caffeinate *bool, keepPartial *bool) (DownloadSettings, error) {
	current, err := s.loadSettings()
	if err != nil {
		return DownloadSettings{}, err
	}
	if strings.TrimSpace(dir) != "" {
		current.Download.Dir = strings.TrimSpace(dir)
	}
	if concurrency > 0 {
		current.Download.Concurrency = concurrency
	}
	if caffeinate != nil {
		current.Download.Caffeinate = caffeinate
	}
	if keepPartial != nil {
		current.Download.KeepPartial = *keepPartial
	}
	if err := s.saveSettings(current); err != nil {
		return DownloadSettings{}, err
	}
	return downloadSettingsFrom(current), nil
}

func (s *Service) SetTranscriptConfig(concurrency int) (TranscriptSettings, error) {
	if concurrency < 1 || concurrency > settings.MaxTranscriptConcurrency {
		return TranscriptSettings{}, fmt.Errorf("transcript.concurrency에는 1~%d 사이의 정수가 필요합니다", settings.MaxTranscriptConcurrency)
	}
	current, err := s.loadSettings()
	if err != nil {
		return TranscriptSettings{}, err
	}
	current.Transcript.Concurrency = concurrency
	if err := s.saveSettings(current); err != nil {
		return TranscriptSettings{}, err
	}
	return transcriptSettingsFrom(current), nil
}

func (s *Service) ConfigSettings() (ConfigSettings, error) {
	current, err := s.loadSettings()
	if err != nil {
		return ConfigSettings{}, err
	}
	return ConfigSettings{
		Reminder: ReminderSettings{
			ListName:        current.Reminder.ListName,
			UseExistingList: current.Reminder.UseExistingList,
			AlarmBeforeMin:  current.Reminder.AlarmBeforeMin,
		},
		Calendar:   calendarSettingsFrom(current),
		Download:   downloadSettingsFrom(current),
		Transcript: transcriptSettingsFrom(current),
		Term:       TermSettings{Value: current.Term.Value, Label: termLabel(current.Term.Value)},
	}, nil
}

func (s *Service) CategoryOptions() (CategoryOptions, error) {
	current, err := s.loadSettings()
	if err != nil {
		return CategoryOptions{}, err
	}
	options, err := s.categoryLister.List()
	if err != nil {
		return CategoryOptions{
			Reminders: uniqueNonEmpty(current.Reminder.ListName, settings.DefaultReminderListName),
			Calendars: uniqueNonEmpty(current.Calendar.AcademicName, current.Calendar.TimetableName, settings.DefaultAcademicCalendarName, settings.DefaultTimetableCalendarName),
		}, nil
	}
	return CategoryOptions{
		Reminders: uniqueNonEmpty(append([]string{current.Reminder.ListName}, options.Reminders...)...),
		Calendars: uniqueNonEmpty(append([]string{current.Calendar.AcademicName, current.Calendar.TimetableName}, options.Calendars...)...),
	}, nil
}

func (s *Service) ResetConfigSettings() (ConfigSettings, error) {
	if err := s.saveSettings(settings.Default()); err != nil {
		return ConfigSettings{}, err
	}
	return s.ConfigSettings()
}

func (s *Service) SetConfigValue(key string, value string) (ConfigSettings, error) {
	key = strings.ToLower(strings.TrimSpace(key))
	value = strings.TrimSpace(value)
	if key == "" {
		return ConfigSettings{}, errors.New("설정 키가 필요합니다")
	}
	current, err := s.loadSettings()
	if err != nil {
		return ConfigSettings{}, err
	}
	switch key {
	case "reminder.name", "reminder.list", "reminder.list-name":
		if value == "" {
			return ConfigSettings{}, errors.New("리마인더 목록 이름은 비워둘 수 없습니다")
		}
		current.Reminder.ListName = value
	case "reminder.use-existing-list":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return ConfigSettings{}, err
		}
		current.Reminder.UseExistingList = parsed
	case "reminder.alarm-before-min", "reminder.alarm-before":
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return ConfigSettings{}, errors.New("reminder.alarm-before-min에는 1 이상의 정수가 필요합니다")
		}
		current.Reminder.AlarmBeforeMin = parsed
	case "calendar.name", "calendar.list", "calendar.list-name":
		if value == "" {
			return ConfigSettings{}, errors.New("캘린더 이름은 비워둘 수 없습니다")
		}
		current.Calendar.AcademicName = value
	case "calendar.use-existing-list":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return ConfigSettings{}, err
		}
		current.Calendar.AcademicUseExistingList = parsed
	case "calendar.academic.name", "academic-calendar.name", "academic-calendar.list":
		if value == "" {
			return ConfigSettings{}, errors.New("학사일정 캘린더 이름은 비워둘 수 없습니다")
		}
		current.Calendar.AcademicName = value
	case "calendar.academic.use-existing-list", "academic-calendar.use-existing-list":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return ConfigSettings{}, err
		}
		current.Calendar.AcademicUseExistingList = parsed
	case "calendar.timetable.name", "timetable-calendar.name", "timetable-calendar.list":
		if value == "" {
			return ConfigSettings{}, errors.New("시간표 캘린더 이름은 비워둘 수 없습니다")
		}
		current.Calendar.TimetableName = value
	case "calendar.timetable.use-existing-list", "timetable-calendar.use-existing-list":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return ConfigSettings{}, err
		}
		current.Calendar.TimetableUseExistingList = parsed
	case "download.dir", "download.path":
		if value == "" {
			return ConfigSettings{}, errors.New("다운로드 폴더 경로가 필요합니다")
		}
		current.Download.Dir = value
	case "download.concurrency", "download.workers":
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return ConfigSettings{}, errors.New("download.concurrency에는 1 이상의 정수가 필요합니다")
		}
		current.Download.Concurrency = parsed
	case "download.caffeinate", "download.prevent-sleep", "download.keep-awake":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return ConfigSettings{}, err
		}
		current.Download.Caffeinate = &parsed
	case "download.keep-partial", "download.resume", "download.partial":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return ConfigSettings{}, err
		}
		current.Download.KeepPartial = parsed
	case "transcript.concurrency", "transcript.workers":
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > settings.MaxTranscriptConcurrency {
			return ConfigSettings{}, fmt.Errorf("transcript.concurrency에는 1~%d 사이의 정수가 필요합니다", settings.MaxTranscriptConcurrency)
		}
		current.Transcript.Concurrency = parsed
	case "term", "term.value":
		normalized, err := normalizeTermValue(value)
		if err != nil {
			return ConfigSettings{}, err
		}
		current.Term.Value = normalized
	default:
		return ConfigSettings{}, fmt.Errorf("지원하지 않는 설정 키입니다: %s", key)
	}
	if err := s.saveSettings(current); err != nil {
		return ConfigSettings{}, err
	}
	return s.ConfigSettings()
}

func (s *Service) loadSettings() (settings.Settings, error) {
	if s.settingsStore == nil {
		return settings.Default(), nil
	}
	return s.settingsStore.Load()
}

func (s *Service) saveSettings(value settings.Settings) error {
	if s.settingsStore == nil {
		return errors.New("settings store가 초기화되지 않았습니다")
	}
	return s.settingsStore.Save(value)
}

func parseConfigBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "t", "1", "yes", "y", "on":
		return true, nil
	case "false", "f", "0", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("boolean 값이 필요합니다: %s", value)
	}
}

type CategoryLister interface {
	List() (category.Options, error)
}
