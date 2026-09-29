package app

import "testing"

func TestCheckedStudentName(t *testing.T) {
	for _, tc := range []struct {
		expected, actual, name, want string
		bad                          bool
	}{
		{"123", "123", " 이효원 ", "이효원", false},
		{"123", "456", "다른 이름", "", true},
		{"123", "123", "  ", "", true},
		{"", "", "이름", "", true},
	} {
		got, err := checkedStudentName(tc.expected, tc.actual, tc.name)
		if got != tc.want || (err != nil) != tc.bad {
			t.Fatalf("got=%q err=%v", got, err)
		}
	}
}
