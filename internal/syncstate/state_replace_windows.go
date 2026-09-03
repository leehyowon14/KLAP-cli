//go:build windows

package syncstate

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func replaceStateFile(temporaryPath string, targetPath string) error {
	replacement, err := windows.UTF16PtrFromString(temporaryPath)
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(targetPath)
	if err != nil {
		return err
	}
	flags := uint32(windows.MOVEFILE_WRITE_THROUGH)
	if _, err := os.Stat(targetPath); err == nil {
		flags |= windows.MOVEFILE_REPLACE_EXISTING
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	backupPath := targetPath + ".backup"
	if err := os.Remove(backupPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return windows.MoveFileEx(replacement, target, flags)
}

func recoverStateFile(targetPath string) error {
	if _, err := os.Stat(targetPath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	backupPath := targetPath + ".backup"
	if _, err := os.Stat(backupPath); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	backup, err := windows.UTF16PtrFromString(backupPath)
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(targetPath)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(backup, target, windows.MOVEFILE_WRITE_THROUGH)
}
