//go:build windows

package account

import (
	"errors"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var replaceFileW = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReplaceFileW")

func replaceRegistryFile(temporaryPath string, registryPath string) error {
	replacement, err := windows.UTF16PtrFromString(temporaryPath)
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(registryPath)
	if err != nil {
		return err
	}

	if _, err := os.Stat(registryPath); errors.Is(err, os.ErrNotExist) {
		return windows.MoveFileEx(replacement, target, windows.MOVEFILE_WRITE_THROUGH)
	} else if err != nil {
		return err
	}
	backupPath := registryPath + ".backup"
	if err := os.Remove(backupPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	backup, err := windows.UTF16PtrFromString(backupPath)
	if err != nil {
		return err
	}

	result, _, callErr := replaceFileW.Call(
		uintptr(unsafe.Pointer(target)),
		uintptr(unsafe.Pointer(replacement)),
		uintptr(unsafe.Pointer(backup)),
		0,
		0,
		0,
	)
	if result == 0 {
		return os.NewSyscallError("ReplaceFileW", callErr)
	}
	// The registry is already committed; a stale backup is safe and retried on
	// the next replacement, so cleanup failure must not turn success into a
	// credential rollback after the registry changed.
	_ = os.Remove(backupPath)
	return nil
}

func recoverRegistryFile(registryPath string) error {
	if _, err := os.Stat(registryPath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	backupPath := registryPath + ".backup"
	if _, err := os.Stat(backupPath); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	backup, err := windows.UTF16PtrFromString(backupPath)
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(registryPath)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(backup, target, windows.MOVEFILE_WRITE_THROUGH)
}
