// Package testsupport contains isolated composition helpers for integration tests.
package testsupport

import (
	"context"
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
	"testing"
)

func NewService(t *testing.T) *app.Service {
	t.Helper()
	t.Setenv("KLAP_CONFIG_DIR", t.TempDir())
	t.Setenv("KLAP_CACHE_DIR", t.TempDir())
	store, err := account.NewStore()
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	settingsStore, err := settings.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	cacheStore, err := cache.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	syncStore, err := syncstate.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	service, err := app.NewService(app.Dependencies{
		Downloader:        download.NewClient(nil),
		Academic:          academic.NewClient(nil),
		Media:             kwcommons.NewClient(nil),
		AssignmentGateway: func(c *klas.Client) app.AssignmentGateway { return c },
		Accounts:          store, Sessions: store, Settings: settingsStore, Cache: cacheStore, SyncState: syncStore,
		Reminder:      reminder.NewMacOSBridge("unused"),
		Calendar:      klapcalendar.NewMacOSBridge("unused"),
		Categories:    category.NewMacOSBridge("unused"),
		Transcriber:   transcript.NewMacOSBridge("unused"),
		NewKlasClient: klas.NewClient,
		Login: func(ctx context.Context, c *klas.Client, id, password string) (klas.Session, error) {
			return c.Login(ctx, id, password)
		},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}
