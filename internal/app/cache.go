package app

import (
	"errors"
	"strings"
	"time"
)

type CacheStatusResult struct {
	Dir   string
	Files int
	Bytes int64
}

type CacheClearResult struct {
	Removed int
}

func (s *Service) CacheStatus() (CacheStatusResult, error) {
	stats, err := s.cacheStore.Stats()
	if err != nil {
		return CacheStatusResult{}, err
	}
	return CacheStatusResult{
		Dir:   stats.Dir,
		Files: stats.Files,
		Bytes: stats.Bytes,
	}, nil
}

func (s *Service) ClearCache() (CacheClearResult, error) {
	removed, err := s.cacheStore.ClearExceptPrefixes("sync-source:")
	if err != nil {
		return CacheClearResult{}, err
	}
	return CacheClearResult{Removed: removed}, nil
}

func (s *Service) ClearCacheScope(scope string) (CacheClearResult, error) {
	normalizedScope := strings.ToLower(strings.TrimSpace(scope))
	if strings.HasPrefix(normalizedScope, "sync-source") {
		return CacheClearResult{}, errors.New("동기화 기준 cache는 삭제할 수 없습니다")
	}
	prefix := cacheScopePrefix(normalizedScope)
	if prefix == "" {
		return s.ClearCache()
	}
	removed, err := s.cacheStore.ClearPrefix(prefix)
	if err != nil {
		return CacheClearResult{}, err
	}
	return CacheClearResult{Removed: removed}, nil
}

func listCacheKey(scope string, studentID string, termValue string, selector string) string {
	return listCacheKeyVersion(scope, "v1", studentID, termValue, selector)
}

func listCacheKeyVersion(scope string, version string, studentID string, termValue string, selector string) string {
	parts := []string{
		strings.TrimSpace(scope),
		strings.TrimSpace(version),
		strings.TrimSpace(studentID),
		strings.TrimSpace(termValue),
		strings.TrimSpace(selector),
	}
	return strings.Join(parts, ":")
}

func listCacheTTL() time.Duration {
	return 5 * time.Minute
}

func cacheScopePrefix(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "", "all":
		return ""
	case "dashboard":
		return "dashboard:"
	case "course", "courses":
		return "course:"
	case "assignment", "assignments":
		return "assignment:"
	case "notice", "notices":
		return "notice:"
	case "lecture", "lectures":
		return "lecture:"
	case "timetable":
		return "timetable:"
	case "attendance":
		return "attendance:"
	case "academic":
		return "academic:"
	case "grade", "grades":
		return "grade:"
	case "rank", "ranks":
		return "rank:"
	case "evaluation", "evaluations":
		return "evaluation:"
	default:
		return strings.TrimSpace(scope) + ":"
	}
}
