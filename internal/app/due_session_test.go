package app

import (
	"context"
	"testing"
)

func TestDueSharesSessionAndPreservesPartialErrors(t *testing.T) {
	s, counts := aggregateSessionFixture(t)
	for run := 1; run <= 2; run++ {
		result, err := s.Due(context.Background(), DueOptions{Refresh: true})
		if err != nil || len(result.Errors) != 2 {
			t.Fatalf("error=%v result=%+v", err, result)
		}
		assertAggregateSessionCounts(t, counts, run)
	}
}
