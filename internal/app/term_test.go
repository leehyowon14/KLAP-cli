package app

import (
	"testing"
	"time"
)

func TestSelectTermRow(t *testing.T) {
	rows := []TermRow{
		{Index: 1, Term: Term{Label: "2026년도 1학기", Value: "2026,1"}},
		{Index: 2, Term: Term{Label: "2025년도 겨울학기", Value: "2025,4"}},
	}

	selected, err := selectTermRow(rows, "2")
	if err != nil {
		t.Fatalf("selectTermRow() by number error = %v", err)
	}
	if selected.Term.Value != "2025,4" {
		t.Fatalf("selectTermRow() by number = %+v", selected)
	}

	selected, err = selectTermRow(rows, "2026,1")
	if err != nil {
		t.Fatalf("selectTermRow() by value error = %v", err)
	}
	if selected.Term.Label != "2026년도 1학기" {
		t.Fatalf("selectTermRow() by value = %+v", selected)
	}

	selected, err = selectTermRow(rows, "겨울")
	if err != nil {
		t.Fatalf("selectTermRow() by label error = %v", err)
	}
	if selected.Term.Value != "2025,4" {
		t.Fatalf("selectTermRow() by label = %+v", selected)
	}
}

func TestNormalizeTermValue(t *testing.T) {
	got, err := normalizeTermValue("2026-1")
	if err != nil {
		t.Fatalf("normalizeTermValue() error = %v", err)
	}
	if got != "2026,1" {
		t.Fatalf("normalizeTermValue() = %q", got)
	}
	if _, err := normalizeTermValue("2026-5"); err == nil {
		t.Fatal("normalizeTermValue() expected error for invalid semester")
	}
}

func TestCurrentAcademicTermValue(t *testing.T) {
	tests := []struct {
		now  time.Time
		want string
	}{
		{now: time.Date(2026, time.June, 7, 0, 0, 0, 0, time.Local), want: "2026,1"},
		{now: time.Date(2026, time.July, 1, 0, 0, 0, 0, time.Local), want: "2026,3"},
		{now: time.Date(2026, time.December, 20, 0, 0, 0, 0, time.Local), want: "2026,4"},
		{now: time.Date(2027, time.January, 10, 0, 0, 0, 0, time.Local), want: "2026,4"},
	}
	for _, tt := range tests {
		if got := currentAcademicTermValue(tt.now); got != tt.want {
			t.Fatalf("currentAcademicTermValue(%s) = %q, want %q", tt.now, got, tt.want)
		}
	}
}

func TestTermLabel(t *testing.T) {
	if got := termLabel("2026,3"); got != "2026년도 여름학기" {
		t.Fatalf("termLabel() = %q", got)
	}
	if got := termLabel("2026,4"); got != "2026년도 겨울학기" {
		t.Fatalf("termLabel() = %q", got)
	}
}
