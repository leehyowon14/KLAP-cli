package app

import (
	"context"
	"testing"
)

func TestAgentSearchReportsFieldsAndPartialSources(t *testing.T) {
	s, _ := aggregateSessionFixture(t)
	r, e := s.QuerySearch(context.Background(), UserOption{}, AgentQuery{Intent: "search"}, "lecture", []string{"assignment", "lecture"})
	if e != nil || r.Coverage.Status != "partial" || r.Selection.MatchedTotal != nil || len(r.SearchedFields) != 2 {
		t.Fatalf("%+v %v", r, e)
	}
	if _, e := s.QuerySearch(context.Background(), UserOption{}, AgentQuery{Intent: "search"}, " ", nil); e == nil {
		t.Fatal("empty query accepted")
	}
}
