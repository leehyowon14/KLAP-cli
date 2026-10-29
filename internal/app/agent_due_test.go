package app

import (
	"context"
	"testing"
)

func TestAgentDueSourceFailuresRemainPartial(t *testing.T) {
	s, _ := aggregateSessionFixture(t)
	r, e := s.QueryDue(context.Background(), UserOption{}, AgentQuery{Intent: "next_deadline"}, []string{"assignment", "lecture", "academic"})
	if e != nil || r.Coverage.Status != "partial" || r.Selection.Status != "uncertain" || r.Selection.MatchedTotal != nil || len(r.Sources) != 3 {
		t.Fatalf("%+v %v", r, e)
	}
}
