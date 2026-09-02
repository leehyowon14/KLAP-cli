package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/kw-klap/klap-cli/internal/klas"
	"github.com/zalando/go-keyring"
)

const (
	keyringService = "klap"
	passwordKind   = "password"
	sessionKind    = "session"
)

type Store struct {
	registryPath string
	keyring      keyringStore
}

type keyringStore interface {
	Set(service string, user string, password string) error
	Get(service string, user string) (string, error)
	Delete(service string, user string) error
}

type systemKeyring struct{}

func (systemKeyring) Set(service string, user string, password string) error {
	return keyring.Set(service, user, password)
}

func (systemKeyring) Get(service string, user string) (string, error) {
	return keyring.Get(service, user)
}

func (systemKeyring) Delete(service string, user string) error {
	return keyring.Delete(service, user)
}

type User struct {
	StudentID string    `json:"studentId"`
	SavedAt   time.Time `json:"savedAt"`
	UserID    string    `json:"userId,omitempty"`
}

type registryFile struct {
	CurrentStudentID string `json:"currentStudentId,omitempty"`
	Users            []User `json:"users"`
}

type PartialFailureError struct {
	Operation      string
	Cause          error
	RollbackErrors []error
}

func (e *PartialFailureError) Error() string {
	return fmt.Sprintf("%s 부분 실패: %v (rollback 실패: %v)", e.Operation, e.Cause, errors.Join(e.RollbackErrors...))
}

func (e *PartialFailureError) Unwrap() error {
	return e.Cause
}

type secretSnapshot struct {
	value  string
	exists bool
}

func NewStore() (*Store, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("config dir 생성 실패: %w", err)
	}

	return newStoreAt(filepath.Join(dir, "users.json"), systemKeyring{}), nil
}

func newStoreAt(registryPath string, secrets keyringStore) *Store {
	return &Store{registryPath: registryPath, keyring: secrets}
}

func configDir() (string, error) {
	if override := os.Getenv("KLAP_CONFIG_DIR"); override != "" {
		return override, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config dir 확인 실패: %w", err)
	}
	return filepath.Join(configDir, "klap"), nil
}

func (s *Store) Save(ctx context.Context, studentID string, password string, session klas.Session) error {
	_ = ctx

	sessionBytes, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("세션 직렬화 실패: %w", err)
	}

	registry, err := s.loadRegistry()
	if err != nil {
		return err
	}
	passwordSnapshot, err := s.snapshotSecret(passwordKind, studentID)
	if err != nil {
		return fmt.Errorf("기존 비밀번호 확인 실패: %w", err)
	}
	sessionSnapshot, err := s.snapshotSecret(sessionKind, studentID)
	if err != nil {
		return fmt.Errorf("기존 세션 확인 실패: %w", err)
	}

	if err := s.keyring.Set(keyringService, keyName(passwordKind, studentID), password); err != nil {
		cause := fmt.Errorf("비밀번호 보안 저장 실패: %w", err)
		return s.rollbackSecrets("계정 저장", studentID, cause, map[string]secretSnapshot{passwordKind: passwordSnapshot})
	}
	if err := s.keyring.Set(keyringService, keyName(sessionKind, studentID), string(sessionBytes)); err != nil {
		cause := fmt.Errorf("세션 보안 저장 실패: %w", err)
		return s.rollbackSecrets("계정 저장", studentID, cause, map[string]secretSnapshot{
			sessionKind:  sessionSnapshot,
			passwordKind: passwordSnapshot,
		})
	}

	updated := false
	for i := range registry.Users {
		if registry.Users[i].StudentID == studentID {
			registry.Users[i].SavedAt = time.Now()
			registry.Users[i].UserID = session.UserID
			updated = true
			break
		}
	}
	if !updated {
		registry.Users = append(registry.Users, User{
			StudentID: studentID,
			SavedAt:   time.Now(),
			UserID:    session.UserID,
		})
	}
	if registry.CurrentStudentID == "" {
		registry.CurrentStudentID = studentID
	}

	sort.Slice(registry.Users, func(i, j int) bool {
		return registry.Users[i].StudentID < registry.Users[j].StudentID
	})

	if err := s.saveRegistry(registry); err != nil {
		return s.rollbackSecrets("계정 저장", studentID, err, map[string]secretSnapshot{
			sessionKind:  sessionSnapshot,
			passwordKind: passwordSnapshot,
		})
	}
	return nil
}

func (s *Store) snapshotSecret(kind string, studentID string) (secretSnapshot, error) {
	value, err := s.keyring.Get(keyringService, keyName(kind, studentID))
	if errors.Is(err, keyring.ErrNotFound) {
		return secretSnapshot{}, nil
	}
	if err != nil {
		return secretSnapshot{}, err
	}
	return secretSnapshot{value: value, exists: true}, nil
}

func (s *Store) rollbackSecrets(operation string, studentID string, cause error, snapshots map[string]secretSnapshot) error {
	var rollbackErrors []error
	for _, kind := range []string{sessionKind, passwordKind} {
		snapshot, ok := snapshots[kind]
		if !ok {
			continue
		}
		var err error
		if snapshot.exists {
			err = s.keyring.Set(keyringService, keyName(kind, studentID), snapshot.value)
		} else {
			err = s.keyring.Delete(keyringService, keyName(kind, studentID))
			if errors.Is(err, keyring.ErrNotFound) {
				err = nil
			}
		}
		if err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("%s rollback 실패: %w", kind, err))
		}
	}
	if len(rollbackErrors) > 0 {
		return &PartialFailureError{
			Operation:      operation,
			Cause:          cause,
			RollbackErrors: rollbackErrors,
		}
	}
	return cause
}

