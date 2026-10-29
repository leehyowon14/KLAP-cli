package app

import (
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"testing"
	"time"
)

func TestAssignmentIntentUnknownAndBoundaries(t *testing.T) {
	now := time.Now()
	w := QueryWindow{From: now, Until: now.Add(time.Hour)}
	for _, tt := range []struct {
		v              klas.Assignment
		intent         string
		match, unknown bool
	}{
		{klas.Assignment{SubmissionStatus: "pending", DueAt: &now}, "due_in_period", true, false},
		{klas.Assignment{SubmissionStatus: "pending", DueAt: &w.Until}, "due_in_period", false, false},
		{klas.Assignment{SubmissionStatus: "unknown", DueAt: &w.Until}, "due_in_period", false, false},
		{klas.Assignment{SubmissionStatus: "unknown", DueAt: &w.Until}, "overdue", false, false},
		{klas.Assignment{SubmissionStatus: "submitted", DueAt: &now}, "due_in_period", false, false},
		{klas.Assignment{SubmissionStatus: "unknown"}, "pending", true, true},
		{klas.Assignment{SubmissionStatus: "pending"}, "overdue", false, true},
	} {
		m, u := assignmentMatches(tt.v, AgentQuery{Intent: tt.intent}, now, &w)
		if m != tt.match || u != tt.unknown {
			t.Fatalf("%+v got %v %v", tt, m, u)
		}
	}
}
