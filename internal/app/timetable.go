package app

import (
	"context"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"sort"
	"strconv"
	"strings"
	"time"
)

type TimetableOptions struct {
	User    UserOption
	Refresh bool
}

type TimetableResult struct {
	Term    Term
	Entries []TimetableEntry
}

func (s *Service) Timetable(ctx context.Context, opts TimetableOptions) (TimetableResult, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return TimetableResult{}, err
	}
	client, term, err := s.latestTerm(ctx, studentID)
	if err != nil {
		return TimetableResult{}, err
	}

	cacheKey := listCacheKeyVersion("timetable", "v2", studentID, term.Value, "")
	if !opts.Refresh {
		var cached TimetableResult
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			return cached, nil
		}
	}

	entries, err := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) ([]klas.TimetableEntry, error) {
		return client.Timetable(ctx, term.Value)
	})
	if err != nil {
		return TimetableResult{}, err
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Online != entries[j].Online {
			return !entries[i].Online
		}
		if entries[i].Weekday != entries[j].Weekday {
			return entries[i].Weekday < entries[j].Weekday
		}
		if entries[i].Period != entries[j].Period {
			return entries[i].Period < entries[j].Period
		}
		return entries[i].SubjectName < entries[j].SubjectName
	})

	result := TimetableResult{
		Term:    term,
		Entries: timetableEntryModels(entries),
	}
	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), result)
	return result, nil
}

type dayClock struct {
	hour   int
	minute int
}

func (c dayClock) Hour() int {
	return c.hour
}

func (c dayClock) Minute() int {
	return c.minute
}

var timetableSinglePeriodTimes = map[int][2]dayClock{
	0:  {dayClock{8, 0}, dayClock{10, 45}},
	1:  {dayClock{9, 0}, dayClock{10, 15}},
	2:  {dayClock{10, 30}, dayClock{11, 45}},
	3:  {dayClock{12, 0}, dayClock{13, 15}},
	4:  {dayClock{13, 30}, dayClock{14, 45}},
	5:  {dayClock{15, 0}, dayClock{16, 15}},
	6:  {dayClock{16, 30}, dayClock{17, 45}},
	7:  {dayClock{18, 0}, dayClock{18, 45}},
	8:  {dayClock{18, 50}, dayClock{19, 35}},
	9:  {dayClock{19, 40}, dayClock{20, 25}},
	10: {dayClock{20, 30}, dayClock{21, 15}},
	11: {dayClock{21, 20}, dayClock{22, 5}},
}

var timetableConsecutiveTimes = map[int]map[int][2]dayClock{
	2: {
		0: {dayClock{8, 0}, dayClock{9, 50}},
		1: {dayClock{9, 0}, dayClock{10, 50}},
		3: {dayClock{12, 0}, dayClock{13, 50}},
		5: {dayClock{15, 0}, dayClock{16, 50}},
	},
	3: {
		0: {dayClock{8, 0}, dayClock{10, 45}},
		6: {dayClock{16, 30}, dayClock{19, 15}},
	},
	4: {
		0: {dayClock{8, 0}, dayClock{11, 50}},
		5: {dayClock{15, 0}, dayClock{18, 50}},
	},
}

func timetablePeriodRange(period int, span int) (dayClock, dayClock, bool) {
	if span <= 1 {
		times, ok := timetableSinglePeriodTimes[period]
		return times[0], times[1], ok
	}
	if byPeriod, ok := timetableConsecutiveTimes[span]; ok {
		if times, ok := byPeriod[period]; ok {
			return times[0], times[1], true
		}
	}
	start, ok := timetableSinglePeriodTimes[period]
	if !ok {
		return dayClock{}, dayClock{}, false
	}
	end, ok := timetableSinglePeriodTimes[period+span-1]
	if !ok {
		return dayClock{}, dayClock{}, false
	}
	return start[0], end[1], true
}

func timetableTermYear(termValue string) string {
	parts := strings.FieldsFunc(strings.TrimSpace(termValue), func(r rune) bool { return r == ',' || r == '-' })
	if len(parts) == 0 {
		return time.Now().Format("2006")
	}
	return parts[0]
}

func timetableTermRange(termValue string) (time.Time, time.Time) {
	year, _ := strconv.Atoi(timetableTermYear(termValue))
	if year <= 0 {
		year = time.Now().Year()
	}
	semester := "1"
	parts := strings.FieldsFunc(strings.TrimSpace(termValue), func(r rune) bool { return r == ',' || r == '-' })
	if len(parts) > 1 {
		semester = strings.TrimSpace(parts[1])
	}
	switch semester {
	case "2":
		return time.Date(year, time.September, 1, 0, 0, 0, 0, time.Local), time.Date(year, time.December, 31, 23, 59, 59, 0, time.Local)
	case "3":
		return time.Date(year, time.June, 22, 0, 0, 0, 0, time.Local), time.Date(year, time.August, 31, 23, 59, 59, 0, time.Local)
	case "4":
		return time.Date(year, time.December, 22, 0, 0, 0, 0, time.Local), time.Date(year+1, time.February, 28, 23, 59, 59, 0, time.Local)
	default:
		return time.Date(year, time.March, 1, 0, 0, 0, 0, time.Local), time.Date(year, time.June, 30, 23, 59, 59, 0, time.Local)
	}
}

func timetableTermRangeFromAcademic(termValue string, events []AcademicEvent, fallbackStart time.Time, fallbackEnd time.Time) (time.Time, time.Time) {
	start := fallbackStart
	end := fallbackEnd
	startMonth, endMonth := timetableTermMonths(termValue)
	for _, event := range events {
		eventStart, eventEnd, ok := academicEventRange(event)
		if !ok {
			continue
		}
		title := strings.TrimSpace(event.Title)
		if strings.Contains(title, "개강") && int(eventStart.Month()) == startMonth {
			start = eventStart
		}
		if strings.Contains(title, "종강") && int(eventStart.Month()) == endMonth {
			end = eventEnd.Add(-time.Second)
		}
	}
	if end.Before(start) {
		return fallbackStart, fallbackEnd
	}
	return start, end
}

func timetableTermMonths(termValue string) (int, int) {
	parts := strings.FieldsFunc(strings.TrimSpace(termValue), func(r rune) bool { return r == ',' || r == '-' })
	if len(parts) > 1 {
		switch strings.TrimSpace(parts[1]) {
		case "2":
			return 9, 12
		case "3":
			return 6, 8
		case "4":
			return 12, 2
		}
	}
	return 3, 6
}

func firstWeekdayOnOrAfter(startAt time.Time, weekday int) time.Time {
	target := time.Weekday(weekday % 7)
	for day := startAt; ; day = day.AddDate(0, 0, 1) {
		if day.Weekday() == target {
			return day
		}
	}
}
