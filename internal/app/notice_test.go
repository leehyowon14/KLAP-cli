package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseNoticeID(t *testing.T) {
	courseIndex, boardNo, masterNo, err := ParseNoticeID("7:1161280:1000000")
	if err != nil {
		t.Fatalf("ParseNoticeID() error = %v", err)
	}
	if courseIndex != 7 || boardNo != "1161280" || masterNo != "1000000" {
		t.Fatalf("ParseNoticeID() = %d, %q, %q", courseIndex, boardNo, masterNo)
	}
}

func TestNoticeResourceIDAcceptsStableAndLegacyIDs(t *testing.T) {
	ref := CourseRef{TermValue: "2026,1", CourseID: "course:id/01"}
	stableID, err := StableNoticeID(ref, "1161280", "1000000")
	if err != nil {
		t.Fatalf("StableNoticeID() error = %v", err)
	}
	locator, boardNo, masterNo, err := parseNoticeResourceID(stableID)
	if err != nil || !locator.Stable || locator.Ref != ref || boardNo != "1161280" || masterNo != "1000000" {
		t.Fatalf("parseNoticeResourceID(stable) = %+v, %q, %q, %v", locator, boardNo, masterNo, err)
	}
	locator, boardNo, masterNo, err = parseNoticeResourceID("7:1161280:1000000")
	if err != nil || locator.Stable || locator.CourseIndex != 7 || boardNo != "1161280" || masterNo != "1000000" {
		t.Fatalf("parseNoticeResourceID(legacy) = %+v, %q, %q, %v", locator, boardNo, masterNo, err)
	}
}

func TestNormalizeCachedNoticeRowsUsesCourseNameAfterReorder(t *testing.T) {
	term := Term{Value: "2026,1", Courses: []Course{
		{Name: "오픈소스", Value: "course-b"},
		{Name: "컴퓨터그래픽스", Value: "course-a"},
	}}
	rows, migrated, err := normalizeCachedNoticeRows([]NoticeRow{{
		ID:         "1:1161280:1000000",
		TermValue:  term.Value,
		CourseName: "컴퓨터그래픽스",
	}}, term)
	if err != nil {
		t.Fatalf("normalizeCachedNoticeRows() error = %v", err)
	}
	if !migrated || len(rows) != 1 {
		t.Fatalf("normalizeCachedNoticeRows() = %+v, migrated %v", rows, migrated)
	}
	ref, remoteParts, stable, err := parseStableCourseResourceID("notice", rows[0].ID, 2)
	if err != nil || !stable || ref.CourseID != "course-a" || len(remoteParts) != 2 || remoteParts[0] != "1161280" || remoteParts[1] != "1000000" {
		t.Fatalf("migrated ID = %q, ref %+v, parts %v, stable %v, error %v", rows[0].ID, ref, remoteParts, stable, err)
	}
	persisted, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(persisted, []byte("1:1161280:1000000")) {
		t.Fatalf("persisted cache contains legacy ID: %s", persisted)
	}
}

func TestNoticeLegacyFilterCacheMustMatchSelectedCourse(t *testing.T) {
	term := Term{Value: "2026,1", Courses: []Course{
		{Name: "B", Value: "course-b"},
		{Name: "A", Value: "course-a"},
	}}
	rows, _, err := normalizeCachedNoticeRows([]NoticeRow{{ID: "1:board:master", CourseName: "A"}}, term)
	if err != nil {
		t.Fatalf("normalizeCachedNoticeRows() error = %v", err)
	}
	selected := []selectedCourse{{Index: 1, Course: term.Courses[0]}}
	if resourceIDsMatchSelected(noticeRowIDs(rows), "notice", 2, selected, false) {
		t.Fatalf("legacy numeric filter cache incorrectly matched current course: %+v", rows)
	}
}

func TestNoticeCacheVersionMissesRawLegacySchema(t *testing.T) {
	courses := []selectedCourse{{Index: 1, Course: Course{Name: "A", Value: "course-a"}}}
	legacyKey := courseResourceListCacheKey("notice", "20260001", "2026,1", courses)
	currentKey := courseResourceListCacheKeyVersion("notice", "v2", "20260001", "2026,1", courses)
	if legacyKey == currentKey || !strings.Contains(currentKey, "notice:v2:") {
		t.Fatalf("notice cache keys = legacy %q, current %q", legacyKey, currentKey)
	}
}
