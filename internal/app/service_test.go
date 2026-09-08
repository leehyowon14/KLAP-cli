package app

import (
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/account"
	"github.com/leehyowon14/KLAP-cli/internal/cache"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
	"github.com/leehyowon14/KLAP-cli/internal/syncstate"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var errDashboardTest = errors.New("dashboard test error")

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestNewServicePropagatesSettingsStoreFailure(t *testing.T) {
	wantErr := errors.New("settings unavailable")
	cacheFactoryCalled := false

	service, err := newService(nil, serviceStoreFactories{
		settings: func() (*settings.Store, error) {
			return nil, wantErr
		},
		cache: func() (*cache.Store, error) {
			cacheFactoryCalled = true
			return nil, nil
		},
		syncState: func() (*syncstate.Store, error) {
			t.Fatal("sync state factory should not run after settings factory failure")
			return nil, nil
		},
	})

	if service != nil {
		t.Fatalf("newService() service = %v, want nil", service)
	}
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "settings store 초기화 실패") {
		t.Fatalf("newService() error = %v", err)
	}
	if cacheFactoryCalled {
		t.Fatal("cache factory should not run after settings factory failure")
	}
}

func TestCourseRefStableResourceIDIgnoresCourseOrder(t *testing.T) {
	termValue := "2026,1"
	firstOrder := []klas.Course{
		{Name: "컴퓨터그래픽스", Value: "U202613951I040013"},
		{Name: "오픈소스소프트웨어실습", Value: "U202613951I040014"},
	}
	secondOrder := []klas.Course{firstOrder[1], firstOrder[0]}

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
	course := klas.Course{Name: "컴퓨터그래픽스", Value: "course:id/01"}
	firstRef, err := NewCourseRef("2026,1", course)
	if err != nil {
		t.Fatalf("NewCourseRef() first error = %v", err)
	}
	secondRef, err := NewCourseRef("2026,2", course)
	if err != nil {
		t.Fatalf("NewCourseRef() second error = %v", err)
	}
	otherCourseRef, err := NewCourseRef("2026,1", klas.Course{Name: "다른 과목", Value: "course:id/02"})
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
	if _, err := NewCourseRef("", klas.Course{Value: "course"}); err == nil {
		t.Fatal("NewCourseRef() expected missing term error")
	}
	if _, err := NewCourseRef("2026,1", klas.Course{}); err == nil {
		t.Fatal("NewCourseRef() expected missing course error")
	}
	if _, _, stable, err := parseStableCourseResourceID("assignment", "assignment:v1:not-base64!:Y291cnNl:Nw", 1); err == nil || !stable {
		t.Fatalf("parseStableCourseResourceID() malformed = stable %v, error %v", stable, err)
	}
	if _, _, stable, err := parseStableCourseResourceID("assignment", "3:7", 1); err != nil || stable {
		t.Fatalf("parseStableCourseResourceID() legacy = stable %v, error %v", stable, err)
	}
}

func TestNewServicePropagatesCacheStoreFailure(t *testing.T) {
	wantErr := errors.New("cache unavailable")

	service, err := newService(nil, serviceStoreFactories{
		settings: func() (*settings.Store, error) {
			return &settings.Store{}, nil
		},
		cache: func() (*cache.Store, error) {
			return nil, wantErr
		},
		syncState: func() (*syncstate.Store, error) {
			t.Fatal("sync state factory should not run after cache factory failure")
			return nil, nil
		},
	})

	if service != nil {
		t.Fatalf("newService() service = %v, want nil", service)
	}
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "cache store 초기화 실패") {
		t.Fatalf("newService() error = %v", err)
	}
}

func TestNewServicePropagatesSyncStateStoreFailure(t *testing.T) {
	wantErr := errors.New("sync state unavailable")
	service, err := newService(nil, serviceStoreFactories{
		settings: func() (*settings.Store, error) { return &settings.Store{}, nil },
		cache:    func() (*cache.Store, error) { return &cache.Store{}, nil },
		syncState: func() (*syncstate.Store, error) {
			return nil, wantErr
		},
	})
	if service != nil {
		t.Fatalf("newService() service = %v, want nil", service)
	}
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "sync state store 초기화 실패") {
		t.Fatalf("newService() error = %v", err)
	}
}

