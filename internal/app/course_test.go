package app

import (
	"strings"
	"testing"
)

func TestCourseRefStableResourceIDIgnoresCourseOrder(t *testing.T) {
	termValue := "2026,1"
	firstOrder := []Course{
		{Name: "컴퓨터그래픽스", Value: "U202613951I040013"},
		{Name: "오픈소스소프트웨어실습", Value: "U202613951I040014"},
	}
	secondOrder := []Course{firstOrder[1], firstOrder[0]}

	firstRef, err := NewCourseRef(termValue, firstOrder[0])
	if err != nil {
		t.Fatalf("NewCourseRef() first error = %v", err)
	}
	secondRef, err := NewCourseRef(termValue, secondOrder[1])
	if err != nil {
		t.Fatalf("NewCourseRef() second error = %v", err)
	}
	firstID, err := stableCourseResourceID("assignment", firstRef, "7")
	if err != nil {
		t.Fatalf("stableCourseResourceID() first error = %v", err)
	}
	secondID, err := stableCourseResourceID("assignment", secondRef, "7")
	if err != nil {
		t.Fatalf("stableCourseResourceID() second error = %v", err)
	}
	if firstID != secondID {
		t.Fatalf("stable ID changed after reorder: %q != %q", firstID, secondID)
	}

	parsedRef, remoteParts, stable, err := parseStableCourseResourceID("assignment", firstID, 1)
	if err != nil || !stable {
		t.Fatalf("parseStableCourseResourceID() = %+v, %v, %v", parsedRef, stable, err)
	}
	if parsedRef != firstRef || len(remoteParts) != 1 || remoteParts[0] != "7" {
		t.Fatalf("parseStableCourseResourceID() = %+v, %v", parsedRef, remoteParts)
	}
}

func TestCourseRefStableResourceIDSeparatesTermsAndCourses(t *testing.T) {
	course := Course{Name: "컴퓨터그래픽스", Value: "course:id/01"}
	firstRef, err := NewCourseRef("2026,1", course)
	if err != nil {
		t.Fatalf("NewCourseRef() first error = %v", err)
	}
	secondRef, err := NewCourseRef("2026,2", course)
	if err != nil {
		t.Fatalf("NewCourseRef() second error = %v", err)
	}
	otherCourseRef, err := NewCourseRef("2026,1", Course{Name: "다른 과목", Value: "course:id/02"})
	if err != nil {
		t.Fatalf("NewCourseRef() other course error = %v", err)
	}
	firstID, _ := stableCourseResourceID("lecture", firstRef, "content:id/1")
	secondID, _ := stableCourseResourceID("lecture", secondRef, "content:id/1")
	otherCourseID, _ := stableCourseResourceID("lecture", otherCourseRef, "content:id/1")
	if firstID == secondID || firstID == otherCourseID || secondID == otherCourseID {
		t.Fatalf("stable IDs collide: %q, %q, %q", firstID, secondID, otherCourseID)
	}

	parsedRef, remoteParts, stable, err := parseStableCourseResourceID("lecture", firstID, 1)
	if err != nil || !stable || parsedRef != firstRef || len(remoteParts) != 1 || remoteParts[0] != "content:id/1" {
		t.Fatalf("parseStableCourseResourceID() = %+v, %v, %v, %v", parsedRef, remoteParts, stable, err)
	}
}

func TestCourseRefRejectsMissingAndMalformedValues(t *testing.T) {
	if _, err := NewCourseRef("", Course{Value: "course"}); err == nil {
		t.Fatal("NewCourseRef() expected missing term error")
	}
	if _, err := NewCourseRef("2026,1", Course{}); err == nil {
		t.Fatal("NewCourseRef() expected missing course error")
	}
	if _, _, stable, err := parseStableCourseResourceID("assignment", "assignment:v1:not-base64!:Y291cnNl:Nw", 1); err == nil || !stable {
		t.Fatalf("parseStableCourseResourceID() malformed = stable %v, error %v", stable, err)
	}
	if _, _, stable, err := parseStableCourseResourceID("assignment", "3:7", 1); err != nil || stable {
		t.Fatalf("parseStableCourseResourceID() legacy = stable %v, error %v", stable, err)
	}
}

