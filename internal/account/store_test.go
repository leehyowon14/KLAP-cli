package account

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/kw-klap/klap-cli/internal/klas"
)

type memoryKeyring struct {
	values map[string]string
}

func newMemoryKeyring() *memoryKeyring {
	return &memoryKeyring{values: make(map[string]string)}
}

func (k *memoryKeyring) Set(service string, user string, password string) error {
	k.values[service+"/"+user] = password
	return nil
}

func (k *memoryKeyring) Get(service string, user string) (string, error) {
	value, ok := k.values[service+"/"+user]
	if !ok {
		return "", errors.New("secret not found")
	}
	return value, nil
}

func (k *memoryKeyring) Delete(service string, user string) error {
	delete(k.values, service+"/"+user)
	return nil
}

func TestStoreUsesInjectedKeyringForCredentials(t *testing.T) {
	secrets := newMemoryKeyring()
	store := newStoreAt(filepath.Join(t.TempDir(), "users.json"), secrets)
	session := klas.Session{UserID: "user-id", Cookies: map[string]string{"SESSION": "cookie"}}

	if err := store.Save(context.Background(), "20260001", "password", session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	password, err := store.LoadPassword(context.Background(), "20260001")
	if err != nil {
		t.Fatalf("LoadPassword() error = %v", err)
	}
	if password != "password" {
		t.Fatalf("LoadPassword() = %q", password)
	}
	gotSession, err := store.LoadSession(context.Background(), "20260001")
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	if gotSession.UserID != session.UserID || gotSession.Cookies["SESSION"] != "cookie" {
		t.Fatalf("LoadSession() = %+v", gotSession)
	}
}

func TestRemoveCurrentSelectsFirstRemainingUser(t *testing.T) {
	store := newStoreAt(filepath.Join(t.TempDir(), "users.json"), newMemoryKeyring())
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
