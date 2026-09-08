package account

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/leehyowon14/KLAP-cli/internal/domain"
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
	session := domain.Session{UserID: "user-id", Cookies: map[string]string{"SESSION": "cookie"}}

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

func TestSaveRegistryAtomicallyReplacesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.json")
	store := newStoreAt(path, newMemoryKeyring())
	want := registryFile{
		CurrentStudentID: "20260002",
		Users:            []User{{StudentID: "20260002", UserID: "user-id"}},
	}

	if err := store.saveRegistry(want); err != nil {
		t.Fatalf("saveRegistry() error = %v", err)
	}
	got, err := store.loadRegistry()
	if err != nil {
		t.Fatalf("loadRegistry() error = %v", err)
	}
	if got.CurrentStudentID != want.CurrentStudentID || len(got.Users) != 1 || got.Users[0].UserID != "user-id" {
		t.Fatalf("loadRegistry() = %+v", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if gotPerm := info.Mode().Perm(); runtime.GOOS == "windows" {
		if gotPerm&0o200 == 0 {
			t.Fatalf("registry permissions = %o, want owner-writable", gotPerm)
		}
	} else if gotPerm != 0o600 {
		t.Fatalf("registry permissions = %o, want 600", gotPerm)
	}
	temporaryFiles, err := filepath.Glob(filepath.Join(dir, ".users-*.tmp"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(temporaryFiles) != 0 {
		t.Fatalf("temporary registry files = %v", temporaryFiles)
	}
}

func TestSaveRegistryFailurePreservesLastValidState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.json")
	store := newStoreAt(path, newMemoryKeyring())
	original := registryFile{
		CurrentStudentID: "20260001",
		Users:            []User{{StudentID: "20260001", UserID: "old-user"}},
	}
	if err := store.saveRegistry(original); err != nil {
		t.Fatalf("saveRegistry() seed error = %v", err)
	}
	wantErr := errors.New("interrupted atomic write")
	backupPath := path + ".backup"
	store.registry = filesystemRegistryWriter{
		replace: func(_ string, registryPath string) error {
			if err := os.Rename(registryPath, backupPath); err != nil {
				t.Fatalf("simulated replacement backup error = %v", err)
			}
			return wantErr
		},
		recover: func(registryPath string) error {
			if _, err := os.Stat(backupPath); errors.Is(err, os.ErrNotExist) {
				return nil
			} else if err != nil {
				return err
			}
			return os.Rename(backupPath, registryPath)
		},
	}

	err := store.saveRegistry(registryFile{
		CurrentStudentID: "20260002",
		Users:            []User{{StudentID: "20260002", UserID: "new-user"}},
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("saveRegistry() error = %v, want %v", err, wantErr)
	}
	store.registry = filesystemRegistryWriter{
		replace: replaceRegistryFile,
		recover: recoverRegistryFile,
	}
	got, loadErr := store.loadRegistry()
	if loadErr != nil {
		t.Fatalf("loadRegistry() error = %v", loadErr)
	}
	if got.CurrentStudentID != original.CurrentStudentID || len(got.Users) != 1 || got.Users[0].UserID != "old-user" {
		t.Fatalf("loadRegistry() after interrupted write = %+v", got)
	}
	temporaryFiles, globErr := filepath.Glob(filepath.Join(filepath.Dir(path), ".users-*.tmp"))
	if globErr != nil {
		t.Fatalf("Glob() error = %v", globErr)
	}
	if len(temporaryFiles) != 0 {
		t.Fatalf("temporary registry files after failure = %v", temporaryFiles)
	}
}

func TestSaveRollsBackAfterPasswordStoreFailure(t *testing.T) {
	wantErr := errors.New("password write failed")
	secrets := newMemoryKeyring()
	secrets.setErrors[secretMapKey(passwordKind, "20260001")] = []error{wantErr}
	store := newStoreAt(filepath.Join(t.TempDir(), "users.json"), secrets)

	err := store.Save(context.Background(), "20260001", "password", domain.Session{UserID: "user-id"})

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
	session := domain.Session{UserID: "user-id", Cookies: map[string]string{"SESSION": "cookie"}}

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

	err := store.Save(context.Background(), "20260001", "password", domain.Session{UserID: "user-id"})

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
	oldSession := domain.Session{UserID: "old-user", Cookies: map[string]string{"SESSION": "old"}}
	if err := store.Save(context.Background(), "20260001", "old-password", oldSession); err != nil {
		t.Fatalf("Save() seed error = %v", err)
	}
	wantErr := errors.New("session update failed")
	secrets.setErrors[secretMapKey(sessionKind, "20260001")] = []error{wantErr}

	err := store.Save(context.Background(), "20260001", "new-password", domain.Session{UserID: "new-user"})

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

	err := store.Save(context.Background(), "20260001", "password", domain.Session{UserID: "user-id"})

	var partial *PartialFailureError
	if !errors.As(err, &partial) {
		t.Fatalf("Save() error type = %T, want *PartialFailureError (%v)", err, err)
	}
	if !errors.Is(err, wantErr) || partial.Operation != "계정 저장" || len(partial.RollbackErrors) != 1 || !errors.Is(partial.RollbackErrors[0], rollbackErr) {
		t.Fatalf("Save() partial failure = %+v", partial)
	}
	if err := store.Save(context.Background(), "20260001", "password", domain.Session{UserID: "user-id"}); err != nil {
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

func TestRemoveRestoresPasswordWhenSessionDeleteFailsAndCanRetry(t *testing.T) {
	secrets := newMemoryKeyring()
	store := newStoreAt(filepath.Join(t.TempDir(), "users.json"), secrets)
	session := domain.Session{UserID: "user-id", Cookies: map[string]string{"SESSION": "cookie"}}
	if err := store.Save(context.Background(), "20260001", "password", session); err != nil {
		t.Fatalf("Save() seed error = %v", err)
	}
	wantErr := errors.New("session delete failed")
	secrets.deleteErrors[secretMapKey(sessionKind, "20260001")] = []error{wantErr}

	err := store.Remove(context.Background(), "20260001")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Remove() first error = %v, want %v", err, wantErr)
	}
	password, passwordErr := store.LoadPassword(context.Background(), "20260001")
	if passwordErr != nil || password != "password" {
		t.Fatalf("LoadPassword() after rollback = %q, %v", password, passwordErr)
	}
	storedSession, sessionErr := store.LoadSession(context.Background(), "20260001")
	if sessionErr != nil || storedSession.UserID != session.UserID {
		t.Fatalf("LoadSession() after rollback = %+v, %v", storedSession, sessionErr)
	}
	users, listErr := store.List(context.Background())
	if listErr != nil || len(users) != 1 {
		t.Fatalf("List() after rollback = %+v, %v", users, listErr)
	}

	if err := store.Remove(context.Background(), "20260001"); err != nil {
		t.Fatalf("Remove() retry error = %v", err)
	}
	users, listErr = store.List(context.Background())
	if listErr != nil || len(users) != 0 {
		t.Fatalf("List() after retry = %+v, %v", users, listErr)
	}
}

func TestRemovePropagatesPasswordDeleteFailureWithoutChangingRegistry(t *testing.T) {
	secrets := newMemoryKeyring()
	store := newStoreAt(filepath.Join(t.TempDir(), "users.json"), secrets)
	if err := store.Save(context.Background(), "20260001", "password", domain.Session{UserID: "user-id"}); err != nil {
		t.Fatalf("Save() seed error = %v", err)
	}
	wantErr := errors.New("password delete failed")
	secrets.deleteErrors[secretMapKey(passwordKind, "20260001")] = []error{wantErr}

	err := store.Remove(context.Background(), "20260001")

	if !errors.Is(err, wantErr) {
		t.Fatalf("Remove() error = %v, want %v", err, wantErr)
	}
	users, listErr := store.List(context.Background())
	if listErr != nil || len(users) != 1 {
		t.Fatalf("List() after delete failure = %+v, %v", users, listErr)
	}
}

func TestRemoveTreatsKeyringNotFoundAsDeleted(t *testing.T) {
	secrets := newMemoryKeyring()
	store := newStoreAt(filepath.Join(t.TempDir(), "users.json"), secrets)
	if err := store.Save(context.Background(), "20260001", "password", domain.Session{UserID: "user-id"}); err != nil {
		t.Fatalf("Save() seed error = %v", err)
	}
	secrets.deleteErrors[secretMapKey(passwordKind, "20260001")] = []error{keyring.ErrNotFound}

	if err := store.Remove(context.Background(), "20260001"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	users, err := store.List(context.Background())
	if err != nil || len(users) != 0 {
		t.Fatalf("List() after remove = %+v, %v", users, err)
	}
}

func TestRemoveReturnsPartialFailureWhenCredentialRollbackFails(t *testing.T) {
	secrets := newMemoryKeyring()
	store := newStoreAt(filepath.Join(t.TempDir(), "users.json"), secrets)
	if err := store.Save(context.Background(), "20260001", "password", domain.Session{UserID: "user-id"}); err != nil {
		t.Fatalf("Save() seed error = %v", err)
	}
	wantErr := errors.New("session delete failed")
	rollbackErr := errors.New("password restore failed")
	secrets.deleteErrors[secretMapKey(sessionKind, "20260001")] = []error{wantErr}
	secrets.setErrors[secretMapKey(passwordKind, "20260001")] = []error{rollbackErr}

	err := store.Remove(context.Background(), "20260001")

	var partial *PartialFailureError
	if !errors.As(err, &partial) {
		t.Fatalf("Remove() error type = %T, want *PartialFailureError (%v)", err, err)
	}
	if !errors.Is(err, wantErr) || partial.Operation != "계정 삭제" || len(partial.RollbackErrors) != 1 || !errors.Is(partial.RollbackErrors[0], rollbackErr) {
		t.Fatalf("Remove() partial failure = %+v", partial)
	}
	users, listErr := store.List(context.Background())
	if listErr != nil || len(users) != 1 {
		t.Fatalf("List() after partial failure = %+v, %v", users, listErr)
	}
	if err := store.Remove(context.Background(), "20260001"); err != nil {
		t.Fatalf("Remove() retry after partial failure error = %v", err)
	}
	users, listErr = store.List(context.Background())
	if listErr != nil || len(users) != 0 {
		t.Fatalf("List() after partial failure retry = %+v, %v", users, listErr)
	}
}
