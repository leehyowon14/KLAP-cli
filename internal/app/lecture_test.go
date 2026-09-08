package app

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/leehyowon14/KLAP-cli/internal/cache"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestLectureCacheHitAttendRefetchesActionPayload(t *testing.T) {
	studentID := "20260001"
	term := Term{
		Label: "2026학년도 1학기",
		Value: "2026,1",
		Courses: []Course{{
			Name:  "테스트 과목",
			Value: "subject-1",
		}},
	}
	selected := []selectedCourse{{Index: 1, Course: term.Courses[0]}}
	ref, err := NewCourseRef(term.Value, term.Courses[0])
	if err != nil {
		t.Fatalf("NewCourseRef() error = %v", err)
	}
	lectureID, err := StableLectureID(ref, "lrn-42")
	if err != nil {
		t.Fatalf("StableLectureID() error = %v", err)
	}
	cacheStore, err := cache.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}
	cacheKey := courseResourceListCacheKeyVersion("lecture", "v2", studentID, term.Value, selected)
	cachedRows := []LectureRow{{
		ID:         lectureID,
		LegacyID:   "1:lrn-42",
		TermValue:  term.Value,
		CourseName: term.Courses[0].Name,
		Lecture: Lecture{
			LearningSeq:  "42",
			Title:        "캐시 강의",
			RequiredTime: "1",
		},
	}}
	if err := cacheStore.Set(cacheKey, listCacheTTL(), cachedRows); err != nil {
		t.Fatalf("cache Set() error = %v", err)
	}

	client, err := klas.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	lectureFetches := 0
	savedPayload := ""
	previousTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var responseBody string
		switch request.URL.Path {
		case "/std/cmn/frame/YearhakgiAtnlcSbjectList.do":
			payload, marshalErr := json.Marshal([]Term{term})
			if marshalErr != nil {
				return nil, marshalErr
			}
			responseBody = string(payload)
		case "/std/lis/evltn/LctrumHomeStdInfo.do":
			responseBody = `{}`
		case "/std/lis/evltn/SelectOnlineCntntsStdList.do":
			lectureFetches++
			responseBody = `[{"grcode":"group-1","subj":"subject-1","year":"2026","hakgi":"1","bunban":"01","lrnSn":"42","ptime":"1","sbjt":"신선 강의"}]`
		case "/std/lis/evltn/SaveLrnStatus.do":
			body, readErr := io.ReadAll(request.Body)
			if readErr != nil {
				return nil, readErr
			}
			savedPayload = string(body)
			responseBody = `"Y"`
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`)), Request: request}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(responseBody)), Request: request}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previousTransport })

	service := &Service{
		sessions: &fakeSessionStore{
			loadSession: func(context.Context, string) (klas.Session, error) {
				return klas.Session{Cookies: map[string]string{"SESSION": "saved"}}, nil
			},
		},
		cacheStore: cacheStore,
		newKlasClient: func() (*klas.Client, error) {
			return client, nil
		},
	}
	rows, err := service.LectureList(context.Background(), LectureListOptions{User: UserOption{StudentID: studentID}})
	if err != nil {
		t.Fatalf("LectureList() error = %v", err)
	}
	if len(rows) != 1 || rows[0].Lecture.Title != "캐시 강의" || lectureFetches != 0 {
		t.Fatalf("LectureList() rows = %+v, lecture fetches = %d", rows, lectureFetches)
	}

	result, err := service.AttendLecture(context.Background(), lectureID, LectureAttendOptions{User: UserOption{StudentID: studentID}})
	if err != nil {
		t.Fatalf("AttendLecture() error = %v", err)
	}
	if lectureFetches != 1 || result.Lecture.Lecture.Title != "신선 강의" || !result.Progress.Completed {
		t.Fatalf("AttendLecture() result = %+v, lecture fetches = %d", result, lectureFetches)
	}
	for _, field := range []string{`"grcode":"group-1"`, `"subj":"subject-1"`, `"lrnSn":"42"`} {
		if !strings.Contains(savedPayload, field) {
			t.Fatalf("SaveLrnStatus payload %q missing %s", savedPayload, field)
		}
	}
}

func TestLectureCacheVersionMissesRawLegacySchema(t *testing.T) {
	courses := []selectedCourse{{Index: 1, Course: Course{Name: "A", Value: "course-a"}}}
	legacyKey := courseResourceListCacheKey("lecture", "20260001", "2026,1", courses)
	currentKey := courseResourceListCacheKeyVersion("lecture", "v2", "20260001", "2026,1", courses)
	if legacyKey == currentKey || !strings.Contains(currentKey, "lecture:v2:") {
		t.Fatalf("lecture cache keys = legacy %q, current %q", legacyKey, currentKey)
	}
}

func TestParseLectureID(t *testing.T) {
	courseIndex, contentID, err := ParseLectureID("7:content-123")
	if err != nil {
		t.Fatalf("ParseLectureID() error = %v", err)
	}
	if courseIndex != 7 || contentID != "content-123" {
		t.Fatalf("ParseLectureID() = %d, %q", courseIndex, contentID)
	}
}

func TestLectureResourceIDAcceptsStableAndLegacyIDs(t *testing.T) {
	ref := CourseRef{TermValue: "2026,1", CourseID: "course:id/01"}
	stableID, err := StableLectureID(ref, "content-123")
	if err != nil {
		t.Fatalf("StableLectureID() error = %v", err)
	}
	locator, lectureKey, err := parseLectureResourceID(stableID)
	if err != nil || !locator.Stable || locator.Ref != ref || lectureKey != "content-123" {
		t.Fatalf("parseLectureResourceID(stable) = %+v, %q, %v", locator, lectureKey, err)
	}
	locator, lectureKey, err = parseLectureResourceID("7:content-123")
	if err != nil || locator.Stable || locator.CourseIndex != 7 || lectureKey != "content-123" {
		t.Fatalf("parseLectureResourceID(legacy) = %+v, %q, %v", locator, lectureKey, err)
	}
}

func TestStableLectureTermValueRejectsMixedTerms(t *testing.T) {
	first, err := StableLectureID(CourseRef{TermValue: "2026,1", CourseID: "course-a"}, "content-a")
	if err != nil {
		t.Fatalf("StableLectureID() first error = %v", err)
	}
	second, err := StableLectureID(CourseRef{TermValue: "2026,2", CourseID: "course-b"}, "content-b")
	if err != nil {
		t.Fatalf("StableLectureID() second error = %v", err)
	}
	if _, err := stableLectureTermValue([]string{first, second}); err == nil {
		t.Fatal("stableLectureTermValue() expected mixed term error")
	}
	got, err := stableLectureTermValue([]string{"1:legacy", first})
	if err != nil || got != "2026,1" {
		t.Fatalf("stableLectureTermValue() = %q, %v", got, err)
	}
}

func TestNewLectureRowIDIgnoresCourseOrder(t *testing.T) {
	lecture := Lecture{ContentID: "content-123", Title: "소개"}
	course := Course{Name: "컴퓨터그래픽스", Value: "course-a"}
	first, err := newLectureRow("2026,1", selectedCourse{Index: 1, Course: course}, lecture)
	if err != nil {
		t.Fatalf("newLectureRow() first error = %v", err)
	}
	second, err := newLectureRow("2026,1", selectedCourse{Index: 4, Course: course}, lecture)
	if err != nil {
		t.Fatalf("newLectureRow() second error = %v", err)
	}
	if first.ID != second.ID || first.LegacyID != "1:content-123" || second.LegacyID != "4:content-123" {
		t.Fatalf("newLectureRow() first %+v, second %+v", first, second)
	}
}

func TestNormalizeCachedLectureRowsUsesCourseNameAfterReorder(t *testing.T) {
	term := Term{Value: "2026,1", Courses: []Course{
		{Name: "오픈소스", Value: "course-b"},
		{Name: "컴퓨터그래픽스", Value: "course-a"},
	}}
	rows, migrated, err := normalizeCachedLectureRows([]LectureRow{{
		ID:         "1:content-123",
		TermValue:  term.Value,
		CourseName: "컴퓨터그래픽스",
		Lecture:    Lecture{ContentID: "content-123"},
	}}, term)
	if err != nil {
		t.Fatalf("normalizeCachedLectureRows() error = %v", err)
	}
	if !migrated || len(rows) != 1 || rows[0].LegacyID != "1:content-123" {
		t.Fatalf("normalizeCachedLectureRows() = %+v, migrated %v", rows, migrated)
	}
	ref, remoteParts, stable, err := parseStableCourseResourceID("lecture", rows[0].ID, 1)
	if err != nil || !stable || ref.CourseID != "course-a" || len(remoteParts) != 1 || remoteParts[0] != "content-123" {
		t.Fatalf("migrated ID = %q, ref %+v, parts %v, stable %v, error %v", rows[0].ID, ref, remoteParts, stable, err)
	}
	persisted, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if bytes.Contains(persisted, []byte("1:content-123")) {
		t.Fatalf("persisted cache contains legacy ID: %s", persisted)
	}
}

func TestLectureLegacyFilterCacheMustMatchSelectedCourse(t *testing.T) {
	term := Term{Value: "2026,1", Courses: []Course{
		{Name: "B", Value: "course-b"},
		{Name: "A", Value: "course-a"},
	}}
	rows, _, err := normalizeCachedLectureRows([]LectureRow{{
		ID:         "1:content",
		CourseName: "A",
		Lecture:    Lecture{ContentID: "content"},
	}}, term)
	if err != nil {
		t.Fatalf("normalizeCachedLectureRows() error = %v", err)
	}
	selected := []selectedCourse{{Index: 1, Course: term.Courses[0]}}
	if resourceIDsMatchSelected(lectureRowIDs(rows), "lecture", 1, selected, false) {
		t.Fatalf("legacy numeric filter cache incorrectly matched current course: %+v", rows)
	}
}

func TestLectureResourceKeyHasStableFallbackWithoutAttendID(t *testing.T) {
	lecture := Lecture{ModuleTitle: "1주차", Title: "오리엔테이션"}
	first := lectureResourceKey(lecture)
	second := lectureResourceKey(lecture)
	if first == "" || first != second || !strings.HasPrefix(first, "meta-") {
		t.Fatalf("lectureResourceKey() = %q, %q", first, second)
	}
}

func TestLectureRowIDMatchesLearningSeqFallback(t *testing.T) {
	lecture := Lecture{LearningSeq: "39769"}
	if got := lectureRowID(1, lecture); got != "1:lrn-39769" {
		t.Fatalf("lectureRowID() = %q", got)
	}
	lecture.ContentID = "content-123"
	if got := lectureRowID(1, lecture); got != "1:content-123" {
		t.Fatalf("lectureRowID() content = %q", got)
	}
}

func TestLectureNeedsAttendance(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.FixedZone("KST", 9*60*60))
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)

	if !lectureNeedsAttendance(Lecture{ContentID: "content", Progress: "20", StartAt: &start, EndAt: &end}, now) {
		t.Fatal("lectureNeedsAttendance() expected true")
	}
	if lectureNeedsAttendance(Lecture{ContentID: "content", Progress: "100", StartAt: &start, EndAt: &end}, now) {
		t.Fatal("lectureNeedsAttendance() expected false for completed lecture")
	}
	if lectureNeedsAttendance(Lecture{ContentID: "", Progress: "20", StartAt: &start, EndAt: &end}, now) {
		t.Fatal("lectureNeedsAttendance() expected false without content id")
	}
	if !lectureNeedsAttendance(Lecture{LearningSeq: "15", AchievedTime: "0", RequiredTime: "10", StartAt: &start, EndAt: &end}, now) {
		t.Fatal("lectureNeedsAttendance() expected true for incomplete learning activity")
	}
	if lectureNeedsAttendance(Lecture{LearningSeq: "15", AchievedTime: "10", RequiredTime: "10", StartAt: &start, EndAt: &end}, now) {
		t.Fatal("lectureNeedsAttendance() expected false for completed learning activity")
	}
}
