package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// collectAgentSources uses one reference time and records source-specific failures.
func (s *Service) collectAgentSources(ctx context.Context, user UserOption, q AgentQuery, kinds []string, years []int) (AgentResult, error) {
	s = s.withRequestSession()
	student, err := s.selectedStudentID(ctx, user)
	if err != nil {
		return AgentResult{}, err
	}
	_, term, err := s.latestTerm(ctx, student)
	if err != nil {
		return AgentResult{}, err
	}
	courses, err := exactQueryCourses(term, q.CourseIDs)
	if err != nil {
		return AgentResult{}, err
	}
	now := q.referenceTime()
	result := newAgentResult(term, courses, q, now)
	result.Sources = map[string]QueryCoverage{}
	failed := map[string]bool{}
	sourceQuery := AgentQuery{Intent: "overview", CourseIDs: q.CourseIDs, asOf: now}
	seen := map[string]bool{}
	for _, kind := range kinds {
		if seen[kind] {
			continue
		}
		seen[kind] = true
		var source AgentResult
		var e error
		switch kind {
		case "lecture":
			source, e = s.QueryLectures(ctx, user, sourceQuery)
		case "assignment":
			source, e = s.QueryAssignments(ctx, user, sourceQuery)
		case "notice":
			source, e = s.QueryNotices(ctx, user, sourceQuery)
		case "academic":
			source.Items = []AgentItem{}
			source.Coverage.FetchedFrom = time.Now().UTC()
			for _, year := range years {
				value, fetchErr := s.AcademicList(ctx, AcademicListOptions{Year: strconv.Itoa(year), Refresh: true})
				if fetchErr != nil {
					e = fetchErr
					break
				}
				for _, event := range value.Events {
					date, valid := academicEventDueAt(event)
					item := AgentItem{Kind: "academic", Title: event.Title, ID: fmt.Sprintf("academic:%x", sha256.Sum256([]byte(event.Year+event.Month+event.Date+event.Title)))}
					if valid {
						item.EndAt = &date
					}
					source.Items = append(source.Items, item)
				}
			}
			source.finish(false)
		default:
			return AgentResult{}, errors.New("unsupported kind")
		}
		if e != nil {
			source.Coverage = QueryCoverage{Status: "partial", Failures: []QueryFailure{{Reason: "lookup_failed"}}, FetchedFrom: now, FetchedUntil: time.Now().UTC()}
			for _, course := range courses {
				failed[course.Course.Value] = true
			}
		}
		result.Sources[kind] = source.Coverage
		for _, failure := range source.Coverage.Failures {
			failure.Reason = kind + ":" + failure.Reason
			result.Coverage.Failures = append(result.Coverage.Failures, failure)
			if failure.CourseID != "" {
				failed[failure.CourseID] = true
			}
		}
		result.Items = append(result.Items, source.Items...)
	}
	result.Coverage.Succeeded = result.Coverage.Requested - len(failed)
	if result.Coverage.Succeeded < 0 {
		result.Coverage.Succeeded = 0
	}
	return result, nil
}
func queryYears(now time.Time, w *QueryWindow) []int {
	start, end := now.Year(), now.Year()+1
	if w != nil {
		start = w.From.Year()
		end = w.Until.Add(-time.Nanosecond).Year()
	}
	years := []int{}
	for y := start; y <= end; y++ {
		years = append(years, y)
	}
	return years
}