func TestSelectedCoursesByNumber(t *testing.T) {
	term := Term{Courses: []Course{
		{Name: "프로그래밍기초", Value: "c1"},
		{Name: "자료구조", Value: "c2"},
	}}

	selected, err := selectedCourses(term, "2")
	if err != nil {
		t.Fatalf("selectedCourses() error = %v", err)
	}
	if len(selected) != 1 || selected[0].Index != 2 || selected[0].Course.Name != "자료구조" {
		t.Fatalf("selectedCourses() = %+v", selected)
	}
}

func TestSelectedCoursesByName(t *testing.T) {
	term := Term{Courses: []Course{
		{Name: "프로그래밍기초", Value: "c1"},
		{Name: "자료구조", Value: "c2"},
	}}

	selected, err := selectedCourses(term, "자료")
	if err != nil {
		t.Fatalf("selectedCourses() error = %v", err)
	}
	if len(selected) != 1 || selected[0].Index != 2 {
		t.Fatalf("selectedCourses() = %+v", selected)
	}
}

func TestResolveResourceCourseRejectsStableIDFromDifferentTerm(t *testing.T) {
	term := Term{Value: "2026,2", Courses: []Course{{Name: "컴퓨터그래픽스", Value: "course-a"}}}
	_, err := resolveResourceCourse(term, courseResourceLocator{
		Ref:    CourseRef{TermValue: "2026,1", CourseID: "course-a"},
		Stable: true,
	})
	if err == nil {
		t.Fatal("resolveResourceCourse() expected term mismatch error")
	}
}

func TestCourseResourceListCacheKeyIgnoresCourseOrderAndFilterAlias(t *testing.T) {
	first := []selectedCourse{
		{Index: 1, Course: Course{Name: "A", Value: "course-a"}},
		{Index: 2, Course: Course{Name: "B", Value: "course-b"}},
	}
	second := []selectedCourse{
		{Index: 1, Course: first[1].Course},
		{Index: 2, Course: first[0].Course},
	}
	firstKey := courseResourceListCacheKey("assignment", "20260001", "2026,1", first)
	secondKey := courseResourceListCacheKey("assignment", "20260001", "2026,1", second)
	if firstKey != secondKey || strings.Contains(firstKey, ":1") {
		t.Fatalf("courseResourceListCacheKey() = %q, %q", firstKey, secondKey)
	}
	byNumber := courseResourceListCacheKey("assignment", "20260001", "2026,1", first[:1])
	byName := courseResourceListCacheKey("assignment", "20260001", "2026,1", []selectedCourse{{Index: 2, Course: first[0].Course}})
	if byNumber != byName {
		t.Fatalf("course filter aliases produced different keys: %q != %q", byNumber, byName)
	}
}

func TestResolveLegacyCachedCourseRejectsDuplicateNames(t *testing.T) {
	term := Term{Value: "2026,1", Courses: []Course{
		{Name: "캡스톤설계", Value: "course-a"},
		{Name: "캡스톤설계", Value: "course-b"},
	}}
	if _, err := resolveLegacyCachedCourse(term, 1, "캡스톤설계"); err == nil {
		t.Fatal("resolveLegacyCachedCourse() expected duplicate name error")
	}
}

func TestResolveLegacyCachedCourseRejectsMissingName(t *testing.T) {
	term := Term{Value: "2026,1", Courses: []Course{{Name: "컴퓨터그래픽스", Value: "course-a"}}}
	if _, err := resolveLegacyCachedCourse(term, 1, ""); err == nil {
		t.Fatal("resolveLegacyCachedCourse() expected missing name error")
	}
}
