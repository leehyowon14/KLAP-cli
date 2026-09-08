package macos

import (
	"context"
	"runtime"
	"sync"
)

// WakeLock prevents idle sleep for the duration of a download. A zero value uses
// caffeinate on macOS and is a no-op on other platforms.
type WakeLock struct{ spec ProcessSpec }

func (w WakeLock) Acquire(ctx context.Context) (func(), error) {
	if runtime.GOOS != "darwin" {
		return func() {}, nil
	}
	spec := w.spec
	if spec.Name == "" {
		spec = ProcessSpec{Name: "caffeinate", Args: []string{"-dims"}}
	}
	process, err := (ProcessRunner{}).start(ctx, spec)
	if err != nil {
		return nil, err
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Losing an optional wake lock after startup does not fail the transfer.
		_ = process.wait(nil)
	}()
	var once sync.Once
	return func() { once.Do(func() { process.cancel(); <-done }) }, nil
}
