package app

import (
	"context"
	"errors"
	"sort"
	"strings"
)

func (s *Service) QuerySearch(ctx context.Context, user UserOption, q AgentQuery, phrase string, kinds []string) (AgentResult, error) {
	phrase = strings.TrimSpace(phrase)
	if phrase == "" || len(phrase) > 800 {
		return AgentResult{}, errors.New("query must contain 1–200 characters")
	}
	if len([]rune(phrase)) > 200 {
		return AgentResult{}, errors.New("query too long")
	}
	if q.IncludeExpired != nil {
		return AgentResult{}, errors.New("include_expired requires lectures unfinished")
	}
	if q.Intent != "search" {
		return AgentResult{}, errors.New("search intent required")
	}
	now := q.referenceTime()
	q.asOf = now
	var w *QueryWindow
	if q.Period != nil {
		value, e := q.Period.Resolve(now)
		if e != nil {
			return AgentResult{}, e
		}
		w = &value
	}
	if len(kinds) == 0 {
		kinds = []string{"notice", "assignment", "lecture"}
	}
	for _, kind := range kinds {
		if kind != "notice" && kind != "assignment" && kind != "lecture" && kind != "academic" {
			return AgentResult{}, errors.New("unsupported search kind")
		}
	}
	years := queryYears(now, w)
	if len(years) > 5 {
		return AgentResult{}, errors.New("academic query period exceeds five years")
	}
	result, err := s.collectAgentSources(ctx, user, q, kinds, years)
	if err != nil {
		return result, err
	}
	result.Query.Period = w
	result.SearchedFields = []string{"title", "course_name"}
	selected := []AgentItem{}
	uncertain := false
	for _, item := range result.Items {
		if !textMatches(phrase, item.Title, result.Courses[item.CourseID]) {
			continue
		}
		if w != nil {
			date := item.EndAt
			if item.Kind == "lecture" {
				date = item.StartAt
			}
			if item.Kind == "notice" {
				date = item.RegisteredAt
			}
			if date == nil {
				uncertain = true
			} else if !w.Contains(*date) {
				continue
			}
		}
		selected = append(selected, item)
	}
	sort.SliceStable(selected, func(i, j int) bool { return selected[i].ID < selected[j].ID })
	result.Items = selected
	result.finish(uncertain)
	return result, nil
}
