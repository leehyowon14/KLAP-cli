package app

import "testing"

func TestQueuedLectureIDValidation(t *testing.T) {
	a, _ := StableLectureID(CourseRef{TermValue: "2026,2", CourseID: "course"}, "lecture")
	b, _ := StableLectureID(CourseRef{TermValue: "2026,1", CourseID: "course"}, "lecture")
	if err := ValidateStableLectureIDs([]string{a}); err != nil {
		t.Fatal(err)
	}
	for _, ids := range [][]string{nil, {"1:2"}, {"lecture:v1:invalid"}, {a, b}, {"notice:v1:invalid"}} {
		if ValidateStableLectureIDs(ids) == nil {
			t.Fatalf("accepted %v", ids)
		}
	}
}
