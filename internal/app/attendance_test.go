package app

import (
	"strings"
	"testing"
)

func TestAttendanceCacheVersionMissesRawLegacySchema(t *testing.T) {
	legacyKey := listCacheKey("attendance", "20260001", "2026,1", "")
	currentKey := listCacheKeyVersion("attendance", "v2", "20260001", "2026,1", "")
	if legacyKey == currentKey || !strings.Contains(currentKey, "attendance:v2:") {
		t.Fatalf("attendance cache keys = legacy %q, current %q", legacyKey, currentKey)
	}
}
