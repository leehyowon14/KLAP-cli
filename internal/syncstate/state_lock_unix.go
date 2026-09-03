//go:build !windows

package syncstate

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

func acquireStateFileLock(path string) (func() error, error) {
	lockFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		_ = lockFile.Close()
		return nil, err
	}
	return func() error {
		return errors.Join(
			syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN),
			lockFile.Close(),
		)
	}, nil
}

func syncStateDirectory(targetPath string) error {
	directory, err := os.Open(filepath.Dir(targetPath))
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}
