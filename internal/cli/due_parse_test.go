package cli

import (
	"testing"
)

func TestDueOptions(t *testing.T) {
	opts, err := dueOptions([]string{"--week", "--refresh", "--user", "20260000"})
	if err != nil {
		t.Fatalf("dueOptions() error = %v", err)
	}
	if opts.Days != 7 || !opts.Refresh || opts.User.StudentID != "20260000" {
		t.Fatalf("dueOptions() = %+v", opts)
	}
	opts, err = dueOptions([]string{"--days", "30"})
	if err != nil {
		t.Fatalf("dueOptions() days error = %v", err)
	}
	if opts.Days != 30 {
		t.Fatalf("dueOptions() days = %d", opts.Days)
	}
}
