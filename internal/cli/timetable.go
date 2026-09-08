package cli

import (
	"context"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strings"
)

func (r Runner) runTimetable(ctx context.Context, service *app.Service, args []string) error {
	if len(args) > 0 && args[0] == "list" {
		args = args[1:]
	} else if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		return fmt.Errorf("unknown timetable command: %s", args[0])
	}

	result, err := service.Timetable(ctx, app.TimetableOptions{
		User:    app.UserOption{StudentID: userFlag(args)},
		Refresh: hasFlag(args, "--refresh"),
	})
	if err != nil {
		return err
	}
	r.printTimetable(result)
	return nil
}

func (r Runner) printTimetable(result app.TimetableResult) {
	_, _ = fmt.Fprintf(r.Out, "%s (%s)\n", result.Term.Label, result.Term.Value)
	if len(result.Entries) == 0 {
		_, _ = fmt.Fprintln(r.Out, "시간표가 없습니다")
		return
	}

	printedOnlineHeader := false
	currentWeekday := 0
	for _, entry := range result.Entries {
		if entry.Online {
			if !printedOnlineHeader {
				_, _ = fmt.Fprintln(r.Out, "\n온라인/미지정")
				printedOnlineHeader = true
			}
			_, _ = fmt.Fprintf(r.Out, "  %s | %s | %s | %s\n",
				formatPeriod(entry.Period, entry.Span),
				entry.SubjectName,
				emptyFallback(entry.Room, "강의실 미지정"),
				emptyFallback(entry.Professor, "교수 미지정"),
			)
			continue
		}

		if entry.Weekday != currentWeekday {
			if currentWeekday != 0 {
				_, _ = fmt.Fprintln(r.Out)
			}
			currentWeekday = entry.Weekday
			_, _ = fmt.Fprintln(r.Out, weekdayLabel(entry.Weekday))
		}
		_, _ = fmt.Fprintf(r.Out, "  %s | %s | %s | %s\n",
			formatPeriod(entry.Period, entry.Span),
			entry.SubjectName,
			emptyFallback(entry.Room, "강의실 미지정"),
			emptyFallback(entry.Professor, "교수 미지정"),
		)
	}
}
