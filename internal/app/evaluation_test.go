package app

import (
	"strings"
	"testing"
)

func TestEvaluationCacheVersionMissesRawLegacySchema(t *testing.T) {
	legacyKey := listCacheKey("evaluation", "20260001", "", "")
	currentKey := listCacheKeyVersion("evaluation", "v2", "20260001", "", "")
	if legacyKey == currentKey || !strings.Contains(currentKey, "evaluation:v2:") {
		t.Fatalf("evaluation cache keys = legacy %q, current %q", legacyKey, currentKey)
	}
}
