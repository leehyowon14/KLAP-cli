package app

import (
	"context"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"sort"
	"time"
)

func noticeWindow(q AgentQuery, now time.Time) (*QueryWindow, error) {
	if q.IncludeExpired != nil {
		return nil, errors.New("include_expired is only supported by lectures unfinished")
	}
	switch q.Intent {
	case "recent":
		if q.Period != nil {
			return nil, errors.New("recent fixes the last seven days")
		}
		return &QueryWindow{From: now.AddDate(0, 0, -7), Until: now}, nil
	case "in_period":
		if q.Period == nil {
			return nil, errors.New("period required")
		}
		w, e := q.Period.Resolve(now)
		return &w, e
	case "overview":
		if q.Period != nil {
			return nil, errors.New("period requires in_period")
		}
		return nil, nil
	default:
		return nil, errors.New("unknown notice intent")
	}
}
func (s *Service) QueryNotices(ctx context.Context, user UserOption, q AgentQuery) (AgentResult, error) {
	now := time.Now().UTC()
	w, err := noticeWindow(q, now)
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
		values, e := executeSessionRequest(ctx, s, student, &client, func(c *klas.Client) ([]klas.Notice, error) { return c.Notices(ctx, term.Value, course.Course) })
		staged := []AgentItem{}
		if e == nil {
			for _, v := range values {
				if w != nil {
					if v.Registered == nil {
						uncertain = true
					} else if !w.Contains(*v.Registered) {
						continue
					}
				}
				id, err := StableNoticeID(CourseRef{TermValue: term.Value, CourseID: course.Course.Value}, v.BoardNo, v.MasterNo)
				if err != nil {
					e = err
					break
				}
				count, valid := validNumber(v.FileCount)
				staged = append(staged, AgentItem{ID: id, Kind: "notice", Title: v.Title, CourseID: course.Course.Value, RegisteredAt: v.Registered, Pinned: v.Top, HasAttachments: valid && count > 0})
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
		if a.RegisteredAt != nil && b.RegisteredAt != nil && !a.RegisteredAt.Equal(*b.RegisteredAt) {
			return a.RegisteredAt.After(*b.RegisteredAt)
		}
		return a.ID < b.ID
	})
	result.finish(uncertain)
	return result, nil
}
