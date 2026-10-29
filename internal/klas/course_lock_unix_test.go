//go:build !windows

package klas

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"
)

func TestCourseProcessLockUsesSharedInode(t *testing.T) {
	key := sha256.Sum256([]byte(t.Name()))
	release, err := lockCourseProcess(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if unlock, err := lockCourseProcess(ctx, key); !errors.Is(err, context.DeadlineExceeded) {
		if unlock != nil {
			unlock()
		}
		release()
		t.Fatalf("%v", err)
	}
	other := sha256.Sum256([]byte("other"))
	unlock, err := lockCourseProcess(context.Background(), other)
	if err != nil {
		t.Fatal(err)
	}
	unlock()
	release()
	unlock, err = lockCourseProcess(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	unlock()
}
