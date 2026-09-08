package app

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type dashboardSyncStub struct {
	t     *testing.T
	ctx   context.Context
	calls []string
	err   error
}

func (s *dashboardSyncStub) record(ctx context.Context, name string, decisions map[string]SyncDecision) {
	s.t.Helper()
	if ctx != s.ctx || decisions["key"] != SyncDecisionKeep {
		s.t.Fatal("context/decisions changed")
	}
	s.calls = append(s.calls, name)
}
func (s *dashboardSyncStub) SyncAssignmentReminders(ctx context.Context, o AssignmentSyncOptions) (ReminderSyncResult, error) {
	s.record(ctx, "assignment", o.Decisions)
	if o.Query.User.StudentID != "student" {
		s.t.Fatal("user lost")
	}
	return ReminderSyncResult{EligibleCount: 1}, s.err
}
func (s *dashboardSyncStub) SyncLectureReminders(ctx context.Context, o LectureSyncOptions) (ReminderSyncResult, error) {
	s.record(ctx, "lecture", o.Decisions)
	if o.Query.User.StudentID != "student" {
		s.t.Fatal("user lost")
	}
	return ReminderSyncResult{EligibleCount: 2}, s.err
}
func (s *dashboardSyncStub) SyncAcademicCalendar(ctx context.Context, o AcademicSyncOptions) (CalendarSyncResult, error) {
	s.record(ctx, "academic", o.Decisions)
	return CalendarSyncResult{EligibleCount: 3}, s.err
}
func (s *dashboardSyncStub) SyncTimetableCalendar(ctx context.Context, o TimetableSyncOptions) (CalendarSyncResult, error) {
	s.record(ctx, "timetable", o.Decisions)
	if o.Query.User.StudentID != "student" {
		s.t.Fatal("user lost")
	}
	return CalendarSyncResult{EligibleCount: 4}, s.err
}

func TestSyncDashboardAttemptsEverySectionAndPreservesResults(t *testing.T) {
	for _, wantErr := range []error{nil, errors.New("section failed"), SyncConflictError{Conflicts: []SyncConflict{{Key: "key"}}}} {
		stub := &dashboardSyncStub{t: t, ctx: context.Background(), err: wantErr}
		result := syncDashboard(stub.ctx, DashboardSyncOptions{User: UserOption{StudentID: "student"}, Decisions: map[string]SyncDecision{"key": SyncDecisionKeep}}, stub)
		if !reflect.DeepEqual(stub.calls, []string{"assignment", "lecture", "academic", "timetable"}) {
			t.Fatalf("order=%v", stub.calls)
		}
		if result.Assignments.EligibleCount != 1 || result.Lectures.EligibleCount != 2 || result.Academic.EligibleCount != 3 || result.Timetable.EligibleCount != 4 {
			t.Fatalf("result=%#v", result)
		}
		for _, got := range []error{result.AssignmentError, result.LectureError, result.AcademicError, result.TimetableError} {
			if !reflect.DeepEqual(got, wantErr) {
				t.Fatalf("error=%v want=%v", got, wantErr)
			}
		}
	}
}
