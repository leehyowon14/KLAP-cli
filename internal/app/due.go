package app

import (
	"context"
	"sort"
	"strings"
	"time"
)

type DueOptions struct {
	User    UserOption
	Days    int
	Refresh bool
}

type DueResult struct {
	From   time.Time
	Until  time.Time
	Items  []DueItem
	Errors []DashboardSectionError
}

type DueItem struct {
	Kind       string
	ID         string
	CourseName string
	Title      string
	DueAt      time.Time
	Status     string
}

func (s *Service) Due(ctx context.Context, opts DueOptions) (DueResult, error) {
	days := opts.Days
	if days <= 0 {
		days = 14
	}
	now := time.Now()
	result := DueResult{
		From:  now,
		Until: now.Add(time.Duration(days) * 24 * time.Hour),
	}
	user := UserOption{StudentID: strings.TrimSpace(opts.User.StudentID)}

	assignments, err := s.AssignmentList(ctx, AssignmentListOptions{User: user, Refresh: opts.Refresh})
	if err != nil {
		result.Errors = append(result.Errors, DashboardSectionError{Section: "과제", Err: err})
	} else {
		for _, row := range assignments {
			if row.Assignment.Submitted || row.Assignment.DueAt == nil || !inDueWindow(*row.Assignment.DueAt, result.From, result.Until) {
				continue
			}
			result.Items = append(result.Items, DueItem{
				Kind:       "과제",
				ID:         row.ID,
				CourseName: row.CourseName,
				Title:      row.Assignment.Title,
				DueAt:      *row.Assignment.DueAt,
				Status:     "미제출",
			})
		}
	}

	lectures, err := s.LectureList(ctx, LectureListOptions{User: user, Refresh: opts.Refresh})
	if err != nil {
		result.Errors = append(result.Errors, DashboardSectionError{Section: "온라인 강의", Err: err})
	} else {
		for _, row := range lectures {
			if !lectureNeedsAttendance(row.Lecture, now) || row.Lecture.EndAt == nil || !inDueWindow(*row.Lecture.EndAt, result.From, result.Until) {
				continue
			}
			result.Items = append(result.Items, DueItem{
				Kind:       "온라인 강의",
				ID:         row.ID,
				CourseName: row.CourseName,
				Title:      firstNonEmpty(row.Lecture.Title, row.Lecture.ModuleTitle),
				DueAt:      *row.Lecture.EndAt,
				Status:     lectureDueStatus(row.Lecture),
			})
		}
	}

	academic, err := s.AcademicList(ctx, AcademicListOptions{Refresh: opts.Refresh})
	if err != nil {
		result.Errors = append(result.Errors, DashboardSectionError{Section: "학사일정", Err: err})
	} else {
		for _, event := range academic.Events {
			dueAt, ok := academicEventDueAt(event)
			if !ok || !inDueWindow(dueAt, result.From, result.Until) {
				continue
			}
			result.Items = append(result.Items, DueItem{
				Kind:   "학사일정",
				Title:  event.Title,
				DueAt:  dueAt,
				Status: event.Note,
			})
		}
	}

	sort.SliceStable(result.Items, func(i, j int) bool {
		if result.Items[i].DueAt.Equal(result.Items[j].DueAt) {
			return result.Items[i].Title < result.Items[j].Title
		}
		return result.Items[i].DueAt.Before(result.Items[j].DueAt)
	})
	return result, nil
}

func inDueWindow(value time.Time, from time.Time, until time.Time) bool {
	return !value.Before(from) && !value.After(until)
}

func futureTime(value *time.Time, now time.Time) bool {
	return value != nil && !value.Before(now)
}

func formatProgressValue(achieved string, required string) string {
	return emptyStatusFallback(achieved, "0") + "/" + emptyStatusFallback(required, "?") + "분"
}

func emptyStatusFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
