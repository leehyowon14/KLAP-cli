package app

import (
	"strings"
	"testing"
)

func TestTimetableCacheVersionMissesRawLegacySchema(t *testing.T) {
	legacyKey := listCacheKey("timetable", "20260001", "2026,1", "")
	currentKey := listCacheKeyVersion("timetable", "v2", "20260001", "2026,1", "")
	if legacyKey == currentKey || !strings.Contains(currentKey, "timetable:v2:") {
		t.Fatalf("timetable cache keys = legacy %q, current %q", legacyKey, currentKey)
	}
}

func TestTimetablePeriodRangeUsesKwangwoonSlots(t *testing.T) {
	start, end, ok := timetablePeriodRange(1, 1)
	if !ok || start.Hour() != 9 || start.Minute() != 0 || end.Hour() != 10 || end.Minute() != 15 {
		t.Fatalf("period 1 = %02d:%02d-%02d:%02d ok=%t", start.Hour(), start.Minute(), end.Hour(), end.Minute(), ok)
	}
	start, end, ok = timetablePeriodRange(8, 1)
	if !ok || start.Hour() != 18 || start.Minute() != 50 || end.Hour() != 19 || end.Minute() != 35 {
		t.Fatalf("period 8 = %02d:%02d-%02d:%02d ok=%t", start.Hour(), start.Minute(), end.Hour(), end.Minute(), ok)
	}
	start, end, ok = timetablePeriodRange(5, 4)
	if !ok || start.Hour() != 15 || start.Minute() != 0 || end.Hour() != 18 || end.Minute() != 50 {
		t.Fatalf("period 5 span 4 = %02d:%02d-%02d:%02d ok=%t", start.Hour(), start.Minute(), end.Hour(), end.Minute(), ok)
	}
}
