//go:build windows

package klas

import "context"

// Windows retains in-process protection; supported production hosts use flock.
func lockCourseProcess(ctx context.Context, key [32]byte) (func(), error) {
	return func() {}, ctx.Err()
}
