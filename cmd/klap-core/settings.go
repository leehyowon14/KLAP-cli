package main

import (
	"context"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

func init() {
	handlers["configure"] = func(ctx context.Context, s *app.Service, r request) error {
		_, err := s.UpdateConfig(app.ConfigUpdate{TimetableCalendarName: &r.TimetableName, AcademicCalendarName: &r.AcademicName, ReminderName: &r.ReminderName, TimetableCalendarUseExistingList: &r.TimetableExisting, AcademicCalendarUseExistingList: &r.AcademicExisting, ReminderUseExistingList: &r.ReminderExisting})
		if err == nil {
			emit("done", nil)
		}
		return err
	}
}
