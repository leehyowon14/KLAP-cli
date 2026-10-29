package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"sort"
	"time"
)

func exactQueryCourses(term Term, ids []string) ([]selectedCourse, error) {
	all, _ := selectedCourses(term, "")
	if len(ids) == 0 {
		return all, nil
	}
	wanted := map[string]bool{}
	for _, id := range ids {
		if id == "" {
			return nil, errors.New("empty course ID")
		}
		wanted[id] = true
	}
	result := []selectedCourse{}
	for _, c := range all {
		if wanted[c.Course.Value] {
			result = append(result, c)
			delete(wanted, c.Course.Value)
		}
	}
	if len(wanted) > 0 {
		return nil, errors.New("unknown course ID in selected term")
	}
	return result, nil
}
func queryFailure(id string, err error) QueryFailure {
	reason := "lookup_failed"
	var remote *klas.Error
	if errors.As(err, &remote) {
		reason = string(remote.Kind)
	}
	return QueryFailure{CourseID: id, Reason: reason}
}
func newAgentResult(term Term, courses []selectedCourse, q AgentQuery, now time.Time) AgentResult {
	names := map[string]string{}
	ids := []string{}
	for _, c := range courses {
		names[c.Course.Value] = c.Course.Name
		ids = append(ids, c.Course.Value)
	}
	return AgentResult{Query: AppliedQuery{Intent: q.Intent, Term: term.Value, AsOf: now, CourseIDs: ids, IncludeExpired: q.IncludeExpired}, Coverage: QueryCoverage{Requested: len(courses), FetchedFrom: time.Now().UTC()}, Courses: names, Items: []AgentItem{}}
}
func validateLectureQuery(q AgentQuery, now time.Time) (*QueryWindow, error) {
	switch q.Intent {
	case "next_opening", "available_now", "due_in_period", "unfinished", "completed", "overview":
	default:
		return nil, errors.New("unknown lecture intent")
	}
	if (q.Intent == "unfinished") != (q.IncludeExpired != nil) {
		return nil, errors.New("include_expired is required only for unfinished")
	}
	if q.Intent == "due_in_period" {
		if q.Period == nil {
			return nil, errors.New("period required")
		}
		w, e := q.Period.Resolve(now)
		return &w, e
	}
	if q.Period != nil {
		return nil, errors.New("period is only supported by due_in_period")
	}
	return nil, nil
}
func lectureAvailability(l Lecture, now time.Time) string {
	if l.StartAt == nil || l.EndAt == nil || l.EndAt.Before(*l.StartAt) {
		return "unknown"
	}
	if now.Before(*l.StartAt) {
		return "upcoming"
	}
	if now.After(*l.EndAt) {
		return "expired"
	}
	return "open"
}
func selectLectureItems(rows []LectureRow, q AgentQuery, now time.Time, w *QueryWindow) ([]AgentItem, bool) {
	items := []AgentItem{}
	uncertain := false
	var earliest *time.Time
	for _, r := range rows {
		l := r.Lecture
		c := LectureCompletion(l)
		availability := lectureAvailability(l, now)
		ref, _, _, err := parseStableCourseResourceID("lecture", r.ID, 1)
		if err != nil {
			uncertain = true
			continue
		}
		item := AgentItem{ID: r.ID, Kind: "lecture", Title: l.Title, CourseID: ref.CourseID, StartAt: l.StartAt, EndAt: l.EndAt, Completion: &c, Availability: availability, Attendable: c.Status == "incomplete" && ValidateLectureAttendance(r, now) == nil, Week: l.WeekNo}
		match := false
		unknown := false
		switch q.Intent {
		case "overview":
			match = true
		case "completed":
			match = c.Status == "completed"
			unknown = c.Status == "unknown"
		case "unfinished":
			match = c.Status != "completed" && (*q.IncludeExpired || availability != "expired")
			unknown = match && (c.Status == "unknown" || availability == "unknown")
		case "available_now":
			match = c.Status != "completed" && availability == "open" && lectureAttendKey(l) != ""
			unknown = c.Status != "completed" && lectureAttendKey(l) != "" && (availability == "open" || availability == "unknown") && (c.Status == "unknown" || availability == "unknown")
		case "due_in_period":
			match = c.Status != "completed" && l.EndAt != nil && w.Contains(*l.EndAt)
			unknown = c.Status != "completed" && (l.EndAt == nil || w.Contains(*l.EndAt)) && (c.Status == "unknown" || availability == "unknown")
		case "next_opening":
			match = c.Status != "completed" && l.StartAt != nil && l.StartAt.After(now)
			unknown = c.Status != "completed" && (l.StartAt == nil || l.StartAt.After(now)) && (c.Status == "unknown" || availability == "unknown")
			if match && c.Status == "incomplete" && availability != "unknown" && (earliest == nil || l.StartAt.Before(*earliest)) {
				earliest = l.StartAt
			}
		}
		if unknown {
			uncertain = true
		}
		if match || unknown {
			items = append(items, item)
		}
	}
	if q.Intent == "next_opening" && earliest != nil {
		filtered := items[:0]
		for _, item := range items {
			if item.StartAt == nil || !item.StartAt.After(*earliest) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if q.Intent == "next_opening" {
		uncertain = false
		for _, item := range items {
			uncertain = uncertain || item.Completion.Status == "unknown" || item.Availability == "unknown"
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i].StartAt, items[j].StartAt
		if a == nil && b != nil {
			return false
		}
		if a != nil && b == nil {
			return true
		}
		if a != nil && b != nil && !a.Equal(*b) {
			return a.Before(*b)
		}
		return items[i].ID < items[j].ID
	})
	return items, uncertain
}

// QueryLectures always fetches every requested course, preserves partial success,
// then selects the semantic result set. Legacy CLI and event lists are unchanged.
func (s *Service) QueryLectures(ctx context.Context, user UserOption, q AgentQuery) (AgentResult, error) {
	now := q.referenceTime()
	w, err := validateLectureQuery(q, now)
	if err != nil {
		return AgentResult{}, err
	}
	student, err := s.selectedStudentID(ctx, user)
	if err != nil {
		return AgentResult{}, err
	}
	client, term, err := s.latestTerm(ctx, student)
	if err != nil {
		return AgentResult{}, err
	}
	courses, err := exactQueryCourses(term, q.CourseIDs)
	if err != nil {
		return AgentResult{}, err
	}
	result := newAgentResult(term, courses, q, now)
	result.Query.Period = w
	rows := []LectureRow{}
	for _, course := range courses {
		values, e := executeSessionRequest(ctx, s, student, &client, func(c *klas.Client) ([]klas.Lecture, error) { return c.Lectures(ctx, term.Value, course.Course) })
		staged := []LectureRow{}
		if e == nil {
			for _, v := range values {
				row, err := newLectureRow(term.Value, course, lectureModel(v))
				if err != nil {
					e = fmt.Errorf("lecture ID: %w", err)
					break
				}
				staged = append(staged, row)
			}
		}
		if e != nil {
			result.Coverage.Failures = append(result.Coverage.Failures, queryFailure(course.Course.Value, e))
			continue
		}
		result.Coverage.Succeeded++
		rows = append(rows, staged...)
	}
	result.Items, err = deduplicateLectureRows(rows, q, now, w)
	if err != nil {
		result.finish(true)
	} else {
		result.finish(false)
	}
	return result, nil
}
func deduplicateLectureRows(rows []LectureRow, q AgentQuery, now time.Time, w *QueryWindow) ([]AgentItem, error) {
	items, unknown := selectLectureItems(rows, q, now, w)
	seen := map[string]bool{}
	result := []AgentItem{}
	for _, item := range items {
		if seen[item.ID] {
			unknown = true
			continue
		}
		seen[item.ID] = true
		result = append(result, item)
	}
	if unknown {
		return result, errors.New("uncertain selection")
	}
	return result, nil
}
