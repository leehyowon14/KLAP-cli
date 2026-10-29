package app

import (
	"context"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"sort"
	"time"
)

func validateAssignmentQuery(q AgentQuery, now time.Time) (*QueryWindow, error) {
	if q.IncludeExpired != nil {
		return nil, errors.New("include_expired is only supported by lectures unfinished")
	}
	switch q.Intent {
	case "pending", "overdue", "submitted", "overview":
		if q.Period != nil {
			return nil, errors.New("period requires due_in_period")
		}
		return nil, nil
	case "due_in_period":
		if q.Period == nil {
			return nil, errors.New("period required")
		}
		w, e := q.Period.Resolve(now)
		return &w, e
	default:
		return nil, errors.New("unknown assignment intent")
	}
}
func assignmentMatches(v klas.Assignment, q AgentQuery, now time.Time, w *QueryWindow) (bool, bool) {
	unknown := v.SubmissionStatus == "unknown" || v.SubmissionStatus == ""
	switch q.Intent {
	case "overview":
		return true, false
	case "submitted":
		return v.SubmissionStatus == "submitted", unknown
	case "pending":
		return v.SubmissionStatus != "submitted", unknown
	case "overdue":
		return v.SubmissionStatus != "submitted" && v.DueAt != nil && v.DueAt.Before(now), v.SubmissionStatus != "submitted" && (unknown || v.DueAt == nil)
	case "due_in_period":
		return v.SubmissionStatus != "submitted" && v.DueAt != nil && w.Contains(*v.DueAt), v.SubmissionStatus != "submitted" && (unknown || v.DueAt == nil)
	}
	return false, false
}
func (s *Service) QueryAssignments(ctx context.Context, user UserOption, q AgentQuery) (AgentResult, error) {
	now := q.referenceTime()
	w, err := validateAssignmentQuery(q, now)
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
	uncertain := false
	for _, course := range courses {
		values, e := executeSessionRequest(ctx, s, student, &client, func(c *klas.Client) ([]klas.Assignment, error) {
			return s.assignmentGateway(c).Assignments(ctx, term.Value, course.Course)
		})
		staged := []AgentItem{}
		if e == nil {
			for _, v := range values {
				match, u := assignmentMatches(v, q, now, w)
				uncertain = uncertain || u
				if !match && !u {
					continue
				}
				id, err := StableAssignmentID(CourseRef{TermValue: term.Value, CourseID: course.Course.Value}, v.OrdSeq)
				if err != nil {
					e = err
					break
				}
				status := v.SubmissionStatus
				if status == "" {
					status = "unknown"
				}
				staged = append(staged, AgentItem{ID: id, Kind: "assignment", Title: v.Title, CourseID: course.Course.Value, StartAt: v.StartAt, EndAt: v.DueAt, Submitted: status})
			}
		}
		if e != nil {
			result.Coverage.Failures = append(result.Coverage.Failures, queryFailure(course.Course.Value, e))
			continue
		}
		result.Coverage.Succeeded++
		result.Items = append(result.Items, staged...)
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		a, b := result.Items[i], result.Items[j]
		if a.EndAt != nil && b.EndAt != nil && !a.EndAt.Equal(*b.EndAt) {
			return a.EndAt.Before(*b.EndAt)
		}
		return a.ID < b.ID
	})
	result.finish(uncertain)
	return result, nil
}
