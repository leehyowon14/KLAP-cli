package account

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestRemoveCurrentSelectsFirstRemainingUser(t *testing.T) {
	store := &Store{registryPath: filepath.Join(t.TempDir(), "users.json")}
	err := store.saveRegistry(registryFile{
		CurrentStudentID: "2024000001",
		Users: []User{
			{StudentID: "2024000001", SavedAt: time.Now()},
			{StudentID: "2024000002", SavedAt: time.Now()},
			{StudentID: "2024000003", SavedAt: time.Now()},
		},
	})
	if err != nil {
		t.Fatalf("saveRegistry() error = %v", err)
	}

	if err := store.Remove(context.Background(), "2024000001"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	got, err := store.Current(context.Background())
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if got != "2024000002" {
		t.Fatalf("Current() = %q, want %q", got, "2024000002")
	}
}
