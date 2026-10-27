package main

import (
	"context"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type academicCalendarEntry struct {
	app.AcademicEvent
	Start string
	End   string
}

func calendarEntries(events []app.AcademicEvent) []academicCalendarEntry {
	rows := make([]academicCalendarEntry, 0, len(events))
	for _, event := range events {
		row := academicCalendarEntry{AcademicEvent: event}
		if start, end, ok := app.AcademicEventRange(event); ok {
			row.Start = start.Format("2006-01-02")
			row.End = end.AddDate(0, 0, -1).Format("2006-01-02")
		}
		rows = append(rows, row)
	}
	return rows
}
func init() {
	handlers["academic-list"] = func(ctx context.Context, s *app.Service, r request) error {
		value, err := s.AcademicList(ctx, app.AcademicListOptions{Year: r.Year, Refresh: true})
		if err == nil {
			emit("result", map[string]any{"Year": value.Year, "SourceURL": value.SourceURL, "Events": calendarEntries(value.Events)})
		}
		return err
	}
}
