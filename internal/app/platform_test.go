package app

import (
	"context"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
	"testing"
)

type fakeWakeLock struct {
	acquire func(context.Context) (func(), error)
}

func (f *fakeWakeLock) Acquire(ctx context.Context) (func(), error) { return f.acquire(ctx) }

func TestDependenciesRequireWakeLock(t *testing.T) {
	for _, value := range []WakeLock{nil, (*fakeWakeLock)(nil)} {
		deps := testDependencies(t)
		deps.WakeLock = value
		if s, err := NewService(deps); s != nil || err == nil {
			t.Fatalf("service=%v err=%v", s, err)
		}
	}
}

func TestWakeLockPolicyAndRelease(t *testing.T) {
	t.Setenv("KLAP_CONFIG_DIR", t.TempDir())
	store, err := settings.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	current := settings.Default()
	disabled := false
	current.Download.Caffeinate = &disabled
	if err := store.Save(current); err != nil {
		t.Fatal(err)
	}
	calls, releases := 0, 0
	ctx := context.Background()
	lock := &fakeWakeLock{acquire: func(got context.Context) (func(), error) {
		calls++
		if got != ctx {
			t.Fatal("context not forwarded")
		}
		return func() { releases++ }, nil
	}}
	service := &Service{settingsStore: store, wakeLock: lock}
	service.acquireWakeLock(ctx)()
	if calls != 0 {
		t.Fatal("disabled lock acquired")
	}
	enabled := true
	current.Download.Caffeinate = &enabled
	if err := store.Save(current); err != nil {
		t.Fatal(err)
	}
	service.acquireWakeLock(ctx)()
	if calls != 1 || releases != 1 {
		t.Fatalf("calls=%d releases=%d", calls, releases)
	}
	lock.acquire = func(context.Context) (func(), error) { return nil, errors.New("optional lock unavailable") }
	service.acquireWakeLock(ctx)()
}
