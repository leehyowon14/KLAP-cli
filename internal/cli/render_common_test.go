package cli

import (
	"testing"
)

func TestHyperlinkURLs(t *testing.T) {
	got := hyperlinkURLs("링크: https://example.com/path.")
	want := "링크: \x1b]8;;https://example.com/path\x1b\\https://example.com/path\x1b]8;;\x1b\\."
	if got != want {
		t.Fatalf("hyperlinkURLs() = %q, want %q", got, want)
	}
}
