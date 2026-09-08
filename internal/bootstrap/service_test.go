package bootstrap

import (
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/account"
	"github.com/leehyowon14/KLAP-cli/internal/cache"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
	"github.com/leehyowon14/KLAP-cli/internal/syncstate"
	"strings"
	"testing"
)

func TestNewServicePropagatesSettingsStoreFailure(t *testing.T) {
	wantErr := errors.New("settings unavailable")
	cacheFactoryCalled := false

	service, err := newService(nil, serviceStoreFactories{
		settings: func() (*settings.Store, error) {
			return nil, wantErr
		},
		cache: func() (*cache.Store, error) {
			cacheFactoryCalled = true
			return nil, nil
		},
		syncState: func() (*syncstate.Store, error) {
			t.Fatal("sync state factory should not run after settings factory failure")
			return nil, nil
		},
	})

	if service != nil {
		t.Fatalf("newService() service = %v, want nil", service)
	}
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "settings store 초기화 실패") {
		t.Fatalf("newService() error = %v", err)
	}
	if cacheFactoryCalled {
		t.Fatal("cache factory should not run after settings factory failure")
	}
}

func TestNewServicePropagatesCacheStoreFailure(t *testing.T) {
	wantErr := errors.New("cache unavailable")

	service, err := newService(nil, serviceStoreFactories{
		settings: func() (*settings.Store, error) {
			return &settings.Store{}, nil
		},
		cache: func() (*cache.Store, error) {
			return nil, wantErr
		},
		syncState: func() (*syncstate.Store, error) {
			t.Fatal("sync state factory should not run after cache factory failure")
			return nil, nil
		},
	})

	if service != nil {
		t.Fatalf("newService() service = %v, want nil", service)
	}
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "cache store 초기화 실패") {
		t.Fatalf("newService() error = %v", err)
	}
}

func TestNewServicePropagatesSyncStateStoreFailure(t *testing.T) {
	wantErr := errors.New("sync state unavailable")
	service, err := newService(nil, serviceStoreFactories{
		settings: func() (*settings.Store, error) { return &settings.Store{}, nil },
		cache:    func() (*cache.Store, error) { return &cache.Store{}, nil },
		syncState: func() (*syncstate.Store, error) {
			return nil, wantErr
		},
	})
	if service != nil {
		t.Fatalf("newService() service = %v, want nil", service)
	}
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "sync state store 초기화 실패") {
		t.Fatalf("newService() error = %v", err)
	}
}

func TestNewServiceInitializesStores(t *testing.T) {
	accountStore := &account.Store{}
	settingsStore := &settings.Store{}
	cacheStore := &cache.Store{}
	syncStateStore, err := syncstate.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}

	service, err := newService(accountStore, serviceStoreFactories{
		settings:  func() (*settings.Store, error) { return settingsStore, nil },
		cache:     func() (*cache.Store, error) { return cacheStore, nil },
		syncState: func() (*syncstate.Store, error) { return syncStateStore, nil },
	})
	if err != nil {
		t.Fatalf("newService() error = %v", err)
	}
	if service == nil {
		t.Fatal("service is nil")
	}
}
