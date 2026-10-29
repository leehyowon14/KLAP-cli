package app

import (
	"testing"
	"time"
)

func TestLectureCompletionEvidence(t *testing.T) {
	for _, tt := range []struct {
		l    Lecture
		want string
	}{
		{Lecture{Progress: "100"}, "completed"}, {Lecture{Progress: "99.9"}, "incomplete"}, {Lecture{Progress: "0"}, "incomplete"},
		{Lecture{}, "unknown"}, {Lecture{Progress: "NaN"}, "unknown"}, {Lecture{Progress: "101"}, "unknown"}, {Lecture{Progress: "-1"}, "unknown"},
		{Lecture{LearningSeq: "a", AchievedTime: "10", RequiredTime: "10"}, "completed"}, {Lecture{LearningSeq: "a", AchievedTime: "9", RequiredTime: "10"}, "incomplete"},
		{Lecture{LearningSeq: "a", AchievedTime: "0", RequiredTime: "0", Progress: "100"}, "unknown"},
		{Lecture{LearningSeq: "a", AchievedTime: "10"}, "unknown"},
	} {
		if got := LectureCompletion(tt.l); got.Status != tt.want {
			t.Fatalf("%+v: %+v want %s", tt.l, got, tt.want)
		}
	}
}
func TestQueryPeriodBoundaries(t *testing.T) {
	now := time.Date(2026, 10, 28, 19, 0, 0, 0, time.FixedZone("KST", 32400))
	w, err := (QueryPeriod{Name: "this_week"}).Resolve(now)
	if err != nil || w.From.Weekday() != time.Monday || w.From.Hour() != 0 || w.Until.Sub(w.From) != 7*24*time.Hour {
		t.Fatalf("%+v %v", w, err)
	}
	if !w.Contains(w.From) || w.Contains(w.Until) {
		t.Fatal("half-open bounds")
	}
	for _, p := range []QueryPeriod{{Name: "bad"}, {Name: "today", From: "2026-01-01"}, {Name: "custom", From: "2026-01-02", Until: "2026-01-01"}, {Name: "custom", From: "bad", Until: "2026-01-01"}} {
		if _, e := p.Resolve(now); e == nil {
			t.Fatalf("accepted %+v", p)
		}
	}
	w, err = (QueryPeriod{Name: "custom", From: "2026-10-28", Until: "2026-10-29"}).Resolve(now)
	if err != nil || !w.From.Equal(time.Date(2026, 10, 27, 15, 0, 0, 0, time.UTC)) {
		t.Fatalf("%+v %v", w, err)
	}
}
