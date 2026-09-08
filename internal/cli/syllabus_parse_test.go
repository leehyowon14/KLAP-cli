package cli

import (
	"testing"
)

func TestSyllabusOptions(t *testing.T) {
	opts, err := syllabusOptions([]string{"I040-3-3951-01", "--term", "2026-1", "--user", "20250000"})
	if err != nil {
		t.Fatalf("syllabusOptions() error = %v", err)
	}
	if opts.Selector != "I040-3-3951-01" || opts.TermValue != "2026-1" || opts.User.StudentID != "20250000" {
		t.Fatalf("syllabusOptions() = %+v", opts)
	}
	if _, err := syllabusOptions([]string{"--term"}); err == nil {
		t.Fatal("syllabusOptions() expected error")
	}
}
