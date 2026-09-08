package app

import (
	"context"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"sort"
	"strings"
	"time"
)

type DashboardOptions struct {
	User    UserOption
	Refresh bool
}

type DashboardResult struct {
	Term           Term
	GeneratedAt    time.Time
	Cached         bool
	CacheCreatedAt time.Time
	Assignments    []AssignmentRow
	Notices        []NoticeRow
	Lectures       []LectureRow
	Courses        []DashboardCourse
	Attendance     DashboardAttendance
	Evaluation     DashboardEvaluation
	SectionErrors  []DashboardSectionError
}

type DashboardCourse struct {
	Index       int
	Name        string
	Assignments []AssignmentRow
	Notices     []NoticeRow
	Lectures    []LectureRow
	Attendance  *DashboardAttendanceRow
	Evaluation  *EvaluationRow
}

type DashboardAttendance struct {
	TotalCourses int
	Rows         []DashboardAttendanceRow
	Completed    int
	Absent       int
	Late         int
	LeaveEarly   int
	Excused      int
	Unknown      int
	DetailErrors int
}

type DashboardAttendanceRow struct {
	Index      int
	Course     AttendanceCourse
	Completed  int
	Absent     int
	Late       int
	LeaveEarly int
	Excused    int
	Unknown    int
	Err        error
}

type DashboardEvaluation struct {
	Term    klas.EvaluationTerm
	Enabled bool
	Done    int
	Pending int
	Rows    []EvaluationRow
}

type DashboardSectionError struct {
	Section string
	Err     error
}

func (s *Service) Dashboard(ctx context.Context, opts DashboardOptions) (DashboardResult, error) {
	s = s.withRequestSession()
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return DashboardResult{}, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	_ = client
	if err != nil {
		return DashboardResult{}, err
	}

	cacheKey := dashboardCacheKey(studentID, term.Value)
	if !opts.Refresh {
		var cached DashboardResult
		hit, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached)
		if cacheErr == nil && ok {
			result, migrated, migrationErr := normalizeCachedDashboardResult(cached, term)
			if migrationErr == nil {
				if migrated {
					_ = s.cacheStore.Set(cacheKey, dashboardCacheTTL(), result)
				}
				result.Cached = true
				result.CacheCreatedAt = hit.CreatedAt
				return result, nil
			}
		}
	}

	user := UserOption{StudentID: studentID}
	result := DashboardResult{
		Term:        term,
		GeneratedAt: time.Now(),
	}
	var assignmentRows []AssignmentRow
	var lectureRows []LectureRow
	var noticeRows []NoticeRow

	assignments, err := s.AssignmentList(ctx, AssignmentListOptions{User: user, Refresh: opts.Refresh})
	if err != nil {
		result.SectionErrors = append(result.SectionErrors, DashboardSectionError{Section: "과제", Err: err})
	} else {
		assignmentRows = assignments
		result.Assignments = dashboardAssignments(assignments, 5)
	}

	lectures, err := s.LectureList(ctx, LectureListOptions{User: user, Refresh: opts.Refresh})
	if err != nil {
		result.SectionErrors = append(result.SectionErrors, DashboardSectionError{Section: "온라인 강의", Err: err})
	} else {
		lectureRows = lectures
		result.Lectures = dashboardLectures(lectures, time.Now(), 5)
	}

	notices, err := s.NoticeList(ctx, NoticeListOptions{User: user, Refresh: opts.Refresh})
	if err != nil {
		result.SectionErrors = append(result.SectionErrors, DashboardSectionError{Section: "공지", Err: err})
	} else {
		noticeRows = notices
		result.Notices = dashboardNotices(notices, 5)
	}

	attendance, err := s.AttendanceList(ctx, AttendanceListOptions{User: user, Refresh: opts.Refresh})
	if err != nil {
		result.SectionErrors = append(result.SectionErrors, DashboardSectionError{Section: "출석", Err: err})
	} else {
		result.Attendance = dashboardAttendance(attendance.Rows)
	}

	evaluation, err := s.EvaluationList(ctx, EvaluationListOptions{User: user, Refresh: opts.Refresh})
	if err != nil {
		result.SectionErrors = append(result.SectionErrors, DashboardSectionError{Section: "수업평가", Err: err})
	} else {
		result.Evaluation = dashboardEvaluation(evaluation)
	}
	result.Courses = dashboardCourses(term.Courses, assignmentRows, lectureRows, noticeRows, result.Attendance, result.Evaluation)

	if dashboardCacheable(result) {
		cacheValue := result
		cacheValue.Cached = false
		cacheValue.CacheCreatedAt = time.Time{}
		_ = s.cacheStore.Set(cacheKey, dashboardCacheTTL(), cacheValue)
	}
	return result, nil
}

