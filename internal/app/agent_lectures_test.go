package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"io"
	"net/http"
	"strings"
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

func TestSummaryAttendanceRequiresKnownIncompleteAndOpen(t *testing.T) {
	now := time.Now()
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)
	id, _ := StableLectureID(CourseRef{TermValue: "2026,2", CourseID: "c"}, "x")
	for _, tt := range []struct {
		progress string
		want     bool
	}{{"0", true}, {"100", false}, {"", false}} {
		rows := []LectureRow{{ID: id, Lecture: Lecture{ContentID: "x", Progress: tt.progress, StartAt: &start, EndAt: &end}}}
		items, _ := selectLectureItems(rows, AgentQuery{Intent: "overview"}, now, nil)
		if items[0].Attendable != tt.want {
			t.Fatalf("%s: %+v", tt.progress, items)
		}
	}
}

func TestLectureQueryPreservesSuccessOnCourseMismatch(t *testing.T) {
	previous := http.DefaultTransport
	defer func() { http.DefaultTransport = previous }()
	selected := ""
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{}`
		switch req.URL.Path {
		case "/std/cmn/frame/YearhakgiAtnlcSbjectList.do":
			body = `[{"value":"2026,2","subjList":[{"value":"good","name":"Good"},{"value":"bad","name":"Bad"}]}]`
		case "/std/lis/evltn/LctrumHomeStdInfo.do":
			var args map[string]string
			_ = json.NewDecoder(req.Body).Decode(&args)
			selected = args["selectSubj"]
		case "/std/lis/evltn/SelectOnlineCntntsStdList.do":
			if selected == "good" {
				body = `[{"subj":"good","lrnSn":"1","sbjt":"kept","ptime":"10","achivTime":"0"}]`
			} else {
				body = `[{"subj":"wrong","lrnSn":"2"}]`
			}
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	s := &Service{settingsStore: readSettingsStub{}, newKlasClient: klas.NewClient, sessions: &fakeSessionStore{loadSession: func(context.Context, string) (klas.Session, error) { return klas.Session{}, nil }}}
	r, e := s.QueryLectures(context.Background(), UserOption{StudentID: "student"}, AgentQuery{Intent: "overview"})
	if e != nil || len(r.Items) != 1 || r.Items[0].Title != "kept" || r.Coverage.Requested != 2 || r.Coverage.Succeeded != 1 || r.Coverage.Status != "partial" || r.Coverage.Failures[0].Reason != "course_mismatch" || r.Selection.MatchedTotal != nil {
		t.Fatalf("%+v %v", r, e)
	}
}

func TestNextOpeningSummaryPayloadBudget(t *testing.T) {
	now := time.Now()
	start := now.Add(time.Hour)
	end := start.Add(time.Hour)
	rows := []LectureRow{}
	for i := 0; i < 40; i++ {
		key := fmt.Sprintf("lecture-%d", i)
		id, _ := StableLectureID(CourseRef{TermValue: "2026,2", CourseID: "course"}, key)
		progress := "100"
		if i >= 36 {
			progress = "0"
		}
		rows = append(rows, LectureRow{ID: id, TermValue: "2026,2", CourseName: "자료구조", Lecture: Lecture{Title: "자료구조 강의", ContentID: key, PlayURL: "https://example.invalid/player/" + key, LearningSeq: key, FileID: "source", Progress: progress, StartAt: &start, EndAt: &end, ModuleTitle: "모듈 제목", WeekNo: "9"}})
	}
	selected, unknown := selectLectureItems(rows, AgentQuery{Intent: "next_opening"}, now, nil)
	if unknown || len(selected) != 4 {
		t.Fatalf("lost tied group %d %v", len(selected), unknown)
	}
	old, _ := json.Marshal(map[string]any{"kind": "klap_lectures", "data": rows})
	compact, _ := json.Marshal(map[string]any{"kind": "klap_lectures", "schema_version": 2, "data": map[string]any{"items": selected, "courses": map[string]string{"course": "자료구조"}}})
	before := 2 * len(old)
	after := len(compact) + 256
	reduction := 1 - float64(after)/float64(before)
	t.Logf("fixture response bytes: before=%d after=%d reduction=%.1f%%; selected=%d", before, after, reduction*100, len(selected))
	if reduction < 0.70 {
		t.Fatal("summary byte reduction below fixture target")
	}
}

func TestUncertainCompletionCannotOverrideKnownTimeScope(t *testing.T) {
	now := time.Now()
	pastStart := now.Add(-2 * time.Hour)
	pastEnd := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	end := now.Add(3 * time.Hour)
	later := now.Add(2 * time.Hour)
	ref := CourseRef{TermValue: "2026,2", CourseID: "c"}
	makeRow := func(key, progress string, start, finish *time.Time) LectureRow {
		id, _ := StableLectureID(ref, key)
		return LectureRow{ID: id, Lecture: Lecture{ContentID: key, Progress: progress, StartAt: start, EndAt: finish}}
	}
	expired := makeRow("expired", "", &pastStart, &pastEnd)
	no := false
	got, u := selectLectureItems([]LectureRow{expired}, AgentQuery{Intent: "unfinished", IncludeExpired: &no}, now, nil)
	if len(got) != 0 || u {
		t.Fatal("known expired leaked through unknown completion")
	}
	confirmed := makeRow("confirmed", "0", &future, &end)
	unknownLater := makeRow("later", "", &later, &end)
	got, u = selectLectureItems([]LectureRow{expired, confirmed, unknownLater}, AgentQuery{Intent: "next_opening"}, now, nil)
	if len(got) != 1 || u || got[0].ID != confirmed.ID {
		t.Fatalf("irrelevant uncertainty: %+v %v", got, u)
	}
	earlier := now.Add(30 * time.Minute)
	unknownEarlier := makeRow("earlier", "", &earlier, &end)
	got, u = selectLectureItems([]LectureRow{unknownEarlier, confirmed}, AgentQuery{Intent: "next_opening"}, now, nil)
	if len(got) != 2 || !u {
		t.Fatalf("known fallback omitted: %+v %v", got, u)
	}
}
