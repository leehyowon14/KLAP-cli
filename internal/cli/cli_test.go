package cli

import "testing"

func TestCourseFilterRequiresValue(t *testing.T) {
	_, err := courseFilter([]string{"--course"})
	if err == nil {
		t.Fatal("courseFilter() expected error")
	}
}

func TestUserFlag(t *testing.T) {
	if got := userFlag([]string{"--user", "20260000"}); got != "20260000" {
		t.Fatalf("userFlag() = %q", got)
	}
}