func dashboardAssignments(rows []AssignmentRow, limit int) []AssignmentRow {
	filtered := make([]AssignmentRow, 0, len(rows))
	now := time.Now()
	for _, row := range rows {
		if row.Assignment.Submitted {
			continue
		}
		if row.Assignment.DueAt != nil && row.Assignment.DueAt.Before(now) {
			continue
		}
		filtered = append(filtered, row)
	}
	return limitAssignments(filtered, limit)
}

func dashboardLectures(rows []LectureRow, now time.Time, limit int) []LectureRow {
	filtered := make([]LectureRow, 0, len(rows))
	for _, row := range rows {
		if !lectureNeedsAttendance(row.Lecture, now) {
			continue
		}
		filtered = append(filtered, row)
	}
	return limitLectures(filtered, limit)
}

func dashboardNotices(rows []NoticeRow, limit int) []NoticeRow {
	sorted := append([]NoticeRow(nil), rows...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := sorted[i].Notice.Registered
		right := sorted[j].Notice.Registered
		if left == nil && right == nil {
			return sorted[i].ID < sorted[j].ID
		}
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		if left.Equal(*right) {
			return sorted[i].ID < sorted[j].ID
		}
		return right.Before(*left)
	})
	return limitNotices(sorted, limit)
}

func dashboardAttendance(rows []AttendanceRow) DashboardAttendance {
	summary := DashboardAttendance{TotalCourses: len(rows)}
	for _, row := range rows {
		item := DashboardAttendanceRow{
			Index:  row.Index,
			Course: row.Course,
			Err:    row.Err,
		}
		if row.Err != nil {
			summary.DetailErrors++
			summary.Rows = append(summary.Rows, item)
			continue
		}
		for _, session := range row.Sessions {
			for _, slot := range session.Slots {
				switch strings.ToUpper(strings.TrimSpace(slot.Mark)) {
				case "O":
					item.Completed++
				case "X":
					item.Absent++
				case "L":
					item.Late++
				case "R":
					item.LeaveEarly++
				case "A":
					item.Excused++
				default:
					if strings.TrimSpace(slot.Mark+slot.Status) != "" {
						item.Unknown++
					}
				}
			}
		}
		summary.Completed += item.Completed
		summary.Absent += item.Absent
		summary.Late += item.Late
		summary.LeaveEarly += item.LeaveEarly
		summary.Excused += item.Excused
		summary.Unknown += item.Unknown
		summary.Rows = append(summary.Rows, item)
	}
	return summary
}

func dashboardEvaluation(result EvaluationListResult) DashboardEvaluation {
	evaluation := DashboardEvaluation{
		Term:    result.Term,
		Enabled: result.Term.TermEnabled,
	}
	for _, row := range result.Rows {
		if row.Course.Evaluated {
			evaluation.Done++
			continue
		}
		evaluation.Pending++
		evaluation.Rows = append(evaluation.Rows, row)
	}
	return evaluation
}

