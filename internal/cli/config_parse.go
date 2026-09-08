package cli

import (
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strconv"
	"strings"
)

func parseConfigUpdate(key string, value string) (app.ConfigUpdate, error) {
	key = strings.ToLower(strings.TrimSpace(key))
	value = strings.TrimSpace(value)
	if key == "" {
		return app.ConfigUpdate{}, errors.New("설정 키가 필요합니다")
	}
	switch key {
	case "reminder.name", "reminder.list", "reminder.list-name":
		if value == "" {
			return app.ConfigUpdate{}, errors.New("리마인더 목록 이름은 비워둘 수 없습니다")
		}
		return app.ConfigUpdate{ReminderName: &value}, nil
	case "reminder.use-existing-list":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return app.ConfigUpdate{}, err
		}
		return app.ConfigUpdate{ReminderUseExistingList: &parsed}, nil
	case "reminder.alarm-before-min", "reminder.alarm-before":
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return app.ConfigUpdate{}, errors.New("reminder.alarm-before-min에는 1 이상의 정수가 필요합니다")
		}
		return app.ConfigUpdate{ReminderAlarmBeforeMin: &parsed}, nil
	case "calendar.name", "calendar.list", "calendar.list-name":
		if value == "" {
			return app.ConfigUpdate{}, errors.New("캘린더 이름은 비워둘 수 없습니다")
		}
		return app.ConfigUpdate{AcademicCalendarName: &value}, nil
	case "calendar.use-existing-list":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return app.ConfigUpdate{}, err
		}
		return app.ConfigUpdate{AcademicCalendarUseExistingList: &parsed}, nil
	case "calendar.academic.name", "academic-calendar.name", "academic-calendar.list":
		if value == "" {
			return app.ConfigUpdate{}, errors.New("학사일정 캘린더 이름은 비워둘 수 없습니다")
		}
		return app.ConfigUpdate{AcademicCalendarName: &value}, nil
	case "calendar.academic.use-existing-list", "academic-calendar.use-existing-list":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return app.ConfigUpdate{}, err
		}
		return app.ConfigUpdate{AcademicCalendarUseExistingList: &parsed}, nil
	case "calendar.timetable.name", "timetable-calendar.name", "timetable-calendar.list":
		if value == "" {
			return app.ConfigUpdate{}, errors.New("시간표 캘린더 이름은 비워둘 수 없습니다")
		}
		return app.ConfigUpdate{TimetableCalendarName: &value}, nil
	case "calendar.timetable.use-existing-list", "timetable-calendar.use-existing-list":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return app.ConfigUpdate{}, err
		}
		return app.ConfigUpdate{TimetableCalendarUseExistingList: &parsed}, nil
	case "download.dir", "download.path":
		if value == "" {
			return app.ConfigUpdate{}, errors.New("다운로드 폴더 경로가 필요합니다")
		}
		return app.ConfigUpdate{DownloadDir: &value}, nil
	case "download.concurrency", "download.workers":
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return app.ConfigUpdate{}, errors.New("download.concurrency에는 1 이상의 정수가 필요합니다")
		}
		return app.ConfigUpdate{DownloadConcurrency: &parsed}, nil
	case "download.caffeinate", "download.prevent-sleep", "download.keep-awake":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return app.ConfigUpdate{}, err
		}
		return app.ConfigUpdate{DownloadCaffeinate: &parsed}, nil
	case "download.keep-partial", "download.resume", "download.partial":
		parsed, err := parseConfigBool(value)
		if err != nil {
			return app.ConfigUpdate{}, err
		}
		return app.ConfigUpdate{DownloadKeepPartial: &parsed}, nil
	case "transcript.concurrency", "transcript.workers":
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > app.MaxTranscriptConcurrency {
			return app.ConfigUpdate{}, fmt.Errorf("transcript.concurrency에는 1~%d 사이의 정수가 필요합니다", app.MaxTranscriptConcurrency)
		}
		return app.ConfigUpdate{TranscriptConcurrency: &parsed}, nil
	case "term", "term.value":
		return app.ConfigUpdate{Term: &value}, nil
	default:
		return app.ConfigUpdate{}, fmt.Errorf("지원하지 않는 설정 키입니다: %s", key)
	}
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
