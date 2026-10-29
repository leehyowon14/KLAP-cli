package app

import (
	"testing"
	"time"
)

func TestLectureIntentSelection(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	start := now.Add(time.Hour)
	end := start.Add(time.Hour)
	past := now.Add(-time.Hour)
	rows := []LectureRow{}
	for _, key := range []string{"a", "b", "c", "d"} {
		id, _ := StableLectureID(CourseRef{TermValue: "2026,2", CourseID: "course"}, key)
		rows = append(rows, LectureRow{ID: id, Lecture: Lecture{Title: "same title", ContentID: key, Progress: "0", StartAt: &start, EndAt: &end}})
	}
	q := AgentQuery{Intent: "next_opening"}
	items, u := selectLectureItems(rows, q, now, nil)
	if len(items) != 4 || u {
		t.Fatalf("%d uncertain %v", len(items), u)
	}
	expired := rows[0]
	expired.ID += "x"
	expired.Lecture.StartAt = &past
	expired.Lecture.EndAt = &past
	rows = append(rows, expired)
	for _, include := range []bool{false, true} {
		q = AgentQuery{Intent: "unfinished", IncludeExpired: &include}
		items, u = selectLectureItems(rows, q, now, nil)
		want := 4
		if include {
			want = 5
		}
		if len(items) != want || u {
			t.Fatalf("include=%v got %d %v", include, len(items), u)
		}
	}
	rows[0].Lecture.Progress = "bad"
	rows[0].Lecture.EndAt = nil
	items, u = selectLectureItems(rows, AgentQuery{Intent: "next_opening"}, now, nil)
	if !u || len(items) != 4 {
		t.Fatalf("unknown lost: %d %v", len(items), u)
	}
}
func TestLectureQueryRejectsIncompatibleArgs(t *testing.T) {
	yes := true
	for _, q := range []AgentQuery{{Intent: "unfinished"}, {Intent: "overview", IncludeExpired: &yes}, {Intent: "due_in_period"}, {Intent: "next_opening", Period: &QueryPeriod{Name: "today"}}, {Intent: "bad"}} {
		if _, err := validateLectureQuery(q, time.Now()); err == nil {
			t.Fatalf("accepted %+v", q)
		}
	}
}
func TestExactQueryCourses(t *testing.T) {
	term := Term{Courses: []Course{{Value: "abc", Name: "name"}}}
	for _, ids := range [][]string{{"name"}, {"1"}, {""}} {
		if _, e := exactQueryCourses(term, ids); e == nil {
			t.Fatalf("accepted %v", ids)
		}
	}
	v, e := exactQueryCourses(term, []string{"abc", "abc"})
	if e != nil || len(v) != 1 {
		t.Fatalf("%v %v", v, e)
	}
}
