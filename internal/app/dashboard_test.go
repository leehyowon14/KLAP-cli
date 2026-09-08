package app

import (
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"strings"
	"testing"
	"time"
)

var errDashboardTest = errors.New("dashboard test error")

func TestDashboardCoursesBuildsCourseScopedSummary(t *testing.T) {
	due := time.Now().Add(24 * time.Hour)
	courses := []Course{{Name: "컴퓨터그래픽스"}, {Name: "오픈소스소프트웨어실습"}}
	assignments := []AssignmentRow{
		{CourseName: "컴퓨터그래픽스", Assignment: Assignment{Title: "과제1", DueAt: &due}},
		{CourseName: "오픈소스소프트웨어실습", Assignment: Assignment{Title: "기말", DueAt: &due}},
	}
	lectures := []LectureRow{
		{CourseName: "오픈소스소프트웨어실습", Lecture: klas.Lecture{Title: "HuggingFace", ContentID: "a", Progress: "0", EndAt: &due}},
	}
	notices := []NoticeRow{
		{CourseName: "컴퓨터그래픽스", Notice: Notice{Title: "공지"}},
	}
	attendance := DashboardAttendance{Rows: []DashboardAttendanceRow{{
		Course:    AttendanceCourse{Name: "컴퓨터그래픽스"},
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
	term := Term{Value: "2026,1", Courses: []Course{{Name: "컴퓨터그래픽스", Value: "course-a"}}}
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

func TestDashboardCacheVersionMissesRawLegacySchema(t *testing.T) {
	legacyKey := dashboardCacheKeyVersion("v2", "20260001", "2026,1")
	currentKey := dashboardCacheKey("20260001", "2026,1")
	if legacyKey == currentKey || !strings.Contains(currentKey, "dashboard:v3:") {
		t.Fatalf("dashboard cache keys = legacy %q, current %q", legacyKey, currentKey)
	}
}

func TestDashboardAssignmentsKeepsUpcomingUnsubmitted(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	rows := []AssignmentRow{
		{ID: "1:past", Assignment: Assignment{Title: "지난 과제", DueAt: &past}},
		{ID: "1:done", Assignment: Assignment{Title: "제출 과제", DueAt: &future, Submitted: true}},
		{ID: "1:todo", Assignment: Assignment{Title: "할 과제", DueAt: &future}},
	}

	got := dashboardAssignments(rows, 5)
	if len(got) != 1 || got[0].ID != "1:todo" {
		t.Fatalf("dashboardAssignments() = %+v", got)
	}
}

func TestDashboardAttendanceCountsMarks(t *testing.T) {
	got := dashboardAttendance([]AttendanceRow{
		{
			Sessions: []AttendanceSession{
				{Slots: []AttendanceSlot{
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
		{ID: "old", Notice: Notice{Registered: &oldDate, Top: true}},
		{ID: "new", Notice: Notice{Registered: &newDate}},
	}

	got := dashboardNotices(rows, 2)
	if len(got) != 2 || got[0].ID != "new" || got[1].ID != "old" {
		t.Fatalf("dashboardNotices() = %+v", got)
	}
}
