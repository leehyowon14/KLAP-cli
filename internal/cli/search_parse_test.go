package cli

import (
	"testing"
)

func TestSearchOptions(t *testing.T) {
	opts, err := searchOptions([]string{"기말", "--type", "assignment", "--refresh", "--user", "20260000"})
	if err != nil {
		t.Fatalf("searchOptions() error = %v", err)
	}
	if opts.Query != "기말" || opts.Type != "assignment" || !opts.Refresh || opts.User.StudentID != "20260000" {
		t.Fatalf("searchOptions() = %+v", opts)
	}
	if _, err := searchOptions([]string{}); err == nil {
		t.Fatal("searchOptions() expected error")
	}
}
