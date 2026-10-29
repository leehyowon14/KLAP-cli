package klas

import "testing"

func TestSubmissionStatusPreservesUnknown(t *testing.T) {
	for _, tt := range []struct{ raw, want string }{{"Y", "submitted"}, {"N", "pending"}, {"", "unknown"}, {"bad", "unknown"}} {
		if got := submissionStatus(tt.raw); got != tt.want {
			t.Fatalf("%s: %s", tt.raw, got)
		}
	}
}
