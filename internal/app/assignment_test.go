package app

import (
	"bytes"
	"encoding/json"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"strings"
	"testing"
)

func TestParseAssignmentID(t *testing.T) {
	courseIndex, ordSeq, err := ParseAssignmentID("3:7")
	if err != nil {
		t.Fatalf("ParseAssignmentID() error = %v", err)
	}
	if courseIndex != 3 || ordSeq != "7" {
		t.Fatalf("ParseAssignmentID() = %d, %q", courseIndex, ordSeq)
	}
}

func TestAssignmentResourceIDAcceptsStableAndLegacyIDs(t *testing.T) {
	ref := CourseRef{TermValue: "2026,1", CourseID: "course:id/01"}
	stableID, err := StableAssignmentID(ref, "7")
	if err != nil {
		t.Fatalf("StableAssignmentID() error = %v", err)
	}
	locator, ordSeq, err := parseAssignmentResourceID(stableID)
	if err != nil || !locator.Stable || locator.Ref != ref || ordSeq != "7" {
		t.Fatalf("parseAssignmentResourceID(stable) = %+v, %q, %v", locator, ordSeq, err)
	}
	locator, ordSeq, err = parseAssignmentResourceID("3:7")
	if err != nil || locator.Stable || locator.CourseIndex != 3 || ordSeq != "7" {
		t.Fatalf("parseAssignmentResourceID(legacy) = %+v, %q, %v", locator, ordSeq, err)
	}
}

func TestNormalizeCachedAssignmentRowsUsesCourseNameAfterReorder(t *testing.T) {
	term := klas.Term{Value: "2026,1", Courses: []klas.Course{
		{Name: "오픈소스", Value: "course-b"},
		{Name: "컴퓨터그래픽스", Value: "course-a"},
	}}
	rows, migrated, err := normalizeCachedAssignmentRows([]AssignmentRow{{
		ID:         "1:7",
		TermValue:  term.Value,
		CourseName: "컴퓨터그래픽스",
	}}, term)
	if err != nil {
		t.Fatalf("normalizeCachedAssignmentRows() error = %v", err)
	}
	if !migrated || len(rows) != 1 || rows[0].LegacyID != "1:7" {
		t.Fatalf("normalizeCachedAssignmentRows() = %+v, migrated %v", rows, migrated)
	}
	ref, _, stable, err := parseStableCourseResourceID("assignment", rows[0].ID, 1)
	if err != nil || !stable || ref.CourseID != "course-a" {
		t.Fatalf("migrated ID = %q, ref %+v, stable %v, error %v", rows[0].ID, ref, stable, err)
	}
	persisted, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(persisted, []byte("1:7")) || bytes.Contains(persisted, []byte("LegacyID")) {
		t.Fatalf("persisted cache contains legacy ID: %s", persisted)
	}
}

func TestAssignmentCacheVersionMissesRawLegacySchema(t *testing.T) {
	courses := []selectedCourse{{Index: 1, Course: klas.Course{Name: "A", Value: "course-a"}}}
	legacyKey := courseResourceListCacheKey("assignment", "20260001", "2026,1", courses)
	currentKey := courseResourceListCacheKeyVersion("assignment", "v2", "20260001", "2026,1", courses)
	if legacyKey == currentKey || !strings.Contains(currentKey, "assignment:v2:") {
		t.Fatalf("assignment cache keys = legacy %q, current %q", legacyKey, currentKey)
	}
}

func TestAssignmentLegacyFilterCacheMustMatchSelectedCourse(t *testing.T) {
	term := klas.Term{Value: "2026,1", Courses: []klas.Course{
		{Name: "B", Value: "course-b"},
		{Name: "A", Value: "course-a"},
	}}
	rows, _, err := normalizeCachedAssignmentRows([]AssignmentRow{{ID: "1:7", CourseName: "A"}}, term)
	if err != nil {
		t.Fatalf("normalizeCachedAssignmentRows() error = %v", err)
	}
	selected := []selectedCourse{{Index: 1, Course: term.Courses[0]}}
	if resourceIDsMatchSelected(assignmentRowIDs(rows), "assignment", 1, selected, false) {
		t.Fatalf("legacy numeric filter cache incorrectly matched current course: %+v", rows)
	}
	if resourceIDsMatchSelected(nil, "assignment", 1, selected, false) {
		t.Fatal("empty filtered legacy cache must not be reused")
	}
}

func TestNormalizeCachedStableAssignmentHydratesCurrentAlias(t *testing.T) {
	term := klas.Term{Value: "2026,1", Courses: []klas.Course{
		{Name: "오픈소스", Value: "course-b"},
		{Name: "컴퓨터그래픽스", Value: "course-a"},
	}}
	id, err := StableAssignmentID(CourseRef{TermValue: term.Value, CourseID: "course-a"}, "7")
	if err != nil {
		t.Fatalf("StableAssignmentID() error = %v", err)
	}
	rows, migrated, err := normalizeCachedAssignmentRows([]AssignmentRow{{ID: id}}, term)
	if err != nil {
		t.Fatalf("normalizeCachedAssignmentRows() error = %v", err)
	}
	if migrated || rows[0].LegacyID != "2:7" {
		t.Fatalf("normalizeCachedAssignmentRows() = %+v, migrated %v", rows, migrated)
	}
}