func dashboardCourses(courses []Course, assignments []AssignmentRow, lectures []LectureRow, notices []NoticeRow, attendance DashboardAttendance, evaluation DashboardEvaluation) []DashboardCourse {
	result := make([]DashboardCourse, 0, len(courses))
	now := time.Now()
	for index, course := range courses {
		name := strings.TrimSpace(course.Name)
		if name == "" {
			name = "과목 확인 필요"
		}
		item := DashboardCourse{
			Index:       index + 1,
			Name:        name,
			Assignments: dashboardAssignments(filterRowsByCourse(assignments, name, func(row AssignmentRow) string { return row.CourseName }), 5),
			Lectures:    dashboardLectures(filterRowsByCourse(lectures, name, func(row LectureRow) string { return row.CourseName }), now, 5),
			Notices:     dashboardNotices(filterRowsByCourse(notices, name, func(row NoticeRow) string { return row.CourseName }), 5),
		}
		if row, ok := dashboardAttendanceForCourse(attendance.Rows, name); ok {
			item.Attendance = &row
		}
		if row, ok := dashboardEvaluationForCourse(evaluation.Rows, name); ok {
			item.Evaluation = &row
		}
		result = append(result, item)
	}
	return result
}

func filterRowsByCourse[T any](rows []T, courseName string, course func(T) string) []T {
	filtered := make([]T, 0)
	for _, row := range rows {
		if strings.TrimSpace(course(row)) == courseName {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func dashboardAttendanceForCourse(rows []DashboardAttendanceRow, courseName string) (DashboardAttendanceRow, bool) {
	for _, row := range rows {
		if strings.TrimSpace(row.Course.Name) == courseName {
			return row, true
		}
	}
	return DashboardAttendanceRow{}, false
}

func dashboardEvaluationForCourse(rows []EvaluationRow, courseName string) (EvaluationRow, bool) {
	for _, row := range rows {
		if strings.TrimSpace(row.Course.Name) == courseName {
			return row, true
		}
	}
	return EvaluationRow{}, false
}

func dashboardCacheKey(studentID string, termValue string) string {
	return dashboardCacheKeyVersion("v3", studentID, termValue)
}

func dashboardCacheKeyVersion(version string, studentID string, termValue string) string {
	return "dashboard:" + strings.TrimSpace(version) + ":" + strings.TrimSpace(studentID) + ":" + strings.TrimSpace(termValue)
}

func dashboardCacheTTL() time.Duration {
	return 5 * time.Minute
}

func dashboardCacheable(result DashboardResult) bool {
	return len(result.SectionErrors) == 0 && result.Attendance.DetailErrors == 0
}

func normalizeCachedDashboardResult(result DashboardResult, term Term) (DashboardResult, bool, error) {
	migrated := false
	assignments, changed, err := normalizeCachedAssignmentRows(result.Assignments, term)
	if err != nil {
		return DashboardResult{}, false, err
	}
	result.Assignments = assignments
	migrated = migrated || changed
	notices, changed, err := normalizeCachedNoticeRows(result.Notices, term)
	if err != nil {
		return DashboardResult{}, false, err
	}
	result.Notices = notices
	migrated = migrated || changed
	lectures, changed, err := normalizeCachedLectureRows(result.Lectures, term)
	if err != nil {
		return DashboardResult{}, false, err
	}
	result.Lectures = lectures
	migrated = migrated || changed
	for index := range result.Courses {
		assignments, changed, err = normalizeCachedAssignmentRows(result.Courses[index].Assignments, term)
		if err != nil {
			return DashboardResult{}, false, err
		}
		result.Courses[index].Assignments = assignments
		migrated = migrated || changed
		notices, changed, err = normalizeCachedNoticeRows(result.Courses[index].Notices, term)
		if err != nil {
			return DashboardResult{}, false, err
		}
		result.Courses[index].Notices = notices
		migrated = migrated || changed
		lectures, changed, err = normalizeCachedLectureRows(result.Courses[index].Lectures, term)
		if err != nil {
			return DashboardResult{}, false, err
		}
		result.Courses[index].Lectures = lectures
		migrated = migrated || changed
	}
	return result, migrated, nil
}
