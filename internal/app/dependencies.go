package app

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/leehyowon14/KLAP-cli/internal/account"
	"github.com/leehyowon14/KLAP-cli/internal/cache"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
)

// MediaResolver resolves public lecture media independently of KLAS authentication.
type MediaResolver interface {
	ResolveLectureMediaURL(context.Context, string) (string, error)
}

type AccountStore interface {
	Save(context.Context, string, string, klas.Session) error
	List(context.Context) ([]account.User, error)
	Current(context.Context) (string, error)
	Select(context.Context, string) error
	Remove(context.Context, string) error
}

type SettingsStore interface {
	Load() (settings.Settings, error)
	Save(settings.Settings) error
}

type CacheStore interface {
	Get(string, any) (cache.Hit, bool, error)
	Set(string, time.Duration, any) error
	Delete(string) (bool, error)
	ClearExceptPrefixes(...string) (int, error)
	ClearPrefix(string) (int, error)
	Stats() (cache.Stats, error)
}

// Dependencies supplies the resources used by the application facade.
// Construction does not access the filesystem, Keychain, network, or bridges.
type Dependencies struct {
	Downloader        Downloader
	Academic          AcademicSource
	Media             MediaResolver
	AssignmentGateway func(*klas.Client) AssignmentGateway
	Accounts          AccountStore
	Sessions          sessionStore
	Settings          SettingsStore
	Cache             CacheStore
	SyncState         syncStateStore
	NewKlasClient     func() (*klas.Client, error)
	Login             func(context.Context, *klas.Client, string, string) (klas.Session, error)
	Reminder          ReminderSyncer
	Calendar          CalendarSyncer
	Categories        CategoryLister
	Transcriber       Transcriber
}

func NewService(deps Dependencies) (*Service, error) {
	for _, dependency := range []struct {
		name  string
		value any
	}{
		{"accounts", deps.Accounts}, {"sessions", deps.Sessions},
		{"settings", deps.Settings}, {"cache", deps.Cache},
		{"sync state", deps.SyncState}, {"KLAS client factory", deps.NewKlasClient},
		{"login", deps.Login},
		{"media", deps.Media},
		{"academic", deps.Academic},
		{"downloader", deps.Downloader},
		{"assignment gateway", deps.AssignmentGateway},
		{"transcript", deps.Transcriber},
		{"category", deps.Categories},
		{"calendar", deps.Calendar},
		{"reminder", deps.Reminder},
	} {
		if missingDependency(dependency.value) {
			return nil, fmt.Errorf("missing application dependency: %s", dependency.name)
		}
	}
	return &Service{
		downloader:        deps.Downloader,
		academic:          deps.Academic,
		media:             deps.Media,
		assignmentGateway: deps.AssignmentGateway,
		store:             deps.Accounts, sessions: deps.Sessions, settingsStore: deps.Settings,
		cacheStore: deps.Cache, syncStateStore: deps.SyncState,
		newKlasClient: deps.NewKlasClient, login: deps.Login,
		reminderSyncer: deps.Reminder, calendarSyncer: deps.Calendar,
		categoryLister: deps.Categories, transcriber: deps.Transcriber,
	}, nil
}

func missingDependency(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