func TestNewServiceInitializesStores(t *testing.T) {
	accountStore := &account.Store{}
	settingsStore := &settings.Store{}
	cacheStore := &cache.Store{}
	syncStateStore, err := syncstate.NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt() error = %v", err)
	}

	service, err := newService(accountStore, serviceStoreFactories{
		settings:  func() (*settings.Store, error) { return settingsStore, nil },
		cache:     func() (*cache.Store, error) { return cacheStore, nil },
		syncState: func() (*syncstate.Store, error) { return syncStateStore, nil },
	})
	if err != nil {
		t.Fatalf("newService() error = %v", err)
	}
	if service.store != accountStore || service.sessions != accountStore || service.settingsStore != settingsStore || service.cacheStore != cacheStore || service.syncStateStore != syncStateStore {
		t.Fatalf("newService() stores = %+v", service)
	}
	if service.newKlasClient == nil || service.login == nil {
		t.Fatal("newService() authentication dependencies are nil")
	}
}

func TestSelectedCoursesByNumber(t *testing.T) {
	term := klas.Term{Courses: []klas.Course{
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
	term := klas.Term{Courses: []klas.Course{
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

func TestFutureTime(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.Local)
	past := now.Add(-time.Minute)
	same := now
	future := now.Add(time.Minute)

	if futureTime(nil, now) {
		t.Fatal("futureTime(nil) should be false")
	}
	if futureTime(&past, now) {
		t.Fatal("past deadline should not be synced")
	}
	if !futureTime(&same, now) || !futureTime(&future, now) {
		t.Fatal("current or future deadline should be synced")
	}
}

func TestDashboardCoursesBuildsCourseScopedSummary(t *testing.T) {
	due := time.Now().Add(24 * time.Hour)
	courses := []klas.Course{{Name: "컴퓨터그래픽스"}, {Name: "오픈소스소프트웨어실습"}}
	assignments := []AssignmentRow{
		{CourseName: "컴퓨터그래픽스", Assignment: klas.Assignment{Title: "과제1", DueAt: &due}},
		{CourseName: "오픈소스소프트웨어실습", Assignment: klas.Assignment{Title: "기말", DueAt: &due}},
	}
	lectures := []LectureRow{
		{CourseName: "오픈소스소프트웨어실습", Lecture: klas.Lecture{Title: "HuggingFace", ContentID: "a", Progress: "0", EndAt: &due}},
	}
	notices := []NoticeRow{
		{CourseName: "컴퓨터그래픽스", Notice: klas.Notice{Title: "공지"}},
	}
	attendance := DashboardAttendance{Rows: []DashboardAttendanceRow{{
		Course:    klas.AttendanceCourse{Name: "컴퓨터그래픽스"},
		Completed: 10,
	}}}

	got := dashboardCourses(courses, assignments, lectures, notices, attendance, DashboardEvaluation{})
	if len(got) != 2 {
		t.Fatalf("len(dashboardCourses) = %d", len(got))
	}
	if got[0].Name != "컴퓨터그래픽스" || len(got[0].Assignments) != 1 || len(got[0].Notices) != 1 || got[0].Attendance == nil {
		t.Fatalf("first dashboard course = %+v", got[0])
	}
	if got[1].Name != "오픈소스소프트웨어실습" || len(got[1].Assignments) != 1 || len(got[1].Lectures) != 1 {
		t.Fatalf("second dashboard course = %+v", got[1])
	}
}

func TestNormalizeCachedDashboardResultMigratesTopLevelAndCourseRows(t *testing.T) {
	term := klas.Term{Value: "2026,1", Courses: []klas.Course{{Name: "컴퓨터그래픽스", Value: "course-a"}}}
	legacyAssignment := AssignmentRow{ID: "1:7", CourseName: "컴퓨터그래픽스"}
	legacyNotice := NoticeRow{ID: "1:board:master", CourseName: "컴퓨터그래픽스"}
	legacyLecture := LectureRow{ID: "1:content", CourseName: "컴퓨터그래픽스", Lecture: klas.Lecture{ContentID: "content"}}
	result, migrated, err := normalizeCachedDashboardResult(DashboardResult{
		Assignments: []AssignmentRow{legacyAssignment},
		Notices:     []NoticeRow{legacyNotice},
		Lectures:    []LectureRow{legacyLecture},
		Courses: []DashboardCourse{{
			Assignments: []AssignmentRow{legacyAssignment},
			Notices:     []NoticeRow{legacyNotice},
			Lectures:    []LectureRow{legacyLecture},
		}},
	}, term)
	if err != nil {
		t.Fatalf("normalizeCachedDashboardResult() error = %v", err)
	}
	if !migrated {
		t.Fatal("normalizeCachedDashboardResult() migrated = false")
	}
	ids := []string{
		result.Assignments[0].ID,
		result.Notices[0].ID,
		result.Lectures[0].ID,
		result.Courses[0].Assignments[0].ID,
		result.Courses[0].Notices[0].ID,
		result.Courses[0].Lectures[0].ID,
	}
	for _, id := range ids {
		if !strings.Contains(id, ":v1:") {
			t.Fatalf("dashboard row ID was not migrated: %q", id)
		}
	}
}

func TestTranscriptBridgeCandidatesPreferBuiltBinary(t *testing.T) {
	got := transcriptBridgeCandidates("bridges/macos")
	want := []string{
		filepath.Join("bridges", "macos", ".build", "release", "TranscriptBridge"),
		filepath.Join("bridges", "macos", ".build", "debug", "TranscriptBridge"),
		filepath.Join("bridges", "macos", "transcribe.swift"),
	}
	if len(got) != len(want) {
		t.Fatalf("transcriptBridgeCandidates() length = %d, want %d: %#v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("transcriptBridgeCandidates()[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestExecutableDirectoryResolvesSymlink(t *testing.T) {
	root := t.TempDir()
	stagedDir := filepath.Join(root, "Caskroom", "klap", "1.0.0")
	if err := os.MkdirAll(stagedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(staged dir) error = %v", err)
	}
	executable := filepath.Join(stagedDir, "klap")
	if err := os.WriteFile(executable, []byte("binary"), 0o755); err != nil {
		t.Fatalf("WriteFile(executable) error = %v", err)
	}
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(bin dir) error = %v", err)
	}
	symlink := filepath.Join(binDir, "klap")
	if err := os.Symlink(executable, symlink); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	want, err := filepath.EvalSymlinks(stagedDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(staged dir) error = %v", err)
	}
	if got := executableDirectory(symlink); got != want {
		t.Fatalf("executableDirectory() = %q, want %q", got, want)
	}
}

func TestExecutableDirectoryFallsBackForMissingTarget(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "bin", "klap")
	if got, want := executableDirectory(executable), filepath.Dir(executable); got != want {
		t.Fatalf("executableDirectory() = %q, want %q", got, want)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestResolveResourceCourseRejectsStableIDFromDifferentTerm(t *testing.T) {
	term := klas.Term{Value: "2026,2", Courses: []klas.Course{{Name: "컴퓨터그래픽스", Value: "course-a"}}}
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
		{Index: 1, Course: klas.Course{Name: "A", Value: "course-a"}},
		{Index: 2, Course: klas.Course{Name: "B", Value: "course-b"}},
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
	term := klas.Term{Value: "2026,1", Courses: []klas.Course{
		{Name: "캡스톤설계", Value: "course-a"},
		{Name: "캡스톤설계", Value: "course-b"},
	}}
	if _, err := resolveLegacyCachedCourse(term, 1, "캡스톤설계"); err == nil {
		t.Fatal("resolveLegacyCachedCourse() expected duplicate name error")
	}
}

func TestResolveLegacyCachedCourseRejectsMissingName(t *testing.T) {
	term := klas.Term{Value: "2026,1", Courses: []klas.Course{{Name: "컴퓨터그래픽스", Value: "course-a"}}}
	if _, err := resolveLegacyCachedCourse(term, 1, ""); err == nil {
		t.Fatal("resolveLegacyCachedCourse() expected missing name error")
	}
}

func TestDashboardCacheVersionMissesRawLegacySchema(t *testing.T) {
	legacyKey := dashboardCacheKeyVersion("v2", "20260001", "2026,1")
	currentKey := dashboardCacheKey("20260001", "2026,1")
	if legacyKey == currentKey || !strings.Contains(currentKey, "dashboard:v3:") {
		t.Fatalf("dashboard cache keys = legacy %q, current %q", legacyKey, currentKey)
	}
}

func TestAcademicEventDueAt(t *testing.T) {
	got, ok := academicEventDueAt(AcademicEvent{
		Year:  "2026",
		Month: "6월",
		Date:  "06.17(수)",
		Title: "기말고사",
	})
	if !ok {
		t.Fatal("academicEventDueAt() expected ok")
	}
	if got.Year() != 2026 || got.Month() != time.June || got.Day() != 17 {
		t.Fatalf("academicEventDueAt() = %s", got)
	}
}

func TestAcademicEventRangeParsesMultiDayEvent(t *testing.T) {
	startAt, endAt, ok := AcademicEventRange(AcademicEvent{
		Year:  "2026",
		Month: "6월",
		Date:  "06.22(월) ~ 06.26(금)",
		Title: "보강주간",
	})
	if !ok {
		t.Fatal("AcademicEventRange() expected ok")
	}
	if startAt.Year() != 2026 || startAt.Month() != time.June || startAt.Day() != 22 {
		t.Fatalf("startAt = %s", startAt)
	}
	if endAt.Year() != 2026 || endAt.Month() != time.June || endAt.Day() != 27 {
		t.Fatalf("endAt = %s", endAt)
	}
}

func TestDashboardAssignmentsKeepsUpcomingUnsubmitted(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	rows := []AssignmentRow{
		{ID: "1:past", Assignment: klas.Assignment{Title: "지난 과제", DueAt: &past}},
		{ID: "1:done", Assignment: klas.Assignment{Title: "제출 과제", DueAt: &future, Submitted: true}},
		{ID: "1:todo", Assignment: klas.Assignment{Title: "할 과제", DueAt: &future}},
	}

	got := dashboardAssignments(rows, 5)
	if len(got) != 1 || got[0].ID != "1:todo" {
		t.Fatalf("dashboardAssignments() = %+v", got)
	}
}

func TestDashboardAttendanceCountsMarks(t *testing.T) {
	got := dashboardAttendance([]AttendanceRow{
		{
			Sessions: []klas.AttendanceSession{
				{Slots: []klas.AttendanceSlot{
					{Mark: "O"},
					{Mark: "X"},
					{Mark: "L"},
					{Mark: "R"},
					{Mark: "A"},
					{Mark: "??"},
				}},
			},
		},
		{Err: errDashboardTest},
	})

	if got.TotalCourses != 2 || got.Completed != 1 || got.Absent != 1 || got.Late != 1 || got.LeaveEarly != 1 || got.Excused != 1 || got.Unknown != 1 || got.DetailErrors != 1 {
		t.Fatalf("dashboardAttendance() = %+v", got)
	}
	if len(got.Rows) != 2 || got.Rows[0].Completed != 1 || got.Rows[0].Unknown != 1 || got.Rows[1].Err == nil {
		t.Fatalf("dashboardAttendance() rows = %+v", got.Rows)
	}
}

func TestDashboardNoticesSortsByRecentDate(t *testing.T) {
	oldDate := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	newDate := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	rows := []NoticeRow{
		{ID: "old", Notice: klas.Notice{Registered: &oldDate, Top: true}},
		{ID: "new", Notice: klas.Notice{Registered: &newDate}},
	}

	got := dashboardNotices(rows, 2)
	if len(got) != 2 || got[0].ID != "new" || got[1].ID != "old" {
		t.Fatalf("dashboardNotices() = %+v", got)
	}
}

func TestParseAcademicEvents(t *testing.T) {
	body := []byte(`
		<!--
		<h3>2025 학사일정</h3>
		<table><tbody><tr><td>3월</td><td>1(토)</td><td>오래된 일정</td><td></td></tr></tbody></table>
		-->
		<h3>2026 학사일정</h3>
		<table><tbody>
			<tr><td rowspan="2">3월</td><td>3(화)</td><td>2026학년도 1학기 개강(학기개시일)</td><td>3월2일(월) - 대체공휴일</td></tr>
			<tr><td>28(토)</td><td>수업일수 4분의 1</td><td>15주 기준</td></tr>
			<tr><td>8월</td><td>3(월)~31(월)</td><td>2학기 복학신청</td></tr>
		</tbody></table>
		<h3>2027 학사일정</h3>
		<table><tbody>
			<tr><td>3월</td><td>2(화)</td><td>2027학년도 1학기 개강</td><td></td></tr>
		</tbody></table>
	`)

	events, err := parseAcademicEvents(body, "2026")
	if err != nil {
		t.Fatalf("parseAcademicEvents() error = %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("parseAcademicEvents() len = %d, want 3: %+v", len(events), events)
	}
	if events[0].Month != "3월" || events[0].Date != "3(화)" || events[0].Title != "2026학년도 1학기 개강(학기개시일)" {
		t.Fatalf("parseAcademicEvents()[0] = %+v", events[0])
	}
	if events[1].Month != "3월" || events[1].Date != "28(토)" || events[1].Note != "15주 기준" {
		t.Fatalf("parseAcademicEvents()[1] = %+v", events[1])
	}
	if events[2].Month != "8월" || events[2].Date != "3(월)~31(월)" || events[2].Title != "2학기 복학신청" {
		t.Fatalf("parseAcademicEvents()[2] = %+v", events[2])
	}
}
