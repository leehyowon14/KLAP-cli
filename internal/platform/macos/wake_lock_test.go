package macos

import (
	"context"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestWakeLockReleaseJoinsAndIsIdempotent(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS wake lock")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	release, err := (WakeLock{spec: helperSpec(t, "hang")}).Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var callers sync.WaitGroup
	for i := 0; i < 4; i++ {
		callers.Go(release)
	}
	callers.Wait()
	if ctx.Err() != nil {
		t.Fatal("release did not promptly join")
	}
}

func TestWakeLockStartFailure(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS wake lock")
	}
	release, err := (WakeLock{spec: ProcessSpec{Name: filepath.Join(t.TempDir(), "missing")}}).Acquire(context.Background())
	if err == nil || release != nil {
		t.Fatalf("release nil=%v err=%v", release == nil, err)
	}
}
