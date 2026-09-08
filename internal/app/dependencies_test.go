package app

import (
	"context"
	"github.com/leehyowon14/KLAP-cli/internal/kwcommons"
	"strings"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/account"
	"github.com/leehyowon14/KLAP-cli/internal/cache"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"github.com/leehyowon14/KLAP-cli/internal/platform/macos"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
)

func testDependencies(t *testing.T) Dependencies {
	t.Helper()
	store := &account.Store{}
	return Dependencies{
		Downloader: &fakeDownloader{download: func(context.Context, string, string, bool, func(int64, int64)) (int64, error) {
			t.Fatal("constructor downloaded file")
			return 0, nil
		}},
		Academic: &fakeAcademicSource{fetch: func(context.Context, string) (AcademicListResult, error) {
			t.Fatal("constructor fetched academic source")
			return AcademicListResult{}, nil
		}},
		Media:             kwcommons.NewClient(nil),
		AssignmentGateway: func(c *klas.Client) AssignmentGateway { return c },
		Accounts:          store, Sessions: store, Settings: &settings.Store{}, Cache: &cache.Store{},
		SyncState:     &fakeSyncStateStore{},
		Reminder:      macos.NewReminderBridge("unused"),
		Calendar:      macos.NewCalendarBridge("unused"),
		Categories:    macos.NewCategoryBridge("unused"),
		Transcriber:   macos.NewTranscriptBridge("unused"),
		NewKlasClient: func() (*klas.Client, error) { t.Fatal("constructor called client factory"); return nil, nil },
		Login: func(context.Context, *klas.Client, string, string) (klas.Session, error) {
			t.Fatal("constructor called login")
			return klas.Session{}, nil
		},
	}
}

func TestDependenciesConstructionDoesNotPerformIO(t *testing.T) {
	deps := testDependencies(t)
	s, err := NewService(deps)
	if err != nil {
		t.Fatal(err)
	}
	if s.store != deps.Accounts || s.sessions != deps.Sessions || s.settingsStore != deps.Settings || s.cacheStore != deps.Cache || s.syncStateStore != deps.SyncState {
		t.Fatal("constructor did not retain injected stores")
	}
}

func TestDependenciesRejectMissingAndTypedNil(t *testing.T) {
	tests := []struct {
		name string
		omit func(*Dependencies)
	}{
		{"accounts", func(d *Dependencies) { d.Accounts = nil }},
		{"accounts", func(d *Dependencies) { d.Accounts = (*account.Store)(nil) }},
		{"sessions", func(d *Dependencies) { d.Sessions = nil }},
		{"sessions", func(d *Dependencies) { d.Sessions = (*account.Store)(nil) }},
		{"settings", func(d *Dependencies) { d.Settings = nil }},
		{"settings", func(d *Dependencies) { d.Settings = (*settings.Store)(nil) }},
		{"cache", func(d *Dependencies) { d.Cache = nil }},
		{"cache", func(d *Dependencies) { d.Cache = (*cache.Store)(nil) }},
		{"sync state", func(d *Dependencies) { d.SyncState = nil }},
		{"sync state", func(d *Dependencies) { d.SyncState = (*fakeSyncStateStore)(nil) }},
		{"KLAS client factory", func(d *Dependencies) { d.NewKlasClient = nil }},
		{"login", func(d *Dependencies) { d.Login = nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := testDependencies(t)
			tt.omit(&deps)
			s, err := NewService(deps)
			if s != nil || err == nil || !strings.Contains(err.Error(), tt.name) {
				t.Fatalf("service=%v error=%v", s, err)
			}
		})
	}
}

func TestDependenciesRequireReminder(t *testing.T) {
	for _, value := range []ReminderSyncer{nil, (*macos.ReminderBridge)(nil)} {
		deps := testDependencies(t)
		deps.Reminder = value
		s, err := NewService(deps)
		if s != nil || err == nil {
			t.Fatalf("service=%v error=%v", s, err)
		}
	}
}

func TestDependenciesRequireCalendar(t *testing.T) {
	for _, value := range []CalendarSyncer{nil, (*macos.CalendarBridge)(nil)} {
		deps := testDependencies(t)
		deps.Calendar = value
		s, err := NewService(deps)
		if s != nil || err == nil {
			t.Fatalf("service=%v error=%v", s, err)
		}
	}
}

func TestDependenciesRequireCategories(t *testing.T) {
	for _, value := range []CategoryLister{nil, (*macos.CategoryBridge)(nil)} {
		deps := testDependencies(t)
		deps.Categories = value
		s, err := NewService(deps)
		if s != nil || err == nil {
			t.Fatalf("service=%v error=%v", s, err)
		}
	}
}

func TestDependenciesRequireTranscriber(t *testing.T) {
	for _, value := range []Transcriber{nil, (*macos.TranscriptBridge)(nil)} {
		deps := testDependencies(t)
		deps.Transcriber = value
		s, err := NewService(deps)
		if s != nil || err == nil {
			t.Fatalf("service=%v error=%v", s, err)
		}
	}
}
