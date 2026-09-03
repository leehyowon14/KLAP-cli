//go:build !windows

package syncstate

import "os"

func replaceStateFile(temporaryPath string, targetPath string) error {
	return os.Rename(temporaryPath, targetPath)
}

func recoverStateFile(string) error {
	return nil
}
