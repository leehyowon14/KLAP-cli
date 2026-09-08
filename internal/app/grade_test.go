package app

import (
	"strings"
	"testing"
)

func TestGradeCacheVersionMissesRawLegacySchema(t *testing.T) {
	legacyKey := listCacheKey("grade", "20260001", "2026,1", "")
	currentKey := listCacheKeyVersion("grade", "v2", "20260001", "2026,1", "")
	if legacyKey == currentKey || !strings.Contains(currentKey, "grade:v2:") {
		t.Fatalf("grade cache keys = legacy %q, current %q", legacyKey, currentKey)
	}
}

func TestRankCacheVersionMissesRawLegacySchema(t *testing.T) {
	legacyKey := listCacheKey("rank", "20260001", "2026,1", "")
	currentKey := listCacheKeyVersion("rank", "v2", "20260001", "2026,1", "")
	if legacyKey == currentKey || !strings.Contains(currentKey, "rank:v2:") {
		t.Fatalf("rank cache keys = legacy %q, current %q", legacyKey, currentKey)
	}
}
