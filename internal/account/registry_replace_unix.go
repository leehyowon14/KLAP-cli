//go:build !windows

package account

import "os"

func replaceRegistryFile(temporaryPath string, registryPath string) error {
	return os.Rename(temporaryPath, registryPath)
}

func recoverRegistryFile(string) error {
	return nil
}
