package cli

import (
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"reflect"
	"testing"
)

func TestParseConfigUpdateAliases(t *testing.T) {
	for _, tt := range []struct {
		key, value, field string
		want              any
	}{
		{key: "reminder.name", value: " name ", field: "ReminderName", want: "name"},
		{key: "reminder.list", value: " name ", field: "ReminderName", want: "name"},
		{key: "reminder.list-name", value: " name ", field: "ReminderName", want: "name"},
		{key: "reminder.use-existing-list", value: "off", field: "ReminderUseExistingList", want: false},
		{key: "reminder.alarm-before-min", value: "2", field: "ReminderAlarmBeforeMin", want: 2},
		{key: "reminder.alarm-before", value: "2", field: "ReminderAlarmBeforeMin", want: 2},
		{key: "calendar.name", value: " name ", field: "AcademicCalendarName", want: "name"},
		{key: "calendar.list", value: " name ", field: "AcademicCalendarName", want: "name"},
		{key: "calendar.list-name", value: " name ", field: "AcademicCalendarName", want: "name"},
		{key: "calendar.use-existing-list", value: "off", field: "AcademicCalendarUseExistingList", want: false},
		{key: "calendar.academic.name", value: " name ", field: "AcademicCalendarName", want: "name"},
		{key: "academic-calendar.name", value: " name ", field: "AcademicCalendarName", want: "name"},
		{key: "academic-calendar.list", value: " name ", field: "AcademicCalendarName", want: "name"},
		{key: "calendar.academic.use-existing-list", value: "off", field: "AcademicCalendarUseExistingList", want: false},
		{key: "academic-calendar.use-existing-list", value: "off", field: "AcademicCalendarUseExistingList", want: false},
		{key: "calendar.timetable.name", value: " name ", field: "TimetableCalendarName", want: "name"},
		{key: "timetable-calendar.name", value: " name ", field: "TimetableCalendarName", want: "name"},
		{key: "timetable-calendar.list", value: " name ", field: "TimetableCalendarName", want: "name"},
		{key: "calendar.timetable.use-existing-list", value: "off", field: "TimetableCalendarUseExistingList", want: false},
		{key: "timetable-calendar.use-existing-list", value: "off", field: "TimetableCalendarUseExistingList", want: false},
		{key: "download.dir", value: " name ", field: "DownloadDir", want: "name"},
		{key: "download.path", value: " name ", field: "DownloadDir", want: "name"},
		{key: "download.concurrency", value: "2", field: "DownloadConcurrency", want: 2},
		{key: "download.workers", value: "2", field: "DownloadConcurrency", want: 2},
		{key: "download.caffeinate", value: "off", field: "DownloadCaffeinate", want: false},
		{key: "download.prevent-sleep", value: "off", field: "DownloadCaffeinate", want: false},
		{key: "download.keep-awake", value: "off", field: "DownloadCaffeinate", want: false},
		{key: "download.keep-partial", value: "off", field: "DownloadKeepPartial", want: false},
		{key: "download.resume", value: "off", field: "DownloadKeepPartial", want: false},
		{key: "download.partial", value: "off", field: "DownloadKeepPartial", want: false},
		{key: "transcript.concurrency", value: "2", field: "TranscriptConcurrency", want: 2},
		{key: "transcript.workers", value: "2", field: "TranscriptConcurrency", want: 2},
		{key: "term", value: "2026,1", field: "Term", want: "2026,1"},
		{key: "term.value", value: "2026,1", field: "Term", want: "2026,1"},
	} {
		update, err := parseConfigUpdate(" "+tt.key+" ", tt.value)
		if err != nil {
			t.Fatal(err)
		}
		value := reflect.ValueOf(update).FieldByName(tt.field)
		if value.IsNil() || !reflect.DeepEqual(value.Elem().Interface(), tt.want) {
			t.Fatalf("%s: %#v", tt.key, update)
		}
		for i := 0; i < reflect.TypeFor[app.ConfigUpdate]().NumField(); i++ {
			if reflect.TypeFor[app.ConfigUpdate]().Field(i).Name != tt.field && !reflect.ValueOf(update).Field(i).IsNil() {
				t.Fatalf("%s modifies extra field", tt.key)
			}
		}
	}
}
func TestParseConfigUpdateRejectsInvalidInput(t *testing.T) {
	for _, pair := range [][2]string{{"", "x"}, {"unknown", "x"}, {"reminder.name", " "}, {"download.concurrency", "0"}, {"download.concurrency", "bad"}, {"reminder.alarm-before", "-1"}, {"transcript.workers", "4"}, {"download.caffeinate", "maybe"}} {
		if _, err := parseConfigUpdate(pair[0], pair[1]); err == nil {
			t.Fatalf("accepted %v", pair)
		}
	}
}
func TestParseConfigBoolSpellings(t *testing.T) {
	for _, v := range []string{"true", "t", "1", "yes", "y", "on"} {
		got, err := parseConfigBool(v)
		if err != nil || !got {
			t.Fatalf("%s: %v %v", v, got, err)
		}
	}
	for _, v := range []string{"false", "f", "0", "no", "n", "off"} {
		got, err := parseConfigBool(v)
		if err != nil || got {
			t.Fatalf("%s: %v %v", v, got, err)
		}
	}
}
