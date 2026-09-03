package syncstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const schemaVersion = 1

type Item struct {
	Hash        string `json:"hash"`
	IgnoredHash string `json:"ignoredHash,omitempty"`
}

type State struct {
	Items map[string]Item `json:"items"`
}

type file struct {
	Version int              `json:"version"`
	Sources map[string]State `json:"sources"`
}

type atomicWriter interface {
	WriteAtomic(path string, data []byte, perm fs.FileMode) error
}

type filesystemAtomicWriter struct {
	replace func(temporaryPath string, targetPath string) error
	recover func(targetPath string) error
	syncDir func(targetPath string) error
}

type Store struct {
	mu     sync.Mutex
	path   string
	writer atomicWriter
}

func NewStore() (*Store, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	return NewStoreAt(dir)
}

func NewStoreAt(dir string) (*Store, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, errors.New("sync state dir가 비어 있습니다")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("sync state dir 생성 실패: %w", err)
	}
	return &Store{
		path: filepath.Join(dir, "sync-state.json"),
		writer: filesystemAtomicWriter{
			replace: replaceStateFile,
			recover: recoverStateFile,
		},
	}, nil
}

func (s *Store) Load(scope string, owner string) (State, bool, error) {
	if s == nil {
		return emptyState(), false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	var state State
	var found bool
	err := withStateFileLock(s.path, func() error {
		stored, exists, err := s.loadFile()
		if err != nil || !exists {
			return err
		}
		state, found = stored.Sources[sourceKey(scope, owner)]
		return nil
	})
	if err != nil {
		return State{}, false, err
	}
	if !found {
		return emptyState(), false, nil
	}
	if state.Items == nil {
		state.Items = map[string]Item{}
	}
	return state, true, nil
}

func (s *Store) Save(scope string, owner string, state State) error {
	if s == nil {
		return errors.New("sync state store가 없습니다")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	return withStateFileLock(s.path, func() error {
		stored, exists, err := s.loadFile()
		if err != nil {
			return err
		}
		if !exists {
			stored = file{Version: schemaVersion, Sources: map[string]State{}}
		}
		if stored.Sources == nil {
			stored.Sources = map[string]State{}
		}
		if state.Items == nil {
			state.Items = map[string]Item{}
		}
		stored.Sources[sourceKey(scope, owner)] = state

		body, err := json.MarshalIndent(stored, "", "  ")
		if err != nil {
			return fmt.Errorf("sync state 직렬화 실패: %w", err)
		}
		if err := s.writer.WriteAtomic(s.path, body, 0o600); err != nil {
			return fmt.Errorf("sync state 저장 실패: %w", err)
		}
		return nil
	})
}

func (s *Store) loadFile() (file, bool, error) {
	if recoverErr := recoverStateFile(s.path); recoverErr != nil {
		return file{}, false, fmt.Errorf("sync state 이전 상태 복구 실패: %w", recoverErr)
	}
	body, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return file{}, false, nil
	}
	if err != nil {
		return file{}, false, fmt.Errorf("sync state 읽기 실패: %w", err)
	}
	var stored file
	if err := json.Unmarshal(body, &stored); err != nil {
		return file{}, false, fmt.Errorf("sync state 파싱 실패: %w", err)
	}
	if stored.Version != schemaVersion {
		return file{}, false, fmt.Errorf("지원하지 않는 sync state schema version: %d", stored.Version)
	}
	if stored.Sources == nil {
		stored.Sources = map[string]State{}
	}
	return stored, true, nil
}

func (w filesystemAtomicWriter) WriteAtomic(path string, data []byte, perm fs.FileMode) (returnErr error) {
	if w.recover != nil {
		if err := w.recover(path); err != nil {
			return fmt.Errorf("sync state 이전 상태 복구 실패: %w", err)
		}
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".sync-state-*.tmp")
	if err != nil {
		return fmt.Errorf("sync state 임시 파일 생성 실패: %w", err)
	}
	temporaryPath := temporary.Name()
	closed := false
	defer func() {
		if !closed {
			if closeErr := temporary.Close(); closeErr != nil {
				returnErr = errors.Join(returnErr, fmt.Errorf("sync state 임시 파일 닫기 실패: %w", closeErr))
			}
		}
		if removeErr := os.Remove(temporaryPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			returnErr = errors.Join(returnErr, fmt.Errorf("sync state 임시 파일 정리 실패: %w", removeErr))
		}
	}()

	if err := temporary.Chmod(perm); err != nil {
		return fmt.Errorf("sync state 임시 파일 권한 설정 실패: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("sync state 임시 파일 쓰기 실패: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync state 임시 파일 동기화 실패: %w", err)
	}
	closed = true
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("sync state 임시 파일 닫기 실패: %w", err)
	}
	replace := w.replace
	if replace == nil {
		replace = replaceStateFile
	}
	if err := replace(temporaryPath, path); err != nil {
		if w.recover != nil {
			if recoverErr := w.recover(path); recoverErr != nil {
				return fmt.Errorf("sync state 교체 및 복구 실패: %w", errors.Join(err, recoverErr))
			}
		}
		return fmt.Errorf("sync state 교체 실패: %w", err)
	}
	syncDir := w.syncDir
	if syncDir == nil {
		syncDir = syncStateDirectory
	}
	if err := syncDir(path); err != nil {
		return fmt.Errorf("sync state directory 동기화 실패: %w", err)
	}
	return nil
}

func withStateFileLock(path string, operation func() error) (returnErr error) {
	release, err := acquireStateFileLock(path + ".lock")
	if err != nil {
		return fmt.Errorf("sync state lock 획득 실패: %w", err)
	}
	defer func() {
		if releaseErr := release(); releaseErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("sync state lock 해제 실패: %w", releaseErr))
		}
	}()
	return operation()
}

func emptyState() State {
	return State{Items: map[string]Item{}}
}

func sourceKey(scope string, owner string) string {
	return strings.TrimSpace(owner) + ":" + strings.TrimSpace(scope)
}

func configDir() (string, error) {
	if override := os.Getenv("KLAP_CONFIG_DIR"); override != "" {
		return override, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config dir 확인 실패: %w", err)
	}
	return filepath.Join(dir, "klap"), nil
}
