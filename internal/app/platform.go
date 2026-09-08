package app

import (
	"context"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
)

// WakeLock is optional at execution time: unavailable sleep prevention must not
// fail a download. Acquire returns a release function that joins its worker.
type WakeLock interface {
	Acquire(context.Context) (func(), error)
}

func (s *Service) acquireWakeLock(ctx context.Context) func() {
	if s.wakeLock == nil {
		return func() {}
	}
	current, err := s.loadSettings()
	if err != nil || !settings.DownloadCaffeinateEnabled(current.Download) {
		return func() {}
	}
	release, err := s.wakeLock.Acquire(ctx)
	if err != nil || release == nil {
		return func() {}
	}
	return release
}
