package app

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var academicDatePattern = regexp.MustCompile(`(?:(\d{1,2})\s*[./]\s*)?(\d{1,2})\s*(?:일|\([^)]*\))?`)

func academicEventDueAt(event AcademicEvent) (time.Time, bool) {
	startAt, _, ok := academicEventRange(event)
	return startAt, ok
}

func AcademicEventRange(event AcademicEvent) (time.Time, time.Time, bool) {
	return academicEventRange(event)
}

func academicEventRange(event AcademicEvent) (time.Time, time.Time, bool) {
	year, err := strconv.Atoi(strings.TrimSpace(event.Year))
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	monthText := strings.TrimSuffix(strings.TrimSpace(event.Month), "월")
	month, err := strconv.Atoi(monthText)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	dateText := strings.TrimSpace(event.Date)
	matches := academicDatePattern.FindAllStringSubmatch(dateText, -1)
	if len(matches) == 0 {
		return time.Time{}, time.Time{}, false
	}
	dates := make([]time.Time, 0, len(matches))
	for _, match := range matches {
		eventMonth := month
		if strings.TrimSpace(match[1]) != "" {
			parsedMonth, monthErr := strconv.Atoi(match[1])
			if monthErr != nil {
				return time.Time{}, time.Time{}, false
			}
			eventMonth = parsedMonth
		}
		day, dayErr := strconv.Atoi(match[2])
		if dayErr != nil {
			return time.Time{}, time.Time{}, false
		}
		dates = append(dates, time.Date(year, time.Month(eventMonth), day, 0, 0, 0, 0, time.Local))
	}
	startAt := dates[0]
	endAt := dates[len(dates)-1]
	if endAt.Before(startAt) {
		endAt = endAt.AddDate(1, 0, 0)
	}
	return startAt, endAt.AddDate(0, 0, 1), true
}
