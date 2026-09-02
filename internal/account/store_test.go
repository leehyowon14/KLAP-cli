package account

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kw-klap/klap-cli/internal/klas"
	"github.com/zalando/go-keyring"
)

type memoryKeyring struct {
	values       map[string]string
	setErrors    map[string][]error
	getErrors    map[string][]error
	deleteErrors map[string][]error
}

func newMemoryKeyring() *memoryKeyring {
	return &memoryKeyring{
		values:       make(map[string]string),
		setErrors:    make(map[string][]error),
		getErrors:    make(map[string][]error),
		deleteErrors: make(map[string][]error),
	}
}

func (k *memoryKeyring) Set(service string, user string, password string) error {
	key := service + "/" + user
	if err := popError(k.setErrors, key); err != nil {
		return err
	}
	k.values[key] = password
	return nil
}

func (k *memoryKeyring) Get(service string, user string) (string, error) {
	key := service + "/" + user
	if err := popError(k.getErrors, key); err != nil {
		return "", err
	}
	value, ok := k.values[key]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return value, nil
}

func (k *memoryKeyring) Delete(service string, user string) error {
	key := service + "/" + user
	if err := popError(k.deleteErrors, key); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			delete(k.values, key)
		}
		return err
	}
	if _, ok := k.values[key]; !ok {
		return keyring.ErrNotFound
	}
	delete(k.values, key)
	return nil
}

func popError(script map[string][]error, key string) error {
	errorsForKey := script[key]
	if len(errorsForKey) == 0 {
		return nil
	}
	err := errorsForKey[0]
	script[key] = errorsForKey[1:]
	return err
}

func secretMapKey(kind string, studentID string) string {
	return keyringService + "/" + keyName(kind, studentID)
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

func TestSaveRollsBackAfterPasswordStoreFailure(t *testing.T) {
	wantErr := errors.New("password write failed")
	secrets := newMemoryKeyring()
	secrets.setErrors[secretMapKey(passwordKind, "20260001")] = []error{wantErr}
	store := newStoreAt(filepath.Join(t.TempDir(), "users.json"), secrets)

	err := store.Save(context.Background(), "20260001", "password", klas.Session{UserID: "user-id"})

	if !errors.Is(err, wantErr) {
		t.Fatalf("Save() error = %v, want %v", err, wantErr)
	}
	if len(secrets.values) != 0 {
		t.Fatalf("Save() credentials after failure = %+v", secrets.values)
	}
	users, listErr := store.List(context.Background())
	if listErr != nil || len(users) != 0 {
		t.Fatalf("List() after failure = %+v, %v", users, listErr)
	}
}

func TestSaveRollsBackPasswordAfterSessionStoreFailureAndCanRetry(t *testing.T) {
	wantErr := errors.New("session write failed")
	secrets := newMemoryKeyring()
	secrets.setErrors[secretMapKey(sessionKind, "20260001")] = []error{wantErr}
	store := newStoreAt(filepath.Join(t.TempDir(), "users.json"), secrets)
	session := klas.Session{UserID: "user-id", Cookies: map[string]string{"SESSION": "cookie"}}

	err := store.Save(context.Background(), "20260001", "password", session)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Save() first error = %v, want %v", err, wantErr)
	}
	if len(secrets.values) != 0 {
		t.Fatalf("Save() credentials after rollback = %+v", secrets.values)
	}

	if err := store.Save(context.Background(), "20260001", "password", session); err != nil {
		t.Fatalf("Save() retry error = %v", err)
	}
	users, err := store.List(context.Background())
	if err != nil || len(users) != 1 || users[0].StudentID != "20260001" {
		t.Fatalf("List() after retry = %+v, %v", users, err)
	}
}

func TestSaveRollsBackCredentialsAfterRegistryFailure(t *testing.T) {
	secrets := newMemoryKeyring()
	store := newStoreAt(filepath.Join(t.TempDir(), "missing", "users.json"), secrets)

	err := store.Save(context.Background(), "20260001", "password", klas.Session{UserID: "user-id"})

	if err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Save() error = %v, want os.ErrNotExist", err)
	}
	if len(secrets.values) != 0 {
		t.Fatalf("Save() credentials after registry failure = %+v", secrets.values)
	}
}

func TestSaveRestoresExistingCredentialsAfterUpdateFailure(t *testing.T) {
	secrets := newMemoryKeyring()
	store := newStoreAt(filepath.Join(t.TempDir(), "users.json"), secrets)
	oldSession := klas.Session{UserID: "old-user", Cookies: map[string]string{"SESSION": "old"}}
	if err := store.Save(context.Background(), "20260001", "old-password", oldSession); err != nil {
		t.Fatalf("Save() seed error = %v", err)
	}
	wantErr := errors.New("session update failed")
	secrets.setErrors[secretMapKey(sessionKind, "20260001")] = []error{wantErr}

	err := store.Save(context.Background(), "20260001", "new-password", klas.Session{UserID: "new-user"})

	if !errors.Is(err, wantErr) {
		t.Fatalf("Save() update error = %v, want %v", err, wantErr)
	}
	password, passwordErr := store.LoadPassword(context.Background(), "20260001")
	if passwordErr != nil || password != "old-password" {
		t.Fatalf("LoadPassword() after rollback = %q, %v", password, passwordErr)
	}
	session, sessionErr := store.LoadSession(context.Background(), "20260001")
	if sessionErr != nil || session.UserID != oldSession.UserID || session.Cookies["SESSION"] != "old" {
		t.Fatalf("LoadSession() after rollback = %+v, %v", session, sessionErr)
	}
	users, listErr := store.List(context.Background())
	if listErr != nil || len(users) != 1 || users[0].UserID != oldSession.UserID {
		t.Fatalf("List() after rollback = %+v, %v", users, listErr)
	}
}

func TestSaveReturnsPartialFailureWhenRollbackFails(t *testing.T) {
	wantErr := errors.New("session write failed")
	rollbackErr := errors.New("password rollback failed")
	secrets := newMemoryKeyring()
	secrets.setErrors[secretMapKey(sessionKind, "20260001")] = []error{wantErr}
	secrets.deleteErrors[secretMapKey(passwordKind, "20260001")] = []error{rollbackErr}
	store := newStoreAt(filepath.Join(t.TempDir(), "users.json"), secrets)

	err := store.Save(context.Background(), "20260001", "password", klas.Session{UserID: "user-id"})

	var partial *PartialFailureError
	if !errors.As(err, &partial) {
		t.Fatalf("Save() error type = %T, want *PartialFailureError (%v)", err, err)
	}
	if !errors.Is(err, wantErr) || partial.Operation != "계정 저장" || len(partial.RollbackErrors) != 1 || !errors.Is(partial.RollbackErrors[0], rollbackErr) {
		t.Fatalf("Save() partial failure = %+v", partial)
	}
	if err := store.Save(context.Background(), "20260001", "password", klas.Session{UserID: "user-id"}); err != nil {
		t.Fatalf("Save() retry after partial failure error = %v", err)
	}
	users, listErr := store.List(context.Background())
	if listErr != nil || len(users) != 1 || users[0].StudentID != "20260001" {
		t.Fatalf("List() after partial failure retry = %+v, %v", users, listErr)
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
