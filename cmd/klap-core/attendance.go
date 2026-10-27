package main

import (
	"context"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type attendanceRow struct {
	Index    int
	Course   app.AttendanceCourse
	Sessions []app.AttendanceSession
	Error    string
}

func attendanceRows(rows []app.AttendanceRow) []attendanceRow {
	result := make([]attendanceRow, 0, len(rows))
	for _, row := range rows {
		value := attendanceRow{Index: row.Index, Course: row.Course, Sessions: row.Sessions}
		if row.Err != nil {
			value.Error = row.Err.Error()
		}
		result = append(result, value)
	}
	return result
}
func init() {
	handlers["attendance-list"] = func(ctx context.Context, s *app.Service, r request) error {
		value, err := s.AttendanceList(ctx, app.AttendanceListOptions{Refresh: true})
		if err == nil {
			emit("result", map[string]any{"Term": value.Term, "Rows": attendanceRows(value.Rows)})
		}
		return err
	}
	handlers["attendance-cdp"] = func(ctx context.Context, s *app.Service, r request) error {
		value, err := s.CdpAttendance(ctx, app.CdpAttendanceOptions{})
		if err == nil {
			emit("result", value)
		}
		return err
	}
}
