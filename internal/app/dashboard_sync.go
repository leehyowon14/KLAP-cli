package app

import "context"

type DashboardSyncOptions struct {
	User      UserOption
	Decisions map[string]SyncDecision
}

type DashboardSyncResult struct {
	Assignments     ReminderSyncResult
	Lectures        ReminderSyncResult
	Academic        CalendarSyncResult
	Timetable       CalendarSyncResult
	AssignmentError error
	LectureError    error
	AcademicError   error
	TimetableError  error
}

type dashboardSyncer interface {
	SyncAssignmentReminders(context.Context, AssignmentListOptions) (ReminderSyncResult, error)
	SyncLectureReminders(context.Context, LectureListOptions) (ReminderSyncResult, error)
	SyncAcademicCalendar(context.Context, AcademicListOptions) (CalendarSyncResult, error)
	SyncTimetableCalendar(context.Context, TimetableOptions) (CalendarSyncResult, error)
}

func (s *Service) SyncDashboard(ctx context.Context, opts DashboardSyncOptions) DashboardSyncResult {
	return syncDashboard(ctx, opts, s)
}

// Every section is attempted in order, retaining independent results and errors.
// Presentation decides how to render conflicts and partial failures.
func syncDashboard(ctx context.Context, opts DashboardSyncOptions, syncer dashboardSyncer) DashboardSyncResult {
	var result DashboardSyncResult
	result.Assignments, result.AssignmentError = syncer.SyncAssignmentReminders(ctx, AssignmentListOptions{User: opts.User, SyncDecisions: opts.Decisions})
	result.Lectures, result.LectureError = syncer.SyncLectureReminders(ctx, LectureListOptions{User: opts.User, SyncDecisions: opts.Decisions})
	result.Academic, result.AcademicError = syncer.SyncAcademicCalendar(ctx, AcademicListOptions{SyncDecisions: opts.Decisions})
	result.Timetable, result.TimetableError = syncer.SyncTimetableCalendar(ctx, TimetableOptions{User: opts.User, SyncDecisions: opts.Decisions})
	return result
}