func (s *Store) List(ctx context.Context) ([]User, error) {
	_ = ctx

	registry, err := s.loadRegistry()
	if err != nil {
		return nil, err
	}
	return registry.Users, nil
}

func (s *Store) Current(ctx context.Context) (string, error) {
	_ = ctx

	registry, err := s.loadRegistry()
	if err != nil {
		return "", err
	}
	return registry.CurrentStudentID, nil
}

func (s *Store) Select(ctx context.Context, studentID string) error {
	_ = ctx

	registry, err := s.loadRegistry()
	if err != nil {
		return err
	}

	for _, user := range registry.Users {
		if user.StudentID == studentID {
			registry.CurrentStudentID = studentID
			return s.saveRegistry(registry)
		}
	}
	return fmt.Errorf("저장된 유저가 없습니다: %s", studentID)
}

func (s *Store) LoadPassword(ctx context.Context, studentID string) (string, error) {
	_ = ctx

	password, err := s.keyring.Get(keyringService, keyName(passwordKind, studentID))
	if err != nil {
		return "", fmt.Errorf("저장된 비밀번호를 읽지 못했습니다: %w", err)
	}
	return password, nil
}

func (s *Store) LoadSession(ctx context.Context, studentID string) (klas.Session, error) {
	_ = ctx

	sessionText, err := s.keyring.Get(keyringService, keyName(sessionKind, studentID))
	if err != nil {
		return klas.Session{}, fmt.Errorf("저장된 세션을 읽지 못했습니다: %w", err)
	}

	var session klas.Session
	if err := json.Unmarshal([]byte(sessionText), &session); err != nil {
		return klas.Session{}, fmt.Errorf("저장된 세션 파싱 실패: %w", err)
	}
	return session, nil
}

func (s *Store) SaveSession(ctx context.Context, studentID string, session klas.Session) error {
	_ = ctx

	sessionBytes, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("세션 직렬화 실패: %w", err)
	}
	if err := s.keyring.Set(keyringService, keyName(sessionKind, studentID), string(sessionBytes)); err != nil {
		return fmt.Errorf("세션 보안 저장 실패: %w", err)
	}
	return nil
}

func (s *Store) Remove(ctx context.Context, studentID string) error {
	_ = ctx

	registry, err := s.loadRegistry()
	if err != nil {
		return err
	}

	filtered := registry.Users[:0]
	found := false
	for _, user := range registry.Users {
		if user.StudentID == studentID {
			found = true
			continue
		}
		filtered = append(filtered, user)
	}
	if !found {
		return fmt.Errorf("저장된 유저가 없습니다: %s", studentID)
	}
	passwordSnapshot, err := s.snapshotSecret(passwordKind, studentID)
	if err != nil {
		return fmt.Errorf("기존 비밀번호 확인 실패: %w", err)
	}
	sessionSnapshot, err := s.snapshotSecret(sessionKind, studentID)
	if err != nil {
		return fmt.Errorf("기존 세션 확인 실패: %w", err)
	}

	if err := s.deleteSecret(passwordKind, studentID); err != nil {
		cause := fmt.Errorf("비밀번호 보안 삭제 실패: %w", err)
		return s.rollbackSecrets("계정 삭제", studentID, cause, map[string]secretSnapshot{passwordKind: passwordSnapshot})
	}
	if err := s.deleteSecret(sessionKind, studentID); err != nil {
		cause := fmt.Errorf("세션 보안 삭제 실패: %w", err)
		return s.rollbackSecrets("계정 삭제", studentID, cause, map[string]secretSnapshot{
			sessionKind:  sessionSnapshot,
			passwordKind: passwordSnapshot,
		})
	}

	registry.Users = filtered
	if registry.CurrentStudentID == studentID {
		registry.CurrentStudentID = ""
		if len(filtered) > 0 {
			registry.CurrentStudentID = filtered[0].StudentID
		}
	}
	if err := s.saveRegistry(registry); err != nil {
		return s.rollbackSecrets("계정 삭제", studentID, err, map[string]secretSnapshot{
			sessionKind:  sessionSnapshot,
			passwordKind: passwordSnapshot,
		})
	}
	return nil
}

func (s *Store) deleteSecret(kind string, studentID string) error {
	err := s.keyring.Delete(keyringService, keyName(kind, studentID))
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

func (s *Store) loadRegistry() (registryFile, error) {
	bytes, err := os.ReadFile(s.registryPath)
	if errors.Is(err, os.ErrNotExist) {
		return registryFile{}, nil
	}
	if err != nil {
		return registryFile{}, fmt.Errorf("유저 registry 읽기 실패: %w", err)
	}

	var registry registryFile
	if err := json.Unmarshal(bytes, &registry); err != nil {
		return registryFile{}, fmt.Errorf("유저 registry 파싱 실패: %w", err)
	}
	return registry, nil
}

func (s *Store) saveRegistry(registry registryFile) error {
	bytes, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("유저 registry 직렬화 실패: %w", err)
	}

	if err := os.WriteFile(s.registryPath, bytes, 0o600); err != nil {
		return fmt.Errorf("유저 registry 저장 실패: %w", err)
	}
	return nil
}

func keyName(kind string, studentID string) string {
	return kind + ":" + studentID
}
