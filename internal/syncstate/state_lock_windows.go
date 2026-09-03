//go:build windows

package syncstate

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func acquireStateFileLock(path string) (func() error, error) {
	lockFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	handle := windows.Handle(lockFile.Fd())
	overlapped := &windows.Overlapped{}
	if err := windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, overlapped); err != nil {
		_ = lockFile.Close()
		return nil, err
	}
	return func() error {
		return errors.Join(
			windows.UnlockFileEx(handle, 0, 1, 0, overlapped),
			lockFile.Close(),
		)
	}, nil
}

func syncStateDirectory(string) error {
	// MoveFileEx is called with write-through semantics. Windows does not expose
	// a portable directory fsync operation.
	return nil
}
