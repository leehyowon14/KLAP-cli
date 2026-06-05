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

func TestHyperlinkURLs(t *testing.T) {
	got := hyperlinkURLs("링크: https://example.com/path.")
	want := "링크: \x1b]8;;https://example.com/path\x1b\\https://example.com/path\x1b]8;;\x1b\\."
	if got != want {
		t.Fatalf("hyperlinkURLs() = %q, want %q", got, want)
	}
}
