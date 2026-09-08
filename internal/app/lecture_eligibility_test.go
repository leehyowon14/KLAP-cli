package app

import (
	"testing"
	"time"
)

func TestValidateLectureAttendance(t *testing.T) {
	now := time.Now()
	start, end := now.Add(-time.Hour), now.Add(time.Hour)
	for _, tc := range []struct {
		name   string
		mutate func(*LectureRow)
		valid  bool
	}{
		{"video", func(r *LectureRow) {}, true},
		{"activity", func(r *LectureRow) { r.Lecture.ContentID = ""; r.Lecture.LearningSeq = "1" }, true},
		{"unknown start", func(r *LectureRow) { r.Lecture.StartAt = nil }, false},
		{"unknown end", func(r *LectureRow) { r.Lecture.EndAt = nil }, false},
		{"future", func(r *LectureRow) { r.Lecture.StartAt = &end }, false},
		{"expired", func(r *LectureRow) { r.Lecture.EndAt = &start }, false},
		{"complete", func(r *LectureRow) { r.Lecture.Progress = "100" }, false},
		{"complete activity", func(r *LectureRow) {
			r.Lecture.ContentID = ""
			r.Lecture.LearningSeq = "1"
			r.Lecture.AchievedTime = "10"
			r.Lecture.RequiredTime = "10"
		}, false},
		{"boundary", func(r *LectureRow) { r.Lecture.StartAt = &now; r.Lecture.EndAt = &now }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := LectureRow{ID: "1:a", Lecture: Lecture{ContentID: "a", StartAt: &start, EndAt: &end}}
			tc.mutate(&r)
			if err := ValidateLectureAttendance(r, now); (err == nil) != tc.valid {
				t.Fatalf("error=%v valid=%v", err, tc.valid)
			}
		})
	}
}
