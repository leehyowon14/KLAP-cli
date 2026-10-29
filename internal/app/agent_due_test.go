package app

import (
	"context"
	"testing"
	"time"
)

func TestAgentDueSourceFailuresRemainPartial(t *testing.T) {
	s, _ := aggregateSessionFixture(t)
	r, e := s.QueryDue(context.Background(), UserOption{}, AgentQuery{Intent: "next_deadline"}, []string{"assignment", "lecture", "academic"})
	if e != nil || r.Coverage.Status != "partial" || r.Selection.Status != "uncertain" || r.Selection.MatchedTotal != nil || len(r.Sources) != 3 {
		t.Fatalf("%+v %v", r, e)
	}
}

func TestNextDeadlinePreservesConfirmedFallback(t *testing.T) {
	now := time.Now()
	earlier := now.Add(time.Hour)
	later := now.Add(2 * time.Hour)
	unknown := AgentItem{ID: "unknown", EndAt: &earlier, Completion: &Completion{Status: "unknown"}}
	confirmed := AgentItem{ID: "confirmed", EndAt: &later, Completion: &Completion{Status: "incomplete"}}
	q := AgentQuery{Intent: "next_deadline"}
	items, u := selectDueCandidates([]AgentItem{unknown, confirmed}, q, now, nil)
	if len(items) != 2 || !u {
		t.Fatalf("lost fallback: %+v %v", items, u)
	}
	unknown.EndAt = &later
	confirmed.EndAt = &earlier
	items, u = selectDueCandidates([]AgentItem{unknown, confirmed}, q, now, nil)
	if len(items) != 1 || u {
		t.Fatalf("irrelevant uncertainty: %+v %v", items, u)
	}
	ties := []AgentItem{}
	for _, id := range []string{"a", "b", "c", "d"} {
		item := confirmed
		item.ID = id
		ties = append(ties, item)
	}
	items, u = selectDueCandidates(ties, q, now, nil)
	if len(items) != 4 || u {
		t.Fatalf("ties dropped: %d %v", len(items), u)
	}
}
