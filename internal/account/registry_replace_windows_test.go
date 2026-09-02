//go:build windows

package account

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceRegistryFileReplacesExistingFile(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "users.json")
	temporaryPath := filepath.Join(dir, ".users-replacement.tmp")
	if err := os.WriteFile(registryPath, []byte("old"), 0o600); err != nil {
		t.Fatalf("WriteFile(registry) error = %v", err)
	}
	if err := os.WriteFile(temporaryPath, []byte("new"), 0o600); err != nil {
		t.Fatalf("WriteFile(replacement) error = %v", err)
	}

	if err := replaceRegistryFile(temporaryPath, registryPath); err != nil {
		t.Fatalf("replaceRegistryFile() error = %v", err)
	}
	got, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("ReadFile(registry) error = %v", err)
	}
	if string(got) != "new" {
		t.Fatalf("registry contents = %q, want new", got)
	}
	if _, err := os.Stat(registryPath + ".backup"); !os.IsNotExist(err) {
		t.Fatalf("registry backup remains after success: %v", err)
	}
}

func TestRecoverRegistryFileRestoresBackupWhenTargetMissing(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "users.json")
	backupPath := registryPath + ".backup"
	if err := os.WriteFile(backupPath, []byte("last-valid"), 0o600); err != nil {
		t.Fatalf("WriteFile(backup) error = %v", err)
	}

	if err := recoverRegistryFile(registryPath); err != nil {
		t.Fatalf("recoverRegistryFile() error = %v", err)
	}
	got, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("ReadFile(registry) error = %v", err)
	}
	if string(got) != "last-valid" {
		t.Fatalf("registry contents = %q, want last-valid", got)
	}
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Fatalf("registry backup remains after recovery: %v", err)
	}
}
