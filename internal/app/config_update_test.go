package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/settings"
)

type configUpdateStore struct {
	value            settings.Settings
	saves            int
	loadErr, saveErr error
}

func (s *configUpdateStore) Load() (settings.Settings, error) { return s.value, s.loadErr }
func (s *configUpdateStore) Save(value settings.Settings) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.value = value
	s.saves++
	return nil
}
func configPointer[T any](value T) *T { return &value }

func TestUpdateConfigTypedFields(t *testing.T) {
	store := &configUpdateStore{value: settings.Default()}
	s := &Service{settingsStore: store}
	result, err := s.UpdateConfig(ConfigUpdate{
		ReminderName: configPointer(" 목록 "), ReminderUseExistingList: configPointer(true), ReminderAlarmBeforeMin: configPointer(60),
		AcademicCalendarName: configPointer("학사"), AcademicCalendarUseExistingList: configPointer(true),
		TimetableCalendarName: configPointer("수업"), TimetableCalendarUseExistingList: configPointer(true),
		DownloadDir: configPointer(" downloads "), DownloadConcurrency: configPointer(9), DownloadCaffeinate: configPointer(false), DownloadKeepPartial: configPointer(true),
		TranscriptConcurrency: configPointer(MaxTranscriptConcurrency), Term: configPointer("2026,1"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.saves != 1 || result.Reminder != (ReminderSettings{ListName: "목록", UseExistingList: true, AlarmBeforeMin: 60}) || result.Calendar != (CalendarSettings{Name: "학사", UseExistingList: true, TimetableName: "수업", TimetableUseExistingList: true}) || result.Download != (DownloadSettings{Dir: "downloads", Concurrency: 9, Caffeinate: false, KeepPartial: true}) || result.Transcript.Concurrency != MaxTranscriptConcurrency || result.Term.Value != "2026,1" {
		t.Fatalf("result=%#v saves=%d", result, store.saves)
	}
	before := store.value
	if _, err = s.UpdateConfig(ConfigUpdate{ReminderName: configPointer("새 목록")}); err != nil {
		t.Fatal(err)
	}
	before.Reminder.ListName = "새 목록"
	if !reflect.DeepEqual(before, store.value) {
		t.Fatal("unmentioned fields changed")
	}
}

func TestUpdateConfigRejectsInvalidValuesBeforeSave(t *testing.T) {
	for _, update := range []ConfigUpdate{
		{ReminderName: configPointer(" ")}, {AcademicCalendarName: configPointer("")}, {TimetableCalendarName: configPointer("")}, {DownloadDir: configPointer("")},
		{ReminderAlarmBeforeMin: configPointer(0)}, {DownloadConcurrency: configPointer(-1)}, {TranscriptConcurrency: configPointer(0)}, {TranscriptConcurrency: configPointer(MaxTranscriptConcurrency + 1)},
		{Term: configPointer("bad")}, {ReminderName: configPointer("valid"), DownloadConcurrency: configPointer(0)},
	} {
		store := &configUpdateStore{value: settings.Default()}
		before := store.value
		if _, err := (&Service{settingsStore: store}).UpdateConfig(update); err == nil {
			t.Fatalf("accepted=%#v", update)
		}
		if store.saves != 0 || !reflect.DeepEqual(before, store.value) {
			t.Fatal("invalid update persisted")
		}
	}
}

func TestUpdateConfigPropagatesStorageFailure(t *testing.T) {
	want := errors.New("storage failure")
	for _, store := range []*configUpdateStore{{loadErr: want}, {value: settings.Default(), saveErr: want}} {
		if _, err := (&Service{settingsStore: store}).UpdateConfig(ConfigUpdate{DownloadKeepPartial: configPointer(false)}); !errors.Is(err, want) {
			t.Fatalf("error=%v", err)
		}
	}
}
