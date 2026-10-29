package app

import (
	"context"
	"errors"
	"sort"
	"time"
)

func (s *Service) QueryDue(ctx context.Context, user UserOption, q AgentQuery, kinds []string) (AgentResult, error) {
	now := q.referenceTime()
	q.asOf = now
	if q.IncludeExpired != nil {
		return AgentResult{}, errors.New("include_expired requires lectures unfinished")
	}
	var w *QueryWindow
	switch q.Intent {
	case "next_deadline":
		if q.Period != nil {
			return AgentResult{}, errors.New("next_deadline does not take period")
		}
	case "in_period":
		if q.Period == nil {
			return AgentResult{}, errors.New("period required")
		}
		value, e := q.Period.Resolve(now)
		if e != nil {
			return AgentResult{}, e
		}
		w = &value
	default:
		return AgentResult{}, errors.New("unknown deadline intent")
	}
	if len(kinds) == 0 {
		kinds = []string{"assignment", "lecture", "academic"}
	}
	for _, kind := range kinds {
		if kind != "assignment" && kind != "lecture" && kind != "academic" {
			return AgentResult{}, errors.New("unsupported deadline kind")
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
	result.Query.Kinds = kinds
	selected, uncertain := selectDueCandidates(result.Items, q, now, w)
	result.Items = selected
	result.finish(uncertain)
	return result, nil
}

func selectDueCandidates(items []AgentItem, q AgentQuery, now time.Time, w *QueryWindow) ([]AgentItem, bool) {
	selected := []AgentItem{}
	uncertain := false
	for _, item := range items {
		if item.Completion != nil && item.Completion.Status == "completed" || item.Submitted == "submitted" {
			continue
		}
		unknown := item.Completion != nil && item.Completion.Status == "unknown" || item.Submitted == "unknown" || item.EndAt == nil
		if item.EndAt != nil {
			if w != nil && !w.Contains(*item.EndAt) {
				continue
			}
			if w == nil && item.EndAt.Before(now) {
				continue
			}
		}
		uncertain = uncertain || unknown
		selected = append(selected, item)
	}
	sort.SliceStable(selected, func(i, j int) bool {
		a, b := selected[i].EndAt, selected[j].EndAt
		if a == nil && b == nil {
			return selected[i].ID < selected[j].ID
		}
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		if !a.Equal(*b) {
			return a.Before(*b)
		}
		return selected[i].ID < selected[j].ID
	})
	if q.Intent == "next_deadline" {
		var first *time.Time
		for _, item := range selected {
			if item.EndAt != nil && (item.Completion == nil || item.Completion.Status != "unknown") && item.Submitted != "unknown" {
				first = item.EndAt
				break
			}
		}
		if first != nil {
			all := selected
			selected = []AgentItem{}
			for _, item := range all {
				if item.EndAt == nil || !item.EndAt.After(*first) {
					selected = append(selected, item)
				}
			}
		}
		uncertain = false
		for _, item := range selected {
			uncertain = uncertain || item.EndAt == nil || item.Completion != nil && item.Completion.Status == "unknown" || item.Submitted == "unknown"
		}
	}

	return selected, uncertain
}
