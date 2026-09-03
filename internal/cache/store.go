package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Store struct {
	dir string
}

type Hit struct {
	CreatedAt time.Time
	ExpiresAt time.Time
}

type Stats struct {
	Dir   string
	Files int
	Bytes int64
}

type entry struct {
	Key       string          `json:"key"`
	CreatedAt time.Time       `json:"createdAt"`
	ExpiresAt time.Time       `json:"expiresAt"`
	Value     json.RawMessage `json:"value"`
}

func NewStore() (*Store, error) {
	dir, err := cacheDir()
	if err != nil {
		return nil, err
	}
	return NewStoreAt(dir)
}

func NewStoreAt(dir string) (*Store, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, errors.New("cache dir가 비어 있습니다")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("cache dir 생성 실패: %w", err)
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Get(key string, target any) (Hit, bool, error) {
	if s == nil {
		return Hit{}, false, nil
	}
	body, err := os.ReadFile(s.pathFor(key))
	if errors.Is(err, os.ErrNotExist) {
		return Hit{}, false, nil
	}
	if err != nil {
		return Hit{}, false, fmt.Errorf("cache 읽기 실패: %w", err)
	}

	var cached entry
	if err := json.Unmarshal(body, &cached); err != nil {
		return Hit{}, false, fmt.Errorf("cache 파싱 실패: %w", err)
	}
	if cached.Key != key {
		return Hit{}, false, errors.New("cache key가 일치하지 않습니다")
	}
	if time.Now().After(cached.ExpiresAt) {
		_ = os.Remove(s.pathFor(key))
		return Hit{}, false, nil
	}
	if err := json.Unmarshal(cached.Value, target); err != nil {
		return Hit{}, false, fmt.Errorf("cache 값 파싱 실패: %w", err)
	}
	return Hit{CreatedAt: cached.CreatedAt, ExpiresAt: cached.ExpiresAt}, true, nil
}

func (s *Store) Set(key string, ttl time.Duration, value any) error {
	if s == nil || ttl <= 0 {
		return nil
	}
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache 값 직렬화 실패: %w", err)
	}

	now := time.Now()
	body, err := json.MarshalIndent(entry{
		Key:       key,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
		Value:     valueBytes,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("cache 직렬화 실패: %w", err)
	}
	if err := os.WriteFile(s.pathFor(key), body, 0o600); err != nil {
		return fmt.Errorf("cache 저장 실패: %w", err)
	}
	return nil
}

func (s *Store) Delete(key string) (bool, error) {
	if s == nil {
		return false, nil
	}
	path := s.pathFor(key)
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cache 읽기 실패: %w", err)
	}
	var cached entry
	if err := json.Unmarshal(body, &cached); err != nil {
		return false, fmt.Errorf("cache 파싱 실패: %w", err)
	}
	if cached.Key != key || s.pathFor(cached.Key) != path {
		return false, errors.New("cache key가 일치하지 않습니다")
	}
	if err := os.Remove(path); err != nil {
		return false, fmt.Errorf("cache 삭제 실패: %w", err)
	}
	return true, nil
}

func (s *Store) Clear() (int, error) {
	return s.ClearPrefix("")
}

func (s *Store) ClearExceptPrefixes(preservedPrefixes ...string) (int, error) {
	if s == nil {
		return 0, nil
	}
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("cache 목록 조회 실패: %w", err)
	}

	count := 0
	for _, item := range entries {
		if item.IsDir() || filepath.Ext(item.Name()) != ".json" {
			continue
		}
		path := filepath.Join(s.dir, item.Name())
		body, err := os.ReadFile(path)
		if err != nil {
			return count, fmt.Errorf("cache 읽기 실패: %w", err)
		}
		var cached entry
		if err := json.Unmarshal(body, &cached); err != nil {
			return count, fmt.Errorf("cache 파싱 실패: %w", err)
		}
		if s.pathFor(cached.Key) != path {
			return count, errors.New("cache key가 일치하지 않습니다")
		}
		preserved := false
		for _, prefix := range preservedPrefixes {
			if prefix != "" && strings.HasPrefix(cached.Key, prefix) {
				preserved = true
				break
			}
		}
		if preserved {
			continue
		}
		if err := os.Remove(path); err != nil {
			return count, fmt.Errorf("cache 삭제 실패: %w", err)
		}
		count++
	}
	return count, nil
}

func (s *Store) ClearPrefix(prefix string) (int, error) {
	if s == nil {
		return 0, nil
	}
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("cache 목록 조회 실패: %w", err)
	}

	count := 0
	for _, item := range entries {
		if item.IsDir() || filepath.Ext(item.Name()) != ".json" {
			continue
		}
		path := filepath.Join(s.dir, item.Name())
		if prefix != "" {
			body, err := os.ReadFile(path)
			if err != nil {
				return count, fmt.Errorf("cache 읽기 실패: %w", err)
			}
			var cached entry
			if err := json.Unmarshal(body, &cached); err != nil {
				return count, fmt.Errorf("cache 파싱 실패: %w", err)
			}
			if s.pathFor(cached.Key) != path {
				return count, errors.New("cache key가 일치하지 않습니다")
			}
			if !strings.HasPrefix(cached.Key, prefix) {
				continue
			}
		}
		if err := os.Remove(path); err != nil {
			return count, fmt.Errorf("cache 삭제 실패: %w", err)
		}
		count++
	}
	return count, nil
}

func (s *Store) Stats() (Stats, error) {
	if s == nil {
		return Stats{}, nil
	}
	stats := Stats{Dir: s.dir}
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, os.ErrNotExist) {
		return stats, nil
	}
	if err != nil {
		return Stats{}, fmt.Errorf("cache 목록 조회 실패: %w", err)
	}
	for _, item := range entries {
		if item.IsDir() || filepath.Ext(item.Name()) != ".json" {
			continue
		}
		info, err := item.Info()
		if err != nil {
			return Stats{}, fmt.Errorf("cache 파일 확인 실패: %w", err)
		}
		stats.Files++
		stats.Bytes += info.Size()
	}
	return stats, nil
}

func (s *Store) pathFor(key string) string {
	hash := sha256.Sum256([]byte(key))
	return filepath.Join(s.dir, hex.EncodeToString(hash[:])+".json")
}

func cacheDir() (string, error) {
	if override := os.Getenv("KLAP_CACHE_DIR"); override != "" {
		return override, nil
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("cache dir 확인 실패: %w", err)
	}
	return filepath.Join(cacheDir, "klap"), nil
}
