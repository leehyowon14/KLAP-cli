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

func TestNativeViewerEligibilityAndStableIdentity(t *testing.T) {
	now := time.Now()
	start, end := now.Add(-time.Hour), now.Add(time.Hour)
	l := Lecture{WeekNo: "2", WeeklySeq: "3", Title: "native", StartAt: &start, EndAt: &end}
	oldID := lectureResourceKey(l)
	l.ViewerSupported = true
	if lectureResourceKey(l) != oldID {
		t.Fatal("native support changed existing identity")
	}
	row := LectureRow{ID: oldID, Lecture: l}
	if err := ValidateLectureAttendance(row, now); err != nil {
		t.Fatal(err)
	}
	if err := ValidateLectureAttendance(row, end.Add(time.Second)); err == nil {
		t.Fatal("expired lecture accepted")
	}
	if err := ValidateLectureAttendance(row, start.Add(-time.Second)); err == nil {
		t.Fatal("future lecture accepted")
	}
	row.Lecture.Progress = "100"
	if err := ValidateLectureAttendance(row, now); err == nil {
		t.Fatal("completed lecture accepted")
	}
	row.Lecture.Progress = "0"
	row.Lecture.ViewerSupported = false
	if err := ValidateLectureAttendance(row, now); err == nil {
		t.Fatal("unsupported lecture accepted")
	}
	l.WeekNo = ""
	l.WeeklySeq = ""
	l.FileID = ""
	l.ViewerSupported = false
	oldID = lectureResourceKey(l)
	l.ViewerSupported = true
	if lectureResourceKey(l) != oldID {
		t.Fatal("metadata fallback identity changed")
	}
}
