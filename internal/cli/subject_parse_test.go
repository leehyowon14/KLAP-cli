package cli

import (
	"testing"
)

func TestSubjectSearchOptions(t *testing.T) {
	opts, err := subjectSearchOptions([]string{"--name", "컴퓨터그래픽스", "--professor", "김동준", "--term", "2026-1"})
	if err != nil {
		t.Fatalf("subjectSearchOptions() error = %v", err)
	}
	if opts.Name != "컴퓨터그래픽스" || opts.Professor != "김동준" || opts.TermValue != "2026-1" {
		t.Fatalf("subjectSearchOptions() = %+v", opts)
	}
	if _, err := subjectSearchOptions([]string{}); err == nil {
		t.Fatal("subjectSearchOptions() expected error")
	}
	if _, err := subjectSearchOptions([]string{"--name", "컴퓨터그래픽스", "--user", "20250000"}); err == nil {
		t.Fatal("subjectSearchOptions() expected error for --user")
	}
}
