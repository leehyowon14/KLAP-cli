package app

import (
	"context"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/account"
	"github.com/leehyowon14/KLAP-cli/internal/cache"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
	"github.com/leehyowon14/KLAP-cli/internal/syncstate"
	"strings"
)

type Service struct {
	store                AccountStore
	sessions             sessionStore
	settingsStore        SettingsStore
	cacheStore           CacheStore
	syncStateStore       syncStateStore
	newKlasClient        func() (*klas.Client, error)
	login                func(context.Context, *klas.Client, string, string) (klas.Session, error)
	reminderBridgePath   string
	calendarBridgePath   string
	categoryBridgePath   string
	transcriptBridgePath string
}

type OpenURLResult struct {
	URL string
}

type serviceStoreFactories struct {
	settings  func() (*settings.Store, error)
	cache     func() (*cache.Store, error)
	syncState func() (*syncstate.Store, error)
}

func NewService(store *account.Store) (*Service, error) {
	return newService(store, serviceStoreFactories{
		settings:  settings.NewStore,
		cache:     cache.NewStore,
		syncState: syncstate.NewStore,
	})
}

func newService(store *account.Store, factories serviceStoreFactories) (*Service, error) {
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
	return &Service{
		store:          store,
		sessions:       store,
		settingsStore:  settingsStore,
		cacheStore:     cacheStore,
		syncStateStore: syncStateStore,
		newKlasClient:  klas.NewClient,
		login: func(ctx context.Context, client *klas.Client, studentID string, password string) (klas.Session, error) {
			return client.Login(ctx, studentID, password)
		},
		reminderBridgePath:   defaultReminderBridgePath(),
		calendarBridgePath:   defaultCalendarBridgePath(),
		categoryBridgePath:   defaultCategoryBridgePath(),
		transcriptBridgePath: defaultTranscriptBridgePath(),
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func uniqueNonEmpty(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func compactNonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
