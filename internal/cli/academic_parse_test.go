package cli

import (
	"testing"
)

func TestAcademicYearFlag(t *testing.T) {
	got, err := academicYearFlag([]string{"--year", "2026"})
	if err != nil {
		t.Fatalf("academicYearFlag() error = %v", err)
	}
	if got != "2026" {
		t.Fatalf("academicYearFlag() = %q", got)
	}
	if _, err := academicYearFlag([]string{"--year", "26"}); err == nil {
		t.Fatal("academicYearFlag() expected error for invalid year")
	}
}
