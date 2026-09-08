package app

import (
	"context"
	"testing"
)

func TestSearchSharesSessionAndPreservesPartialErrors(t *testing.T) {
	s, counts := aggregateSessionFixture(t)
	for run := 1; run <= 2; run++ {
		result, err := s.Search(context.Background(), SearchOptions{Query: "course", Refresh: true})
		if err != nil || len(result.Errors) != 3 || len(result.Courses) != 1 {
			t.Fatalf("error=%v result=%+v", err, result)
		}
		assertAggregateSessionCounts(t, counts, run)
	}
}
