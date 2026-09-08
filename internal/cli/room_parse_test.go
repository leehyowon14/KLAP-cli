package cli

import (
	"testing"
)

func TestRoomQueryOptions(t *testing.T) {
	opts, err := roomQueryOptions([]string{"연", "102", "--term", "2026-1", "--building", "연구", "--refresh", "--user", "20260000"})
	if err != nil {
		t.Fatalf("roomQueryOptions() error = %v", err)
	}
	if opts.Room != "연 102" || opts.TermValue != "2026-1" || opts.Building != "연구" || !opts.Refresh || opts.User.StudentID != "20260000" {
		t.Fatalf("roomQueryOptions() = %+v", opts)
	}
	if _, err := roomQueryOptions([]string{"--term", "2026-1"}); err == nil {
		t.Fatal("roomQueryOptions() expected missing room error")
	}
}

func TestRoomIndexOptions(t *testing.T) {
	opts, err := roomIndexOptions([]string{"--term", "2026-1", "--building", "비마", "--user", "20260000"})
	if err != nil {
		t.Fatalf("roomIndexOptions() error = %v", err)
	}
	if opts.TermValue != "2026-1" || opts.Building != "비마" || opts.User.StudentID != "20260000" {
		t.Fatalf("roomIndexOptions() = %+v", opts)
	}
	if _, err := roomIndexOptions([]string{"비마관"}); err == nil {
		t.Fatal("roomIndexOptions() expected unknown positional error")
	}
}

func TestRoomAvailableOptions(t *testing.T) {
	opts, err := roomAvailableOptions([]string{"--day", "금요일", "--duration", "1-3", "--term", "2026-1", "--building", "새빛", "--refresh", "--user", "20260000"})
	if err != nil {
		t.Fatalf("roomAvailableOptions() error = %v", err)
	}
	if opts.Day != "금요일" || opts.Duration != "1-3" || opts.TermValue != "2026-1" || opts.Building != "새빛" || !opts.Refresh || opts.User.StudentID != "20260000" {
		t.Fatalf("roomAvailableOptions() = %+v", opts)
	}
	if _, err := roomAvailableOptions([]string{"--day", "금"}); err == nil {
		t.Fatal("roomAvailableOptions() expected missing duration error")
	}
}
