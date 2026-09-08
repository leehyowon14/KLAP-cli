package klas

import (
	"testing"
)

func TestSplitYearHakgi(t *testing.T) {
	year, hakgi := splitYearHakgi("2026,1")
	if year != "2026" || hakgi != "1" {
		t.Fatalf("splitYearHakgi() = %q, %q", year, hakgi)
	}

	year, hakgi = splitYearHakgi("2026-2")
	if year != "2026" || hakgi != "2" {
		t.Fatalf("splitYearHakgi() hyphen = %q, %q", year, hakgi)
	}
}

func TestNormalizeTermLabelSeasonSemesters(t *testing.T) {
	if got := normalizeTermLabel("2026년도 3학기", "2026,3"); got != "2026년도 여름학기" {
		t.Fatalf("normalizeTermLabel() summer = %q", got)
	}
	if got := normalizeTermLabel("", "2026,4"); got != "2026년도 겨울학기" {
		t.Fatalf("normalizeTermLabel() winter = %q", got)
	}
	if got := normalizeTermLabel("2026년도 1학기", "2026,1"); got != "2026년도 1학기" {
		t.Fatalf("normalizeTermLabel() regular = %q", got)
	}
}
