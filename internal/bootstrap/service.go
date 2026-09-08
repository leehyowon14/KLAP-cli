package bootstrap

import (
	"context"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/academic"
	"github.com/leehyowon14/KLAP-cli/internal/account"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"github.com/leehyowon14/KLAP-cli/internal/cache"
	klapcalendar "github.com/leehyowon14/KLAP-cli/internal/calendar"
	category "github.com/leehyowon14/KLAP-cli/internal/category"
	"github.com/leehyowon14/KLAP-cli/internal/download"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"github.com/leehyowon14/KLAP-cli/internal/kwcommons"
	reminder "github.com/leehyowon14/KLAP-cli/internal/reminder"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
	"github.com/leehyowon14/KLAP-cli/internal/syncstate"
	transcript "github.com/leehyowon14/KLAP-cli/internal/transcript"
)

type serviceStoreFactories struct {
	settings  func() (*settings.Store, error)
	cache     func() (*cache.Store, error)
	syncState func() (*syncstate.Store, error)
}

func NewService() (*app.Service, error) {
	store, err := account.NewStore()
	if err != nil {
		return nil, err
	}
	return newService(store, serviceStoreFactories{settings: settings.NewStore, cache: cache.NewStore, syncState: syncstate.NewStore})
}

func newService(store *account.Store, factories serviceStoreFactories) (*app.Service, error) {
	settingsStore, err := factories.settings()
	if err != nil {
		return nil, fmt.Errorf("settings store 초기화 실패: %w", err)
	}
	cacheStore, err := factories.cache()
	if err != nil {
		return nil, fmt.Errorf("cache store 초기화 실패: %w", err)
	}
	syncStateStore, err := factories.syncState()
	if err != nil {
		return nil, fmt.Errorf("sync state store 초기화 실패: %w", err)
	}
	return app.NewService(app.Dependencies{
		Downloader:        download.NewClient(nil),
		Academic:          academic.NewClient(nil),
		Media:             kwcommons.NewClient(nil),
		AssignmentGateway: func(c *klas.Client) app.AssignmentGateway { return c },
		Accounts:          store,
		Sessions:          store,
		Settings:          settingsStore,
		Cache:             cacheStore,
		SyncState:         syncStateStore,
		NewKlasClient:     klas.NewClient,
		Login: func(ctx context.Context, client *klas.Client, studentID string, password string) (klas.Session, error) {
			return client.Login(ctx, studentID, password)
		},
		Reminder:    reminder.NewMacOSBridge(defaultReminderBridgePath()),
		Calendar:    klapcalendar.NewMacOSBridge(defaultCalendarBridgePath()),
		Categories:  category.NewMacOSBridge(defaultCategoryBridgePath()),
		Transcriber: transcript.NewMacOSBridge(defaultTranscriptBridgePath()),
	})
}
