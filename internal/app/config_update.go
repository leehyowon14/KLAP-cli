package app

import (
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
	"strings"
)

const MaxTranscriptConcurrency = settings.MaxTranscriptConcurrency

// ConfigUpdate changes only explicitly supplied fields. Validation completes
// before persistence, so an invalid multi-field update never partially saves.
type ConfigUpdate struct {
	ReminderName                     *string
	ReminderUseExistingList          *bool
	ReminderAlarmBeforeMin           *int
	AcademicCalendarName             *string
	AcademicCalendarUseExistingList  *bool
	TimetableCalendarName            *string
	TimetableCalendarUseExistingList *bool
	DownloadDir                      *string
	DownloadConcurrency              *int
	DownloadCaffeinate               *bool
	DownloadKeepPartial              *bool
	TranscriptConcurrency            *int
	Term                             *string
}

func (s *Service) UpdateConfig(update ConfigUpdate) (ConfigSettings, error) {
	current, err := s.loadSettings()
	if err != nil {
		return ConfigSettings{}, err
	}
	if update.ReminderName != nil {
		value := *update.ReminderName
		value = strings.TrimSpace(value)
		if value == "" {
			return ConfigSettings{}, errors.New("리마인더 목록 이름은 비워둘 수 없습니다")
		}
		current.Reminder.ListName = value
	}
	if update.ReminderUseExistingList != nil {
		value := *update.ReminderUseExistingList
		current.Reminder.UseExistingList = value
	}
	if update.ReminderAlarmBeforeMin != nil {
		value := *update.ReminderAlarmBeforeMin
		if value < 1 {
			return ConfigSettings{}, errors.New("리마인더 알림 시간이 허용 범위를 벗어났습니다")
		}
		current.Reminder.AlarmBeforeMin = value
	}
	if update.AcademicCalendarName != nil {
		value := *update.AcademicCalendarName
		value = strings.TrimSpace(value)
		if value == "" {
			return ConfigSettings{}, errors.New("학사일정 캘린더 이름은 비워둘 수 없습니다")
		}
		current.Calendar.AcademicName = value
	}
	if update.AcademicCalendarUseExistingList != nil {
		value := *update.AcademicCalendarUseExistingList
		current.Calendar.AcademicUseExistingList = value
	}
	if update.TimetableCalendarName != nil {
		value := *update.TimetableCalendarName
		value = strings.TrimSpace(value)
		if value == "" {
			return ConfigSettings{}, errors.New("시간표 캘린더 이름은 비워둘 수 없습니다")
		}
		current.Calendar.TimetableName = value
	}
	if update.TimetableCalendarUseExistingList != nil {
		value := *update.TimetableCalendarUseExistingList
		current.Calendar.TimetableUseExistingList = value
	}
	if update.DownloadDir != nil {
		value := *update.DownloadDir
		value = strings.TrimSpace(value)
		if value == "" {
			return ConfigSettings{}, errors.New("다운로드 폴더 경로는 비워둘 수 없습니다")
		}
		current.Download.Dir = value
	}
	if update.DownloadConcurrency != nil {
		value := *update.DownloadConcurrency
		if value < 1 {
			return ConfigSettings{}, errors.New("다운로드 동시 작업 수가 허용 범위를 벗어났습니다")
		}
		current.Download.Concurrency = value
	}
	if update.DownloadCaffeinate != nil {
		value := *update.DownloadCaffeinate
		current.Download.Caffeinate = &value
	}
	if update.DownloadKeepPartial != nil {
		value := *update.DownloadKeepPartial
		current.Download.KeepPartial = value
	}
	if update.TranscriptConcurrency != nil {
		value := *update.TranscriptConcurrency
		if value < 1 || value > MaxTranscriptConcurrency {
			return ConfigSettings{}, errors.New("전사 동시 작업 수가 허용 범위를 벗어났습니다")
		}
		current.Transcript.Concurrency = value
	}
	if update.Term != nil {
		value := *update.Term
		normalized, err := normalizeTermValue(value)
		if err != nil {
			return ConfigSettings{}, err
		}
		value = normalized
		current.Term.Value = value
	}
	if err := s.saveSettings(current); err != nil {
		return ConfigSettings{}, err
	}
	return s.ConfigSettings()
}
