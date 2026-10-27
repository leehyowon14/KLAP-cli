package main

import (
	"context"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

func init() { handlers["sync"] = syncCommand }
func syncCommand(ctx context.Context, s *app.Service, r request) error {
	v := s.SyncDashboard(ctx, app.DashboardSyncOptions{Decisions: r.Decisions})
	problems := []string{}
	for _, e := range []error{v.AssignmentError, v.LectureError, v.AcademicError, v.TimetableError} {
		if e != nil {
			problems = append(problems, e.Error())
		}
	}
	conflicts := append(v.Assignments.Conflicts, v.Lectures.Conflicts...)
	conflicts = append(conflicts, v.Academic.Conflicts...)
	conflicts = append(conflicts, v.Timetable.Conflicts...)
	emit("result", map[string]any{"errors": problems, "conflicts": conflicts})

	return nil
}
